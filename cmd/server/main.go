package main

import (
	"context"
	"log"
	"time"

	"crypto-tracker-trader/internal/api"
	"crypto-tracker-trader/internal/client/coingecko"
	"crypto-tracker-trader/internal/client/defi/uniswap"
	"crypto-tracker-trader/internal/config"
	"crypto-tracker-trader/internal/service"
	"crypto-tracker-trader/internal/store"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if cfg.MasterDatabaseURL == "" {
		log.Fatal("MASTER_DATABASE_URL environment variable is not set")
	}
	if cfg.EthereumNodeURL == "" {
		log.Fatal("ETHEREUM_NODE_URL environment variable is not set")
	}
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}
	if cfg.EncryptionKey == "" {
		log.Fatal("ENCRYPTION_KEY environment variable is not set (must be 32-byte hex, 64 chars)")
	}

	r := gin.Default()

	// Shared DB pool.
	dbPool, err := pgxpool.Connect(context.Background(), cfg.MasterDatabaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbPool.Close()

	// Stores.
	portfolioStore := store.NewPortfolioStore(dbPool)
	userStore := store.NewUserStore(dbPool)
	walletStore := store.NewWalletStore(dbPool)
	priceStore := store.NewPriceStore(dbPool)
	exchangeStore := store.NewExchangeStore(dbPool)
	defiStore := store.NewDefiStore(dbPool)

	// Ethereum client.
	ethClient, err := ethclient.Dial(cfg.EthereumNodeURL)
	if err != nil {
		log.Fatalf("Failed to connect to Ethereum node: %v", err)
	}
	defer ethClient.Close()

	// Services.
	portfolioService := service.NewPortfolioService(portfolioStore)
	blockchainService := service.NewBlockchainDataFetcherService(ethClient, portfolioStore)
	userService := service.NewUserService(userStore)
	walletService := service.NewWalletService(walletStore)
	priceService := service.NewPriceService(coingecko.New(), priceStore, "coingecko")
	exchangeService := service.NewExchangeService(exchangeStore, cfg.EncryptionKey)
	uniswapClient := uniswap.New(ethClient)
	defiService := service.NewDefiSyncService(defiStore, walletStore, uniswapClient)
	portfolioAggregator := service.NewPortfolioAggregationService(walletStore, priceStore, exchangeStore, defiStore)

	// Background sync context (cancelled on exit).
	syncCtx, stopSync := context.WithCancel(context.Background())
	defer stopSync()

	priceService.StartSync(syncCtx, cfg.PriceSymbols, time.Duration(cfg.PriceSyncIntervalS)*time.Second)
	exchangeService.StartSync(syncCtx, time.Duration(cfg.ExchangeSyncIntervalS)*time.Second)
	defiService.StartSync(syncCtx, time.Duration(cfg.DefiSyncIntervalS)*time.Second)

	// API.
	apiHandler := api.NewAPI(api.APIConfig{
		PortfolioService:    portfolioService,
		BlockchainService:   blockchainService,
		UserService:         userService,
		WalletService:       walletService,
		PriceService:        priceService,
		ExchangeService:     exchangeService,
		DefiService:         defiService,
		PortfolioAggregator: portfolioAggregator,
		JWTSecret:           cfg.JWTSecret,
		JWTExpiryHours:      cfg.JWTExpiryHours,
		PriceSymbols:        cfg.PriceSymbols,
	})
	apiHandler.RegisterRoutes(r)

	addr := ":" + cfg.Port
	log.Printf("Starting server on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}

package main

import (
	"context"
	"log"

	"crypto-tracker-trader/internal/api"
	"crypto-tracker-trader/internal/config"
	"crypto-tracker-trader/internal/service"
	"crypto-tracker-trader/internal/store"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4"
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

	r := gin.Default()

	// Establish a single database connection
	dbConn, err := pgx.Connect(context.Background(), cfg.MasterDatabaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	if err := dbConn.Close(context.Background()); err != nil {
		log.Printf("Error closing database connection: %v", err)
	}

	// Create the store, service, and API
	portfolioStore := store.NewPortfolioStore(dbConn)
	defer portfolioStore.Close()

	userStore := store.NewUserStore(dbConn)
	defer userStore.Close()

	ethClient, err := ethclient.Dial(cfg.EthereumNodeURL)
	if err != nil {
		log.Fatalf("Failed to connect to Ethereum node: %v", err)
	}
	defer ethClient.Close()

	portfolioService := service.NewPortfolioService(portfolioStore)
	blockchainDataFetcherService := service.NewBlockchainDataFetcherService(ethClient, portfolioStore)
	userService := service.NewUserService(userStore) // userService implements service.UserManager
	apiHandler := api.NewAPI(portfolioService, blockchainDataFetcherService, userService)

	// Register the routes
	apiHandler.RegisterRoutes(r)

	addr := ":" + cfg.Port
	log.Printf("Starting server on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}

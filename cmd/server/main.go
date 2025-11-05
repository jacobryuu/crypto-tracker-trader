package main

import (
    "log"

    "crypto-tracker-trader/internal/api"
    "crypto-tracker-trader/internal/config"
    "crypto-tracker-trader/internal/service"
    "crypto-tracker-trader/internal/store"
    "github.com/gin-gonic/gin"
    "github.com/ethereum/go-ethereum/ethclient"
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

    // Create the store, service, and API
    portfolioStore := store.NewPortfolioStore(cfg.MasterDatabaseURL)
    defer portfolioStore.Close()

    ethClient, err := ethclient.Dial(cfg.EthereumNodeURL)
    if err != nil {
        log.Fatalf("Failed to connect to Ethereum node: %v", err)
    }
    defer ethClient.Close()

    portfolioService := service.NewPortfolioService(portfolioStore)
    blockchainDataFetcherService := service.NewBlockchainDataFetcherService(ethClient, portfolioStore)
    apiHandler := api.NewAPI(portfolioService, blockchainDataFetcherService)

    // Register the routes
    apiHandler.RegisterRoutes(r)

    addr := ":" + cfg.Port
    log.Printf("Starting server on %s", addr)
    if err := r.Run(addr); err != nil {
        log.Fatalf("server exited: %v", err)
    }
}

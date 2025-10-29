package main

import (
    "log"

    "crypto-tracker-trader/internal/api"
    "crypto-tracker-trader/internal/config"
    "crypto-tracker-trader/internal/service"
    "crypto-tracker-trader/internal/store"
    "github.com/gin-gonic/gin"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("failed to load config: %v", err)
    }

    if cfg.DatabaseURL == "" {
        log.Fatal("DATABASE_URL environment variable is not set")
    }

    r := gin.Default()

    // Create the store, service, and API
    portfolioStore := store.NewPortfolioStore(cfg.DatabaseURL)
    defer portfolioStore.Close()

    portfolioService := service.NewPortfolioService(portfolioStore)
    apiHandler := api.NewAPI(portfolioService)

    // Register the routes
    apiHandler.RegisterRoutes(r)

    addr := ":" + cfg.Port
    log.Printf("Starting server on %s", addr)
    if err := r.Run(addr); err != nil {
        log.Fatalf("server exited: %v", err)
    }
}

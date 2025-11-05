package api

import (
    "log"
    "net/http"

    "crypto-tracker-trader/internal/service"
    "github.com/gin-gonic/gin"
    "github.com/ethereum/go-ethereum/common"
)

type API struct {
    portfolioService service.PortfolioManager
    blockchainDataFetcherService service.BlockchainDataFetcher
}

func NewAPI(portfolioService service.PortfolioManager, blockchainDataFetcherService service.BlockchainDataFetcher) *API {
    return &API{
        portfolioService: portfolioService,
        blockchainDataFetcherService: blockchainDataFetcherService,
    }
}

// FetchETHBalanceAndSave handles fetching ETH balance from blockchain and saving it to DB.
func (a *API) FetchETHBalanceAndSave(c *gin.Context) {
    addressParam := c.Param("address")
    if !common.IsHexAddress(addressParam) {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Ethereum address format"})
        return
    }
    address := common.HexToAddress(addressParam)

    err := a.blockchainDataFetcherService.FetchAndSaveETHBalance(c.Request.Context(), address)
    if err != nil {
        log.Printf("Error fetching and saving ETH balance for address %s: %v", address.Hex(), err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch and save ETH balance"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "ETH balance fetched and saved successfully"})
}
func (a *API) RegisterRoutes(r *gin.Engine) {
    r.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status":"ok"})
    })

    v1 := r.Group("/api/v1")
    {
        portfolio := v1.Group("/portfolio")
        {
            portfolio.GET("/history", a.GetPortfolioHistory)
        }
        blockchain := v1.Group("/blockchain")
        {
            blockchain.POST("/fetch-eth-balance/:address", a.FetchETHBalanceAndSave)
        }
    }
}

func (a *API) GetPortfolioHistory(c *gin.Context) {
    history, err := a.portfolioService.GetPortfolioHistory()
    if err != nil {
        // Log the detailed error for internal monitoring
        log.Printf("Error getting portfolio history: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
        return
    }
    c.JSON(http.StatusOK, history)
}

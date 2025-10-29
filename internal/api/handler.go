package api

import (
    "net/http"

    "crypto-tracker-trader/internal/service"
    "github.com/gin-gonic/gin"
)

type API struct {
    portfolioService *service.PortfolioService
}

func NewAPI(portfolioService *service.PortfolioService) *API {
    return &API{
        portfolioService: portfolioService,
    }
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
    }
}

func (a *API) GetPortfolioHistory(c *gin.Context) {
    history, err := a.portfolioService.GetPortfolioHistory()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, history)
}

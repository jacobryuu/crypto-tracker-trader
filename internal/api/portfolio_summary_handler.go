package api

import (
	"log"
	"net/http"

	"crypto-tracker-trader/internal/api/middleware"

	"github.com/gin-gonic/gin"
)

// GetPortfolioSummary returns the unified portfolio view for the authenticated user.
func (a *API) GetPortfolioSummary(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	summary, err := a.portfolioAggregator.GetSummary(c.Request.Context(), userID)
	if err != nil {
		log.Printf("GetPortfolioSummary user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compute portfolio summary"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

// GetPortfolioAllocation returns only the per-symbol breakdown (subset of summary).
func (a *API) GetPortfolioAllocation(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	summary, err := a.portfolioAggregator.GetSummary(c.Request.Context(), userID)
	if err != nil {
		log.Printf("GetPortfolioAllocation user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compute allocation"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"total_usd": summary.TotalUSD,
		"by_symbol": summary.BySymbol,
	})
}

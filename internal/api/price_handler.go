package api

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"crypto-tracker-trader/internal/api/middleware"
	"crypto-tracker-trader/internal/store"

	"github.com/gin-gonic/gin"
)

// GetLatestPrice returns the most recently stored USD price for a symbol.
// GET /api/v1/prices/:symbol
func (a *API) GetLatestPrice(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is required"})
		return
	}

	price, err := a.priceService.GetLatestPrice(c.Request.Context(), symbol)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "no price data found for symbol"})
			return
		}
		log.Printf("Error fetching latest price for %s: %v", symbol, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve price"})
		return
	}
	c.JSON(http.StatusOK, price)
}

// GetPriceHistory returns recent price records for a symbol.
// GET /api/v1/prices/:symbol/history?limit=N
func (a *API) GetPriceHistory(c *gin.Context) {
	symbol := c.Param("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is required"})
		return
	}

	limit := 100
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}

	history, err := a.priceService.GetPriceHistory(c.Request.Context(), symbol, limit)
	if err != nil {
		log.Printf("Error fetching price history for %s: %v", symbol, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve price history"})
		return
	}
	c.JSON(http.StatusOK, history)
}

// TriggerPriceSync manually triggers a price sync for the configured symbols.
// POST /api/v1/prices/sync  (protected, admin-like utility endpoint)
func (a *API) TriggerPriceSync(c *gin.Context) {
	_, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	go func() {
		if err := a.priceService.FetchAndSave(c.Request.Context(), a.priceSymbols); err != nil {
			log.Printf("Manual price sync error: %v", err)
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{"message": "price sync triggered"})
}

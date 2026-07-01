package api

import (
	"log"
	"net/http"
	"strconv"

	"crypto-tracker-trader/internal/api/middleware"
	"crypto-tracker-trader/internal/store"

	"github.com/gin-gonic/gin"
)

// AddExchangeCredential registers a new exchange API key pair for the authenticated user.
func (a *API) AddExchangeCredential(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	var req struct {
		Exchange  string `json:"exchange" binding:"required"`
		APIKey    string `json:"api_key" binding:"required"`
		APISecret string `json:"api_secret" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cred, err := a.exchangeService.AddCredential(c.Request.Context(), userID, req.Exchange, req.APIKey, req.APISecret)
	if err != nil {
		log.Printf("AddExchangeCredential: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, cred)
}

// ListExchangeCredentials returns all exchange credentials for the authenticated user.
func (a *API) ListExchangeCredentials(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	creds, err := a.exchangeService.GetCredentials(c.Request.Context(), userID)
	if err != nil {
		log.Printf("ListExchangeCredentials: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve credentials"})
		return
	}
	c.JSON(http.StatusOK, creds)
}

// DeleteExchangeCredential removes an exchange credential by ID.
func (a *API) DeleteExchangeCredential(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credential id"})
		return
	}

	if err := a.exchangeService.DeleteCredential(c.Request.Context(), id, userID); err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "credential not found"})
			return
		}
		log.Printf("DeleteExchangeCredential: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete credential"})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// SyncExchangeBalances triggers a manual balance sync for a specific credential.
func (a *API) SyncExchangeBalances(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credential id"})
		return
	}

	balances, err := a.exchangeService.SyncBalances(c.Request.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "credential not found"})
			return
		}
		log.Printf("SyncExchangeBalances: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sync failed"})
		return
	}
	c.JSON(http.StatusOK, balances)
}

// GetExchangeBalances returns the latest stored balances for the authenticated user.
func (a *API) GetExchangeBalances(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	balances, err := a.exchangeService.GetBalances(c.Request.Context(), userID)
	if err != nil {
		log.Printf("GetExchangeBalances: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve balances"})
		return
	}
	c.JSON(http.StatusOK, balances)
}

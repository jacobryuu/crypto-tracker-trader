package api

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"crypto-tracker-trader/internal/api/middleware"
	"crypto-tracker-trader/internal/store"

	"github.com/gin-gonic/gin"
)

// ListWallets returns all wallets belonging to the authenticated user.
func (a *API) ListWallets(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	wallets, err := a.walletService.GetWallets(userID)
	if err != nil {
		log.Printf("Error listing wallets for user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve wallets"})
		return
	}
	c.JSON(http.StatusOK, wallets)
}

// AddWallet registers a new wallet for the authenticated user.
func (a *API) AddWallet(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Chain   string `json:"chain"   binding:"required"`
		Address string `json:"address" binding:"required"`
		Label   string `json:"label"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wallet, err := a.walletService.AddWallet(userID, req.Chain, req.Address, req.Label)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "already registered") || strings.Contains(msg, "unsupported chain") {
			c.JSON(http.StatusConflict, gin.H{"error": msg})
			return
		}
		log.Printf("Error adding wallet for user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add wallet"})
		return
	}
	c.JSON(http.StatusCreated, wallet)
}

// DeleteWallet removes a wallet owned by the authenticated user.
func (a *API) DeleteWallet(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	walletID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet id"})
		return
	}

	if err := a.walletService.DeleteWallet(walletID, userID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
			return
		}
		log.Printf("Error deleting wallet %d for user %d: %v", walletID, userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete wallet"})
		return
	}
	c.Status(http.StatusNoContent)
}

// ListWalletAssets returns token assets for a specific wallet owned by the authenticated user.
func (a *API) ListWalletAssets(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	walletID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet id"})
		return
	}

	assets, err := a.walletService.GetWalletAssets(walletID, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
			return
		}
		log.Printf("Error listing assets for wallet %d, user %d: %v", walletID, userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve wallet assets"})
		return
	}
	c.JSON(http.StatusOK, assets)
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

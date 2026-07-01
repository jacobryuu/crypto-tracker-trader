package api

import (
	"log"
	"net/http"
	"strconv"

	"crypto-tracker-trader/internal/api/middleware"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
)

// GetDefiPositions returns all DeFi positions for the authenticated user.
func (a *API) GetDefiPositions(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	positions, err := a.defiService.GetPositions(c.Request.Context(), userID)
	if err != nil {
		log.Printf("GetDefiPositions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve DeFi positions"})
		return
	}
	c.JSON(http.StatusOK, positions)
}

// SyncDefiPositions triggers a manual DeFi sync for a specific wallet address.
func (a *API) SyncDefiPositions(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	walletIDStr := c.Param("id")
	walletID, err := strconv.ParseUint(walletIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet id"})
		return
	}

	addressParam := c.Query("address")
	if !common.IsHexAddress(addressParam) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address query param must be a valid Ethereum hex address"})
		return
	}

	// Verify wallet ownership via wallet service
	wallets, err := a.walletService.GetWallets(userID)
	if err != nil {
		log.Printf("SyncDefiPositions: getting wallets: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify wallet ownership"})
		return
	}
	owned := false
	for _, w := range wallets {
		if w.ID == walletID {
			owned = true
			break
		}
	}
	if !owned {
		c.JSON(http.StatusForbidden, gin.H{"error": "wallet not found or not owned by user"})
		return
	}

	addr := common.HexToAddress(addressParam)
	if err := a.defiService.SyncPositions(c.Request.Context(), walletID, addr); err != nil {
		log.Printf("SyncDefiPositions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DeFi sync failed"})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "DeFi sync triggered for wallet", "wallet_id": walletID})
}

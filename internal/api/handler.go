package api

import (
	"log"
	"net/http"

	"crypto-tracker-trader/internal/service"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
)

type API struct {
	portfolioService             service.PortfolioManager
	blockchainDataFetcherService service.BlockchainDataFetcher
	userService                  service.UserManager
}

func NewAPI(portfolioService service.PortfolioManager, blockchainDataFetcherService service.BlockchainDataFetcher, userService service.UserManager) *API {
	return &API{
		portfolioService:             portfolioService,
		blockchainDataFetcherService: blockchainDataFetcherService,
		userService:                  userService,
	}
}

// RegisterUser handles user registration.
func (a *API) RegisterUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := a.userService.RegisterUser(req.Username, req.Email, req.Password)
	if err != nil {
		if err.Error() == "username already taken" || err.Error() == "email already taken" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		log.Printf("Error registering user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully", "user_id": user.ID, "username": user.Username})
}

// LoginUser handles user login.
func (a *API) LoginUser(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := a.userService.LoginUser(req.Email, req.Password)
	if err != nil {
		if err.Error() == "invalid credentials" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		log.Printf("Error logging in user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to login user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Login successful", "user_id": user.ID, "username": user.Username})
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
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
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
		auth := v1.Group("/auth")
		{
			auth.POST("/register", a.RegisterUser)
			auth.POST("/login", a.LoginUser)
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

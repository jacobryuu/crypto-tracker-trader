package api

import (
	"log"
	"net/http"

	"crypto-tracker-trader/internal/api/middleware"
	"crypto-tracker-trader/internal/auth"
	"crypto-tracker-trader/internal/service"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
)

// APIConfig holds all dependencies for the API server.
type APIConfig struct {
	PortfolioService    service.PortfolioManager
	BlockchainService   service.BlockchainDataFetcher
	UserService         service.UserManager
	WalletService       service.WalletManager
	PriceService        service.PriceManager
	ExchangeService     service.ExchangeManager
	DefiService         service.DefiManager
	PortfolioAggregator service.PortfolioAggregator
	JWTSecret           string
	JWTExpiryHours      int
	PriceSymbols        []string
}

// API is the central HTTP handler.
type API struct {
	portfolioService             service.PortfolioManager
	blockchainDataFetcherService service.BlockchainDataFetcher
	userService                  service.UserManager
	walletService                service.WalletManager
	priceService                 service.PriceManager
	exchangeService              service.ExchangeManager
	defiService                  service.DefiManager
	portfolioAggregator          service.PortfolioAggregator
	jwtSecret                    string
	jwtExpiryHours               int
	priceSymbols                 []string
}

// NewAPI constructs the API from an APIConfig.
func NewAPI(cfg APIConfig) *API {
	return &API{
		portfolioService:             cfg.PortfolioService,
		blockchainDataFetcherService: cfg.BlockchainService,
		userService:                  cfg.UserService,
		walletService:                cfg.WalletService,
		priceService:                 cfg.PriceService,
		exchangeService:              cfg.ExchangeService,
		defiService:                  cfg.DefiService,
		portfolioAggregator:          cfg.PortfolioAggregator,
		jwtSecret:                    cfg.JWTSecret,
		jwtExpiryHours:               cfg.JWTExpiryHours,
		priceSymbols:                 cfg.PriceSymbols,
	}
}

// RegisterUser handles user registration and returns a JWT on success.
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
	token, err := auth.GenerateToken(user.ID, a.jwtSecret, a.jwtExpiryHours)
	if err != nil {
		log.Printf("Error generating token for user %d: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message":      "User registered successfully",
		"user_id":      user.ID,
		"username":     user.Username,
		"access_token": token,
	})
}

// LoginUser authenticates a user and returns a JWT on success.
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
	token, err := auth.GenerateToken(user.ID, a.jwtSecret, a.jwtExpiryHours)
	if err != nil {
		log.Printf("Error generating token for user %d: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":      "Login successful",
		"user_id":      user.ID,
		"username":     user.Username,
		"access_token": token,
	})
}

// FetchETHBalanceAndSave handles fetching ETH balance from blockchain and saving it to DB.
func (a *API) FetchETHBalanceAndSave(c *gin.Context) {
	addressParam := c.Param("address")
	if !common.IsHexAddress(addressParam) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Ethereum address format"})
		return
	}
	address := common.HexToAddress(addressParam)
	if err := a.blockchainDataFetcherService.FetchAndSaveETHBalance(c.Request.Context(), address); err != nil {
		log.Printf("Error fetching and saving ETH balance for address %s: %v", address.Hex(), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch and save ETH balance"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ETH balance fetched and saved successfully"})
}

func (a *API) GetPortfolioHistory(c *gin.Context) {
	history, err := a.portfolioService.GetPortfolioHistory()
	if err != nil {
		log.Printf("Error getting portfolio history: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An internal server error occurred"})
		return
	}
	c.JSON(http.StatusOK, history)
}

func (a *API) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")
	{
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", a.RegisterUser)
			authGroup.POST("/login", a.LoginUser)
		}

		protected := v1.Group("")
		protected.Use(middleware.Auth(a.jwtSecret))
		{
			// Portfolio
			portfolio := protected.Group("/portfolio")
			{
				portfolio.GET("/history", a.GetPortfolioHistory)
				portfolio.GET("/summary", a.GetPortfolioSummary)
				portfolio.GET("/allocation", a.GetPortfolioAllocation)
			}

			// Blockchain
			blockchain := protected.Group("/blockchain")
			{
				blockchain.POST("/fetch-eth-balance/:address", a.FetchETHBalanceAndSave)
			}

			// Wallets
			wallets := protected.Group("/wallets")
			{
				wallets.POST("", a.AddWallet)
				wallets.GET("", a.ListWallets)
				wallets.DELETE("/:id", a.DeleteWallet)
				wallets.GET("/:id/assets", a.ListWalletAssets)
				wallets.POST("/:id/defi/sync", a.SyncDefiPositions)
			}

			// Prices
			prices := protected.Group("/prices")
			{
				prices.GET("/:symbol", a.GetLatestPrice)
				prices.GET("/:symbol/history", a.GetPriceHistory)
				prices.POST("/sync", a.TriggerPriceSync)
			}

			// Exchanges
			exchanges := protected.Group("/exchanges")
			{
				exchanges.POST("", a.AddExchangeCredential)
				exchanges.GET("", a.ListExchangeCredentials)
				exchanges.DELETE("/:id", a.DeleteExchangeCredential)
				exchanges.POST("/:id/sync", a.SyncExchangeBalances)
				exchanges.GET("/balances", a.GetExchangeBalances)
			}

			// DeFi
			defi := protected.Group("/defi")
			{
				defi.GET("/positions", a.GetDefiPositions)
			}
		}
	}
}

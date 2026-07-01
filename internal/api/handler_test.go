package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"crypto-tracker-trader/internal/auth"
	"crypto-tracker-trader/internal/model"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const testJWTSecret = "test-secret-key-32-bytes-padding!"
const testJWTExpiry = 24

// MockBlockchainDataFetcher is a mock implementation of service.BlockchainDataFetcher
type MockBlockchainDataFetcher struct {
	mock.Mock
}

func (m *MockBlockchainDataFetcher) FetchAndSaveETHBalance(ctx context.Context, address common.Address) error {
	args := m.Called(ctx, address)
	return args.Error(0)
}

// MockPortfolioManager is a mock implementation of service.PortfolioManager
type MockPortfolioManager struct {
	mock.Mock
}

func (m *MockPortfolioManager) GetPortfolioHistory() ([]model.PortfolioSnapshot, error) {
	args := m.Called()
	return args.Get(0).([]model.PortfolioSnapshot), args.Error(1)
}

// MockUserManager is a mock implementation of service.UserManager
type MockUserManager struct {
	mock.Mock
}

func (m *MockUserManager) RegisterUser(username, email, password string) (*model.User, error) {
	args := m.Called(username, email, password)
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserManager) LoginUser(email, password string) (*model.User, error) {
	args := m.Called(email, password)
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserManager) GetUserByID(userID uint64) (*model.User, error) {
	args := m.Called(userID)
	return args.Get(0).(*model.User), args.Error(1)
}

// MockWalletManager is a mock implementation of service.WalletManager
type MockWalletManager struct {
	mock.Mock
}

func (m *MockWalletManager) AddWallet(userID uint64, chain, address, label string) (*model.UserWallet, error) {
	args := m.Called(userID, chain, address, label)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.UserWallet), args.Error(1)
}

func (m *MockWalletManager) GetWallets(userID uint64) ([]model.UserWallet, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.UserWallet), args.Error(1)
}

func (m *MockWalletManager) DeleteWallet(walletID, userID uint64) error {
	args := m.Called(walletID, userID)
	return args.Error(0)
}

func (m *MockWalletManager) GetWalletAssets(walletID, userID uint64) ([]model.UserAsset, error) {
	args := m.Called(walletID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.UserAsset), args.Error(1)
}

// MockPriceManager is a mock implementation of service.PriceManager
type MockPriceManager struct {
	mock.Mock
}

func (m *MockPriceManager) GetLatestPrice(ctx context.Context, symbol string) (*model.AssetPrice, error) {
	args := m.Called(ctx, symbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AssetPrice), args.Error(1)
}

func (m *MockPriceManager) GetPriceHistory(ctx context.Context, symbol string, limit int) ([]model.AssetPrice, error) {
	args := m.Called(ctx, symbol, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.AssetPrice), args.Error(1)
}

func (m *MockPriceManager) FetchAndSave(ctx context.Context, symbols []string) error {
	return m.Called(ctx, symbols).Error(0)
}

// MockExchangeManager is a mock implementation of service.ExchangeManager
type MockExchangeManager struct {
	mock.Mock
}

func (m *MockExchangeManager) AddCredential(ctx context.Context, userID uint64, exchange, apiKey, apiSecret string) (*model.ExchangeCredential, error) {
	args := m.Called(ctx, userID, exchange, apiKey, apiSecret)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ExchangeCredential), args.Error(1)
}

func (m *MockExchangeManager) GetCredentials(ctx context.Context, userID uint64) ([]model.ExchangeCredential, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.ExchangeCredential), args.Error(1)
}

func (m *MockExchangeManager) DeleteCredential(ctx context.Context, credentialID, userID uint64) error {
	return m.Called(ctx, credentialID, userID).Error(0)
}

func (m *MockExchangeManager) SyncBalances(ctx context.Context, credentialID uint64) ([]model.ExchangeBalance, error) {
	args := m.Called(ctx, credentialID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.ExchangeBalance), args.Error(1)
}

func (m *MockExchangeManager) GetBalances(ctx context.Context, userID uint64) ([]model.ExchangeBalance, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.ExchangeBalance), args.Error(1)
}

func (m *MockExchangeManager) StartSync(ctx context.Context, interval time.Duration) {}

// MockDefiManager is a mock implementation of service.DefiManager
type MockDefiManager struct {
	mock.Mock
}

func (m *MockDefiManager) SyncPositions(ctx context.Context, walletID uint64, address common.Address) error {
	return m.Called(ctx, walletID, address).Error(0)
}

func (m *MockDefiManager) GetPositions(ctx context.Context, userID uint64) ([]model.UserDefiPosition, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.UserDefiPosition), args.Error(1)
}

func (m *MockDefiManager) StartSync(ctx context.Context, interval time.Duration) {}

// MockPortfolioAggregator is a mock implementation of service.PortfolioAggregator
type MockPortfolioAggregator struct {
	mock.Mock
}

func (m *MockPortfolioAggregator) GetSummary(ctx context.Context, userID uint64) (*model.PortfolioSummary, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.PortfolioSummary), args.Error(1)
}

// newTestAPI is a helper that creates an API with test JWT config and blank mocks.
func newTestAPI(pm *MockPortfolioManager, bd *MockBlockchainDataFetcher, um *MockUserManager) (*API, *MockWalletManager) {
	wm := new(MockWalletManager)
	return NewAPI(APIConfig{
		PortfolioService:    pm,
		BlockchainService:   bd,
		UserService:         um,
		WalletService:       wm,
		PriceService:        new(MockPriceManager),
		ExchangeService:     new(MockExchangeManager),
		DefiService:         new(MockDefiManager),
		PortfolioAggregator: new(MockPortfolioAggregator),
		JWTSecret:           testJWTSecret,
		JWTExpiryHours:      testJWTExpiry,
		PriceSymbols:        []string{"BTC", "ETH"},
	}), wm
}

func TestGetPortfolioHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockPortfolioManager := new(MockPortfolioManager)
	mockBlockchainDataFetcher := new(MockBlockchainDataFetcher)
	mockUserManager := new(MockUserManager)
	apiHandler, _ := newTestAPI(mockPortfolioManager, mockBlockchainDataFetcher, mockUserManager)

	r := gin.Default()
	apiHandler.RegisterRoutes(r)

	snapshot := model.PortfolioSnapshot{
		Timestamp: time.Now(),
		Assets: []model.PortfolioAsset{
			{AssetID: "BTC", Quantity: "1", Value: "50000"},
		},
		TotalValue: "50000",
	}
	mockPortfolioManager.On("GetPortfolioHistory").Return([]model.PortfolioSnapshot{snapshot}, nil)

	// Portfolio history now requires auth — provide a valid token.
	token := mustGenerateToken(t, 1, testJWTSecret)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/portfolio/history", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []model.PortfolioSnapshot
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response, 1)
	assert.Equal(t, snapshot.TotalValue, response[0].TotalValue)
	mockPortfolioManager.AssertExpectations(t)

	// Without token → 401.
	rNoAuth := gin.Default()
	apiHandler2, _ := newTestAPI(new(MockPortfolioManager), new(MockBlockchainDataFetcher), new(MockUserManager))
	apiHandler2.RegisterRoutes(rNoAuth)
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/portfolio/history", nil)
	w2 := httptest.NewRecorder()
	rNoAuth.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusUnauthorized, w2.Code)

	// Service error → 500.
	rError := gin.Default()
	mockPM2 := new(MockPortfolioManager)
	apiHandler3, _ := newTestAPI(mockPM2, new(MockBlockchainDataFetcher), new(MockUserManager))
	apiHandler3.RegisterRoutes(rError)
	mockPM2.On("GetPortfolioHistory").Return([]model.PortfolioSnapshot{}, errors.New("failed to fetch history"))
	token3 := mustGenerateToken(t, 1, testJWTSecret)
	req3, _ := http.NewRequest(http.MethodGet, "/api/v1/portfolio/history", nil)
	req3.Header.Set("Authorization", "Bearer "+token3)
	w3 := httptest.NewRecorder()
	rError.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusInternalServerError, w3.Code)
	assert.Contains(t, w3.Body.String(), "An internal server error occurred")
}

func TestFetchETHBalanceAndSave(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testAddress := common.HexToAddress("0x1234567890123456789012345678901234567890")

	t.Run("success", func(t *testing.T) {
		r := gin.Default()
		mockBD := new(MockBlockchainDataFetcher)
		apiHandler, _ := newTestAPI(new(MockPortfolioManager), mockBD, new(MockUserManager))
		apiHandler.RegisterRoutes(r)
		mockBD.On("FetchAndSaveETHBalance", mock.Anything, testAddress).Return(nil).Once()

		token := mustGenerateToken(t, 1, testJWTSecret)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/blockchain/fetch-eth-balance/"+testAddress.Hex(), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ETH balance fetched and saved successfully")
		mockBD.AssertExpectations(t)
	})

	t.Run("invalid_address", func(t *testing.T) {
		r := gin.Default()
		apiHandler, _ := newTestAPI(new(MockPortfolioManager), new(MockBlockchainDataFetcher), new(MockUserManager))
		apiHandler.RegisterRoutes(r)

		token := mustGenerateToken(t, 1, testJWTSecret)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/blockchain/fetch-eth-balance/invalid-address", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid Ethereum address format")
	})

	t.Run("service_error", func(t *testing.T) {
		r := gin.Default()
		mockBD := new(MockBlockchainDataFetcher)
		apiHandler, _ := newTestAPI(new(MockPortfolioManager), mockBD, new(MockUserManager))
		apiHandler.RegisterRoutes(r)
		mockBD.On("FetchAndSaveETHBalance", mock.Anything, testAddress).Return(errors.New("service error")).Once()

		token := mustGenerateToken(t, 1, testJWTSecret)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/blockchain/fetch-eth-balance/"+testAddress.Hex(), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockBD.AssertExpectations(t)
	})
}

func TestUserRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	setup := func() (*gin.Engine, *MockUserManager) {
		r := gin.Default()
		mockUM := new(MockUserManager)
		apiHandler, _ := newTestAPI(new(MockPortfolioManager), new(MockBlockchainDataFetcher), mockUM)
		apiHandler.RegisterRoutes(r)
		return r, mockUM
	}

	t.Run("successful_registration", func(t *testing.T) {
		r, mockUM := setup()
		testUser := &model.User{ID: 1, Username: "testuser", Email: "test@example.com"}
		mockUM.On("RegisterUser", "testuser", "test@example.com", "password123").Return(testUser, nil).Once()

		body, _ := json.Marshal(gin.H{"username": "testuser", "email": "test@example.com", "password": "password123"})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), `"message":"User registered successfully"`)
		assert.Contains(t, w.Body.String(), `"user_id":1`)
		assert.Contains(t, w.Body.String(), `"username":"testuser"`)
		assert.Contains(t, w.Body.String(), `"access_token"`)
		mockUM.AssertExpectations(t)
	})

	t.Run("missing_fields", func(t *testing.T) {
		r, mockUM := setup()
		body, _ := json.Marshal(gin.H{"username": "testuser", "email": "test@example.com"})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Key: 'Password' Error:Field validation for 'Password' failed on the 'required' tag")
		mockUM.AssertNotCalled(t, "RegisterUser", mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestUserLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	setup := func() (*gin.Engine, *MockUserManager) {
		r := gin.Default()
		mockUM := new(MockUserManager)
		apiHandler, _ := newTestAPI(new(MockPortfolioManager), new(MockBlockchainDataFetcher), mockUM)
		apiHandler.RegisterRoutes(r)
		return r, mockUM
	}

	t.Run("successful_login", func(t *testing.T) {
		r, mockUM := setup()
		testUser := &model.User{ID: 1, Username: "testuser", Email: "test@example.com"}
		mockUM.On("LoginUser", "test@example.com", "password123").Return(testUser, nil).Once()

		body, _ := json.Marshal(gin.H{"email": "test@example.com", "password": "password123"})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"message":"Login successful"`)
		assert.Contains(t, w.Body.String(), `"user_id":1`)
		assert.Contains(t, w.Body.String(), `"username":"testuser"`)
		assert.Contains(t, w.Body.String(), `"access_token"`)
		mockUM.AssertExpectations(t)
	})

	t.Run("incorrect_credentials", func(t *testing.T) {
		r, mockUM := setup()
		mockUM.On("LoginUser", "wrong@example.com", "wrongpass").
			Return((*model.User)(nil), errors.New("invalid credentials")).Once()

		body, _ := json.Marshal(gin.H{"email": "wrong@example.com", "password": "wrongpass"})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), `"error":"invalid credentials"`)
		mockUM.AssertExpectations(t)
	})
}

// mustGenerateToken generates a JWT for tests; fails the test on error.
func mustGenerateToken(t *testing.T, userID uint64, secret string) string {
	t.Helper()
	token, err := auth.GenerateToken(userID, secret, testJWTExpiry)
	if err != nil {
		t.Fatalf("failed to generate test token: %v", err)
	}
	return token
}

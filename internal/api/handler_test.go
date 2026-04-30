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

	"crypto-tracker-trader/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ethereum/go-ethereum/common"
)

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

func TestGetPortfolioHistory(t *testing.T) {
	// Set up
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	mockPortfolioManager := new(MockPortfolioManager)
	mockBlockchainDataFetcher := new(MockBlockchainDataFetcher)
	mockUserManager := new(MockUserManager) // Initialize mockUserManager
	apiHandler := NewAPI(mockPortfolioManager, mockBlockchainDataFetcher, mockUserManager)

	apiHandler.RegisterRoutes(r)

	// Mock data
	snapshot := model.PortfolioSnapshot{
		Timestamp: time.Now(),
		Assets: []model.PortfolioAsset{
			{AssetID: "BTC", Quantity: "1", Value: "50000"},
		},
		TotalValue: "50000",
	}
	snapshots := []model.PortfolioSnapshot{snapshot}

	mockPortfolioManager.On("GetPortfolioHistory").Return(snapshots, nil)

	// Make request
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/portfolio/history", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response []model.PortfolioSnapshot
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Len(t, response, 1)
	assert.Equal(t, snapshot.TotalValue, response[0].TotalValue)
	assert.Len(t, response[0].Assets, 1)
	assert.Equal(t, snapshot.Assets[0].AssetID, response[0].Assets[0].AssetID)

	mockPortfolioManager.AssertExpectations(t)

	// Test case for error from service
	rError := gin.Default()                                                               // Create a new gin.Engine for the error case
	mockPortfolioManager = new(MockPortfolioManager)                                      // Reset mock
	mockBlockchainDataFetcher = new(MockBlockchainDataFetcher)                            // Reset mock
	mockUserManager = new(MockUserManager)                                                // Reset mockUserManager
	apiHandler = NewAPI(mockPortfolioManager, mockBlockchainDataFetcher, mockUserManager) // Pass new mock
	apiHandler.RegisterRoutes(rError)                                                     // Register routes on the new engine

	expectedError := errors.New("failed to fetch history")
	mockPortfolioManager.On("GetPortfolioHistory").Return([]model.PortfolioSnapshot{}, expectedError)

	req, _ = http.NewRequest(http.MethodGet, "/api/v1/portfolio/history", nil)
	w = httptest.NewRecorder()
	rError.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var errorResponse gin.H
	err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
	assert.NoError(t, err)
	assert.Equal(t, "An internal server error occurred", errorResponse["error"])

	mockPortfolioManager.AssertExpectations(t)
}

func TestFetchETHBalanceAndSave(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// r is now created inside each t.Run block

	mockPortfolioManager := new(MockPortfolioManager)
	// mockBlockchainDataFetcher is now created inside each t.Run block

	testAddress := common.HexToAddress("0x1234567890123456789012345678901234567890")

	// Test case 1: Successful fetch and save
	t.Run("success", func(t *testing.T) {
		r := gin.Default() // Create a new gin.Engine for this sub-test
		mockBlockchainDataFetcher := new(MockBlockchainDataFetcher)
		mockUserManager := new(MockUserManager) // Initialize mockUserManager for this sub-test
		apiHandler := NewAPI(mockPortfolioManager, mockBlockchainDataFetcher, mockUserManager)
		apiHandler.RegisterRoutes(r)

		mockBlockchainDataFetcher.On("FetchAndSaveETHBalance", mock.Anything, testAddress).Return(nil).Once()

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/blockchain/fetch-eth-balance/"+testAddress.Hex(), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ETH balance fetched and saved successfully")
		mockBlockchainDataFetcher.AssertExpectations(t)
	})

	// Test case 2: Invalid address
	t.Run("invalid_address", func(t *testing.T) {
		r := gin.Default() // Create a new gin.Engine for this sub-test
		mockBlockchainDataFetcher := new(MockBlockchainDataFetcher)
		mockUserManager := new(MockUserManager) // Initialize mockUserManager for this sub-test
		apiHandler := NewAPI(mockPortfolioManager, mockBlockchainDataFetcher, mockUserManager)
		apiHandler.RegisterRoutes(r)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/blockchain/fetch-eth-balance/invalid-address", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid Ethereum address format")
		mockBlockchainDataFetcher.AssertNotCalled(t, "FetchAndSaveETHBalance", mock.Anything, mock.Anything)
	})

	// Test case 3: Error from service
	t.Run("service_error", func(t *testing.T) {
		r := gin.Default() // Create a new gin.Engine for this sub-test
		mockBlockchainDataFetcher := new(MockBlockchainDataFetcher)
		mockUserManager := new(MockUserManager) // Initialize mockUserManager for this sub-test
		apiHandler := NewAPI(mockPortfolioManager, mockBlockchainDataFetcher, mockUserManager)
		apiHandler.RegisterRoutes(r)

		mockBlockchainDataFetcher.On("FetchAndSaveETHBalance", mock.Anything, testAddress).Return(errors.New("service error")).Once()

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/blockchain/fetch-eth-balance/"+testAddress.Hex(), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to fetch and save ETH balance")
		mockBlockchainDataFetcher.AssertExpectations(t)
	})
}

func TestUserRegistration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	setup := func() (*gin.Engine, *MockUserManager) {
		r := gin.Default()
		mockPortfolioManager := new(MockPortfolioManager)
		mockBlockchainDataFetcher := new(MockBlockchainDataFetcher)
		mockUserManager := new(MockUserManager)
		apiHandler := NewAPI(mockPortfolioManager, mockBlockchainDataFetcher, mockUserManager)
		apiHandler.RegisterRoutes(r)
		return r, mockUserManager
	}

	// Test case: Successful registration
	t.Run("successful_registration", func(t *testing.T) {
		r, mockUserManager := setup()
		testUser := &model.User{ID: 1, Username: "testuser", Email: "test@example.com"}
		mockUserManager.On("RegisterUser", "testuser", "test@example.com", "password123").Return(testUser, nil).Once()

		registrationData := gin.H{
			"username": "testuser",
			"email":    "test@example.com",
			"password": "password123",
		}
		jsonValue, _ := json.Marshal(registrationData)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(jsonValue))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), `"message":"User registered successfully"`)
		assert.Contains(t, w.Body.String(), `"user_id":1`)
		assert.Contains(t, w.Body.String(), `"username":"testuser"`)
		mockUserManager.AssertExpectations(t)
	})

	// Test case: Registration with missing fields (example)
	t.Run("missing_fields", func(t *testing.T) {
		r, mockUserManager := setup()
		registrationData := gin.H{
			"username": "testuser",
			"email":    "test@example.com",
			// Missing password
		}
		jsonValue, _ := json.Marshal(registrationData)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(jsonValue))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code) // Expect 400 Bad Request
		assert.Contains(t, w.Body.String(), "Key: 'Password' Error:Field validation for 'Password' failed on the 'required' tag")
		mockUserManager.AssertNotCalled(t, "RegisterUser", mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestUserLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	setup := func() (*gin.Engine, *MockUserManager) {
		r := gin.Default()
		mockPortfolioManager := new(MockPortfolioManager)
		mockBlockchainDataFetcher := new(MockBlockchainDataFetcher)
		mockUserManager := new(MockUserManager)
		apiHandler := NewAPI(mockPortfolioManager, mockBlockchainDataFetcher, mockUserManager)
		apiHandler.RegisterRoutes(r)
		return r, mockUserManager
	}

	// Test case: Successful login
	t.Run("successful_login", func(t *testing.T) {
		r, mockUserManager := setup()
		testUser := &model.User{ID: 1, Username: "testuser", Email: "test@example.com"}
		mockUserManager.On("LoginUser", "test@example.com", "password123").Return(testUser, nil).Once()

		loginData := gin.H{
			"email":    "test@example.com",
			"password": "password123",
		}
		jsonValue, _ := json.Marshal(loginData)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(jsonValue))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"message":"Login successful"`)
		assert.Contains(t, w.Body.String(), `"user_id":1`)
		assert.Contains(t, w.Body.String(), `"username":"testuser"`)
		mockUserManager.AssertExpectations(t)
	})

	// Test case: Login with incorrect credentials
	t.Run("incorrect_credentials", func(t *testing.T) {
		r, mockUserManager := setup()
		mockUserManager.On("LoginUser", "wrong@example.com", "wrongpass").Return((*model.User)(nil), errors.New("invalid credentials")).Once()

		loginData := gin.H{
			"email":    "wrong@example.com",
			"password": "wrongpass",
		}
		jsonValue, _ := json.Marshal(loginData)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(jsonValue))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code) // Expect 401 Unauthorized
		assert.Contains(t, w.Body.String(), `"error":"invalid credentials"`)
		mockUserManager.AssertExpectations(t)
	})
}

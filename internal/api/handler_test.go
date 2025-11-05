package api

import (
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

func TestGetPortfolioHistory(t *testing.T) {
	// Set up
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	mockPortfolioManager := new(MockPortfolioManager)
	mockBlockchainDataFetcher := new(MockBlockchainDataFetcher)
	apiHandler := NewAPI(mockPortfolioManager, mockBlockchainDataFetcher)

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
	rError := gin.Default()                                              // Create a new gin.Engine for the error case
	mockPortfolioManager = new(MockPortfolioManager)                     // Reset mock
	mockBlockchainDataFetcher = new(MockBlockchainDataFetcher)           // Reset mock
	apiHandler = NewAPI(mockPortfolioManager, mockBlockchainDataFetcher) // Pass new mock
	apiHandler.RegisterRoutes(rError)                                    // Register routes on the new engine

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
		apiHandler := NewAPI(mockPortfolioManager, mockBlockchainDataFetcher)
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
		apiHandler := NewAPI(mockPortfolioManager, mockBlockchainDataFetcher)
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
		apiHandler := NewAPI(mockPortfolioManager, mockBlockchainDataFetcher)
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

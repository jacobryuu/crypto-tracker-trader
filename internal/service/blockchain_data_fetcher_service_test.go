package service

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"crypto-tracker-trader/internal/store"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEthClient is a mock implementation of EthClientInterface
type MockEthClient struct {
	mock.Mock
}

func (m *MockEthClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	args := m.Called(ctx, number)
	return args.Get(0).(*types.Header), args.Error(1)
}

func (m *MockEthClient) BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error) {
	args := m.Called(ctx, account, blockNumber)
	return args.Get(0).(*big.Int), args.Error(1)
}

func TestBlockchainDataFetcherService_FetchAndSaveETHBalance(t *testing.T) {
	// Setup mocks
	mockEthClient := new(MockEthClient)
	mockPortfolioStore := new(store.MockPortfolioStore)

	service := NewBlockchainDataFetcherService(mockEthClient, mockPortfolioStore)

	ctx := context.Background()
	testAddress := common.HexToAddress("0x1234567890123456789012345678901234567890")

	// Test case 1: Successful fetch and save
	t.Run("success", func(t *testing.T) {
		expectedBlockNumber := big.NewInt(100)
		expectedBalanceWei := big.NewInt(0).Mul(big.NewInt(10), big.NewInt(1e18)) // 10 ETH

		mockEthClient.On("HeaderByNumber", ctx, mock.Anything).Return(&types.Header{Number: expectedBlockNumber}, nil).Once()
		mockEthClient.On("BalanceAt", ctx, testAddress, expectedBlockNumber).Return(expectedBalanceWei, nil).Once()
		mockPortfolioStore.On("AddSnapshot", mock.AnythingOfType("model.PortfolioSnapshot")).Return(nil).Once()

		err := service.FetchAndSaveETHBalance(ctx, testAddress)
		assert.NoError(t, err)

		mockEthClient.AssertExpectations(t)
		mockPortfolioStore.AssertExpectations(t)
	})

	// Test case 2: Error getting latest block header
	t.Run("error_get_header", func(t *testing.T) {
		expectedError := errors.New("failed to get header")
		mockEthClient.On("HeaderByNumber", ctx, mock.Anything).Return(&types.Header{}, expectedError).Once()

		err := service.FetchAndSaveETHBalance(ctx, testAddress)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get latest block header")

		mockEthClient.AssertExpectations(t)
		mockPortfolioStore.AssertNotCalled(t, "AddSnapshot")
	})

	// Test case 3: Error getting balance at block
	t.Run("error_get_balance", func(t *testing.T) {
		expectedBlockNumber := big.NewInt(100)
		expectedError := errors.New("failed to get balance")
		mockEthClient.On("HeaderByNumber", ctx, mock.Anything).Return(&types.Header{Number: expectedBlockNumber}, nil).Once()
		mockEthClient.On("BalanceAt", ctx, testAddress, expectedBlockNumber).Return(big.NewInt(0), expectedError).Once()

		err := service.FetchAndSaveETHBalance(ctx, testAddress)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get ETH balance")

		mockEthClient.AssertExpectations(t)
		mockPortfolioStore.AssertNotCalled(t, "AddSnapshot")
	})

	// Test case 4: Error adding snapshot to store
	t.Run("error_add_snapshot", func(t *testing.T) {
		expectedBlockNumber := big.NewInt(100)
		expectedBalanceWei := big.NewInt(0).Mul(big.NewInt(10), big.NewInt(1e18)) // 10 ETH
		expectedError := errors.New("failed to add snapshot")

		mockEthClient.On("HeaderByNumber", ctx, mock.Anything).Return(&types.Header{Number: expectedBlockNumber}, nil).Once()
		mockEthClient.On("BalanceAt", ctx, testAddress, expectedBlockNumber).Return(expectedBalanceWei, nil).Once()
		mockPortfolioStore.On("AddSnapshot", mock.AnythingOfType("model.PortfolioSnapshot")).Return(expectedError).Once()

		err := service.FetchAndSaveETHBalance(ctx, testAddress)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to add portfolio snapshot")

		mockEthClient.AssertExpectations(t)
		mockPortfolioStore.AssertExpectations(t)
	})
}

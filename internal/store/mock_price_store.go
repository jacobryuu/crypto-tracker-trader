package store

import (
	"crypto-tracker-trader/internal/model"

	"github.com/stretchr/testify/mock"
)

// MockPriceStore is a testify mock for PriceStoreInterface.
type MockPriceStore struct {
	mock.Mock
}

func (m *MockPriceStore) SavePrice(symbol, priceUSD, source string) error {
	return m.Called(symbol, priceUSD, source).Error(0)
}

func (m *MockPriceStore) GetLatestPrice(symbol string) (*model.AssetPrice, error) {
	args := m.Called(symbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.AssetPrice), args.Error(1)
}

func (m *MockPriceStore) GetPriceHistory(symbol string, limit int) ([]model.AssetPrice, error) {
	args := m.Called(symbol, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.AssetPrice), args.Error(1)
}

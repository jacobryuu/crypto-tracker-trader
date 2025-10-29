package store

import (
    "crypto-tracker-trader/internal/model"
    "github.com/stretchr/testify/mock"
)

type MockPortfolioStore struct {
    mock.Mock
}

func (m *MockPortfolioStore) AddSnapshot(snapshot model.PortfolioSnapshot) error {
    args := m.Called(snapshot)
    return args.Error(0)
}

func (m *MockPortfolioStore) GetHistory() ([]model.PortfolioSnapshot, error) {
    args := m.Called()
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).([]model.PortfolioSnapshot), args.Error(1)
}

func (m *MockPortfolioStore) Close() {
    m.Called()
}

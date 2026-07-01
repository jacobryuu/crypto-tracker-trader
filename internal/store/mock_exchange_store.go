package store

import (
	"crypto-tracker-trader/internal/model"

	"github.com/stretchr/testify/mock"
)

type MockExchangeStore struct {
	mock.Mock
}

func (m *MockExchangeStore) CreateCredential(cred *model.ExchangeCredential) error {
	return m.Called(cred).Error(0)
}

func (m *MockExchangeStore) GetCredentialsByUserID(userID uint64) ([]model.ExchangeCredential, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.ExchangeCredential), args.Error(1)
}

func (m *MockExchangeStore) GetCredentialByID(id uint64) (*model.ExchangeCredential, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ExchangeCredential), args.Error(1)
}

func (m *MockExchangeStore) GetAllActiveCredentials() ([]model.ExchangeCredential, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.ExchangeCredential), args.Error(1)
}

func (m *MockExchangeStore) DeleteCredential(id, userID uint64) error {
	return m.Called(id, userID).Error(0)
}

func (m *MockExchangeStore) UpsertBalances(credentialID, userID uint64, balances []model.ExchangeBalance) error {
	return m.Called(credentialID, userID, balances).Error(0)
}

func (m *MockExchangeStore) GetBalancesByUserID(userID uint64) ([]model.ExchangeBalance, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.ExchangeBalance), args.Error(1)
}

func (m *MockExchangeStore) GetBalancesByCredentialID(credentialID uint64) ([]model.ExchangeBalance, error) {
	args := m.Called(credentialID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.ExchangeBalance), args.Error(1)
}

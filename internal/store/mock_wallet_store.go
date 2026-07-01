package store

import (
	"crypto-tracker-trader/internal/model"

	"github.com/stretchr/testify/mock"
)

// MockWalletStore is a testify mock for WalletStoreInterface.
type MockWalletStore struct {
	mock.Mock
}

func (m *MockWalletStore) CreateWallet(wallet *model.UserWallet) error {
	args := m.Called(wallet)
	return args.Error(0)
}

func (m *MockWalletStore) GetWalletsByUserID(userID uint64) ([]model.UserWallet, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.UserWallet), args.Error(1)
}

func (m *MockWalletStore) GetWalletByID(id uint64) (*model.UserWallet, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.UserWallet), args.Error(1)
}

func (m *MockWalletStore) DeleteWallet(walletID, userID uint64) error {
	args := m.Called(walletID, userID)
	return args.Error(0)
}

func (m *MockWalletStore) GetAssetsByWalletID(walletID uint64) ([]model.UserAsset, error) {
	args := m.Called(walletID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.UserAsset), args.Error(1)
}

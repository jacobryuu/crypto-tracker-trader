package store

import (
	"crypto-tracker-trader/internal/model"

	"github.com/stretchr/testify/mock"
)

type MockDefiStore struct {
	mock.Mock
}

func (m *MockDefiStore) UpsertPosition(pos *model.UserDefiPosition) error {
	return m.Called(pos).Error(0)
}

func (m *MockDefiStore) GetPositionsByWalletID(walletID uint64) ([]model.UserDefiPosition, error) {
	args := m.Called(walletID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.UserDefiPosition), args.Error(1)
}

func (m *MockDefiStore) GetPositionsByUserID(userID uint64) ([]model.UserDefiPosition, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.UserDefiPosition), args.Error(1)
}

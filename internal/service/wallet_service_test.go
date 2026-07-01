package service

import (
	"errors"
	"testing"

	"crypto-tracker-trader/internal/model"
	"crypto-tracker-trader/internal/store"

	storemod "crypto-tracker-trader/internal/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newWalletService(ms *storemod.MockWalletStore) *WalletService {
	return NewWalletService(ms)
}

func TestWalletService_AddWallet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ms := new(storemod.MockWalletStore)
		ms.On("CreateWallet", mock.AnythingOfType("*model.UserWallet")).Return(nil)
		svc := newWalletService(ms)
		wallet, err := svc.AddWallet(1, "ethereum", "0xABC", "main")
		require.NoError(t, err)
		assert.Equal(t, "ethereum", wallet.Chain)
		assert.Equal(t, "0xABC", wallet.Address)
		ms.AssertExpectations(t)
	})

	t.Run("unsupported chain returns error without DB call", func(t *testing.T) {
		ms := new(storemod.MockWalletStore)
		svc := newWalletService(ms)
		_, err := svc.AddWallet(1, "unknown-chain", "0xABC", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported chain")
		ms.AssertNotCalled(t, "CreateWallet")
	})

	t.Run("empty address returns error without DB call", func(t *testing.T) {
		ms := new(storemod.MockWalletStore)
		svc := newWalletService(ms)
		_, err := svc.AddWallet(1, "ethereum", "", "")
		assert.Error(t, err)
		ms.AssertNotCalled(t, "CreateWallet")
	})

	t.Run("duplicate address returns friendly error", func(t *testing.T) {
		ms := new(storemod.MockWalletStore)
		ms.On("CreateWallet", mock.AnythingOfType("*model.UserWallet")).
			Return(errors.New("unique constraint violation"))
		svc := newWalletService(ms)
		_, err := svc.AddWallet(1, "ethereum", "0xDUP", "")
		assert.ErrorContains(t, err, "already registered")
	})

	t.Run("chain name is normalized to lowercase", func(t *testing.T) {
		ms := new(storemod.MockWalletStore)
		ms.On("CreateWallet", mock.AnythingOfType("*model.UserWallet")).Return(nil)
		svc := newWalletService(ms)
		wallet, err := svc.AddWallet(1, "ETHEREUM", "0xABC", "")
		require.NoError(t, err)
		assert.Equal(t, "ethereum", wallet.Chain)
	})
}

func TestWalletService_GetWallets(t *testing.T) {
	t.Run("returns wallets", func(t *testing.T) {
		ms := new(storemod.MockWalletStore)
		expected := []model.UserWallet{{ID: 1, UserID: 5, Chain: "ethereum"}}
		ms.On("GetWalletsByUserID", uint64(5)).Return(expected, nil)
		svc := newWalletService(ms)
		wallets, err := svc.GetWallets(5)
		require.NoError(t, err)
		assert.Len(t, wallets, 1)
	})

	t.Run("nil result becomes empty slice", func(t *testing.T) {
		ms := new(storemod.MockWalletStore)
		ms.On("GetWalletsByUserID", uint64(9)).Return(nil, nil)
		svc := newWalletService(ms)
		wallets, err := svc.GetWallets(9)
		require.NoError(t, err)
		assert.Empty(t, wallets)
		assert.NotNil(t, wallets)
	})
}

func TestWalletService_DeleteWallet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ms := new(storemod.MockWalletStore)
		ms.On("DeleteWallet", uint64(3), uint64(1)).Return(nil)
		svc := newWalletService(ms)
		assert.NoError(t, svc.DeleteWallet(3, 1))
	})

	t.Run("not found propagates ErrNotFound", func(t *testing.T) {
		ms := new(storemod.MockWalletStore)
		ms.On("DeleteWallet", uint64(99), uint64(1)).Return(store.ErrNotFound)
		svc := newWalletService(ms)
		err := svc.DeleteWallet(99, 1)
		assert.ErrorIs(t, err, store.ErrNotFound)
	})
}

func TestWalletService_GetWalletAssets(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ms := new(storemod.MockWalletStore)
		wallet := &model.UserWallet{ID: 2, UserID: 1}
		assets := []model.UserAsset{{ID: 1, WalletID: 2, Symbol: "ETH"}}
		ms.On("GetWalletByID", uint64(2)).Return(wallet, nil)
		ms.On("GetAssetsByWalletID", uint64(2)).Return(assets, nil)
		svc := newWalletService(ms)
		result, err := svc.GetWalletAssets(2, 1)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "ETH", result[0].Symbol)
	})

	t.Run("wallet belongs to different user returns ErrNotFound", func(t *testing.T) {
		ms := new(storemod.MockWalletStore)
		wallet := &model.UserWallet{ID: 2, UserID: 99}
		ms.On("GetWalletByID", uint64(2)).Return(wallet, nil)
		svc := newWalletService(ms)
		_, err := svc.GetWalletAssets(2, 1)
		assert.ErrorIs(t, err, store.ErrNotFound)
		ms.AssertNotCalled(t, "GetAssetsByWalletID")
	})

	t.Run("nil assets becomes empty slice", func(t *testing.T) {
		ms := new(storemod.MockWalletStore)
		wallet := &model.UserWallet{ID: 2, UserID: 1}
		ms.On("GetWalletByID", uint64(2)).Return(wallet, nil)
		ms.On("GetAssetsByWalletID", uint64(2)).Return(nil, nil)
		svc := newWalletService(ms)
		result, err := svc.GetWalletAssets(2, 1)
		require.NoError(t, err)
		assert.Empty(t, result)
		assert.NotNil(t, result)
	})
}

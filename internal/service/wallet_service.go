package service

import (
	"errors"
	"fmt"
	"strings"

	"crypto-tracker-trader/internal/model"
	"crypto-tracker-trader/internal/store"
)

// supportedChains is the set of valid chain identifiers.
var supportedChains = map[string]bool{
	"ethereum": true,
	"polygon":  true,
	"arbitrum": true,
	"optimism": true,
	"solana":   true,
}

// WalletService implements WalletManager.
type WalletService struct {
	walletStore store.WalletStoreInterface
}

// NewWalletService creates a new WalletService.
func NewWalletService(ws store.WalletStoreInterface) *WalletService {
	return &WalletService{walletStore: ws}
}

// AddWallet validates and registers a new wallet for the given user.
func (s *WalletService) AddWallet(userID uint64, chain, address, label string) (*model.UserWallet, error) {
	chain = strings.ToLower(strings.TrimSpace(chain))
	address = strings.TrimSpace(address)

	if !supportedChains[chain] {
		return nil, fmt.Errorf("unsupported chain: %s", chain)
	}
	if address == "" {
		return nil, errors.New("wallet address must not be empty")
	}

	wallet := &model.UserWallet{
		UserID:  userID,
		Chain:   chain,
		Address: address,
		Label:   label,
	}
	if err := s.walletStore.CreateWallet(wallet); err != nil {
		// Surface duplicate address error clearly.
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return nil, errors.New("wallet address already registered for this chain")
		}
		return nil, fmt.Errorf("add wallet: %w", err)
	}
	return wallet, nil
}

// GetWallets returns all wallets for the given user.
func (s *WalletService) GetWallets(userID uint64) ([]model.UserWallet, error) {
	wallets, err := s.walletStore.GetWalletsByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("get wallets: %w", err)
	}
	if wallets == nil {
		wallets = []model.UserWallet{}
	}
	return wallets, nil
}

// DeleteWallet removes a wallet, verifying ownership via userID.
func (s *WalletService) DeleteWallet(walletID, userID uint64) error {
	if err := s.walletStore.DeleteWallet(walletID, userID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return store.ErrNotFound
		}
		return fmt.Errorf("delete wallet: %w", err)
	}
	return nil
}

// GetWalletAssets returns assets for a wallet, verifying the wallet belongs to userID.
func (s *WalletService) GetWalletAssets(walletID, userID uint64) ([]model.UserAsset, error) {
	wallet, err := s.walletStore.GetWalletByID(walletID)
	if err != nil {
		return nil, fmt.Errorf("get wallet: %w", err)
	}
	if wallet.UserID != userID {
		return nil, store.ErrNotFound
	}
	assets, err := s.walletStore.GetAssetsByWalletID(walletID)
	if err != nil {
		return nil, fmt.Errorf("get wallet assets: %w", err)
	}
	if assets == nil {
		assets = []model.UserAsset{}
	}
	return assets, nil
}

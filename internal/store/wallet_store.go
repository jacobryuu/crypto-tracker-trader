package store

import (
	"context"
	"fmt"
	"time"

	"crypto-tracker-trader/internal/model"

	"github.com/jackc/pgx/v4/pgxpool"
)

// WalletStore implements WalletStoreInterface for PostgreSQL.
type WalletStore struct {
	db *pgxpool.Pool
}

// NewWalletStore creates a new WalletStore.
func NewWalletStore(db *pgxpool.Pool) *WalletStore {
	return &WalletStore{db: db}
}

// CreateWallet inserts a new wallet row and sets the generated ID on the struct.
func (s *WalletStore) CreateWallet(wallet *model.UserWallet) error {
	wallet.CreatedAt = time.Now()
	wallet.UpdatedAt = time.Now()
	err := s.db.QueryRow(context.Background(),
		`INSERT INTO user_wallets (user_id, chain, address, label, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		wallet.UserID, wallet.Chain, wallet.Address, wallet.Label,
		wallet.CreatedAt, wallet.UpdatedAt,
	).Scan(&wallet.ID)
	if err != nil {
		return fmt.Errorf("create wallet: %w", err)
	}
	return nil
}

// GetWalletsByUserID returns all wallets belonging to the given user.
func (s *WalletStore) GetWalletsByUserID(userID uint64) ([]model.UserWallet, error) {
	rows, err := s.db.Query(context.Background(),
		`SELECT id, user_id, chain, address, label, created_at, updated_at
		 FROM user_wallets WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query wallets: %w", err)
	}
	defer rows.Close()

	var wallets []model.UserWallet
	for rows.Next() {
		var w model.UserWallet
		if err := rows.Scan(&w.ID, &w.UserID, &w.Chain, &w.Address, &w.Label, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan wallet: %w", err)
		}
		wallets = append(wallets, w)
	}
	return wallets, rows.Err()
}

// GetWalletByID returns a single wallet by its primary key.
func (s *WalletStore) GetWalletByID(id uint64) (*model.UserWallet, error) {
	var w model.UserWallet
	err := s.db.QueryRow(context.Background(),
		`SELECT id, user_id, chain, address, label, created_at, updated_at
		 FROM user_wallets WHERE id = $1`,
		id,
	).Scan(&w.ID, &w.UserID, &w.Chain, &w.Address, &w.Label, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get wallet by id: %w", err)
	}
	return &w, nil
}

// DeleteWallet removes a wallet only if it belongs to the given user.
// Returns an error if the wallet does not exist or does not belong to the user.
func (s *WalletStore) DeleteWallet(walletID, userID uint64) error {
	tag, err := s.db.Exec(context.Background(),
		`DELETE FROM user_wallets WHERE id = $1 AND user_id = $2`,
		walletID, userID,
	)
	if err != nil {
		return fmt.Errorf("delete wallet: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetAssetsByWalletID returns all token assets for the given wallet.
func (s *WalletStore) GetAssetsByWalletID(walletID uint64) ([]model.UserAsset, error) {
	rows, err := s.db.Query(context.Background(),
		`SELECT id, wallet_id, chain, token_address, symbol, balance, updated_at
		 FROM user_assets WHERE wallet_id = $1 ORDER BY symbol`,
		walletID,
	)
	if err != nil {
		return nil, fmt.Errorf("query assets: %w", err)
	}
	defer rows.Close()

	var assets []model.UserAsset
	for rows.Next() {
		var a model.UserAsset
		if err := rows.Scan(&a.ID, &a.WalletID, &a.Chain, &a.TokenAddr, &a.Symbol, &a.Balance, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan asset: %w", err)
		}
		assets = append(assets, a)
	}
	return assets, rows.Err()
}

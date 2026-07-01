package service

import (
	"context"
	"math/big"
	"time"

	"crypto-tracker-trader/internal/model"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// UserStoreInterface defines the methods for interacting with user storage.
type UserStoreInterface interface {
	CreateUser(user *model.User, credential *model.UserCredential, authProvider *model.UserAuthProvider) error
	GetUserByUsername(username string) (*model.User, *model.UserCredential, error)
	GetUserByEmail(email string) (*model.User, *model.UserCredential, error)
	GetUserByID(userID uint64) (*model.User, *model.UserCredential, error)
}

// EthClientInterface defines the methods of ethclient.Client that BlockchainDataFetcherService uses.
type EthClientInterface interface {
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	BalanceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (*big.Int, error)
}

// BlockchainDataFetcher defines the interface for fetching blockchain data.
type BlockchainDataFetcher interface {
	FetchAndSaveETHBalance(ctx context.Context, address common.Address) error
}

// PortfolioManager defines the interface for portfolio operations.
type PortfolioManager interface {
	GetPortfolioHistory() ([]model.PortfolioSnapshot, error)
}

// UserManager defines the interface for user management operations.
type UserManager interface {
	RegisterUser(username, email, password string) (*model.User, error)
	LoginUser(email, password string) (*model.User, error)
	GetUserByID(userID uint64) (*model.User, error)
}

// WalletManager defines the interface for wallet management operations.
type WalletManager interface {
	AddWallet(userID uint64, chain, address, label string) (*model.UserWallet, error)
	GetWallets(userID uint64) ([]model.UserWallet, error)
	DeleteWallet(walletID, userID uint64) error
	GetWalletAssets(walletID, userID uint64) ([]model.UserAsset, error)
}

// PriceFetcher is the interface for external price data providers.
type PriceFetcher interface {
	FetchPrices(ctx context.Context, symbols []string) (map[string]string, error)
}

// PriceManager defines the interface for price data operations.
type PriceManager interface {
	GetLatestPrice(ctx context.Context, symbol string) (*model.AssetPrice, error)
	GetPriceHistory(ctx context.Context, symbol string, limit int) ([]model.AssetPrice, error)
	FetchAndSave(ctx context.Context, symbols []string) error
}

// ExchangeManager defines the interface for exchange credential and balance management.
type ExchangeManager interface {
	AddCredential(ctx context.Context, userID uint64, exchange, apiKey, apiSecret string) (*model.ExchangeCredential, error)
	GetCredentials(ctx context.Context, userID uint64) ([]model.ExchangeCredential, error)
	DeleteCredential(ctx context.Context, credentialID, userID uint64) error
	SyncBalances(ctx context.Context, credentialID uint64) ([]model.ExchangeBalance, error)
	GetBalances(ctx context.Context, userID uint64) ([]model.ExchangeBalance, error)
	StartSync(ctx context.Context, interval time.Duration)
}

// DefiManager defines the interface for DeFi position management.
type DefiManager interface {
	SyncPositions(ctx context.Context, walletID uint64, address common.Address) error
	GetPositions(ctx context.Context, userID uint64) ([]model.UserDefiPosition, error)
	StartSync(ctx context.Context, interval time.Duration)
}

// PortfolioAggregator computes the unified portfolio view across all data sources.
type PortfolioAggregator interface {
	GetSummary(ctx context.Context, userID uint64) (*model.PortfolioSummary, error)
}

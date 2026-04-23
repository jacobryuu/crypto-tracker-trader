package service

import (
	"context"
	"math/big"

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

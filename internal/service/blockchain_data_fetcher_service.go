package service

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"crypto-tracker-trader/internal/model"
	"crypto-tracker-trader/internal/store"

	"github.com/ethereum/go-ethereum/common"
)

// BlockchainDataFetcherService implements the BlockchainDataFetcher interface.
type BlockchainDataFetcherService struct {
	ethClient      EthClientInterface
	portfolioStore store.PortfolioStoreInterface
}

// NewBlockchainDataFetcherService creates a new BlockchainDataFetcherService.
func NewBlockchainDataFetcherService(ethClient EthClientInterface, portfolioStore store.PortfolioStoreInterface) BlockchainDataFetcher {
	return &BlockchainDataFetcherService{
		ethClient:      ethClient,
		portfolioStore: portfolioStore,
	}
}

// FetchAndSaveETHBalance fetches the ETH balance for a given address at the latest block and saves it to the portfolio store.
func (s *BlockchainDataFetcherService) FetchAndSaveETHBalance(ctx context.Context, address common.Address) error {
	// Get the latest block number
	header, err := s.ethClient.HeaderByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get latest block header: %w", err)
	}
	blockNumber := header.Number

	// Get the ETH balance at the latest block
	balance, err := s.ethClient.BalanceAt(ctx, address, blockNumber)
	if err != nil {
		return fmt.Errorf("failed to get ETH balance for address %s at block %s: %w", address.Hex(), blockNumber.String(), err)
	}

	// Convert balance from Wei to ETH (1 ETH = 10^18 Wei)
	// For simplicity, we'll store it as a big.Int and let the model handle conversion if needed.
	// Or, convert to a more human-readable format (e.g., float64) for storage if NUMERIC type in DB is used.
	// Given the DB schema uses NUMERIC(20, 8), we should convert it to a decimal string.
	ethBalance := new(big.Float).SetInt(balance)
	divisor := new(big.Float).SetInt(big.NewInt(1e18)) // 10^18
	ethBalance = ethBalance.Quo(ethBalance, divisor)

	// Create a portfolio snapshot
	snapshot := model.PortfolioSnapshot{
		Timestamp:  time.Now(),
		TotalValue: ethBalance.String(), // Store as string to match NUMERIC type in DB
		Assets: []model.PortfolioAsset{
			{
				AssetID:  "ETH",
				Quantity: ethBalance.String(), // Assuming quantity can also be a string representation of big.Float
				Value:    ethBalance.String(),
			},
		},
	}

	// Save the snapshot to the portfolio store
	if err := s.portfolioStore.AddSnapshot(snapshot); err != nil {
		return fmt.Errorf("failed to add portfolio snapshot: %w", err)
	}

	return nil
}

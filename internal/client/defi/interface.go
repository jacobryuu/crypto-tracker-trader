package defi

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// PositionType classifies a DeFi position.
type PositionType string

const (
	PositionTypeLP      PositionType = "lp"
	PositionTypeLending PositionType = "lending"
	PositionTypeFarming PositionType = "farming"
)

// Position represents a single DeFi protocol position.
type Position struct {
	Protocol     string          `json:"protocol"`
	PositionType PositionType    `json:"position_type"`
	Data         json.RawMessage `json:"data"` // protocol-specific JSON
	UpdatedAt    time.Time       `json:"updated_at"`
}

// DefiProtocolClient fetches positions from a DeFi protocol for a wallet address.
type DefiProtocolClient interface {
	GetPositions(ctx context.Context, walletAddr common.Address) ([]Position, error)
	ProtocolName() string
}

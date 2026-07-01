package uniswap

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"crypto-tracker-trader/internal/client/defi"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

// mainnetNFPMAddr is the Uniswap V3 NonfungiblePositionManager on Ethereum mainnet.
const mainnetNFPMAddr = "0xC36442b4a4522E871399CD717aBDD847Ab11FE88"

// EthCaller abstracts the eth_call needed to interact with contracts.
type EthCaller interface {
	CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

// Client implements defi.DefiProtocolClient for Uniswap V3.
type Client struct {
	eth      EthCaller
	nfpmAddr common.Address
}

// New creates a Uniswap V3 client using the default mainnet contract address.
func New(eth EthCaller) *Client {
	return &Client{eth: eth, nfpmAddr: common.HexToAddress(mainnetNFPMAddr)}
}

// NewWithAddress allows injecting a custom contract address (for testing/forks).
func NewWithAddress(eth EthCaller, nfpmAddr common.Address) *Client {
	return &Client{eth: eth, nfpmAddr: nfpmAddr}
}

func (c *Client) ProtocolName() string { return "uniswap-v3" }

// ABI function selectors (keccak256(sig)[0:4]):
//   balanceOf(address)              → 70a08231
//   tokenOfOwnerByIndex(address,uint256) → 2f745c59
//   positions(uint256)              → 99fbab88
var (
	selBalanceOf            = []byte{0x70, 0xa0, 0x82, 0x31}
	selTokenOfOwnerByIndex  = []byte{0x2f, 0x74, 0x5c, 0x59}
	selPositions            = []byte{0x99, 0xfb, 0xab, 0x88}
)

// GetPositions returns all Uniswap V3 LP positions owned by walletAddr.
func (c *Client) GetPositions(ctx context.Context, walletAddr common.Address) ([]defi.Position, error) {
	balance, err := c.balanceOf(ctx, walletAddr)
	if err != nil {
		return nil, fmt.Errorf("uniswap: balanceOf: %w", err)
	}
	if balance.Sign() == 0 {
		return nil, nil
	}

	positions := make([]defi.Position, 0, int(balance.Int64()))
	for i := int64(0); i < balance.Int64(); i++ {
		tokenID, err := c.tokenOfOwnerByIndex(ctx, walletAddr, big.NewInt(i))
		if err != nil {
			return nil, fmt.Errorf("uniswap: tokenOfOwnerByIndex(%d): %w", i, err)
		}
		pos, err := c.positions(ctx, tokenID)
		if err != nil {
			return nil, fmt.Errorf("uniswap: positions(%s): %w", tokenID.String(), err)
		}
		// Skip positions with zero liquidity (closed positions still hold the NFT)
		liq := new(big.Int)
		liq.SetString(pos.Liquidity, 10)
		if liq.Sign() == 0 {
			continue
		}
		data, _ := json.Marshal(pos)
		positions = append(positions, defi.Position{
			Protocol:     "uniswap-v3",
			PositionType: defi.PositionTypeLP,
			Data:         data,
			UpdatedAt:    time.Now(),
		})
	}
	return positions, nil
}

// UniswapPosition holds the decoded data from the positions() call.
type UniswapPosition struct {
	TokenID    string `json:"token_id"`
	Token0     string `json:"token0"`
	Token1     string `json:"token1"`
	Fee        uint64 `json:"fee"`
	TickLower  int64  `json:"tick_lower"`
	TickUpper  int64  `json:"tick_upper"`
	Liquidity  string `json:"liquidity"`
	Owed0      string `json:"tokens_owed0"`
	Owed1      string `json:"tokens_owed1"`
}

// balanceOf calls ERC-721 balanceOf on the NFPM.
func (c *Client) balanceOf(ctx context.Context, owner common.Address) (*big.Int, error) {
	calldata := make([]byte, 36)
	copy(calldata[:4], selBalanceOf)
	copy(calldata[16:36], owner.Bytes()) // right-aligned in 32-byte word
	ret, err := c.eth.CallContract(ctx, ethereum.CallMsg{To: &c.nfpmAddr, Data: calldata}, nil)
	if err != nil {
		return nil, err
	}
	if len(ret) < 32 {
		return big.NewInt(0), nil
	}
	return new(big.Int).SetBytes(ret[:32]), nil
}

// tokenOfOwnerByIndex returns the tokenId at a given index for the owner.
func (c *Client) tokenOfOwnerByIndex(ctx context.Context, owner common.Address, index *big.Int) (*big.Int, error) {
	calldata := make([]byte, 68)
	copy(calldata[:4], selTokenOfOwnerByIndex)
	copy(calldata[16:36], owner.Bytes())
	idxBytes := index.Bytes()
	copy(calldata[68-len(idxBytes):], idxBytes)
	ret, err := c.eth.CallContract(ctx, ethereum.CallMsg{To: &c.nfpmAddr, Data: calldata}, nil)
	if err != nil {
		return nil, err
	}
	if len(ret) < 32 {
		return nil, fmt.Errorf("unexpected short response")
	}
	return new(big.Int).SetBytes(ret[:32]), nil
}

// positions decodes the return value of NonfungiblePositionManager.positions(tokenId).
// Return layout (12 × 32-byte words):
//
//	[0]  nonce (uint96)
//	[1]  operator (address)
//	[2]  token0 (address)
//	[3]  token1 (address)
//	[4]  fee (uint24)
//	[5]  tickLower (int24)
//	[6]  tickUpper (int24)
//	[7]  liquidity (uint128)
//	[8]  feeGrowthInside0LastX128 (uint256)
//	[9]  feeGrowthInside1LastX128 (uint256)
//	[10] tokensOwed0 (uint128)
//	[11] tokensOwed1 (uint128)
func (c *Client) positions(ctx context.Context, tokenID *big.Int) (*UniswapPosition, error) {
	calldata := make([]byte, 36)
	copy(calldata[:4], selPositions)
	idBytes := tokenID.Bytes()
	copy(calldata[36-len(idBytes):], idBytes)

	ret, err := c.eth.CallContract(ctx, ethereum.CallMsg{To: &c.nfpmAddr, Data: calldata}, nil)
	if err != nil {
		return nil, err
	}
	if len(ret) < 384 { // 12 × 32
		return nil, fmt.Errorf("uniswap: positions response too short: %d bytes", len(ret))
	}

	word := func(i int) []byte { return ret[i*32 : i*32+32] }
	addrWord := func(i int) common.Address {
		return common.BytesToAddress(word(i)[12:])
	}
	uintWord := func(i int) *big.Int { return new(big.Int).SetBytes(word(i)) }
	// ABI sign-extends all signed integers to 256 bits (two's complement).
	signedWord := func(i int) int64 {
		v := new(big.Int).SetBytes(word(i))
		if v.Bit(255) == 1 {
			max256 := new(big.Int).Lsh(big.NewInt(1), 256)
			v.Sub(v, max256)
		}
		return v.Int64()
	}

	return &UniswapPosition{
		TokenID:   tokenID.String(),
		Token0:    addrWord(2).Hex(),
		Token1:    addrWord(3).Hex(),
		Fee:       uintWord(4).Uint64(),
		TickLower: signedWord(5),
		TickUpper: signedWord(6),
		Liquidity: uintWord(7).String(),
		Owed0:     uintWord(10).String(),
		Owed1:     uintWord(11).String(),
	}, nil
}

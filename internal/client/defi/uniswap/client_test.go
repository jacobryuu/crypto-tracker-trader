package uniswap

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockEthCaller mocks EthCaller.
type mockEthCaller struct{ mock.Mock }

func (m *mockEthCaller) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	args := m.Called(ctx, call, blockNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func makeWord(v *big.Int) []byte {
	b := make([]byte, 32)
	vb := v.Bytes()
	copy(b[32-len(vb):], vb)
	return b
}

func makeAddressWord(addr common.Address) []byte {
	b := make([]byte, 32)
	copy(b[12:], addr.Bytes())
	return b
}

func TestGetPositions_NoNFTs(t *testing.T) {
	eth := new(mockEthCaller)
	c := New(eth)
	owner := common.HexToAddress("0x1234567890123456789012345678901234567890")

	// balanceOf returns 0
	eth.On("CallContract", mock.Anything, mock.MatchedBy(func(msg ethereum.CallMsg) bool {
		return len(msg.Data) == 36 && msg.Data[0] == 0x70
	}), mock.Anything).Return(makeWord(big.NewInt(0)), nil)

	positions, err := c.GetPositions(context.Background(), owner)
	require.NoError(t, err)
	assert.Empty(t, positions)
}

func TestGetPositions_WithLPPosition(t *testing.T) {
	eth := new(mockEthCaller)
	nfpmAddr := common.HexToAddress(mainnetNFPMAddr)
	c := NewWithAddress(eth, nfpmAddr)
	owner := common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	// balanceOf → 1
	eth.On("CallContract", mock.Anything, mock.MatchedBy(func(msg ethereum.CallMsg) bool {
		return len(msg.Data) >= 4 && msg.Data[0] == 0x70 && msg.Data[1] == 0xa0
	}), mock.Anything).Return(makeWord(big.NewInt(1)), nil).Once()

	// tokenOfOwnerByIndex → tokenId = 42
	eth.On("CallContract", mock.Anything, mock.MatchedBy(func(msg ethereum.CallMsg) bool {
		return len(msg.Data) >= 4 && msg.Data[0] == 0x2f && msg.Data[1] == 0x74
	}), mock.Anything).Return(makeWord(big.NewInt(42)), nil).Once()

	// positions → 12 words with non-zero liquidity
	posData := buildPositionsResponse(
		common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"), // WETH
		common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"), // USDC
		3000,   // 0.3% fee
		-60000, // tickLower
		60000,  // tickUpper
		big.NewInt(1000000000),                // liquidity
		big.NewInt(500),                       // owed0
		big.NewInt(250),                       // owed1
	)
	eth.On("CallContract", mock.Anything, mock.MatchedBy(func(msg ethereum.CallMsg) bool {
		return len(msg.Data) >= 4 && msg.Data[0] == 0x99 && msg.Data[1] == 0xfb
	}), mock.Anything).Return(posData, nil).Once()

	positions, err := c.GetPositions(context.Background(), owner)
	require.NoError(t, err)
	require.Len(t, positions, 1)
	assert.Equal(t, "uniswap-v3", positions[0].Protocol)

	var lp UniswapPosition
	require.NoError(t, json.Unmarshal(positions[0].Data, &lp))
	assert.Equal(t, "42", lp.TokenID)
	assert.Equal(t, uint64(3000), lp.Fee)
	assert.Equal(t, int64(-60000), lp.TickLower)
	assert.Equal(t, int64(60000), lp.TickUpper)
	assert.Equal(t, "1000000000", lp.Liquidity)
}

func TestGetPositions_SkipsZeroLiquidity(t *testing.T) {
	eth := new(mockEthCaller)
	c := New(eth)
	owner := common.HexToAddress("0x1111111111111111111111111111111111111111")

	eth.On("CallContract", mock.Anything, mock.MatchedBy(func(msg ethereum.CallMsg) bool {
		return msg.Data[0] == 0x70
	}), mock.Anything).Return(makeWord(big.NewInt(1)), nil).Once()

	eth.On("CallContract", mock.Anything, mock.MatchedBy(func(msg ethereum.CallMsg) bool {
		return msg.Data[0] == 0x2f
	}), mock.Anything).Return(makeWord(big.NewInt(7)), nil).Once()

	// positions with zero liquidity → should be skipped
	posData := buildPositionsResponse(
		common.Address{}, common.Address{}, 3000, 0, 0,
		big.NewInt(0), big.NewInt(0), big.NewInt(0),
	)
	eth.On("CallContract", mock.Anything, mock.MatchedBy(func(msg ethereum.CallMsg) bool {
		return msg.Data[0] == 0x99
	}), mock.Anything).Return(posData, nil).Once()

	positions, err := c.GetPositions(context.Background(), owner)
	require.NoError(t, err)
	assert.Empty(t, positions, "zero-liquidity positions should be filtered out")
}

// buildPositionsResponse constructs the 384-byte (12×32) return value of positions().
func buildPositionsResponse(token0, token1 common.Address, fee uint64, tickLower, tickUpper int64, liquidity, owed0, owed1 *big.Int) []byte {
	buf := make([]byte, 384)
	// word 0: nonce
	// word 1: operator
	// word 2: token0
	copy(buf[2*32+12:], token0.Bytes())
	// word 3: token1
	copy(buf[3*32+12:], token1.Bytes())
	// word 4: fee
	binary.BigEndian.PutUint64(buf[4*32+24:], fee)
	// word 5: tickLower (signed 24-bit, encoded as two's complement in 256 bits)
	encodeSigned(buf[5*32:], tickLower)
	// word 6: tickUpper
	encodeSigned(buf[6*32:], tickUpper)
	// word 7: liquidity
	lb := liquidity.Bytes()
	copy(buf[7*32+32-len(lb):], lb)
	// word 10: tokensOwed0
	o0 := owed0.Bytes()
	copy(buf[10*32+32-len(o0):], o0)
	// word 11: tokensOwed1
	o1 := owed1.Bytes()
	copy(buf[11*32+32-len(o1):], o1)
	return buf
}

func encodeSigned(dst []byte, v int64) {
	if v >= 0 {
		b := big.NewInt(v).Bytes()
		copy(dst[32-len(b):], b)
	} else {
		// two's complement of 256-bit representation
		pos := new(big.Int).Neg(big.NewInt(v))
		max := new(big.Int).Lsh(big.NewInt(1), 256)
		twos := new(big.Int).Sub(max, pos)
		b := twos.Bytes()
		copy(dst[32-len(b):], b)
	}
}

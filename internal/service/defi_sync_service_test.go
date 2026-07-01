package service

import (
	"context"
	"errors"
	"testing"

	"crypto-tracker-trader/internal/client/defi"
	"crypto-tracker-trader/internal/model"
	"crypto-tracker-trader/internal/store"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockDefiProtocol fakes defi.DefiProtocolClient.
type mockDefiProtocol struct{ mock.Mock }

func (m *mockDefiProtocol) GetPositions(ctx context.Context, addr common.Address) ([]defi.Position, error) {
	args := m.Called(ctx, addr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]defi.Position), args.Error(1)
}
func (m *mockDefiProtocol) ProtocolName() string { return "mock-defi" }

func TestDefiSyncService_SyncPositions_Success(t *testing.T) {
	mockDS := new(store.MockDefiStore)
	mockWS := new(store.MockWalletStore)
	proto := new(mockDefiProtocol)
	svc := NewDefiSyncService(mockDS, mockWS, proto)

	addr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	proto.On("GetPositions", mock.Anything, addr).Return([]defi.Position{
		{Protocol: "mock-defi", PositionType: defi.PositionTypeLP, Data: []byte(`{"test":"data"}`)},
	}, nil)
	mockDS.On("UpsertPosition", mock.AnythingOfType("*model.UserDefiPosition")).Return(nil)

	err := svc.SyncPositions(context.Background(), 1, addr)
	require.NoError(t, err)
	mockDS.AssertExpectations(t)
}

func TestDefiSyncService_SyncPositions_ProtocolError_Continues(t *testing.T) {
	mockDS := new(store.MockDefiStore)
	mockWS := new(store.MockWalletStore)
	proto := new(mockDefiProtocol)
	svc := NewDefiSyncService(mockDS, mockWS, proto)

	addr := common.HexToAddress("0xdeadbeef00000000000000000000000000000000")
	proto.On("GetPositions", mock.Anything, addr).Return(nil, errors.New("node unreachable"))

	// Should not return an error — per-protocol errors are logged and skipped.
	err := svc.SyncPositions(context.Background(), 1, addr)
	assert.NoError(t, err)
}

func TestDefiSyncService_GetPositions(t *testing.T) {
	mockDS := new(store.MockDefiStore)
	mockWS := new(store.MockWalletStore)
	svc := NewDefiSyncService(mockDS, mockWS)

	mockDS.On("GetPositionsByUserID", uint64(1)).Return([]model.UserDefiPosition{
		{ID: 1, WalletID: 2, Protocol: "uniswap-v3", PositionType: "lp"},
	}, nil)

	positions, err := svc.GetPositions(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, positions, 1)
	assert.Equal(t, "uniswap-v3", positions[0].Protocol)
}

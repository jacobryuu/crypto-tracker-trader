package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"crypto-tracker-trader/internal/model"
	"crypto-tracker-trader/internal/store"

	storemod "crypto-tracker-trader/internal/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- mock PriceFetcher ---

type mockPriceFetcher struct{ mock.Mock }

func (m *mockPriceFetcher) FetchPrices(ctx context.Context, symbols []string) (map[string]string, error) {
	args := m.Called(ctx, symbols)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]string), args.Error(1)
}

// --- helpers ---

func newTestPriceService(mf *mockPriceFetcher, ms *storemod.MockPriceStore) *PriceService {
	return NewPriceService(mf, ms, "test")
}

// --- tests ---

func TestPriceService_FetchAndSave_Success(t *testing.T) {
	mf := new(mockPriceFetcher)
	ms := new(storemod.MockPriceStore)
	svc := newTestPriceService(mf, ms)

	prices := map[string]string{"BTC": "65000.00000000", "ETH": "3500.00000000"}
	mf.On("FetchPrices", mock.Anything, []string{"BTC", "ETH"}).Return(prices, nil)
	ms.On("SavePrice", "BTC", "65000.00000000", "test").Return(nil)
	ms.On("SavePrice", "ETH", "3500.00000000", "test").Return(nil)

	err := svc.FetchAndSave(context.Background(), []string{"BTC", "ETH"})
	require.NoError(t, err)
	mf.AssertExpectations(t)
	ms.AssertExpectations(t)
}

func TestPriceService_FetchAndSave_FetcherError(t *testing.T) {
	mf := new(mockPriceFetcher)
	ms := new(storemod.MockPriceStore)
	svc := newTestPriceService(mf, ms)

	mf.On("FetchPrices", mock.Anything, mock.Anything).Return(nil, errors.New("network error"))

	err := svc.FetchAndSave(context.Background(), []string{"BTC"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network error")
	ms.AssertNotCalled(t, "SavePrice")
}

func TestPriceService_FetchAndSave_SaveErrorContinues(t *testing.T) {
	// SavePrice failure for one symbol should not abort saving others.
	mf := new(mockPriceFetcher)
	ms := new(storemod.MockPriceStore)
	svc := newTestPriceService(mf, ms)

	prices := map[string]string{"BTC": "65000.00000000", "ETH": "3500.00000000"}
	mf.On("FetchPrices", mock.Anything, mock.Anything).Return(prices, nil)
	ms.On("SavePrice", "BTC", "65000.00000000", "test").Return(errors.New("db error"))
	ms.On("SavePrice", "ETH", "3500.00000000", "test").Return(nil)

	// FetchAndSave itself should not return error even if individual saves fail.
	err := svc.FetchAndSave(context.Background(), []string{"BTC", "ETH"})
	assert.NoError(t, err)
	ms.AssertExpectations(t)
}

func TestPriceService_GetLatestPrice_Success(t *testing.T) {
	mf := new(mockPriceFetcher)
	ms := new(storemod.MockPriceStore)
	svc := newTestPriceService(mf, ms)

	expected := &model.AssetPrice{Symbol: "BTC", PriceUSD: "65000.00000000", Source: "test"}
	ms.On("GetLatestPrice", "BTC").Return(expected, nil)

	price, err := svc.GetLatestPrice(context.Background(), "BTC")
	require.NoError(t, err)
	assert.Equal(t, "65000.00000000", price.PriceUSD)
}

func TestPriceService_GetLatestPrice_NotFound(t *testing.T) {
	mf := new(mockPriceFetcher)
	ms := new(storemod.MockPriceStore)
	svc := newTestPriceService(mf, ms)

	ms.On("GetLatestPrice", "UNKNOWN").Return(nil, store.ErrNotFound)

	_, err := svc.GetLatestPrice(context.Background(), "UNKNOWN")
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestPriceService_GetPriceHistory_EmptyBecomesSlice(t *testing.T) {
	mf := new(mockPriceFetcher)
	ms := new(storemod.MockPriceStore)
	svc := newTestPriceService(mf, ms)

	ms.On("GetPriceHistory", "BTC", 10).Return(nil, nil)

	history, err := svc.GetPriceHistory(context.Background(), "BTC", 10)
	require.NoError(t, err)
	assert.NotNil(t, history)
	assert.Empty(t, history)
}

func TestPriceService_StartSync_StopsOnContextCancel(t *testing.T) {
	mf := new(mockPriceFetcher)
	ms := new(storemod.MockPriceStore)
	svc := newTestPriceService(mf, ms)

	// Allow any number of FetchPrices + SavePrice calls.
	mf.On("FetchPrices", mock.Anything, mock.Anything).Return(map[string]string{"BTC": "1.0"}, nil).Maybe()
	ms.On("SavePrice", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	ctx, cancel := context.WithCancel(context.Background())
	svc.StartSync(ctx, []string{"BTC"}, 50*time.Millisecond)

	time.Sleep(120 * time.Millisecond) // Let at least one tick run.
	cancel()
	time.Sleep(20 * time.Millisecond) // Allow goroutine to observe cancellation.

	mf.AssertCalled(t, "FetchPrices", mock.Anything, mock.Anything)
}

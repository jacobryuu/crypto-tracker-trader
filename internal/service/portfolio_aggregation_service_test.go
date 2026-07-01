package service

import (
	"context"
	"testing"
	"time"

	"crypto-tracker-trader/internal/model"
	"crypto-tracker-trader/internal/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAggregationService() (*PortfolioAggregationService, *store.MockWalletStore, *store.MockPriceStore, *store.MockExchangeStore, *store.MockDefiStore) {
	ws := new(store.MockWalletStore)
	ps := new(store.MockPriceStore)
	es := new(store.MockExchangeStore)
	ds := new(store.MockDefiStore)
	svc := NewPortfolioAggregationService(ws, ps, es, ds)
	return svc, ws, ps, es, ds
}

func TestGetSummary_EmptyPortfolio(t *testing.T) {
	svc, ws, _, es, ds := newTestAggregationService()

	ws.On("GetWalletsByUserID", uint64(1)).Return([]model.UserWallet{}, nil)
	es.On("GetBalancesByUserID", uint64(1)).Return([]model.ExchangeBalance{}, nil)
	es.On("GetCredentialsByUserID", uint64(1)).Return([]model.ExchangeCredential{}, nil)
	ds.On("GetPositionsByUserID", uint64(1)).Return([]model.UserDefiPosition{}, nil)

	summary, err := svc.GetSummary(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "0.00000000", summary.TotalUSD)
	assert.Empty(t, summary.BySymbol)
}

func TestGetSummary_WalletAssets(t *testing.T) {
	svc, ws, ps, es, ds := newTestAggregationService()

	ws.On("GetWalletsByUserID", uint64(1)).Return([]model.UserWallet{
		{ID: 1, UserID: 1, Chain: "ethereum"},
	}, nil)
	ws.On("GetAssetsByWalletID", uint64(1)).Return([]model.UserAsset{
		{Symbol: "ETH", Balance: "2.5"},
		{Symbol: "USDC", Balance: "1000"},
	}, nil)
	ps.On("GetLatestPrice", "ETH").Return(&model.AssetPrice{Symbol: "ETH", PriceUSD: "3000.00000000", FetchedAt: time.Now()}, nil)
	ps.On("GetLatestPrice", "USDC").Return(&model.AssetPrice{Symbol: "USDC", PriceUSD: "1.00000000", FetchedAt: time.Now()}, nil)
	es.On("GetBalancesByUserID", uint64(1)).Return([]model.ExchangeBalance{}, nil)
	es.On("GetCredentialsByUserID", uint64(1)).Return([]model.ExchangeCredential{}, nil)
	ds.On("GetPositionsByUserID", uint64(1)).Return([]model.UserDefiPosition{}, nil)

	summary, err := svc.GetSummary(context.Background(), 1)
	require.NoError(t, err)
	// 2.5 ETH × $3000 + 1000 USDC × $1 = $7500 + $1000 = $8500
	assert.Equal(t, "8500.00000000", summary.TotalUSD)
	assert.Equal(t, "8500.00000000", summary.WalletUSD)
	assert.Equal(t, "0.00000000", summary.ExchangeUSD)
	assert.Len(t, summary.BySymbol, 2)
}

func TestGetSummary_ExchangeBalances(t *testing.T) {
	svc, ws, ps, es, ds := newTestAggregationService()

	ws.On("GetWalletsByUserID", uint64(1)).Return([]model.UserWallet{}, nil)
	es.On("GetBalancesByUserID", uint64(1)).Return([]model.ExchangeBalance{
		{ID: 1, CredentialID: 1, UserID: 1, Symbol: "BTC", FreeBalance: "0.5", LockedBalance: "0"},
	}, nil)
	es.On("GetCredentialsByUserID", uint64(1)).Return([]model.ExchangeCredential{
		{ID: 1, Exchange: "binance"},
	}, nil)
	ps.On("GetLatestPrice", "BTC").Return(&model.AssetPrice{Symbol: "BTC", PriceUSD: "60000.00000000", FetchedAt: time.Now()}, nil)
	ds.On("GetPositionsByUserID", uint64(1)).Return([]model.UserDefiPosition{}, nil)

	summary, err := svc.GetSummary(context.Background(), 1)
	require.NoError(t, err)
	// 0.5 BTC × $60000 = $30000
	assert.Equal(t, "30000.00000000", summary.TotalUSD)
	assert.Equal(t, "30000.00000000", summary.ExchangeUSD)
	assert.NotEmpty(t, summary.BySource)
	assert.Equal(t, "exchange:binance", summary.BySource[0].Source)
}

func TestGetSummary_UnknownSymbolPrice(t *testing.T) {
	svc, ws, ps, es, ds := newTestAggregationService()

	ws.On("GetWalletsByUserID", uint64(1)).Return([]model.UserWallet{
		{ID: 1, Chain: "ethereum"},
	}, nil)
	ws.On("GetAssetsByWalletID", uint64(1)).Return([]model.UserAsset{
		{Symbol: "UNKNOWN_TOKEN", Balance: "100"},
	}, nil)
	ps.On("GetLatestPrice", "UNKNOWN_TOKEN").Return(nil, store.ErrNotFound)
	es.On("GetBalancesByUserID", uint64(1)).Return([]model.ExchangeBalance{}, nil)
	es.On("GetCredentialsByUserID", uint64(1)).Return([]model.ExchangeCredential{}, nil)
	ds.On("GetPositionsByUserID", uint64(1)).Return([]model.UserDefiPosition{}, nil)

	summary, err := svc.GetSummary(context.Background(), 1)
	require.NoError(t, err)
	// Unknown token: price = 0, total = 0
	assert.Equal(t, "0.00000000", summary.TotalUSD)
	ps.AssertExpectations(t)
}

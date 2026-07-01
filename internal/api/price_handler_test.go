package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"crypto-tracker-trader/internal/model"
	"crypto-tracker-trader/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func priceSetup(t *testing.T) (*gin.Engine, *MockPriceManager, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	prm := new(MockPriceManager)
	a := NewAPI(APIConfig{
		PortfolioService:    new(MockPortfolioManager),
		BlockchainService:   new(MockBlockchainDataFetcher),
		UserService:         new(MockUserManager),
		WalletService:       new(MockWalletManager),
		PriceService:        prm,
		ExchangeService:     new(MockExchangeManager),
		DefiService:         new(MockDefiManager),
		PortfolioAggregator: new(MockPortfolioAggregator),
		JWTSecret:           testJWTSecret,
		JWTExpiryHours:      testJWTExpiry,
		PriceSymbols:        []string{"BTC", "ETH"},
	})
	a.RegisterRoutes(r)
	token := mustGenerateToken(t, 1, testJWTSecret)
	return r, prm, token
}

func TestGetLatestPrice(t *testing.T) {
	t.Run("success returns price", func(t *testing.T) {
		r, prm, token := priceSetup(t)
		expected := &model.AssetPrice{
			ID: 1, Symbol: "BTC", PriceUSD: "65000.00000000",
			Source: "coingecko", FetchedAt: time.Now(),
		}
		prm.On("GetLatestPrice", mock.Anything, "BTC").Return(expected, nil)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodGet, "/api/v1/prices/BTC", nil, token))

		assert.Equal(t, http.StatusOK, w.Code)
		var resp model.AssetPrice
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "BTC", resp.Symbol)
		assert.Equal(t, "65000.00000000", resp.PriceUSD)
		prm.AssertExpectations(t)
	})

	t.Run("symbol not found returns 404", func(t *testing.T) {
		r, prm, token := priceSetup(t)
		prm.On("GetLatestPrice", mock.Anything, "UNKNOWN").Return(nil, store.ErrNotFound)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodGet, "/api/v1/prices/UNKNOWN", nil, token))

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		r, prm, token := priceSetup(t)
		prm.On("GetLatestPrice", mock.Anything, "BTC").Return(nil, errors.New("db down"))

		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodGet, "/api/v1/prices/BTC", nil, token))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("no token returns 401", func(t *testing.T) {
		r, _, _ := priceSetup(t)
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/prices/BTC", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestGetPriceHistory(t *testing.T) {
	t.Run("returns history slice", func(t *testing.T) {
		r, prm, token := priceSetup(t)
		history := []model.AssetPrice{
			{ID: 1, Symbol: "ETH", PriceUSD: "3500.00000000", Source: "coingecko"},
			{ID: 2, Symbol: "ETH", PriceUSD: "3450.00000000", Source: "coingecko"},
		}
		prm.On("GetPriceHistory", mock.Anything, "ETH", 100).Return(history, nil)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodGet, "/api/v1/prices/ETH/history", nil, token))

		assert.Equal(t, http.StatusOK, w.Code)
		var resp []model.AssetPrice
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Len(t, resp, 2)
	})

	t.Run("custom limit parameter", func(t *testing.T) {
		r, prm, token := priceSetup(t)
		prm.On("GetPriceHistory", mock.Anything, "BTC", 10).Return([]model.AssetPrice{}, nil)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodGet, "/api/v1/prices/BTC/history?limit=10", nil, token))

		assert.Equal(t, http.StatusOK, w.Code)
		prm.AssertExpectations(t)
	})

	t.Run("invalid limit uses default 100", func(t *testing.T) {
		r, prm, token := priceSetup(t)
		prm.On("GetPriceHistory", mock.Anything, "BTC", 100).Return([]model.AssetPrice{}, nil)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodGet, "/api/v1/prices/BTC/history?limit=bad", nil, token))

		assert.Equal(t, http.StatusOK, w.Code)
		prm.AssertExpectations(t)
	})
}

func TestTriggerPriceSync(t *testing.T) {
	t.Run("returns 202 accepted", func(t *testing.T) {
		r, prm, token := priceSetup(t)
		prm.On("FetchAndSave", mock.Anything, mock.Anything).Return(nil).Maybe()

		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodPost, "/api/v1/prices/sync", nil, token))

		assert.Equal(t, http.StatusAccepted, w.Code)
		assert.Contains(t, w.Body.String(), "price sync triggered")
	})
}

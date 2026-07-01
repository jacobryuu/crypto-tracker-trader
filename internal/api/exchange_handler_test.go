package api

import (
	"bytes"
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

func exchangeSetup(t *testing.T) (*gin.Engine, *MockExchangeManager, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	em := new(MockExchangeManager)
	a := NewAPI(APIConfig{
		PortfolioService:    new(MockPortfolioManager),
		BlockchainService:   new(MockBlockchainDataFetcher),
		UserService:         new(MockUserManager),
		WalletService:       new(MockWalletManager),
		PriceService:        new(MockPriceManager),
		ExchangeService:     em,
		DefiService:         new(MockDefiManager),
		PortfolioAggregator: new(MockPortfolioAggregator),
		JWTSecret:           testJWTSecret,
		JWTExpiryHours:      testJWTExpiry,
		PriceSymbols:        []string{"BTC"},
	})
	a.RegisterRoutes(r)
	token := mustGenerateToken(t, 1, testJWTSecret)
	return r, em, token
}

func TestAddExchangeCredential(t *testing.T) {
	t.Run("success returns 201", func(t *testing.T) {
		r, em, token := exchangeSetup(t)
		cred := &model.ExchangeCredential{ID: 1, UserID: 1, Exchange: "binance", IsActive: true, CreatedAt: time.Now()}
		em.On("AddCredential", mock.Anything, uint64(1), "binance", "my-api-key", "my-secret").Return(cred, nil)

		body, _ := json.Marshal(gin.H{"exchange": "binance", "api_key": "my-api-key", "api_secret": "my-secret"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodPost, "/api/v1/exchanges", body, token))

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp model.ExchangeCredential
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "binance", resp.Exchange)
		em.AssertExpectations(t)
	})

	t.Run("missing fields returns 400", func(t *testing.T) {
		r, _, token := exchangeSetup(t)
		body, _ := json.Marshal(gin.H{"exchange": "binance"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodPost, "/api/v1/exchanges", body, token))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("no token returns 401", func(t *testing.T) {
		r, _, _ := exchangeSetup(t)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/exchanges", bytes.NewBufferString(`{}`))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestListExchangeCredentials(t *testing.T) {
	t.Run("success returns list", func(t *testing.T) {
		r, em, token := exchangeSetup(t)
		em.On("GetCredentials", mock.Anything, uint64(1)).Return([]model.ExchangeCredential{
			{ID: 1, Exchange: "binance"},
		}, nil)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodGet, "/api/v1/exchanges", nil, token))

		assert.Equal(t, http.StatusOK, w.Code)
		var resp []model.ExchangeCredential
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Len(t, resp, 1)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		r, em, token := exchangeSetup(t)
		em.On("GetCredentials", mock.Anything, uint64(1)).Return(nil, errors.New("db error"))

		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodGet, "/api/v1/exchanges", nil, token))
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestDeleteExchangeCredential(t *testing.T) {
	t.Run("success returns 204", func(t *testing.T) {
		r, em, token := exchangeSetup(t)
		em.On("DeleteCredential", mock.Anything, uint64(1), uint64(1)).Return(nil)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodDelete, "/api/v1/exchanges/1", nil, token))
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		r, em, token := exchangeSetup(t)
		em.On("DeleteCredential", mock.Anything, uint64(99), uint64(1)).Return(store.ErrNotFound)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodDelete, "/api/v1/exchanges/99", nil, token))
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestSyncExchangeBalances(t *testing.T) {
	t.Run("success returns balances", func(t *testing.T) {
		r, em, token := exchangeSetup(t)
		em.On("SyncBalances", mock.Anything, uint64(1)).Return([]model.ExchangeBalance{
			{Symbol: "BTC", FreeBalance: "0.5"},
		}, nil)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodPost, "/api/v1/exchanges/1/sync", nil, token))
		assert.Equal(t, http.StatusOK, w.Code)
		var balances []model.ExchangeBalance
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &balances))
		assert.Len(t, balances, 1)
	})
}

func TestGetExchangeBalances(t *testing.T) {
	t.Run("success returns stored balances", func(t *testing.T) {
		r, em, token := exchangeSetup(t)
		em.On("GetBalances", mock.Anything, uint64(1)).Return([]model.ExchangeBalance{
			{Symbol: "ETH", FreeBalance: "5.0"},
		}, nil)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodGet, "/api/v1/exchanges/balances", nil, token))
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

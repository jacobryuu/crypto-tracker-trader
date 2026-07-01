package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"crypto-tracker-trader/internal/model"
	"crypto-tracker-trader/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// walletSetup creates a router with auth middleware and wallet routes using mock services.
func walletSetup(t *testing.T) (*gin.Engine, *MockWalletManager, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	wm := new(MockWalletManager)
	api := NewAPI(APIConfig{
		PortfolioService:    new(MockPortfolioManager),
		BlockchainService:   new(MockBlockchainDataFetcher),
		UserService:         new(MockUserManager),
		WalletService:       wm,
		PriceService:        new(MockPriceManager),
		ExchangeService:     new(MockExchangeManager),
		DefiService:         new(MockDefiManager),
		PortfolioAggregator: new(MockPortfolioAggregator),
		JWTSecret:           testJWTSecret,
		JWTExpiryHours:      testJWTExpiry,
		PriceSymbols:        []string{"BTC", "ETH"},
	})
	api.RegisterRoutes(r)
	token := mustGenerateToken(t, 1, testJWTSecret)
	return r, wm, token
}

func authReq(method, path string, body []byte, token string) *http.Request {
	var req *http.Request
	if body != nil {
		req, _ = http.NewRequest(method, path, bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, path, nil)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func TestAddWallet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, wm, token := walletSetup(t)
		wallet := &model.UserWallet{ID: 1, UserID: 1, Chain: "ethereum", Address: "0xABC", Label: "main"}
		wm.On("AddWallet", uint64(1), "ethereum", "0xABC", "main").Return(wallet, nil)

		body, _ := json.Marshal(gin.H{"chain": "ethereum", "address": "0xABC", "label": "main"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodPost, "/api/v1/wallets", body, token))

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), `"chain":"ethereum"`)
		wm.AssertExpectations(t)
	})

	t.Run("missing required fields returns 400", func(t *testing.T) {
		r, wm, token := walletSetup(t)
		body, _ := json.Marshal(gin.H{"chain": "ethereum"}) // missing address
		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodPost, "/api/v1/wallets", body, token))

		assert.Equal(t, http.StatusBadRequest, w.Code)
		wm.AssertNotCalled(t, "AddWallet")
	})

	t.Run("duplicate address returns 409", func(t *testing.T) {
		r, wm, token := walletSetup(t)
		wm.On("AddWallet", uint64(1), "ethereum", "0xDUP", "").
			Return(nil, errors.New("wallet address already registered for this chain"))

		body, _ := json.Marshal(gin.H{"chain": "ethereum", "address": "0xDUP"})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodPost, "/api/v1/wallets", body, token))

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("no token returns 401", func(t *testing.T) {
		r, _, _ := walletSetup(t)
		body, _ := json.Marshal(gin.H{"chain": "ethereum", "address": "0xABC"})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/wallets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestListWallets(t *testing.T) {
	t.Run("success returns wallets", func(t *testing.T) {
		r, wm, token := walletSetup(t)
		wallets := []model.UserWallet{{ID: 1, Chain: "ethereum", Address: "0xABC"}}
		wm.On("GetWallets", uint64(1)).Return(wallets, nil)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodGet, "/api/v1/wallets", nil, token))

		assert.Equal(t, http.StatusOK, w.Code)
		var resp []model.UserWallet
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Len(t, resp, 1)
	})

	t.Run("service error returns 500", func(t *testing.T) {
		r, wm, token := walletSetup(t)
		wm.On("GetWallets", uint64(1)).Return(nil, errors.New("db error"))

		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodGet, "/api/v1/wallets", nil, token))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestDeleteWallet(t *testing.T) {
	t.Run("success returns 204", func(t *testing.T) {
		r, wm, token := walletSetup(t)
		wm.On("DeleteWallet", uint64(5), uint64(1)).Return(nil)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodDelete, "/api/v1/wallets/5", nil, token))

		assert.Equal(t, http.StatusNoContent, w.Code)
		wm.AssertExpectations(t)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		r, wm, token := walletSetup(t)
		wm.On("DeleteWallet", uint64(99), uint64(1)).Return(store.ErrNotFound)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodDelete, "/api/v1/wallets/99", nil, token))

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		r, _, token := walletSetup(t)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodDelete, "/api/v1/wallets/abc", nil, token))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestListWalletAssets(t *testing.T) {
	t.Run("success returns assets", func(t *testing.T) {
		r, wm, token := walletSetup(t)
		assets := []model.UserAsset{{ID: 1, WalletID: 2, Symbol: "ETH", Balance: "1000000000000000000"}}
		wm.On("GetWalletAssets", uint64(2), uint64(1)).Return(assets, nil)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodGet, "/api/v1/wallets/2/assets", nil, token))

		assert.Equal(t, http.StatusOK, w.Code)
		var resp []model.UserAsset
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Len(t, resp, 1)
		assert.Equal(t, "ETH", resp[0].Symbol)
	})

	t.Run("wallet not found returns 404", func(t *testing.T) {
		r, wm, token := walletSetup(t)
		wm.On("GetWalletAssets", uint64(99), uint64(1)).Return(nil, store.ErrNotFound)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodGet, "/api/v1/wallets/99/assets", nil, token))

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("invalid wallet id returns 400", func(t *testing.T) {
		r, _, token := walletSetup(t)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, authReq(http.MethodGet, "/api/v1/wallets/xyz/assets", nil, token))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestWalletRoutes_Unauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	api := NewAPI(APIConfig{
		PortfolioService:    new(MockPortfolioManager),
		BlockchainService:   new(MockBlockchainDataFetcher),
		UserService:         new(MockUserManager),
		WalletService:       new(MockWalletManager),
		PriceService:        new(MockPriceManager),
		ExchangeService:     new(MockExchangeManager),
		DefiService:         new(MockDefiManager),
		PortfolioAggregator: new(MockPortfolioAggregator),
		JWTSecret:           testJWTSecret,
		JWTExpiryHours:      testJWTExpiry,
		PriceSymbols:        []string{"BTC"},
	})
	api.RegisterRoutes(r)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/wallets"},
		{http.MethodPost, "/api/v1/wallets"},
		{http.MethodDelete, "/api/v1/wallets/1"},
		{http.MethodGet, "/api/v1/wallets/1/assets"},
	}
	for _, route := range routes {
		req, _ := http.NewRequest(route.method, route.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code, "expected 401 for %s %s", route.method, route.path)
	}
}

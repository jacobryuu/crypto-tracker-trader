package binance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBalances_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v3/account", r.URL.Path)
		assert.NotEmpty(t, r.URL.Query().Get("timestamp"))
		assert.NotEmpty(t, r.URL.Query().Get("signature"))
		assert.Equal(t, "test-api-key", r.Header.Get("X-MBX-APIKEY"))

		json.NewEncoder(w).Encode(map[string]interface{}{
			"balances": []map[string]string{
				{"asset": "BTC", "free": "0.50000000", "locked": "0.00000000"},
				{"asset": "ETH", "free": "5.00000000", "locked": "1.00000000"},
				{"asset": "DUST", "free": "0.00000000", "locked": "0.00000000"}, // should be filtered
			},
		})
	}))
	defer srv.Close()

	client := NewWithBaseURL("test-api-key", "test-secret", srv.URL)
	balances, err := client.GetBalances(context.Background())
	require.NoError(t, err)
	assert.Len(t, balances, 2, "zero-balance DUST entry should be filtered out")
	assert.Equal(t, "BTC", balances[0].Symbol)
	assert.Equal(t, "0.50000000", balances[0].Free)
	assert.Equal(t, "ETH", balances[1].Symbol)
	assert.Equal(t, "1.00000000", balances[1].Locked)
}

func TestGetBalances_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": -2014,
			"msg":  "API-key format invalid.",
		})
	}))
	defer srv.Close()

	client := NewWithBaseURL("bad-key", "bad-secret", srv.URL)
	_, err := client.GetBalances(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "-2014")
}

func TestGetTradeHistory_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v3/myTrades", r.URL.Path)
		assert.Equal(t, "BTCUSDT", r.URL.Query().Get("symbol"))

		json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"symbol":   "BTCUSDT",
				"id":       12345,
				"price":    "60000.00",
				"qty":      "0.01",
				"isBuyer":  true,
				"time":     time.Now().UnixMilli(),
			},
			{
				"symbol":  "BTCUSDT",
				"id":      12346,
				"price":   "61000.00",
				"qty":     "0.005",
				"isBuyer": false,
				"time":    time.Now().UnixMilli(),
			},
		})
	}))
	defer srv.Close()

	client := NewWithBaseURL("test-key", "test-secret", srv.URL)
	trades, err := client.GetTradeHistory(context.Background(), "BTCUSDT", time.Now().Add(-24*time.Hour))
	require.NoError(t, err)
	assert.Len(t, trades, 2)
	assert.Equal(t, "BUY", trades[0].Side)
	assert.Equal(t, "SELL", trades[1].Side)
	assert.Equal(t, "12345", trades[0].OrderID)
}

func TestSign_HMAC(t *testing.T) {
	c := &Client{apiSecret: "NhqPtmdSJYdKjVHjA7PZj4Mge3R5YNiP1e3UZjInClVN65XAbvqqM6A7H5fATj0j"}
	// Known vector from Binance docs
	result := c.sign("symbol=LTCBTC&side=BUY&type=LIMIT&timeInForce=GTC&quantity=1&price=0.1&recvWindow=5000&timestamp=1499827319559")
	assert.Equal(t, "c8db56825ae71d6d79447849e617115f4a920fa2acdcab2b053c4b2838bd6b71", result)
}

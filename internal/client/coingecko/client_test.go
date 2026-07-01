package coingecko

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockServer(t *testing.T, status int, body interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}))
}

func TestFetchPrices_Success(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]map[string]float64{
		"bitcoin":  {"usd": 65000.12345678},
		"ethereum": {"usd": 3500.99},
	})
	defer srv.Close()

	c := NewWithBaseURL(srv.URL)
	prices, err := c.FetchPrices(context.Background(), []string{"BTC", "ETH"})

	require.NoError(t, err)
	assert.Equal(t, "65000.12345678", prices["BTC"])
	assert.Equal(t, "3500.99000000", prices["ETH"])
}

func TestFetchPrices_UnknownSymbolSkipped(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]map[string]float64{
		"bitcoin": {"usd": 65000.0},
	})
	defer srv.Close()

	c := NewWithBaseURL(srv.URL)
	prices, err := c.FetchPrices(context.Background(), []string{"BTC", "UNKNOWN_TOKEN"})

	require.NoError(t, err)
	assert.Contains(t, prices, "BTC")
	assert.NotContains(t, prices, "UNKNOWN_TOKEN")
}

func TestFetchPrices_EmptySymbols(t *testing.T) {
	c := NewWithBaseURL("http://unused")
	prices, err := c.FetchPrices(context.Background(), []string{})
	require.NoError(t, err)
	assert.Empty(t, prices)
}

func TestFetchPrices_AllUnknownSymbols(t *testing.T) {
	c := NewWithBaseURL("http://unused")
	prices, err := c.FetchPrices(context.Background(), []string{"UNKNOWN1", "UNKNOWN2"})
	require.NoError(t, err)
	assert.Empty(t, prices)
}

func TestFetchPrices_SymbolNormalization(t *testing.T) {
	srv := mockServer(t, http.StatusOK, map[string]map[string]float64{
		"bitcoin": {"usd": 65000.0},
	})
	defer srv.Close()

	c := NewWithBaseURL(srv.URL)
	// Lowercase and spaces are normalised.
	prices, err := c.FetchPrices(context.Background(), []string{"btc", " eth "})
	require.NoError(t, err)
	assert.Contains(t, prices, "BTC")
}

func TestFetchPrices_ServerError(t *testing.T) {
	srv := mockServer(t, http.StatusInternalServerError, nil)
	defer srv.Close()

	c := NewWithBaseURL(srv.URL)
	_, err := c.FetchPrices(context.Background(), []string{"BTC"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status 500")
}

func TestFetchPrices_RateLimitRetry(t *testing.T) {
	attempt := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]map[string]float64{"bitcoin": {"usd": 1.0}})
	}))
	defer srv.Close()

	c := NewWithBaseURL(srv.URL)
	// Use a very short backoff for tests by calling with a cancelable context.
	prices, err := c.FetchPrices(context.Background(), []string{"BTC"})
	require.NoError(t, err)
	assert.Contains(t, prices, "BTC")
	assert.Equal(t, 2, attempt, "should retry once after 429")
}

func TestFetchPrices_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Never respond — force context cancellation to kick in.
		<-r.Context().Done()
	}))
	defer srv.Close()

	c := NewWithBaseURL(srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := c.FetchPrices(ctx, []string{"BTC"})
	assert.Error(t, err)
}

func TestBuildIDParams(t *testing.T) {
	ids, byID := buildIDParams([]string{"BTC", "ETH", "UNKNOWN"})
	assert.Len(t, ids, 2)
	assert.Equal(t, "BTC", byID["bitcoin"])
	assert.Equal(t, "ETH", byID["ethereum"])
	assert.NotContains(t, byID, "UNKNOWN")
}

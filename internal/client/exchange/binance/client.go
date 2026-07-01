package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"crypto-tracker-trader/internal/client/exchange"
)

const defaultBaseURL = "https://api.binance.com"

// Client implements exchange.ExchangeClient for Binance REST API v3.
type Client struct {
	apiKey    string
	apiSecret string
	baseURL   string
	http      *http.Client
}

// New returns a Binance client targeting the production endpoint.
func New(apiKey, apiSecret string) *Client {
	return &Client{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		baseURL:   defaultBaseURL,
		http:      &http.Client{Timeout: 10 * time.Second},
	}
}

// NewWithBaseURL returns a Binance client with a custom base URL (useful for tests/mocking).
func NewWithBaseURL(apiKey, apiSecret, baseURL string) *Client {
	return &Client{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		baseURL:   baseURL,
		http:      &http.Client{Timeout: 5 * time.Second},
	}
}

// sign computes HMAC-SHA256(queryString, apiSecret) as a lowercase hex string.
func (c *Client) sign(queryString string) string {
	mac := hmac.New(sha256.New, []byte(c.apiSecret))
	mac.Write([]byte(queryString))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

// GetBalances fetches account balances, skipping dust (zero-balance) entries.
func (c *Client) GetBalances(ctx context.Context) ([]exchange.Balance, error) {
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	query := "timestamp=" + ts
	sig := c.sign(query)

	reqURL := fmt.Sprintf("%s/api/v3/account?%s&signature=%s", c.baseURL, query, sig)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("binance: building request: %w", err)
	}
	req.Header.Set("X-MBX-APIKEY", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("binance: HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return nil, fmt.Errorf("binance: API error %d: %s", apiErr.Code, apiErr.Msg)
	}

	var account struct {
		Balances []struct {
			Asset  string `json:"asset"`
			Free   string `json:"free"`
			Locked string `json:"locked"`
		} `json:"balances"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
		return nil, fmt.Errorf("binance: decoding account: %w", err)
	}

	balances := make([]exchange.Balance, 0, len(account.Balances))
	for _, b := range account.Balances {
		if b.Free == "0.00000000" && b.Locked == "0.00000000" {
			continue
		}
		balances = append(balances, exchange.Balance{
			Symbol: b.Asset,
			Free:   b.Free,
			Locked: b.Locked,
		})
	}
	return balances, nil
}

// GetTradeHistory fetches executed trades for a symbol since the given time (up to 1000 records).
func (c *Client) GetTradeHistory(ctx context.Context, symbol string, since time.Time) ([]exchange.Trade, error) {
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	query := fmt.Sprintf("symbol=%s&startTime=%d&limit=1000&timestamp=%s",
		symbol, since.UnixMilli(), ts)
	sig := c.sign(query)

	reqURL := fmt.Sprintf("%s/api/v3/myTrades?%s&signature=%s", c.baseURL, query, sig)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("binance: building request: %w", err)
	}
	req.Header.Set("X-MBX-APIKEY", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("binance: HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return nil, fmt.Errorf("binance: API error %d: %s", apiErr.Code, apiErr.Msg)
	}

	var raw []struct {
		Symbol   string `json:"symbol"`
		ID       int64  `json:"id"`
		Price    string `json:"price"`
		Qty      string `json:"qty"`
		IsBuyer  bool   `json:"isBuyer"`
		TradeTime int64 `json:"time"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("binance: decoding trades: %w", err)
	}

	trades := make([]exchange.Trade, len(raw))
	for i, t := range raw {
		side := "SELL"
		if t.IsBuyer {
			side = "BUY"
		}
		trades[i] = exchange.Trade{
			Symbol:   t.Symbol,
			OrderID:  strconv.FormatInt(t.ID, 10),
			Price:    t.Price,
			Quantity: t.Qty,
			Side:     side,
			Time:     time.UnixMilli(t.TradeTime),
		}
	}
	return trades, nil
}

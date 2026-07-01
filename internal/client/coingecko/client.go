// Package coingecko provides a CoinGecko REST API client that implements
// the service.PriceFetcher interface.
package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL    = "https://api.coingecko.com/api/v3"
	defaultHTTPTimeout = 10 * time.Second
	// CoinGecko free tier: 30 calls/min → burst of 1 call per 2s is safe.
	minRequestInterval = 2 * time.Second
)

// symbolToID maps uppercase ticker symbols to CoinGecko coin IDs.
var symbolToID = map[string]string{
	"BTC":   "bitcoin",
	"ETH":   "ethereum",
	"BNB":   "binancecoin",
	"SOL":   "solana",
	"USDT":  "tether",
	"USDC":  "usd-coin",
	"ADA":   "cardano",
	"DOGE":  "dogecoin",
	"TRX":   "tron",
	"DOT":   "polkadot",
	"MATIC": "matic-network",
	"LTC":   "litecoin",
	"LINK":  "chainlink",
	"AVAX":  "avalanche-2",
	"UNI":   "uniswap",
	"ATOM":  "cosmos",
	"XLM":   "stellar",
	"BCH":   "bitcoin-cash",
	"FIL":   "filecoin",
	"NEAR":  "near",
}

// Client fetches USD prices from CoinGecko.
type Client struct {
	baseURL     string
	httpClient  *http.Client
	lastRequest time.Time
}

// New creates a CoinGecko client with default settings.
func New() *Client {
	return &Client{
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: defaultHTTPTimeout},
	}
}

// NewWithBaseURL creates a client pointing at a custom base URL (useful for tests).
func NewWithBaseURL(baseURL string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: defaultHTTPTimeout},
	}
}

// FetchPrices returns a map[SYMBOL]priceUSDString for the requested symbols.
// Symbols not present in the known mapping are silently skipped.
// Rate-limiting: waits until minRequestInterval has elapsed since the last call.
func (c *Client) FetchPrices(ctx context.Context, symbols []string) (map[string]string, error) {
	ids, symbolByID := buildIDParams(symbols)
	if len(ids) == 0 {
		return map[string]string{}, nil
	}

	// Honour rate limit.
	c.waitForRateLimit(ctx)

	url := fmt.Sprintf("%s/simple/price?ids=%s&vs_currencies=usd",
		c.baseURL, strings.Join(ids, ","))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.doWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	c.lastRequest = time.Now()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	// CoinGecko response: { "bitcoin": { "usd": 65000.12 }, ... }
	var raw map[string]map[string]float64
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	result := make(map[string]string, len(symbolByID))
	for id, usd := range raw {
		symbol, ok := symbolByID[id]
		if !ok {
			continue
		}
		result[symbol] = strconv.FormatFloat(usd["usd"], 'f', 8, 64)
	}
	return result, nil
}

// doWithRetry performs the HTTP request, retrying on 429 with exponential back-off.
func (c *Client) doWithRetry(req *http.Request) (*http.Response, error) {
	backoff := time.Second
	for attempt := 0; attempt < 3; attempt++ {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("http request: %w", err)
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(backoff):
				backoff *= 2
				continue
			}
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
		}
		return resp, nil
	}
	return nil, fmt.Errorf("max retries exceeded")
}

// waitForRateLimit sleeps until minRequestInterval has passed since lastRequest.
func (c *Client) waitForRateLimit(ctx context.Context) {
	if c.lastRequest.IsZero() {
		return
	}
	elapsed := time.Since(c.lastRequest)
	if elapsed < minRequestInterval {
		select {
		case <-ctx.Done():
		case <-time.After(minRequestInterval - elapsed):
		}
	}
}

// buildIDParams converts symbols to CoinGecko IDs and returns:
//   - ids: slice of CoinGecko IDs to include in the query
//   - symbolByID: reverse map from ID → SYMBOL (uppercase)
func buildIDParams(symbols []string) ([]string, map[string]string) {
	ids := make([]string, 0, len(symbols))
	symbolByID := make(map[string]string, len(symbols))
	for _, sym := range symbols {
		upper := strings.ToUpper(strings.TrimSpace(sym))
		id, ok := symbolToID[upper]
		if !ok {
			continue
		}
		ids = append(ids, id)
		symbolByID[id] = upper
	}
	return ids, symbolByID
}

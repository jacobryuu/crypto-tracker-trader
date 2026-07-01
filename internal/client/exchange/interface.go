package exchange

import (
	"context"
	"time"
)

// Balance represents an asset balance at an exchange.
type Balance struct {
	Symbol  string `json:"symbol"`
	Free    string `json:"free"`
	Locked  string `json:"locked"`
}

// Trade represents a single executed trade.
type Trade struct {
	Symbol   string    `json:"symbol"`
	OrderID  string    `json:"order_id"`
	Price    string    `json:"price"`
	Quantity string    `json:"quantity"`
	Side     string    `json:"side"` // "BUY" or "SELL"
	Time     time.Time `json:"time"`
}

// ExchangeClient is the interface for fetching data from a centralised exchange.
type ExchangeClient interface {
	GetBalances(ctx context.Context) ([]Balance, error)
	GetTradeHistory(ctx context.Context, symbol string, since time.Time) ([]Trade, error)
}

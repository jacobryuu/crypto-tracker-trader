package model

import "time"

// AssetPrice holds a single price snapshot for a cryptocurrency symbol.
type AssetPrice struct {
	ID        uint64    `json:"id"`
	Symbol    string    `json:"symbol"`
	PriceUSD  string    `json:"price_usd"`
	Source    string    `json:"source"`
	FetchedAt time.Time `json:"fetched_at"`
}

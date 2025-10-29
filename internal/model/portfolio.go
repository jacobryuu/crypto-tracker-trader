package model

import "time"

type PortfolioAsset struct {
    AssetID  string  `json:"asset_id"`
    Quantity float64 `json:"quantity"`
    Value    float64 `json:"value"` // Value in a reference currency (e.g., USD)
}

type PortfolioSnapshot struct {
    Timestamp time.Time        `json:"timestamp"`
    Assets    []PortfolioAsset `json:"assets"`
    TotalValue float64          `json:"total_value"`
}

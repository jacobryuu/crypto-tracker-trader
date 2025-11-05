package model

import "time"

type PortfolioAsset struct {
    AssetID  string `json:"asset_id"`
    Quantity string `json:"quantity"`
    Value    string `json:"value"` // Value in a reference currency (e.g., USD)
}

type PortfolioSnapshot struct {
    Timestamp time.Time        `json:"timestamp"`
    Assets    []PortfolioAsset `json:"assets"`
    TotalValue string          `json:"total_value"`
}

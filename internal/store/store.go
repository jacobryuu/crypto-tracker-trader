package store

import "crypto-tracker-trader/internal/model"

type PortfolioStoreInterface interface {
    AddSnapshot(snapshot model.PortfolioSnapshot) error
    GetHistory() ([]model.PortfolioSnapshot, error)
    Close()
}

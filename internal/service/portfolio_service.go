package service

import (
	"crypto-tracker-trader/internal/model"
	"crypto-tracker-trader/internal/store"
)

// PortfolioService implements the PortfolioManager interface.
type PortfolioService struct {
	portfolioStore store.PortfolioStoreInterface
}

func NewPortfolioService(portfolioStore store.PortfolioStoreInterface) PortfolioManager {
	return &PortfolioService{
		portfolioStore: portfolioStore,
	}
}

func (s *PortfolioService) GetPortfolioHistory() ([]model.PortfolioSnapshot, error) {
	return s.portfolioStore.GetHistory()
}

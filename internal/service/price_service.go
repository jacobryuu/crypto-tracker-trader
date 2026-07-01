package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"crypto-tracker-trader/internal/model"
	"crypto-tracker-trader/internal/store"
)

const defaultPriceSyncInterval = 60 * time.Second

// PriceService orchestrates price fetching from an external provider and
// persisting the results. It also runs a background sync loop.
type PriceService struct {
	fetcher    PriceFetcher
	priceStore store.PriceStoreInterface
	source     string
}

// NewPriceService creates a new PriceService.
// source identifies the data provider (e.g. "coingecko").
func NewPriceService(fetcher PriceFetcher, priceStore store.PriceStoreInterface, source string) *PriceService {
	if source == "" {
		source = "coingecko"
	}
	return &PriceService{fetcher: fetcher, priceStore: priceStore, source: source}
}

// FetchAndSave fetches current prices for the given symbols and persists them.
func (s *PriceService) FetchAndSave(ctx context.Context, symbols []string) error {
	prices, err := s.fetcher.FetchPrices(ctx, symbols)
	if err != nil {
		return fmt.Errorf("fetch prices: %w", err)
	}
	for symbol, priceUSD := range prices {
		if err := s.priceStore.SavePrice(symbol, priceUSD, s.source); err != nil {
			log.Printf("PriceService: failed to save price for %s: %v", symbol, err)
		}
	}
	return nil
}

// GetLatestPrice returns the most recent price for the given symbol.
func (s *PriceService) GetLatestPrice(ctx context.Context, symbol string) (*model.AssetPrice, error) {
	price, err := s.priceStore.GetLatestPrice(symbol)
	if err != nil {
		return nil, fmt.Errorf("get latest price: %w", err)
	}
	return price, nil
}

// GetPriceHistory returns recent price records for the given symbol.
func (s *PriceService) GetPriceHistory(ctx context.Context, symbol string, limit int) ([]model.AssetPrice, error) {
	prices, err := s.priceStore.GetPriceHistory(symbol, limit)
	if err != nil {
		return nil, fmt.Errorf("get price history: %w", err)
	}
	if prices == nil {
		prices = []model.AssetPrice{}
	}
	return prices, nil
}

// StartSync launches a background goroutine that calls FetchAndSave on the
// given interval until ctx is cancelled.
// It performs an immediate fetch on start, then ticks at each interval.
func (s *PriceService) StartSync(ctx context.Context, symbols []string, interval time.Duration) {
	if interval <= 0 {
		interval = defaultPriceSyncInterval
	}
	go func() {
		// Fetch immediately on startup.
		if err := s.FetchAndSave(ctx, symbols); err != nil {
			log.Printf("PriceService: initial sync error: %v", err)
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("PriceService: background sync stopped")
				return
			case <-ticker.C:
				if err := s.FetchAndSave(ctx, symbols); err != nil {
					log.Printf("PriceService: sync error: %v", err)
				}
			}
		}
	}()
}

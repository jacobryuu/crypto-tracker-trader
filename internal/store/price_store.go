package store

import (
	"context"
	"fmt"
	"strings"

	"crypto-tracker-trader/internal/model"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// PriceStore implements PriceStoreInterface for PostgreSQL.
type PriceStore struct {
	db *pgxpool.Pool
}

// NewPriceStore creates a new PriceStore.
func NewPriceStore(db *pgxpool.Pool) *PriceStore {
	return &PriceStore{db: db}
}

// SavePrice inserts a new price record for the given symbol.
func (s *PriceStore) SavePrice(symbol, priceUSD, source string) error {
	_, err := s.db.Exec(context.Background(),
		`INSERT INTO asset_prices (symbol, price_usd, source) VALUES ($1, $2, $3)`,
		strings.ToUpper(symbol), priceUSD, source,
	)
	if err != nil {
		return fmt.Errorf("save price: %w", err)
	}
	return nil
}

// GetLatestPrice returns the most recently saved price for the given symbol.
// Returns ErrNotFound if no price record exists.
func (s *PriceStore) GetLatestPrice(symbol string) (*model.AssetPrice, error) {
	var p model.AssetPrice
	err := s.db.QueryRow(context.Background(),
		`SELECT id, symbol, price_usd, source, fetched_at
		 FROM asset_prices
		 WHERE symbol = $1
		 ORDER BY fetched_at DESC
		 LIMIT 1`,
		strings.ToUpper(symbol),
	).Scan(&p.ID, &p.Symbol, &p.PriceUSD, &p.Source, &p.FetchedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get latest price: %w", err)
	}
	return &p, nil
}

// GetPriceHistory returns the last `limit` price records for the given symbol,
// newest first.
func (s *PriceStore) GetPriceHistory(symbol string, limit int) ([]model.AssetPrice, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(context.Background(),
		`SELECT id, symbol, price_usd, source, fetched_at
		 FROM asset_prices
		 WHERE symbol = $1
		 ORDER BY fetched_at DESC
		 LIMIT $2`,
		strings.ToUpper(symbol), limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get price history: %w", err)
	}
	defer rows.Close()

	var prices []model.AssetPrice
	for rows.Next() {
		var p model.AssetPrice
		if err := rows.Scan(&p.ID, &p.Symbol, &p.PriceUSD, &p.Source, &p.FetchedAt); err != nil {
			return nil, fmt.Errorf("scan price: %w", err)
		}
		prices = append(prices, p)
	}
	return prices, rows.Err()
}

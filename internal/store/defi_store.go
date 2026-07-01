package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"crypto-tracker-trader/internal/model"

	"github.com/jackc/pgx/v4/pgxpool"
	"gorm.io/datatypes"
)

// DefiStore handles persistence for user DeFi positions.
type DefiStore struct {
	pool *pgxpool.Pool
}

func NewDefiStore(pool *pgxpool.Pool) *DefiStore {
	return &DefiStore{pool: pool}
}

// UpsertPosition inserts or updates a DeFi position.
// Uniqueness is keyed on (wallet_id, protocol, position_type).
func (s *DefiStore) UpsertPosition(pos *model.UserDefiPosition) error {
	data, err := json.Marshal(pos.PositionJSON)
	if err != nil {
		return fmt.Errorf("defi store: marshalling position json: %w", err)
	}
	const q = `
		INSERT INTO user_defi_positions (wallet_id, protocol, position_type, position_json, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (wallet_id, protocol, position_type) DO UPDATE SET
			position_json = EXCLUDED.position_json,
			updated_at    = EXCLUDED.updated_at
		RETURNING id`
	return s.pool.QueryRow(context.Background(), q,
		pos.WalletID, pos.Protocol, pos.PositionType, data, time.Now(),
	).Scan(&pos.ID)
}

func (s *DefiStore) GetPositionsByWalletID(walletID uint64) ([]model.UserDefiPosition, error) {
	const q = `
		SELECT id, wallet_id, protocol, position_type, position_json, updated_at
		FROM user_defi_positions WHERE wallet_id = $1
		ORDER BY protocol, position_type`
	return s.queryPositions(q, walletID)
}

func (s *DefiStore) GetPositionsByUserID(userID uint64) ([]model.UserDefiPosition, error) {
	const q = `
		SELECT dp.id, dp.wallet_id, dp.protocol, dp.position_type, dp.position_json, dp.updated_at
		FROM user_defi_positions dp
		JOIN user_wallets w ON w.id = dp.wallet_id
		WHERE w.user_id = $1
		ORDER BY dp.protocol, dp.position_type`
	return s.queryPositions(q, userID)
}

func (s *DefiStore) queryPositions(query string, arg interface{}) ([]model.UserDefiPosition, error) {
	rows, err := s.pool.Query(context.Background(), query, arg)
	if err != nil {
		return nil, fmt.Errorf("defi store: query: %w", err)
	}
	defer rows.Close()

	var positions []model.UserDefiPosition
	for rows.Next() {
		var p model.UserDefiPosition
		var raw []byte
		if err := rows.Scan(&p.ID, &p.WalletID, &p.Protocol, &p.PositionType, &raw, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.PositionJSON = datatypes.JSON(raw)
		positions = append(positions, p)
	}
	return positions, rows.Err()
}

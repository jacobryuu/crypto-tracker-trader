package store

import (
    "context"
    "log"
    "time"

    "crypto-tracker-trader/internal/model"
    "github.com/jackc/pgx/v4"
)

type PortfolioStore struct {
    db *pgx.Conn
}

func NewPortfolioStore(databaseUrl string) *PortfolioStore {
    conn, err := pgx.Connect(context.Background(), databaseUrl)
    if err != nil {
        log.Fatalf("Unable to connect to database: %v\n", err)
    }

    return &PortfolioStore{db: conn}
}

func (s *PortfolioStore) Close() {
    s.db.Close(context.Background())
}

func (s *PortfolioStore) AddSnapshot(snapshot model.PortfolioSnapshot) error {
    tx, err := s.db.Begin(context.Background())
    if err != nil {
        return err
    }
    defer tx.Rollback(context.Background())

    var snapshotID int
    err = tx.QueryRow(context.Background(),
        "INSERT INTO portfolio_snapshots (timestamp, total_value) VALUES ($1, $2) RETURNING id",
        snapshot.Timestamp, snapshot.TotalValue).Scan(&snapshotID)
    if err != nil {
        return err
    }

    for _, asset := range snapshot.Assets {
        _, err := tx.Exec(context.Background(),
            "INSERT INTO portfolio_assets (snapshot_id, asset_id, quantity, value) VALUES ($1, $2, $3, $4)",
            snapshotID, asset.AssetID, asset.Quantity, asset.Value)
        if err != nil {
            return err
        }
    }

    return tx.Commit(context.Background())
}

func (s *PortfolioStore) GetHistory() ([]model.PortfolioSnapshot, error) {
    rows, err := s.db.Query(context.Background(),
        `SELECT s.id, s.timestamp, s.total_value, a.asset_id, a.quantity, a.value
         FROM portfolio_snapshots s
         JOIN portfolio_assets a ON s.id = a.snapshot_id
         ORDER BY s.timestamp DESC`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    snapshotsMap := make(map[int]*model.PortfolioSnapshot)

    for rows.Next() {
        var snapshotID int
        var timestamp time.Time
        var totalValue float64
        var asset model.PortfolioAsset

        err := rows.Scan(&snapshotID, &timestamp, &totalValue, &asset.AssetID, &asset.Quantity, &asset.Value)
        if err != nil {
            return nil, err
        }

        if _, ok := snapshotsMap[snapshotID]; !ok {
            snapshotsMap[snapshotID] = &model.PortfolioSnapshot{
                Timestamp:  timestamp,
                TotalValue: totalValue,
                Assets:     []model.PortfolioAsset{},
            }
        }
        snapshotsMap[snapshotID].Assets = append(snapshotsMap[snapshotID].Assets, asset)
    }

    // Convert map to slice and respect order
    var snapshots []model.PortfolioSnapshot
    // A bit of a hack to get ordered keys
    var keys []int
    for k := range snapshotsMap {
        keys = append(keys, k)
    }
    // sort.Sort(sort.Reverse(sort.IntSlice(keys)))

    for _, k := range keys {
        snapshots = append(snapshots, *snapshotsMap[k])
    }

    return snapshots, nil
}

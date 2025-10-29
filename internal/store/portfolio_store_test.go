package store

import (
    "context"
    "testing"
    "time"

    "crypto-tracker-trader/internal/model"
    "github.com/jackc/pgx/v4"
    "github.com/stretchr/testify/assert"
)

const testDatabaseURL = "postgres://user:password@localhost:5532/testdb?sslmode=disable" // Assuming a test database

func setupTestDB(t *testing.T) *pgx.Conn {
    conn, err := pgx.Connect(context.Background(), testDatabaseURL)
    if err != nil {
        t.Fatalf("Unable to connect to test database: %v", err)
    }

    // Clear existing tables
    _, err = conn.Exec(context.Background(), "DROP TABLE IF EXISTS portfolio_assets CASCADE;")
    assert.NoError(t, err)
    _, err = conn.Exec(context.Background(), "DROP TABLE IF EXISTS portfolio_snapshots CASCADE;")
    assert.NoError(t, err)

    // Create tables
    _, err = conn.Exec(context.Background(), `
        CREATE TABLE portfolio_snapshots (
            id SERIAL PRIMARY KEY,
            timestamp TIMESTAMPTZ NOT NULL,
            total_value NUMERIC(20, 8) NOT NULL
        );

        CREATE TABLE portfolio_assets (
            id SERIAL PRIMARY KEY,
            snapshot_id INTEGER NOT NULL REFERENCES portfolio_snapshots(id) ON DELETE CASCADE,
            asset_id VARCHAR(255) NOT NULL,
            quantity NUMERIC(20, 8) NOT NULL,
            value NUMERIC(20, 8) NOT NULL
        );
    `)
    assert.NoError(t, err)

    return conn
}

func teardownTestDB(t *testing.T, conn *pgx.Conn) {
    _, err := conn.Exec(context.Background(), "DROP TABLE IF EXISTS portfolio_assets CASCADE;")
    assert.NoError(t, err)
    _, err = conn.Exec(context.Background(), "DROP TABLE IF EXISTS portfolio_snapshots CASCADE;")
    assert.NoError(t, err)
    conn.Close(context.Background())
}

func TestPortfolioStore(t *testing.T) {
    conn := setupTestDB(t)
    defer teardownTestDB(t, conn)

    store := &PortfolioStore{db: conn} // Use the existing PortfolioStore struct with the test connection

    // Test AddSnapshot and GetHistory
    snapshot1 := model.PortfolioSnapshot{
        Timestamp: time.Now(),
        Assets: []model.PortfolioAsset{
            {AssetID: "BTC", Quantity: 1, Value: 50000},
        },
        TotalValue: 50000,
    }
    err := store.AddSnapshot(snapshot1)
    assert.NoError(t, err)

    history, err := store.GetHistory()
    assert.NoError(t, err)
    assert.Len(t, history, 1)
    assert.Equal(t, snapshot1.TotalValue, history[0].TotalValue)

    snapshot2 := model.PortfolioSnapshot{
        Timestamp: time.Now(),
        Assets: []model.PortfolioAsset{
            {AssetID: "ETH", Quantity: 10, Value: 30000},
        },
        TotalValue: 30000,
    }
    err = store.AddSnapshot(snapshot2)
    assert.NoError(t, err)

    history, err = store.GetHistory()
    assert.NoError(t, err)
    assert.Len(t, history, 2)
}

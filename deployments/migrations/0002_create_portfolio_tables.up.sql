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

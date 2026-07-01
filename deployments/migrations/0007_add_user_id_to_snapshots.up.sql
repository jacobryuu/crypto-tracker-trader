ALTER TABLE portfolio_snapshots ADD COLUMN IF NOT EXISTS user_id BIGINT REFERENCES users(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_portfolio_snapshots_user_id ON portfolio_snapshots(user_id);

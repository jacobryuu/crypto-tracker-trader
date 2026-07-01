DROP INDEX IF EXISTS idx_portfolio_snapshots_user_id;
ALTER TABLE portfolio_snapshots DROP COLUMN IF EXISTS user_id;

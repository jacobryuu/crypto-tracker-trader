-- Creates portfolio tables, with snapshots linked to a user.
-- Creates portfolio-related tables.

-- 1) portfolios: ユーザーが作成するポートフォリオを表します。各ポートフォリオは users テーブルを参照します。
CREATE TABLE IF NOT EXISTS portfolios (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL DEFAULT 'default', -- ポートフォリオの名称
    wallet_address VARCHAR(255) NOT NULL, -- 関連するウォレットアドレス
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_portfolios_user FOREIGN KEY(user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

-- インデックス: ユーザー毎の検索やウォレットでの検索を高速化
CREATE INDEX IF NOT EXISTS idx_portfolios_user_id ON portfolios(user_id);
CREATE INDEX IF NOT EXISTS idx_portfolios_wallet_address ON portfolios(wallet_address);

-- 2) portfolio_snapshots: ポートフォリオのスナップショット（時点の合計値など）
CREATE TABLE IF NOT EXISTS portfolio_snapshots (
    id SERIAL PRIMARY KEY,
    portfolio_id INTEGER NOT NULL,
    total_value_eth NUMERIC(38, 18) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_snapshot_portfolio FOREIGN KEY(portfolio_id)
        REFERENCES portfolios(id)
        ON DELETE CASCADE
);

-- インデックス
CREATE INDEX IF NOT EXISTS idx_portfolio_snapshots_portfolio_id ON portfolio_snapshots(portfolio_id);


-- 3) portfolio_assets: スナップショットに含まれる各アセットの明細
CREATE TABLE IF NOT EXISTS portfolio_assets (
    id SERIAL PRIMARY KEY,
    snapshot_id INTEGER NOT NULL,
    asset_symbol VARCHAR(50) NOT NULL,
    quantity NUMERIC(38, 18) NOT NULL,
    value_eth NUMERIC(38, 18) NOT NULL,
    CONSTRAINT fk_snapshot FOREIGN KEY(snapshot_id)
        REFERENCES portfolio_snapshots(id)
        ON DELETE CASCADE
);

-- インデックス
CREATE INDEX IF NOT EXISTS idx_snapshot_id ON portfolio_assets(snapshot_id);

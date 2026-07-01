CREATE TABLE exchange_credentials (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exchange VARCHAR(50) NOT NULL,
    api_key_encrypted BYTEA NOT NULL,
    api_secret_encrypted BYTEA NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, exchange)
);

CREATE TABLE exchange_balances (
    id BIGSERIAL PRIMARY KEY,
    credential_id BIGINT NOT NULL REFERENCES exchange_credentials(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    symbol VARCHAR(20) NOT NULL,
    free_balance NUMERIC(38, 8) NOT NULL DEFAULT 0,
    locked_balance NUMERIC(38, 8) NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(credential_id, symbol)
);

CREATE INDEX idx_exchange_balances_user ON exchange_balances(user_id);

CREATE TABLE asset_prices (
    id         BIGSERIAL    PRIMARY KEY,
    symbol     VARCHAR(20)  NOT NULL,
    price_usd  NUMERIC(38, 8) NOT NULL,
    source     VARCHAR(50)  NOT NULL DEFAULT 'coingecko',
    fetched_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Speeds up "latest price for symbol" queries and history lookups.
CREATE INDEX idx_asset_prices_symbol_fetched ON asset_prices (symbol, fetched_at DESC);

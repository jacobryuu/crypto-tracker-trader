CREATE INDEX IF NOT EXISTS idx_user_defi_positions_wallet ON user_defi_positions(wallet_id);
CREATE INDEX IF NOT EXISTS idx_user_defi_positions_protocol ON user_defi_positions(protocol);
ALTER TABLE user_defi_positions ADD CONSTRAINT uq_defi_wallet_protocol_type UNIQUE (wallet_id, protocol, position_type);

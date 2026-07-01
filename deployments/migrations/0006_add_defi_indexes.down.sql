ALTER TABLE user_defi_positions DROP CONSTRAINT IF EXISTS uq_defi_wallet_protocol_type;
DROP INDEX IF EXISTS idx_user_defi_positions_protocol;
DROP INDEX IF EXISTS idx_user_defi_positions_wallet;

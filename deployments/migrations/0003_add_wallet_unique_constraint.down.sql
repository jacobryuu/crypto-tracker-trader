ALTER TABLE user_wallets
    DROP CONSTRAINT IF EXISTS uq_user_wallets_user_chain_address;

-- Add unique constraint to prevent duplicate wallet addresses per chain per user.
ALTER TABLE user_wallets
    ADD CONSTRAINT uq_user_wallets_user_chain_address
    UNIQUE (user_id, chain, address);

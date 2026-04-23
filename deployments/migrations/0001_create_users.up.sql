CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

COMMENT ON TABLE users IS 'ユーザーの基本情報を管理するテーブル';
COMMENT ON COLUMN users.id IS 'ユーザーID';
COMMENT ON COLUMN users.username IS 'ユーザー名（ユニーク）';
COMMENT ON COLUMN users.email IS 'メールアドレス（ユニーク）';
COMMENT ON COLUMN users.is_active IS 'アクティブ状態';
COMMENT ON COLUMN users.created_at IS 'レコード作成日時';
COMMENT ON COLUMN users.updated_at IS 'レコード更新日時';


CREATE TABLE user_credentials (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    password_hash VARCHAR(255),
    mfa_enabled BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

COMMENT ON TABLE user_credentials IS 'ユーザーの認証情報（パスワード、MFA）を保持する';
COMMENT ON COLUMN user_credentials.user_id IS 'users.id への外部キー';
COMMENT ON COLUMN user_credentials.password_hash IS 'ハッシュ化されたパスワード';
COMMENT ON COLUMN user_credentials.mfa_enabled IS '多要素認証の有効状態';


CREATE TABLE user_auth_providers (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    provider VARCHAR(30) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

COMMENT ON TABLE user_auth_providers IS '外部認証プロバイダ（Google/Apple/Web3など）を管理';
COMMENT ON COLUMN user_auth_providers.provider IS 'プロバイダ名（google, apple, metamask など）';
COMMENT ON COLUMN user_auth_providers.provider_user_id IS 'プロバイダ側のユーザーID';


CREATE TABLE user_wallets (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    chain VARCHAR(20) NOT NULL,
    address VARCHAR(150) NOT NULL,
    label VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

COMMENT ON TABLE user_wallets IS 'ユーザーごとのウォレットを管理（複数チェーン対応）';
COMMENT ON COLUMN user_wallets.chain IS 'チェーン名（ethereum, polygon, solana 等）';
COMMENT ON COLUMN user_wallets.address IS 'ウォレットアドレス';
COMMENT ON COLUMN user_wallets.label IS '任意のラベル（Main wallet など）';


CREATE TABLE user_assets (
    id BIGSERIAL PRIMARY KEY,
    wallet_id BIGINT NOT NULL,
    chain VARCHAR(20) NOT NULL,
    token_address VARCHAR(150),
    symbol VARCHAR(20),
    balance NUMERIC(78, 0) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    FOREIGN KEY (wallet_id) REFERENCES user_wallets(id)
);

COMMENT ON TABLE user_assets IS 'ウォレット内のトークン残高を保持';
COMMENT ON COLUMN user_assets.token_address IS 'ERC20やSPLなどのトークンコントラクトアドレス';
COMMENT ON COLUMN user_assets.balance IS 'トークン残高（精度のためNUMERIC）';


CREATE TABLE user_nfts (
    id BIGSERIAL PRIMARY KEY,
    wallet_id BIGINT NOT NULL,
    chain VARCHAR(20) NOT NULL,
    contract_address VARCHAR(150) NOT NULL,
    token_id VARCHAR(100) NOT NULL,
    metadata_json JSONB,
    updated_at TIMESTAMPTZ NOT NULL,
    FOREIGN KEY (wallet_id) REFERENCES user_wallets(id)
);

COMMENT ON TABLE user_nfts IS 'NFT 所有情報を管理';
COMMENT ON COLUMN user_nfts.contract_address IS 'NFTコントラクトアドレス';
COMMENT ON COLUMN user_nfts.token_id IS 'NFTのTokenID';
COMMENT ON COLUMN user_nfts.metadata_json IS 'NFTメタデータ（JSON）';

CREATE TABLE user_defi_positions (
    id BIGSERIAL PRIMARY KEY,
    wallet_id BIGINT NOT NULL,
    protocol VARCHAR(100) NOT NULL,
    position_type VARCHAR(50),
    position_json JSONB,
    updated_at TIMESTAMPTZ NOT NULL,
    FOREIGN KEY (wallet_id) REFERENCES user_wallets(id)
);

COMMENT ON TABLE user_defi_positions IS 'DeFiプロトコルのポジション情報を管理';
COMMENT ON COLUMN user_defi_positions.position_type IS 'ポジション種別（pool, lending, farming 等）';
COMMENT ON COLUMN user_defi_positions.position_json IS 'プロトコル固有の情報（JSONB）';

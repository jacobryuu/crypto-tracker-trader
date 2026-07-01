# crypto-tracker-trader プロジェクト 調査・計画ドキュメント

作成日: 2026-06-30
最終更新: 2026-06-30

---

## 1. プロジェクト概要

複数の暗号資産取引所（Binance、Coinbase、OKX、Bybit 等）と DeFi プロトコルの資産を統合管理・可視化する Go 製バックエンドアプリケーション。

- **言語**: Go 1.25.2
- **Web フレームワーク**: Gin
- **DB**: PostgreSQL（pgx/v4 + pgxpool）
- **ブロックチェーン**: go-ethereum（Ethereum ノード接続）
- **認証**: bcrypt（パスワードハッシュ）
- **デプロイ**: Docker / Docker Compose

---

## 2. 現在のアーキテクチャ

```
cmd/server/main.go          ← エントリーポイント
internal/
  api/handler.go            ← HTTP ハンドラ (Gin)
  service/                  ← ビジネスロジック層
    interfaces.go
    user_service.go
    portfolio_service.go
    blockchain_data_fetcher_service.go
  store/                    ← DB アクセス層 (Repository)
    store.go                ← PortfolioStoreInterface 定義
    portfolio_store.go
    user_store.go
    mock_store.go
  model/                    ← データモデル
    user.go
    portfolio.go
  config/config.go          ← 環境変数ロード
deployments/
  migrations/               ← SQL マイグレーションファイル
  Dockerfile
  docker-compose.yml
docs/
  plan.md                   ← 本ドキュメント
```

---

## 3. 現在の API エンドポイント

| Method | Path | 説明 |
|--------|------|------|
| GET | /health | ヘルスチェック |
| POST | /api/v1/auth/register | ユーザー登録 |
| POST | /api/v1/auth/login | ログイン |
| GET | /api/v1/portfolio/history | ポートフォリオ履歴取得 |
| POST | /api/v1/blockchain/fetch-eth-balance/:address | ETH残高取得＆保存 |

---

## 4. DB スキーマ

### マイグレーションファイル（実行順）
```
0001_create_users.up.sql          ← ユーザー系テーブル
0002_create_portfolio_tables.up.sql ← ポートフォリオ系テーブル
```

### ユーザー系テーブル
- `users` — 基本情報
- `user_credentials` — パスワードハッシュ、MFA
- `user_auth_providers` — Google / Apple / Web3 等の外部認証
- `user_wallets` — ウォレット（複数チェーン対応）
- `user_assets` — トークン残高
- `user_nfts` — NFT 所有情報
- `user_defi_positions` — DeFi ポジション

### ポートフォリオ系テーブル
- `portfolio_snapshots` — 時点スナップショット（timestamp, total_value）
- `portfolio_assets` — スナップショット内アセット明細

---

## 5. 修正済みバグ ✅

### ~~🔴 Critical（動作不可レベル）~~

1. ~~**main.go: DB接続を即座にCloseしてからStoreに渡していた**~~
   → `pgxpool.Connect()` に切り替え。プールを `defer pool.Close()` で一元管理。
   二重 Close バグ（両ストアが同じ接続を Close）も解消。

2. ~~**migration ファイルの重複・矛盾**~~
   → `0001_create_portfolio_tables.up.sql` を `0002_create_portfolio_tables.up.sql` にリネーム。
   スキーマ不一致だった `0002_create_portfolios.up.sql` を削除。

3. ~~**GetHistory の結果順序が非確定的**~~
   → map のソートを有効化（降順 snapshot ID）。

---

## 6. 残存課題・改善計画（優先度順）

### Phase 2: 認証・セキュリティ強化

| # | タスク | 対象ファイル |
|---|--------|------------|
| 2-1 | JWT トークン発行（ログイン・登録時） | internal/service/user_service.go, internal/api/handler.go |
| 2-2 | JWT 認証ミドルウェア実装 | internal/api/middleware.go (新規) |
| 2-3 | 認証が必要なルートに middleware を適用 | internal/api/handler.go |

### Phase 3: 機能補完

| # | タスク | 対象ファイル |
|---|--------|------------|
| 3-1 | ポートフォリオ履歴のユーザー紐付け（現在全件返却） | internal/api/handler.go, internal/service/portfolio_service.go |
| 3-2 | ウォレット管理 API の実装（モデル/DBは定義済み） | internal/api/handler.go, internal/store/wallet_store.go（新規）|
| 3-3 | SlaveDatabaseURL の活用（読み取りクエリの分離） | internal/config/config.go, store 層 |

### Phase 4: 取引所 API 連携（将来）

| # | タスク |
|---|--------|
| 4-1 | 取引所 API クライアントの抽象インターフェース設計 |
| 4-2 | Binance API 連携実装 |
| 4-3 | Coinbase / OKX / Bybit API 連携実装 |

---

## 7. 技術的推奨事項

- **エラーハンドリング**: `err.Error()` の文字列比較でエラー判別しているため、`errors.Is` / カスタムエラー型に統一する
- **ロガー**: `log.Printf` ではなく構造化ロガー（zerolog / zap）の導入を検討
- **Migration ツール**: README では `migrate` CLI を推奨しているが、Makefile に migrate コマンドが未追加
- **テスト**: service 層のユニットテストあり。store 層は integration test が推奨（`TEST_DATABASE_URL` 環境変数で接続先を制御）

---

## 1. プロジェクト概要

複数の暗号資産取引所（Binance、Coinbase、OKX、Bybit 等）と DeFi プロトコルの資産を統合管理・可視化する Go 製バックエンドアプリケーション。

- **言語**: Go 1.25.2
- **Web フレームワーク**: Gin
- **DB**: PostgreSQL（pgx/v4）
- **ブロックチェーン**: go-ethereum（Ethereum ノード接続）
- **認証**: bcrypt（パスワードハッシュ）
- **デプロイ**: Docker / Docker Compose

---

## 2. 現在のアーキテクチャ

```
cmd/server/main.go          ← エントリーポイント
internal/
  api/handler.go            ← HTTP ハンドラ (Gin)
  service/                  ← ビジネスロジック層
    interfaces.go
    user_service.go
    portfolio_service.go
    blockchain_data_fetcher_service.go
  store/                    ← DB アクセス層 (Repository)
    store.go                ← PortfolioStoreInterface 定義
    portfolio_store.go
    user_store.go
    mock_store.go
  model/                    ← データモデル
    user.go
    portfolio.go
  config/config.go          ← 環境変数ロード
deployments/
  migrations/               ← SQL マイグレーションファイル
  Dockerfile
  docker-compose.yml
```

---

## 3. 現在の API エンドポイント

| Method | Path | 説明 |
|--------|------|------|
| GET | /health | ヘルスチェック |
| POST | /api/v1/auth/register | ユーザー登録 |
| POST | /api/v1/auth/login | ログイン |
| GET | /api/v1/portfolio/history | ポートフォリオ履歴取得 |
| POST | /api/v1/blockchain/fetch-eth-balance/:address | ETH残高取得＆保存 |

---

## 4. DB スキーマ

### ユーザー系テーブル（migration: 0001_create_users.up.sql）
- `users` — 基本情報
- `user_credentials` — パスワードハッシュ、MFA
- `user_auth_providers` — Google / Apple / Web3 等の外部認証
- `user_wallets` — ウォレット（複数チェーン対応）
- `user_assets` — トークン残高
- `user_nfts` — NFT 所有情報
- `user_defi_positions` — DeFi ポジション

### ポートフォリオ系テーブル（migration: 0002_create_portfolios.up.sql）
- `portfolios` — user_id に紐づくポートフォリオ
- `portfolio_snapshots` — 時点スナップショット
- `portfolio_assets` — スナップショット内アセット明細

---

## 5. 発見した問題点・バグ

### 🔴 Critical（動作不可レベル）

1. **main.go: DB接続を即座にCloseしてからStoreに渡している**
   ```go
   dbConn, err := pgx.Connect(...)
   if err := dbConn.Close(...); err != nil { ... } // ← 接続をすぐClose
   portfolioStore := store.NewPortfolioStore(dbConn)  // ← 閉じた接続を使用
   ```
   → 起動直後に全DBアクセスが失敗する。

2. **portfolio_store.go と migration の SQL スキーマ不一致**
   - store では `portfolio_snapshots(timestamp, total_value)` を参照
   - migration 0002 では `total_value_eth` と `created_at`（`timestamp` カラムなし）
   - → INSERT/SELECT が失敗する。

3. **migration ファイルの重複・矛盾**
   - `0001_create_portfolio_tables.up.sql` と `0002_create_portfolios.up.sql` の両方にポートフォリオ系テーブル定義がある
   - → 実行順次第でエラーになる可能性がある。

### 🟠 High（機能欠如）

4. **JWT 認証が未実装**
   - ログイン成功時にトークンを返さない（user_id のみ返却）
   - 認証が必要な API ルートにミドルウェアが存在しない

5. **DB 接続プーリングが未使用**
   - `pgx.Conn`（単一接続）を使用。本番では `pgxpool.Pool` を使うべき。

6. **ポートフォリオ履歴がユーザーに紐づいていない**
   - `GET /portfolio/history` はユーザーIDを受け取らず全件返す。

7. **GetHistory の結果順序が非確定的**
   - map のイテレーション順は保証されない（ソートがコメントアウトされている）

### 🟡 Medium（実装途中）

8. **ウォレット管理 API が未実装**
   - `UserWallet`, `UserAsset`, `UserNFT`, `UserDefiPosition` モデルは存在するが API エンドポイントなし

9. **取引所 API 連携が未実装**
   - Binance / Coinbase / OKX / Bybit 等のサポートは設計上の目標だが未着手

10. **SlaveDatabaseURL が設定されているが使われていない**
    - Config に `SlaveDatabaseURL` フィールドがあるが参照箇所なし

---

## 6. 改善・実装計画（優先度順）

### Phase 1: バグ修正（最優先）

| # | タスク | 対象ファイル |
|---|--------|------------|
| 1-1 | main.go の DB 接続即時 Close バグを修正 | cmd/server/main.go |
| 1-2 | portfolio_store の SQL を migration スキーマに合わせる | internal/store/portfolio_store.go または migration |
| 1-3 | migration ファイルの重複を整理 | deployments/migrations/ |

### Phase 2: 認証・セキュリティ強化

| # | タスク | 対象ファイル |
|---|--------|------------|
| 2-1 | JWT トークン発行（ログイン・登録時） | internal/service/user_service.go, internal/api/handler.go |
| 2-2 | JWT 認証ミドルウェア実装 | internal/api/middleware.go (新規) |
| 2-3 | 認証が必要なルートに middleware を適用 | internal/api/handler.go |

### Phase 3: DB・インフラ改善

| # | タスク | 対象ファイル |
|---|--------|------------|
| 3-1 | pgxpool への移行（接続プーリング） | internal/store/*.go, cmd/server/main.go |
| 3-2 | ポートフォリオ履歴のユーザー紐付け | internal/api/handler.go, internal/service/portfolio_service.go |
| 3-3 | GetHistory のソート修正 | internal/store/portfolio_store.go |

### Phase 4: ウォレット管理 API の実装

| # | タスク | 対象ファイル |
|---|--------|------------|
| 4-1 | ウォレット CRUD API | internal/api/handler.go |
| 4-2 | ウォレット Store 実装 | internal/store/wallet_store.go (新規) |
| 4-3 | ウォレット Service 実装 | internal/service/wallet_service.go (新規) |

### Phase 5: 取引所 API 連携（将来）

| # | タスク |
|---|--------|
| 5-1 | 取引所 API クライアントの抽象インターフェース設計 |
| 5-2 | Binance API 連携実装 |
| 5-3 | Coinbase API 連携実装 |
| 5-4 | OKX / Bybit API 連携実装 |

---

## 7. 技術的推奨事項

- **エラーハンドリング**: 現在 `err.Error()` の文字列比較でエラー判別をしているため、`errors.Is` / カスタムエラー型に統一する
- **ロガー**: `log.Printf` ではなく構造化ロガー（zerolog / zap）の導入を検討
- **Migration ツール**: README では `migrate` CLI を推奨しているが、Makefile に migrate コマンドが未追加
- **テスト**: mock_store.go が存在しサービス層のテストはあるが、ストア層は integration test が推奨

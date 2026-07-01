# 実装計画：マルチ取引所・DeFi 統合 暗号資産管理バックエンド

作成日: 2026-06-30

---

## Executive Summary

本プロジェクトは、複数の暗号資産取引所（Binance、Coinbase、OKX、Bybit 等）および DeFi プロトコルのオンチェーン資産を一元管理・可視化する Go 製 REST API バックエンドの構築を目的とする。

現時点ではユーザー管理・Ethereum 残高取得・ポートフォリオ履歴の基盤が実装済みだが、JWT 認証・取引所 API 連携・DeFi データ取得・価格集計など中核機能は未実装である。本計画は既存コードベースを活かしながら 6 フェーズで段階的に機能を拡充する。

**最終目標**: ユーザーが複数取引所と複数ウォレットを登録し、リアルタイムに近い統合ポートフォリオを参照できる API の提供。

---

## Requirement Analysis Snapshot

### Goals

1. ユーザーが複数の取引所アカウント（API キー管理）を登録・管理できる
2. ユーザーが複数のブロックチェーンウォレットを登録・管理できる
3. 各取引所 REST API から残高・取引履歴を定期取得できる
4. オンチェーン資産（ERC-20、NFT、DeFi ポジション）を取得・保存できる
5. 外部価格データ（CoinGecko / CoinMarketCap 等）を取得し USD 換算できる
6. ユーザーごとの統合ポートフォリオ（総資産額・資産構成・履歴）を API として提供する
7. JWT ベースの認証で API を保護する

### Constraints

| 種別 | 内容 |
|------|------|
| 技術 | Go 1.25.2、Gin、PostgreSQL、既存コードベースを継続使用 |
| 技術 | go-ethereum で Ethereum チェーンの直接読み取りは可能 |
| セキュリティ | 取引所 API キーは暗号化して保存必須（AES-256-GCM 推奨） |
| コンプライアンス | 各取引所の利用規約に沿った API 呼び出しレート制限の遵守が必須 |
| 外部依存 | Ethereum ノード URL（Infura/Alchemy または自前ノード） |
| 外部依存 | 価格データプロバイダのレート制限（CoinGecko 無料: 30req/min） |

### Assumptions

1. マルチチェーン対応の初期スコープは **Ethereum のみ**とし、他チェーン（Polygon、Solana 等）は Phase 6 以降
2. 取引所連携の初期スコープは **Binance のみ**とし、他取引所は順次追加
3. バックグラウンドジョブには **定期実行（cron / ticker）** を使用し、外部ジョブキューは不要
4. フロントエンドは本プロジェクトのスコープ外
5. 本番 DB は PostgreSQL、ローカル開発は Docker Compose で完結させる

### Missing Information

| 項目 | 影響度 | 対処 |
|------|-------|------|
| 対応取引所の優先順位 | 高 | Binance 優先と仮定し計画 |
| 価格データプロバイダの選定 | 中 | CoinGecko（無料）で計画、差し替え可能な抽象化を設ける |
| データ更新頻度の要件 | 中 | 残高: 5 分間隔、価格: 1 分間隔と仮定 |
| API キー暗号化のマスターキー管理方法 | 高 | 環境変数による注入と仮定（KMS 導入は Open Question） |
| マルチテナント分離要件（SaaS か内部ツールか）| 中 | 内部ツールと仮定しシンプルな user_id フィルタで対応 |

---

## Assumptions

- 各取引所は REST API（v1）を使用し、WebSocket はスコープ外
- DeFi は Uniswap V2/V3 の LP ポジション読み取りを代表実装とする
- API キーは取引所ごとに `Read-Only` 権限のみ要求する
- バックグラウンドジョブはアプリプロセス内の goroutine + ticker で実装する（Phase 4 でキューへの移行を検討）

---

## Milestones

| フェーズ | マイルストーン | 期間目安 | 完了条件 |
|---------|--------------|---------|---------|
| Phase 1 | JWT 認証基盤 | 1 週間 | 全保護エンドポイントで Bearer トークン検証が通る |
| Phase 2 | ウォレット管理 API | 1 週間 | ウォレット CRUD・資産一覧取得 API が動作する |
| Phase 3 | 価格データ統合 | 1 週間 | CoinGecko から価格取得し DB に保存・USD 換算が可能 |
| Phase 4 | 取引所 API 連携（Binance） | 2 週間 | Binance 残高の定期取得・保存・取得 API が動作する |
| Phase 5 | DeFi プロトコル統合 | 2 週間 | Uniswap LP ポジション・ERC-20 残高の取得・保存が動作する |
| Phase 6 | 統合ポートフォリオ API | 1 週間 | 全資産の統合ビュー・履歴グラフ用データを API で提供する |

---

## Detailed Tasks

---

### Phase 1: JWT 認証基盤（1 週間）

#### 目的
現在ログイン API はユーザー情報を返すだけでトークンを発行しない。全フェーズの基盤となる認証レイヤーを最初に完成させる。

#### タスク

**1-1. JWT ライブラリの追加**
- `github.com/golang-jwt/jwt/v4` を `go.mod` に追加
- Config に `JWT_SECRET` と `JWT_EXPIRY_HOURS` を追加

**1-2. JWT トークン発行（ログイン・登録レスポンス修正）**
- `internal/service/user_service.go`: `LoginUser` / `RegisterUser` の戻り値にトークン文字列を追加
- `internal/api/handler.go`: レスポンスに `access_token` フィールドを追加

**1-3. JWT 認証ミドルウェアの実装**
- `internal/api/middleware/auth.go`（新規）を作成
- `Authorization: Bearer <token>` ヘッダを検証
- 検証成功時に `userID` を Gin context にセット
- 失敗時は `401 Unauthorized` を返す

**1-4. ルートへのミドルウェア適用**
- `internal/api/handler.go`: `/api/v1/portfolio/*` と `/api/v1/blockchain/*` および今後追加する全保護ルートに middleware を適用

**1-5. ユニットテスト**
- middleware のテスト（有効トークン・期限切れ・署名不一致・missing ヘッダ）
- `handler_test.go` の既存テストを認証付きに更新

#### 完了条件
```
POST /api/v1/auth/login → { "access_token": "eyJ..." } を返す
GET  /api/v1/portfolio/history（tokenなし） → 401
GET  /api/v1/portfolio/history（有効token） → 200
```

---

### Phase 2: ウォレット管理 API（1 週間）

#### 目的
`user_wallets` / `user_assets` / `user_nfts` / `user_defi_positions` テーブルは設計済みだが API が存在しない。フロントエンドとの接続に必要なウォレット CRUD を実装する。

#### タスク

**2-1. ウォレット Store の実装**
- `internal/store/wallet_store.go`（新規）: CRUD + 資産一覧取得
- インターフェース: `WalletStoreInterface` を `internal/store/store.go` に追加

**2-2. ウォレット Service の実装**
- `internal/service/wallet_service.go`（新規）
- ウォレット追加時に重複アドレス検証を実施
- `internal/service/interfaces.go` に `WalletManager` インターフェースを追加

**2-3. ウォレット API ハンドラの実装**
- `internal/api/wallet_handler.go`（新規）
- エンドポイント:
  - `POST /api/v1/wallets` — ウォレット登録
  - `GET  /api/v1/wallets` — ウォレット一覧
  - `DELETE /api/v1/wallets/:id` — ウォレット削除
  - `GET  /api/v1/wallets/:id/assets` — 資産一覧

**2-4. DB マイグレーション**
- `0003_create_wallet_indexes.up.sql`（新規）: `user_wallets(user_id, chain, address)` にユニーク制約追加

**2-5. テスト**
- wallet_store の統合テスト
- wallet_handler のユニットテスト（モック使用）

---

### Phase 3: 価格データ統合（1 週間）

#### 目的
すべての資産を USD で統一換算するために、外部価格データの定期取得・保存の仕組みを構築する。

#### タスク

**3-1. 価格データ DB スキーマ**
- `0004_create_price_data.up.sql`（新規）
  ```sql
  CREATE TABLE asset_prices (
      id BIGSERIAL PRIMARY KEY,
      symbol VARCHAR(20) NOT NULL,
      price_usd NUMERIC(38, 8) NOT NULL,
      source VARCHAR(50) NOT NULL,
      fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
  );
  CREATE INDEX idx_asset_prices_symbol_fetched ON asset_prices(symbol, fetched_at DESC);
  ```

**3-2. 価格プロバイダの抽象インターフェース**
- `internal/service/price_fetcher.go`（新規）
  ```go
  type PriceFetcher interface {
      FetchPrices(ctx context.Context, symbols []string) (map[string]decimal.Decimal, error)
  }
  ```

**3-3. CoinGecko クライアントの実装**
- `internal/client/coingecko/client.go`（新規）
- レート制限対応（30 req/min、指数バックオフ）
- シンボルマッピング（BTC → bitcoin 等）

**3-4. 価格取得サービス + 定期実行**
- `internal/service/price_service.go`（新規）
- 1 分ごとに主要銘柄の価格を取得して `asset_prices` に保存
- goroutine + time.Ticker で実装
- 取得対象銘柄は `user_assets.symbol` から動的に収集

**3-5. price_store の実装**
- `internal/store/price_store.go`（新規）: 最新価格取得・履歴保存

**3-6. API エンドポイント**
- `GET /api/v1/prices/:symbol` — 最新価格返却

---

### Phase 4: 取引所 API 連携 — Binance 優先（2 週間）

#### 目的
ユーザーが登録した Binance API キーを使って残高・取引履歴を定期取得する。

#### タスク

**4-1. 取引所 API キー管理 DB スキーマ**
- `0005_create_exchange_credentials.up.sql`（新規）
  ```sql
  CREATE TABLE exchange_credentials (
      id BIGSERIAL PRIMARY KEY,
      user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
      exchange VARCHAR(50) NOT NULL,
      api_key_encrypted BYTEA NOT NULL,
      api_secret_encrypted BYTEA NOT NULL,
      is_active BOOLEAN DEFAULT TRUE,
      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      UNIQUE(user_id, exchange)
  );
  ```

**4-2. API キー暗号化ユーティリティ**
- `internal/crypto/aes.go`（新規）
- AES-256-GCM による暗号化・復号
- マスターキーは環境変数 `ENCRYPTION_KEY`（32 バイト hex）から注入

**4-3. 取引所クライアント抽象インターフェース**
- `internal/client/exchange/interface.go`（新規）
  ```go
  type ExchangeClient interface {
      GetBalances(ctx context.Context) ([]Balance, error)
      GetTradeHistory(ctx context.Context, symbol string, since time.Time) ([]Trade, error)
  }
  ```

**4-4. Binance クライアント実装**
- `internal/client/exchange/binance/client.go`（新規）
- Binance REST API v3 対応
- HMAC-SHA256 署名
- レート制限対応（1200 req/min weight ベース）

**4-5. 取引所残高取得サービス**
- `internal/service/exchange_sync_service.go`（新規）
- 5 分ごとに全ユーザーの取引所残高を取得・保存
- `user_assets` テーブルに upsert

**4-6. 取引所設定 API**
- `internal/api/exchange_handler.go`（新規）
  - `POST /api/v1/exchanges` — API キー登録
  - `GET  /api/v1/exchanges` — 登録済み取引所一覧
  - `DELETE /api/v1/exchanges/:id` — API キー削除
  - `POST /api/v1/exchanges/:id/sync` — 手動同期トリガー

**4-7. テスト**
- Binance クライアントのモックテスト
- 暗号化ユーティリティのユニットテスト
- exchange_sync_service のテスト

---

### Phase 5: DeFi プロトコル統合（2 週間）

#### 目的
Ethereum オンチェーンの ERC-20 残高・Uniswap LP ポジション・Aave 貸出ポジションを取得する。

#### タスク

**5-1. ERC-20 残高取得の拡張**
- `internal/service/blockchain_data_fetcher_service.go` を拡張
- ERC-20 balanceOf() の呼び出し
- 対象トークンリスト（上位 100 銘柄）をコンフィグで管理

**5-2. DeFi プロトコル抽象インターフェース**
- `internal/client/defi/interface.go`（新規）
  ```go
  type DefiProtocolClient interface {
      GetPositions(ctx context.Context, walletAddr common.Address) ([]DefiPosition, error)
  }
  ```

**5-3. Uniswap V3 クライアント**
- `internal/client/defi/uniswap/client.go`（新規）
- NonfungiblePositionManager コントラクトから LP ポジション読み取り
- 未収手数料の計算

**5-4. Aave V3 クライアント（オプション）**
- `internal/client/defi/aave/client.go`（新規）
- AaveDataProvider コントラクトから貸出・借入ポジション読み取り

**5-5. DeFi 同期サービス**
- `internal/service/defi_sync_service.go`（新規）
- 15 分ごとにウォレットの DeFi ポジションを取得・保存
- `user_defi_positions` テーブルに upsert

**5-6. DB マイグレーション**
- `0006_add_defi_protocol_index.up.sql`（新規）

**5-7. テスト**
- コントラクト呼び出しのモックテスト（go-ethereum のモック使用）

---

### Phase 6: 統合ポートフォリオ API（1 週間）

#### 目的
全ソース（取引所・オンチェーン・DeFi）の資産を統合し、ユーザーごとのポートフォリオビューを提供する。

#### タスク

**6-1. ポートフォリオ集計サービス**
- `internal/service/portfolio_aggregation_service.go`（新規）
- 全資産を symbol ごとに集計し USD 換算
- 取引所・ウォレット・DeFi の内訳も保持

**6-2. ポートフォリオ API の拡張**
- `GET /api/v1/portfolio/summary` — 現在の総資産・内訳（取引所別・チェーン別）
- `GET /api/v1/portfolio/history` — ユーザー紐付けで時系列スナップショット返却（既存を修正）
- `GET /api/v1/portfolio/allocation` — 銘柄別比率

**6-3. ポートフォリオスナップショット定期保存**
- 1 時間ごとに `portfolio_snapshots` にスナップショットを保存
- user_id を snapshot に紐付け（migration 修正）

**6-4. パフォーマンス最適化**
- 集計クエリのインデックス確認
- Redis キャッシュの検討（summary API は TTL 30 秒）

---

## Risks and Mitigations

| リスク | 発生確率 | 影響度 | 対策 |
|--------|---------|-------|------|
| 取引所 API のレート制限超過 | 高 | 高 | クライアントに指数バックオフ・キューイングを実装。1 ユーザーあたり取得間隔を調整 |
| API キー漏洩 | 低 | 最高 | AES-256-GCM 暗号化 + 環境変数管理。DB には暗号文のみ保存。ログへの平文出力禁止 |
| 取引所 API の仕様変更 | 中 | 中 | 取引所クライアントを抽象インターフェースの後ろに隠蔽。各クライアントを独立テスト |
| Ethereum ノード障害 | 中 | 中 | ノード URL を複数設定可能にし、フォールバック機能を追加（Infura + Alchemy） |
| 価格データプロバイダの無料枠超過 | 中 | 中 | キャッシュ活用。プロバイダをインターフェースで抽象化し差し替え可能に |
| DB 負荷増大（定期バッチ書き込み） | 中 | 中 | upsert + インデックス最適化。将来的には TimescaleDB への移行を検討 |
| goroutine リーク（バックグラウンドジョブ） | 低 | 中 | context キャンセルを全 goroutine に伝播。Graceful Shutdown を実装 |

---

## Success Metrics

| 指標 | 目標値 |
|------|-------|
| API レスポンスタイム（portfolio/summary） | P95 < 500ms |
| 残高データの鮮度 | 最大 5 分遅延 |
| 取引所 API エラー率 | < 1% |
| テストカバレッジ（service 層） | ≥ 80% |
| セキュリティ: API キーの平文露出 | ゼロ件 |

---

## Validation Strategy

### Phase 1 (JWT)
- `go test ./internal/api/... ./internal/service/...` — 全ユニットテスト
- curl で 401/200 の動作確認

### Phase 2〜3 (Wallet / Price)
- Docker Compose でローカル DB を起動し統合テスト実施
- `go test ./internal/store/...` — store 統合テスト

### Phase 4 (Exchange)
- Binance Testnet または API モックサーバーで動作確認
- 暗号化ユーティリティのラウンドトリップテスト（暗号化→復号→一致確認）

### Phase 5 (DeFi)
- Anvil（Foundry）でローカル Ethereum フォークを起動しコントラクト呼び出しテスト

### Phase 6 (Portfolio)
- E2E シナリオ: ユーザー登録 → ウォレット登録 → 取引所登録 → portfolio/summary 確認
- スナップショット保存の時系列データ正確性確認

### ロールバック戦略
- 各マイグレーションに対応する `.down.sql` を必ず作成
- フィーチャーフラグ（環境変数）で各フェーズの機能を ON/OFF 可能にする
- DB マイグレーションは `migrate` CLI でバージョン管理

---

## Open Questions

1. **取引所の優先順位**: Binance の次は Coinbase か OKX か？
2. **マスターキー管理**: 環境変数注入か AWS KMS / HashiCorp Vault を使うか？
3. **価格プロバイダ**: CoinGecko 無料枠で十分か？CoinMarketCap / Chainlink も検討するか？
4. **マルチチェーン対応**: Polygon / Arbitrum / Solana の優先順位は？
5. **レート制限の共有**: 同一 IP から複数ユーザーの取引所 API を叩く場合の戦略は？
6. **通知機能**: 残高変動やエラー発生時の Slack / メール通知は必要か？
7. **管理者向け API**: ユーザー管理・バッチ実行トリガーの管理 API は必要か？

---

## Quality Rubric Scores (Final)

| 評価軸 | スコア | 根拠 |
|--------|--------|------|
| **Completeness** | 5/5 | 認証・ウォレット・価格・取引所・DeFi・統合ビューの全フェーズを網羅。各タスクに具体的なファイル名・テーブル・エンドポイントを明記 |
| **Feasibility** | 4/5 | 既存 Go コードベースと整合するライブラリ（golang-jwt, go-ethereum）のみ使用。ただし取引所 API レート制限の実運用調整が必要 |
| **Risk Coverage** | 4/5 | API キー漏洩・レート制限・ノード障害・goroutine リークを識別しミティゲーションを提示。KMS 導入判断は Open Question として残存 |
| **Testability** | 4/5 | Phase ごとの検証戦略・ロールバック手順を定義。Anvil によるオンチェーンテストを含む。取引所モックサーバーの整備に工数依存 |
| **Maintainability** | 5/5 | 取引所・DeFi・価格プロバイダ全てをインターフェースで抽象化。各クライアントを独立パッケージに分離することで、新規取引所追加時の変更範囲を最小化 |

**総合: 22/25**

---

## Refinement Notes

### 初稿から改善した点

1. **API キー暗号化を Phase 4 の最初のタスクに格上げ**
   - 初稿では取引所クライアント実装の後に配置していたが、セキュリティ上の重要性から先行させた

2. **価格データ統合（Phase 3）を取引所連携（Phase 4）より先に配置**
   - 取引所残高の USD 換算には価格データが必要なため、依存関係を整理した

3. **DeFi インターフェースの抽象化を明示**
   - 初稿では Uniswap のみの具体実装を示していたが、Aave 等の追加を見越して抽象レイヤーを先に定義するよう変更

4. **Graceful Shutdown / context キャンセル伝播**
   - goroutine リークリスクへの対策として、バックグラウンドジョブに context を明示的に伝播させる要件を追加

5. **ロールバック戦略に `.down.sql` 必須化とフィーチャーフラグを追加**
   - 初稿では migration ロールバックのみだったが、機能フラグによる段階的リリースを追加して運用リスクを低減

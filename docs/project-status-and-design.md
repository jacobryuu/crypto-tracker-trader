# crypto-tracker-trader プロジェクト 現状・設計ドキュメント

作成日: 2026-09-02

> 本ドキュメントは、プロジェクトの**最終完成図（目標アーキテクチャ）**・**実装状況（As-Is）**・**設計の意図とその理由**を整理したものです。
> 計画立案の詳細については [`implementation-plan.md`](./implementation-plan.md) を、開発初期の問題調査については [`plan.md`](./plan.md) を参照してください。

---

## 1. 本ドキュメントの目的

`plan.md`（初期調査・バグ報告）と `implementation-plan.md`（6 フェーズ実装計画）は「計画」を記した文書であり、**計画が実際にどこまで実装されたか** の進捗記録はありません。本ドキュメントは以下の 3 点を補完します。

1. **最終完成図**: このプロジェクトが完成したときのシステム全体像・機能・DB・API（将来像）
2. **実装状況**: 現在のコードベースに実装済みの機能と、未実装・部分実装のギャップ
3. **設計の意図**: 各設計判断の背景と、なぜその方式を採用したか

---

## 2. 最終完成図（Target Architecture）

### 2.1 システム概要

暗号資産トラッカー（兼トレーダー）のバックエンド API。ユーザーが複数の**取引所アカウント**（Binance / Coinbase / OKX / Bybit）と複数の**ブロックチェーンウォレット**（Ethereum / Polygon / Arbitrum / Optimism / Solana）を登録し、オンチェーン資産（ETH・ERC-20・NFT・DeFi ポジション）を自動取得して、**USD 建ての統合ポートフォリオ**を提供する。

完成時の姿は「**暗号資産版の資産管理アプリ API**（MoneyForward や Personal Capital の暗号通貨特化型）」であり、フロントエンドはスコープ外（バックエンド API のみを提供）。

### 2.2 最終アーキテクチャ図

```
┌─────────────────────────────────────────────────────────────────────┐
│                        HTTP Client (フロントエンド)                     │
└───────────────────────────────┬─────────────────────────────────────┘
                                │ REST (JSON) + Bearer JWT
┌───────────────────────────────▼─────────────────────────────────────┐
│                        internal/api (Gin)                            │
│  handler.go / wallet_handler.go / price_handler.go                   │
│  exchange_handler.go / defi_handler.go / portfolio_summary_handler.go│
│  middleware/auth.go (JWT検証 → userID を ctx に注入)                   │
└───────────────────────────────┬─────────────────────────────────────┘
                                │ コンシューマー側インターフェース
┌───────────────────────────────▼─────────────────────────────────────┐
│                      internal/service (業務ロジック)                   │
│  user_service        … auth / user 管理                              │
│  wallet_service      … ウォレット CRUD・所有権チェック                   │
│  price_service       … 価格取得・定期同期（ticker）                    │
│  exchange_service    … キー暗号化・残高同期（ticker）                  │
│  defi_sync_service   … DeFi ポジション同期（ticker）                   │
│  portfolio_aggregation_service … 統合ポートフォリオ集計                  │
│  blockchain_data_fetcher_service … オンチェーン残高取得                 │
└───────┬──────────────────┬────────────────────┬──────────────────────┘
        │ 外部連携（抽象インターフェースの背後）    │ Finde 側インターフェース     │
        │                                        │ (store/store.go)
┌───────▼──────┐  ┌──────────────┐  ┌─────────────▼──────────────┐
│ PriceFetcher │  │ ExchangeClient│ │      internal/store (pgxpool) │
│ coingecko    │  │ binance       │ │  user/wallet/price/exchange/   │
│ (将来: cMC等) │  │ (将来: cb/okx/bybit)  │ │  defi/portfolio store       │
└──────────────┘  └──────────────┘  └─────────────┬──────────────┘
┌──────────────┐  ┌──────────────┐                │
│ DefiProtocol │  │ EthClient    │        ┌───────▼────────┐
│  Client      │  │ (go-ethereum)│        │  PostgreSQL 15 │
│  uniswap     │  │              │        └────────────────┘
│ (将来: aave) │  └──────────────┘
└──────────────┘
```

### 2.3 最終機能セット

| カテゴリ | 目標機能 |
|---------|---------|
| 認証 | 登録・ログイン、JWT 発行/検証、全保護ルートの認可 |
| ウォレット | マルチチェーン（ETH/Polygon/Arbitrum/Optimism/Solana）の CRUD、重複防止 |
| オンチェーン取得 | ネイティブ残高 + ERC-20 balanceOf + NFT 所有 + 他チェーン |
| 価格データ | CoinGecko（差し替え可能な Provider 抽象）、USD 換算の基盤 |
| 取引所 | Binance/Coinbase/OKX/Bybit 残高・取引履歴の定期取得、API キー暗号化保存 |
| DeFi | Uniswap V2/V3 LP ポジション、Aave V3 貸出/借入、未収手数料計算 |
| ポートフォリオ | 総資産・銘柄別比率（allocation）・ソース別内訳・時系列履歴（1 時間スナップショット） |
| 非同期処理 | 価格 1 分 / 残高 5 分 / DeFi 15 分 / スナップショット 1 時間 の ticker 自動同期 |

### 2.4 最終 DB スキーマ（完成形）

個別のテーブルはすべてマイグレーション済み。最終形は「資産情報の取得元を一意に紐づけられる」構造です。

```
users / user_credentials / user_auth_providers   … ユーザー + 認証 (MFA・外部認証は将来)
user_wallets(user_id, chain, address UNIQUE)      … ウォレット登録（複数チェーン）
user_assets / user_nfts                           … オンチェーン資産
user_defi_positions                               … DeFi ポジション（protocol×type UNIQUE）
exchange_credentials(user_id, exchange UNIQUE)    … 暗号化 API キー
exchange_balances(credential_id, symbol UNIQUE)   … 取引所残高
asset_prices(symbol, fetched_at)                  … 価格履歴
portfolio_snapshots(user_id) / portfolio_assets   … 時系列スナップショット
```

### 2.5 最終 API サーフェス（目標）

| Method | Path | 説明 |
|--------|------|------|
| POST | `/api/v1/auth/register` | 登録 → access_token |
| POST | `/api/v1/auth/login` | ログイン → access_token |
| GET | `/api/v1/portfolio/summary` | 統合総資産・内訳（実装済み） |
| GET | `/api/v1/portfolio/allocation` | 銘柄別比率（実装済み） |
| GET | `/api/v1/portfolio/history` | ユーザー紐付けの時系列履歴（※要修正） |
| GET/POST/DELETE | `/api/v1/wallets...` | ウォレット管理（実装済み） |
| POST | `/api/v1/wallets/:id/assets/fetch` | ERC-20 残高取得（未実装） |
| GET | `/api/v1/defi/positions` | DeFi ポジション一覧（実装済み） |
| GET/POST/DELETE | `/api/v1/exchanges...` | 取引所キー管理・同期（実装済み） |
| GET | `/api/v1/prices/:symbol` | 最新価格（実装済み） |
| POST | `/api/v1/blockchain/fetch-eth-balance/:address` | ETH 残高取得＆スナップショット |

---

## 3. 実装状況（As-Is）

### 3.1 実装済み機能マトリクス

`implementation-plan.md` の 6 フェーズに対する進捗です。

| Phase | 機能 | 状態 | 主要ファイル |
|-------|------|------|--------------|
| Phase 1 | JWT 認証（発行・検証・ミドルウェア適用） | ✅ 完了 | `internal/auth/jwt.go`, `internal/api/middleware/auth.go` |
| Phase 2 | ウォレット管理 API（CRUD・資産一覧） | ✅ 完了 | `wallet_service.go`, `wallet_handler.go`, migration 0003 |
| Phase 3 | 価格データ統合（CoinGecko・1 分同期・価格 API） | ✅ 完了 | `price_service.go`, `client/coingecko`, migration 0004 |
| Phase 4 | 取引所連携（Binance・キー暗号化・5 分同期） | ✅ ほぼ完了 | `exchange_service.go`, `client/exchange/binance`, migration 0005 |
| Phase 5 | DeFi 統合（Uniswap V3） | 🟡 部分 | `defi_sync_service.go`, `client/defi/uniswap`, migration 0006 |
| Phase 6 | 統合ポートフォリオ API | 🟡 部分 | `portfolio_aggregation_service.go`, migration 0007 |

追加実装（計画外で対応済み）
- 価格履歴 API（`GET /prices/:symbol/history`）と手動同期トリガー（`POST /prices/sync`）
- 取引所残高一覧 API（`GET /exchanges/balances`）

### 3.2 現在のモジュール構成（実コード反映）

```
cmd/server/main.go                    … 全依存の配線（DIコンテナなしの手動コンストラクタ注入）
internal/
  api/handler.go                      … ルート登録・auth・blockchain・portfolio
  api/wallet_handler.go / price_handler.go / exchange_handler.go
  api/defi_handler.go / portfolio_summary_handler.go
  api/middleware/auth.go              … Bearer JWT 検証
  auth/jwt.go                         … HS256 トークン generate/validate
  client/coingecko/client.go          … PriceFetcher 実装（レート制限・指数バックオフ）
  client/exchange/interface.go        … ExchangeClient 抽象
  client/exchange/binance/client.go   … Binance v3（HMAC-SHA256）
  client/defi/interface.go            … DefiProtocolClient 抽象
  client/defi/uniswap/client.go       … Uniswap V3 NFPM（eth_call）
  config/config.go                    … 環境変数ロード（godotenv）
  crypto/aes.go                       … AES-256-GCM 鍵暗号化
  model/                              … データモデル（金額は文字列保持）
  service/interfaces.go               … コンシューマー側インターフェース
  service/*_service.go                … 業務ロジック + ticker バックグラウンド同期
  store/store.go                      … プロデューサー側インターフェース
  store/user_store.go / wallet_store.go / price_store.go
  store/exchange_store.go / defi_store.go / portfolio_store.go
  store/mock_*.go                     … testify モック
deployments/migrations/0001..0007     … .up/.down ペア（0003以降）
```

### 3.3 実装済み API エンドポイント（`handler.go:157` の `RegisterRoutes` より）

**認証不要**（`/health`, `/api/v1/auth/*`）

| Method | Path |
|--------|------|
| GET | `/health` |
| POST | `/api/v1/auth/register` |
| POST | `/api/v1/auth/login` |

**認証必須**（`middleware.Auth` 適用）

| Method | Path |
|--------|------|
| GET | `/api/v1/portfolio/history` |
| GET | `/api/v1/portfolio/summary` |
| GET | `/api/v1/portfolio/allocation` |
| POST | `/api/v1/blockchain/fetch-eth-balance/:address` |
| POST | `/api/v1/wallets` |
| GET | `/api/v1/wallets` |
| DELETE | `/api/v1/wallets/:id` |
| GET | `/api/v1/wallets/:id/assets` |
| POST | `/api/v1/wallets/:id/defi/sync` |
| GET | `/api/v1/prices/:symbol` |
| GET | `/api/v1/prices/:symbol/history` |
| POST | `/api/v1/prices/sync` |
| POST | `/api/v1/exchanges` |
| GET | `/api/v1/exchanges` |
| DELETE | `/api/v1/exchanges/:id` |
| POST | `/api/v1/exchanges/:id/sync` |
| GET | `/api/v1/exchanges/balances` |
| GET | `/api/v1/defi/positions` |

### 3.4 バックグラウンドジョブ（`main.go:75-80`）

`syncCtx`（`context.WithCancel`）+ `time.Ticker` による goroutine。`defer stopSync()` で終了時にキャンセル。

| ジョブ | 間隔（デフォルト） | 動作 |
|--------|------------------|------|
| 価格同期 `priceService.StartSync` | 60 秒 | 起動直後に 1 回 + tick ごとに `FetchAndSave` |
| 取引所残高同期 `exchangeService.StartSync` | 300 秒 | 全有効クレデンシャルの残高を再取得・upsert |
| DeFi 同期 `defiService.StartSync` | 900 秒 | 起動時・定期実行するが **`runSync` はスタブ**（下記 3.5 参照） |

### 3.5 未実装・部分実装のギャップ

| # | ギャップ | 現状の実コード | 影響 |
|---|---------|--------------|------|
| G1 | **DeFi 定期同期はスタブ** | `defi_sync_service.go:87-93` の `runSync()` はログ出力のみ。walletStore に全ウォレット列挙 API がなく、API トリガー（`POST /wallets/:id/defi/sync`）でのみ同期される | 定期で DeFi 残高が更新されない |
| G2 | **DeFi の USD 評価が $0** | `portfolio_aggregation_service.go:116-132` で `position_json` をパースせず、ソース内訳のみ記録（保守的に 0） | 総資産に DeFi が含まれない |
| G3 | **`GET /portfolio/history` がユーザー未分離** | `handler.go:147` `GetPortfolioHistory` は userID を受渡ししない。`portfolio_store.go:59` `GetHistory` は全件返却。migration 0007 で `user_id` カラムは追加済み | 他ユーザーのスナップショットが見える |
| G4 | **スナップショット定期保存なし** | 1 時間スナップショットジョブは未実装（0007 でカラム追加のみ）。スナップショットは ETH 取得（`POST /blockchain/fetch-eth-balance`）時のみ | 時系列履歴がほぼ空 |
| G5 | **Coinbase / OKX / Bybit 未実装** | `exchange_service.go:21-28` `DefaultExchangeClientFactory` は binance のみ | 他取引所が使えない |
| G6 | **Aave / Uniswap V2 未実装** | Uniswap V3 のみ（Aave は計画上もオプション） | DeFi の網羅性 |
| G7 | **ERC-20・NFT・他チェーン取得未実装** | `blockchain_data_fetcher_service.go` はネイティブ ETH 残高のみ | トークン残高が取引所残高に依存 |
| G8 | **`SLAVE_DATABASE_URL` が未使用** | `config.go:18` でロードのみ。読み取りリクエストの分離なし | スケール時の読み取り分散不可 |
| G9 | **取引履歴が API 公開されていない** | `binance.GetTradeHistory` は実装済み（`client.go:107`）だが API ハンドラなし。`GetTradeHistory` をモデルへの保存もなし | トレード機能が使えない |
| G10 | **`configs/config.go` は死んだスタブ** | ルート直下 `configs/` の `package config` は現在の `internal/config` と二重定義（コンパイル上は別ディレクトリなので共存） | 削除推奨 |
| G11 | **`user_credentials.password_hash` に非 NULL 制約なし** | 外部認証想定で nullable。bcrypt のみ使用中 | 誤運用の余地 |
| G12 | **エラーハンドリングが文字列比較** | `handler.go:75-76` などで `err.Error() == "..."` で分岐 | `errors.Is` / センチネルエラーへの統一が望ましい |

### 3.6 既知の技術的負債・推奨事項

`plan.md` の「技術的推奨事項」と重複しますが、現在も該当するものは以下です。

- 構造化ロガー（zerolog / slog）未導入。`log.Printf` が散在
- Makefile に `migrate` コマンドが未追加（README では CLI 推奨）
- store 層の統合テストが未整備（service 層は `mock_*` でテスト済み）
- Dockerfile は `golang:1.25-alpine` のまま（`go.mod` は 1.26.4）— 軽微なドリフト
- `PortfolioStore.Close()`（`portfolio_store.go:23`）と `main.go` の `dbPool.Close()`（`main.go:46`）の二重 Close リスク。現在は main だけが Close を呼ぶため実被害なし

---

## 4. 設計の意図と設計理由（Design Intent & Rationale）

### 4.1 レイヤード + ヘキサゴナル風アーキテクチャ

**構成**: HTTP（api）→ サービス（service）→ ストア（store）→ PostgreSQL、と単一方向の依存関係にしている。

**理由**:
- ビジネスロジックを HTTP や DB の詳細から分離し、**各層を独立にテスト**できる（`mock_*` store を使う service 単体テストがそれを可能にしている）
- 将来 SQL → TimescaleDB、Gin → 別 WebFW など層内の差し替えに備える
- DI フレームワークを使わず `main.go` で手動配線するのは、**依存が少なく画面推移が単純**な小〜中規模プロジェクトに適した選択。ステップアップで Wire 等への移行も容易

### 4.2 インターフェース駆動設計（両方向の抽象）

- **コンシューマー側**（api が service を使うための `service/interfaces.go`）
- **プロデューサー側**（service が store を使うための `store/store.go`）
- **外部連携の抽象**: `PriceFetcher` / `ExchangeClient` / `DefiProtocolClient` / `EthClientInterface`

**理由**:
- 「Binance しか実装していない」段階でも、将来の Coinbase/OKX/Bybit 追加時に**変更範囲がホットスポット（ファクトリ1箇所 + クライアント1パッケージ）に閉じる**
- 価格プロバイダ（CoinGecko → CoinMarketCap）の差し替え、Ethereum ノード（Infura → Alchemy）の変更も同様に局所化
- `NewWithBaseURL`（binance）、`NewExchangeServiceWithFactory`（exchange service）等の**テスト用コンストラクタ**を提供することで、実ネットワーク・実 DB なしのテストを可能にしている

### 4.3 金額・数量の精度設計（NUMERIC + 文字列 + big.Float）

**設計**: DB は `NUMERIC` 型、Go のモデル・API レスポンス・集計は **外部との境界で文字列**または `big.Float` / `big.Int` を利用。`float64` を資産金額に一切使わない（`portfolio_aggregation_service.go` の `parseDecimal` / `formatFloat`、`exchange_balances.free_balance NUMERIC(38,8)`、`user_assets.balance NUMERIC(78,0)`）。

**理由**:
- 暗号資産は桁数が大きい（wei: 10^18、大量供給トークン）かつ小数が絡むため、**float64 の丸め誤差は金額として許容できない**
- 文字列で受け渡しすることで、クライアント側が任意精度（decimal.js 等）で扱う選択肢を残す
- `NUMERIC(78,0)` は uint256 の最大値 `2^256-1`（≈10^77）をちょうど収められる最大桁として選定（オンチェーン値の丸めなし保存）

### 4.4 セキュリティ設計

| 領域 | 実装 | 理由 |
|------|------|------|
| パスワード | bcrypt ハッシュ保存 | 逆算困難・コスト調整可能 |
| トークン | JWT HS256、`JWT_SECRET` を env 注入、24h 期限 | ステートレス認証。対称鍵方式は小規模でも運用可能（RSA/ED25519 は将来の多検証者要件で検討） |
| API キー | AES-256-GCM で暗号化して `exchange_credentials` に格納、`ENCRYPTION_KEY`（32byte hex）を env 注入 | **DB 流出時もキー平文が漏れない**。GCM は認証付き暗号で改ざん検知も可能 |
| 応答 | `model.ExchangeCredential` の暗号バイト列は JSON 対象外（`json:"-"`）、サービス層でも nil 化（`exchange_service.go:86-90`） | 暗号文すら API レスポンスに載せない |
| 認可 | JWT に含む userID と各クエリの `WHERE user_id` を常に一致させる。ウォレット資産取得では所有権チェック実施（`wallet_service.go:88`） | マルチテナント間での資産の横断閲覧防止 |

### 4.5 バックグラウンド同期ジョブ

**設計**: アプリプロセス内 goroutine + `context.WithCancel` + `time.Ticker`。価格 60s / 残高 300s / DeFi 900s。`main.go` の `defer stopSync()` でグレースフル停止。

**理由**:
- 外部ジョブキュー（Redis/Celery 等）を導入せず**依存を最小化**。個人〜チーム規模で十分
- context キャンセルを全 goroutine に伝播させ、**goroutine リーク**と `ticker` の停止を保証（実装計画の Refinement で明記された要件）
- 残高更新は取引所が実質ステートマシン的に返すため、5 分間隔でも十分新鮮。間隔は env で調整可能にした

### 4.6 マルチテナント分離（user_id 軸）

**設計**: テナント分離要件を「SaaS」ではなく「内部ツール前提のシンプルな user_id フィルタ」と仮定。取引所・ウォレットはすべて `user_id` でフィルタ、`portfolio_snapshots.user_id`（0007）もその延長。

**理由**:
- 実装計画の Missing Information / Open Question で「内部ツールと仮定」とした。この仮定が崩れる（SaaS 化）場合は、テナント ID の一元管理や row-level security への移行が必要になる
- 一方で G3 のとおり `portfolio/history` は未対応で、**分離の穴が残っている**（最優先修正候補）

### 4.7 DeFi 評価の「保守的」設計（G2 の背景）

**設計**: 集計サービスは DeFi ポジションの `position_json` を解釈せず、USD を **0** としてソース内訳だけ記録する。

**理由**:
- Uniswap LP の価値計算はトークン残高の on-chain 換算・未収手数料の算出が必要で、**JSON の乱解析で「間違った金額」を出すリスク**を避けた意図的決定
- 「$0 と誤表示」は「過大表示」より害が少なく、集計の整合性（`wallet + exchange + defi = total`）を保てる
- 将来はシェア算出（`balanceOf`）やオラクル価格を組み合わせる拡張ポイントとして、`PositionJSON` に raw データを保持し続けている

### 4.8 DB マイグレーション方針

**設計**: `deployments/migrations/` に連番 + `.up` / `.down` ペア。WebFW 側に ORM マイグレーションを使わず（gorm はモデルタグのみで DB 操作不使用）、SQL ファイルを `migrate` CLI で適用。

**理由**:
- `plan.md` で「migration ファイルの重複・矛盾」という実際の事故を踏まえ、**バージョン管理はツールに一任**しつつ直感的な差分レビューを確保
- `.down.sql` を揃えることでロールバック戦略を担保（実装計画の Validation Strategy で必須化）
- スキーマ変更（0003 のユニーク制約、0006 のインデックス、0007 の user_id）はすべてこの仕組みで適用済み

### 4.9 テスト戦略

- service 層: `store/mock_*.go`（testify）によるユニットテスト
- HTTP 層: handler ユニットテスト
- クライアント: `NewWithBaseURL` / `NewWithAddress` で外部 API・ノードを模擬
- pre-commit で `gofmt / goimports / go vet / golangci-lint / gosec / go test` を強制

**理由**: 外部依存（取引所・ノード・価格）が不安定なため、「実サービスに依存しない再現可能なテスト」を優先。将来は store 層の統合テスト（`TEST_DATABASE_URL`）と Anvil によるオンチェーン E2E を追加予定（実装計画 Phase 5 参照）。

### 4.10 トレードオフと代替案

| 選択 | トレードオフ | 代替案・その場合 |
|------|-------------|-----------------|
| 金額を文字列/big で扱う | コードが冗長になり、計算効率が落ちる | `shopspring/decimal` 等のライブラリ導入で簡潔化 |
| HS256 対称 JWT | 鍵を複数サービスで共有できず、失効管理が弱い | RS256/ES256 + 公開鍵検証（JWKS）へ移行可能 |
| プロセス内 ticker | 複数インスタンス化すると同期が重複する | `ON CONFLICT DO NOTHING` の upssert で冪等性は担保。スケール時はジョブキューへ |
| AES-256-GCM の env キー | キーローテーション・監査が弱い | AWS KMS / Vault によるマスターキー管理（Open Question として残存） |
| 全外部依存を抽象化 | ボイラープレート増、トレースが分断（1 層挟む） | 抽象を最小限に絞る。ただし取引所追加（G5）のコストを再調査 |

---

## 5. ロードマップ（残作業、優先度順）

実装状況（3.5）に対応する推奨の次アクションです。

| 優先度 | タスク | 対応ギャップ |
|--------|--------|--------------|
| P0 | `portfolio/history` を user_id でフィルタ（service/store に userID を伝播） | G3 |
| P0 | 1 時間スナップショットジョブを実装（集計サービスを呼んで保存） | G4 |
| P1 | DeFi 定期同期を実装（walletStore に全ウォレット列挙 API を追加し `runSync` を実装） | G1 |
| P1 | DeFi の USD 評価を実装（LP シェア × トークン価格） | G2 |
| P2 | 取引履歴を DB 保存 + API 公開、`user_assets` への反映 | G9 |
| P2 | Coinbase クライアント追加（警告: `DefaultExchangeClientFactory` のみ変更） | G5 |
| P3 | ERC-20 / NFT 取得、`SLAVE_DATABASE_URL` の読み書き分離 | G7, G8 |
| P3 | 構造化ロガー導入、エラーの `errors.Is` 統一、`configs/config.go` 削除 | G10, G12 |

---

## 付録: 環境変数一覧（`internal/config`）

| 変数 | 必須 | デフォルト | 説明 |
|------|------|-----------|------|
| `MASTER_DATABASE_URL` | ✅ | — | PostgreSQL 接続文字列 |
| `ETHEREUM_NODE_URL` | ✅ | — | JSON-RPC ノード |
| `JWT_SECRET` | ✅ | — | JWT 署名シークレット |
| `ENCRYPTION_KEY` | ✅ | — | AES-256-GCM キー（64 hex 文字） |
| `PORT` | | `8080` | HTTP ポート |
| `JWT_EXPIRY_HOURS` | | `24` | トークン有効時間 |
| `PRICE_SYMBOLS` | | `BTC,ETH,BNB,SOL,USDT,USDC,ADA,DOGE` | 価格追跡銘柄 |
| `PRICE_SYNC_INTERVAL_S` | | `60` | 価格同期間隔 |
| `EXCHANGE_SYNC_INTERVAL_S` | | `300` | 残高同期間隔 |
| `DEFI_SYNC_INTERVAL_S` | | `900` | DeFi 同期間隔 |
| `SLAVE_DATABASE_URL` | | — | 読み取り専用 DB（未使用） |
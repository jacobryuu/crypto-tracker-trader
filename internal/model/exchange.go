package model

import "time"

// ExchangeCredential holds encrypted API credentials for a user's exchange account.
type ExchangeCredential struct {
	ID                 uint64    `json:"id"`
	UserID             uint64    `json:"user_id"`
	Exchange           string    `json:"exchange"`
	APIKeyEncrypted    []byte    `json:"-"`
	APISecretEncrypted []byte    `json:"-"`
	IsActive           bool      `json:"is_active"`
	CreatedAt          time.Time `json:"created_at"`
}

// ExchangeBalance holds the last-known balance for a symbol at a given exchange.
type ExchangeBalance struct {
	ID            uint64    `json:"id"`
	CredentialID  uint64    `json:"credential_id"`
	UserID        uint64    `json:"user_id"`
	Symbol        string    `json:"symbol"`
	FreeBalance   string    `json:"free_balance"`
	LockedBalance string    `json:"locked_balance"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// PortfolioSummary is the aggregated portfolio view for a user.
type PortfolioSummary struct {
	UserID      uint64            `json:"user_id"`
	TotalUSD    string            `json:"total_usd"`
	WalletUSD   string            `json:"wallet_usd"`
	ExchangeUSD string            `json:"exchange_usd"`
	DefiUSD     string            `json:"defi_usd"`
	BySymbol    []SymbolPosition  `json:"by_symbol"`
	BySource    []SourceBreakdown `json:"by_source"`
	ComputedAt  time.Time         `json:"computed_at"`
}

// SymbolPosition holds aggregated qty and USD value for a single asset symbol.
type SymbolPosition struct {
	Symbol   string `json:"symbol"`
	TotalQty string `json:"total_qty"`
	PriceUSD string `json:"price_usd"`
	ValueUSD string `json:"value_usd"`
	PctShare string `json:"pct_share"`
}

// SourceBreakdown describes the USD contribution from a single data source.
type SourceBreakdown struct {
	Source   string `json:"source"` // "wallet", "exchange:binance", "defi:uniswap-v3"
	ValueUSD string `json:"value_usd"`
	PctShare string `json:"pct_share"`
}

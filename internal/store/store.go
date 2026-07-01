package store

import "crypto-tracker-trader/internal/model"

type PortfolioStoreInterface interface {
	AddSnapshot(snapshot model.PortfolioSnapshot) error
	GetHistory() ([]model.PortfolioSnapshot, error)
	Close()
}

type WalletStoreInterface interface {
	CreateWallet(wallet *model.UserWallet) error
	GetWalletsByUserID(userID uint64) ([]model.UserWallet, error)
	GetWalletByID(id uint64) (*model.UserWallet, error)
	DeleteWallet(walletID, userID uint64) error
	GetAssetsByWalletID(walletID uint64) ([]model.UserAsset, error)
}

type PriceStoreInterface interface {
	SavePrice(symbol, priceUSD, source string) error
	GetLatestPrice(symbol string) (*model.AssetPrice, error)
	GetPriceHistory(symbol string, limit int) ([]model.AssetPrice, error)
}

type ExchangeStoreInterface interface {
	CreateCredential(cred *model.ExchangeCredential) error
	GetCredentialsByUserID(userID uint64) ([]model.ExchangeCredential, error)
	GetCredentialByID(id uint64) (*model.ExchangeCredential, error)
	GetAllActiveCredentials() ([]model.ExchangeCredential, error)
	DeleteCredential(id, userID uint64) error
	UpsertBalances(credentialID, userID uint64, balances []model.ExchangeBalance) error
	GetBalancesByUserID(userID uint64) ([]model.ExchangeBalance, error)
	GetBalancesByCredentialID(credentialID uint64) ([]model.ExchangeBalance, error)
}

type DefiStoreInterface interface {
	UpsertPosition(pos *model.UserDefiPosition) error
	GetPositionsByWalletID(walletID uint64) ([]model.UserDefiPosition, error)
	GetPositionsByUserID(userID uint64) ([]model.UserDefiPosition, error)
}

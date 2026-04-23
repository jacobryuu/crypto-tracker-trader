package model

import (
	"time"

	"gorm.io/datatypes"
)

type User struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Credentials   UserCredential     `gorm:"foreignKey:UserID"`
	AuthProviders []UserAuthProvider `gorm:"foreignKey:UserID"`
	Wallets       []UserWallet       `gorm:"foreignKey:UserID"`
}

type UserCredential struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	UserID       uint64    `json:"user_id"`
	PasswordHash string    `json:"-"`
	MFAEnabled   bool      `json:"mfa_enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserAuthProvider struct {
	ID             uint64    `gorm:"primaryKey" json:"id"`
	UserID         uint64    `json:"user_id"`
	Provider       string    `json:"provider"`
	ProviderUserID string    `json:"provider_user_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type UserWallet struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	UserID    uint64    `json:"user_id"`
	Chain     string    `json:"chain"`
	Address   string    `json:"address"`
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Assets       []UserAsset        `gorm:"foreignKey:WalletID"`
	NFTs         []UserNFT          `gorm:"foreignKey:WalletID"`
	DefiPosition []UserDefiPosition `gorm:"foreignKey:WalletID"`
}

type UserAsset struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	WalletID  uint64    `json:"wallet_id"`
	Chain     string    `json:"chain"`
	TokenAddr string    `json:"token_address"`
	Symbol    string    `json:"symbol"`
	Balance   string    `json:"balance"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserNFT struct {
	ID           uint64         `gorm:"primaryKey" json:"id"`
	WalletID     uint64         `json:"wallet_id"`
	Chain        string         `json:"chain"`
	ContractAddr string         `json:"contract_address"`
	TokenID      string         `json:"token_id"`
	MetadataJSON datatypes.JSON `json:"metadata_json"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type UserDefiPosition struct {
	ID           uint64         `gorm:"primaryKey" json:"id"`
	WalletID     uint64         `json:"wallet_id"`
	Protocol     string         `json:"protocol"`
	PositionType string         `json:"position_type"`
	PositionJSON datatypes.JSON `json:"position_json"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

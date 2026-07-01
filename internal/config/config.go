package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// DefaultPriceSymbols is the set of symbols tracked when PRICE_SYMBOLS is not set.
var DefaultPriceSymbols = []string{"BTC", "ETH", "BNB", "SOL", "USDT", "USDC", "ADA", "DOGE"}

type Config struct {
	Port                   string
	MasterDatabaseURL      string
	SlaveDatabaseURL       string
	EthereumNodeURL        string
	JWTSecret              string
	JWTExpiryHours         int
	PriceSymbols           []string
	PriceSyncIntervalS     int // seconds between price sync runs (default 60)
	EncryptionKey          string // 32-byte hex for AES-256-GCM (64 hex chars)
	ExchangeSyncIntervalS  int // seconds between exchange balance sync (default 300)
	DefiSyncIntervalS      int // seconds between DeFi position sync (default 900)
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Printf("Error loading .env file, using system environment variables: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	jwtExpiryHours := 24
	if v := os.Getenv("JWT_EXPIRY_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			jwtExpiryHours = n
		}
	}

	priceSymbols := DefaultPriceSymbols
	if v := os.Getenv("PRICE_SYMBOLS"); v != "" {
		var syms []string
		for _, s := range strings.Split(v, ",") {
			if t := strings.TrimSpace(strings.ToUpper(s)); t != "" {
				syms = append(syms, t)
			}
		}
		if len(syms) > 0 {
			priceSymbols = syms
		}
	}

	priceSyncIntervalS := 60
	if v := os.Getenv("PRICE_SYNC_INTERVAL_S"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			priceSyncIntervalS = n
		}
	}

	exchangeSyncIntervalS := 300
	if v := os.Getenv("EXCHANGE_SYNC_INTERVAL_S"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			exchangeSyncIntervalS = n
		}
	}

	defiSyncIntervalS := 900
	if v := os.Getenv("DEFI_SYNC_INTERVAL_S"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			defiSyncIntervalS = n
		}
	}

	return &Config{
		Port:                  port,
		MasterDatabaseURL:     os.Getenv("MASTER_DATABASE_URL"),
		SlaveDatabaseURL:      os.Getenv("SLAVE_DATABASE_URL"),
		EthereumNodeURL:       os.Getenv("ETHEREUM_NODE_URL"),
		JWTSecret:             os.Getenv("JWT_SECRET"),
		JWTExpiryHours:        jwtExpiryHours,
		PriceSymbols:          priceSymbols,
		PriceSyncIntervalS:    priceSyncIntervalS,
		EncryptionKey:         os.Getenv("ENCRYPTION_KEY"),
		ExchangeSyncIntervalS: exchangeSyncIntervalS,
		DefiSyncIntervalS:     defiSyncIntervalS,
	}, nil
}

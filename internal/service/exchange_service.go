package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	appCrypto "crypto-tracker-trader/internal/crypto"
	"crypto-tracker-trader/internal/client/exchange"
	"crypto-tracker-trader/internal/client/exchange/binance"
	"crypto-tracker-trader/internal/model"
	"crypto-tracker-trader/internal/store"
)

// ExchangeClientFactory creates an ExchangeClient for the given exchange name and credentials.
type ExchangeClientFactory func(exchangeName, apiKey, apiSecret string) (exchange.ExchangeClient, error)

// DefaultExchangeClientFactory supports Binance.
func DefaultExchangeClientFactory(exchangeName, apiKey, apiSecret string) (exchange.ExchangeClient, error) {
	switch strings.ToLower(exchangeName) {
	case "binance":
		return binance.New(apiKey, apiSecret), nil
	default:
		return nil, fmt.Errorf("unsupported exchange: %s", exchangeName)
	}
}

// ExchangeService handles exchange credential management and balance synchronisation.
type ExchangeService struct {
	store         store.ExchangeStoreInterface
	encryptionKey string
	clientFactory ExchangeClientFactory
}

// NewExchangeService creates an ExchangeService with the default Binance client factory.
func NewExchangeService(s store.ExchangeStoreInterface, encryptionKey string) *ExchangeService {
	return &ExchangeService{
		store:         s,
		encryptionKey: encryptionKey,
		clientFactory: DefaultExchangeClientFactory,
	}
}

// NewExchangeServiceWithFactory allows injecting a custom client factory (useful for testing).
func NewExchangeServiceWithFactory(s store.ExchangeStoreInterface, encryptionKey string, f ExchangeClientFactory) *ExchangeService {
	return &ExchangeService{store: s, encryptionKey: encryptionKey, clientFactory: f}
}

// AddCredential encrypts the API credentials and persists them.
func (s *ExchangeService) AddCredential(ctx context.Context, userID uint64, exchangeName, apiKey, apiSecret string) (*model.ExchangeCredential, error) {
	exchangeName = strings.ToLower(exchangeName)
	if exchangeName == "" || apiKey == "" || apiSecret == "" {
		return nil, fmt.Errorf("exchange, api_key and api_secret are required")
	}

	encKey, err := appCrypto.Encrypt(s.encryptionKey, []byte(apiKey))
	if err != nil {
		return nil, fmt.Errorf("exchange service: encrypting api key: %w", err)
	}
	encSecret, err := appCrypto.Encrypt(s.encryptionKey, []byte(apiSecret))
	if err != nil {
		return nil, fmt.Errorf("exchange service: encrypting api secret: %w", err)
	}

	cred := &model.ExchangeCredential{
		UserID:             userID,
		Exchange:           exchangeName,
		APIKeyEncrypted:    encKey,
		APISecretEncrypted: encSecret,
		IsActive:           true,
	}
	if err := s.store.CreateCredential(cred); err != nil {
		return nil, fmt.Errorf("exchange service: saving credential: %w", err)
	}
	return cred, nil
}

// GetCredentials returns the user's exchange credentials (without encrypted key bytes).
func (s *ExchangeService) GetCredentials(ctx context.Context, userID uint64) ([]model.ExchangeCredential, error) {
	creds, err := s.store.GetCredentialsByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("exchange service: get credentials: %w", err)
	}
	// Strip encrypted bytes from the response (security: don't expose even ciphertext)
	for i := range creds {
		creds[i].APIKeyEncrypted = nil
		creds[i].APISecretEncrypted = nil
	}
	return creds, nil
}

// DeleteCredential removes an exchange credential (and cascades to balances).
func (s *ExchangeService) DeleteCredential(ctx context.Context, credentialID, userID uint64) error {
	err := s.store.DeleteCredential(credentialID, userID)
	if err == store.ErrNotFound {
		return store.ErrNotFound
	}
	return err
}

// SyncBalances fetches live balances for the given credential and upserts them.
func (s *ExchangeService) SyncBalances(ctx context.Context, credentialID uint64) ([]model.ExchangeBalance, error) {
	cred, err := s.store.GetCredentialByID(credentialID)
	if err != nil {
		return nil, fmt.Errorf("exchange service: loading credential: %w", err)
	}

	apiKey, err := appCrypto.Decrypt(s.encryptionKey, cred.APIKeyEncrypted)
	if err != nil {
		return nil, fmt.Errorf("exchange service: decrypting api key: %w", err)
	}
	apiSecret, err := appCrypto.Decrypt(s.encryptionKey, cred.APISecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("exchange service: decrypting api secret: %w", err)
	}

	client, err := s.clientFactory(cred.Exchange, string(apiKey), string(apiSecret))
	if err != nil {
		return nil, fmt.Errorf("exchange service: creating client for %s: %w", cred.Exchange, err)
	}

	rawBalances, err := client.GetBalances(ctx)
	if err != nil {
		return nil, fmt.Errorf("exchange service: fetching balances from %s: %w", cred.Exchange, err)
	}

	modelBalances := make([]model.ExchangeBalance, len(rawBalances))
	for i, b := range rawBalances {
		modelBalances[i] = model.ExchangeBalance{
			CredentialID:  credentialID,
			UserID:        cred.UserID,
			Symbol:        strings.ToUpper(b.Symbol),
			FreeBalance:   b.Free,
			LockedBalance: b.Locked,
		}
	}

	if err := s.store.UpsertBalances(credentialID, cred.UserID, modelBalances); err != nil {
		return nil, fmt.Errorf("exchange service: upserting balances: %w", err)
	}
	return modelBalances, nil
}

// GetBalances returns the latest known balances for a user across all exchanges.
func (s *ExchangeService) GetBalances(ctx context.Context, userID uint64) ([]model.ExchangeBalance, error) {
	return s.store.GetBalancesByUserID(userID)
}

// StartSync launches a goroutine that periodically syncs all active credentials.
func (s *ExchangeService) StartSync(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		s.runSync(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runSync(ctx)
			}
		}
	}()
}

func (s *ExchangeService) runSync(ctx context.Context) {
	creds, err := s.store.GetAllActiveCredentials()
	if err != nil {
		log.Printf("exchange sync: loading credentials: %v", err)
		return
	}
	for _, cred := range creds {
		if ctx.Err() != nil {
			return
		}
		if _, err := s.SyncBalances(ctx, cred.ID); err != nil {
			log.Printf("exchange sync: credential %d (%s): %v", cred.ID, cred.Exchange, err)
		}
	}
}

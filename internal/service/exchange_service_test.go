package service

import (
	"context"
	"errors"
	"testing"
	"time"

	appCrypto "crypto-tracker-trader/internal/crypto"
	"crypto-tracker-trader/internal/client/exchange"
	"crypto-tracker-trader/internal/model"
	"crypto-tracker-trader/internal/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testEncryptionKey = "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"

// mockExchangeClient fakes the exchange client for tests.
type mockExchangeClient struct {
	mock.Mock
}

func (m *mockExchangeClient) GetBalances(ctx context.Context) ([]exchange.Balance, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]exchange.Balance), args.Error(1)
}

func (m *mockExchangeClient) GetTradeHistory(ctx context.Context, symbol string, since time.Time) ([]exchange.Trade, error) {
	args := m.Called(ctx, symbol, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]exchange.Trade), args.Error(1)
}

func newTestExchangeService(t *testing.T) (*ExchangeService, *store.MockExchangeStore, *mockExchangeClient) {
	t.Helper()
	mockStore := new(store.MockExchangeStore)
	mockClient := new(mockExchangeClient)
	svc := NewExchangeServiceWithFactory(mockStore, testEncryptionKey, func(exch, key, secret string) (exchange.ExchangeClient, error) {
		return mockClient, nil
	})
	return svc, mockStore, mockClient
}

func TestAddCredential_Success(t *testing.T) {
	svc, mockStore, _ := newTestExchangeService(t)
	mockStore.On("CreateCredential", mock.AnythingOfType("*model.ExchangeCredential")).
		Return(nil).Once()

	cred, err := svc.AddCredential(context.Background(), 1, "binance", "my-api-key", "my-secret")
	require.NoError(t, err)
	assert.Equal(t, "binance", cred.Exchange)
	assert.NotNil(t, cred.APIKeyEncrypted)

	// Encrypted value should decrypt back to original
	plain, err := appCrypto.Decrypt(testEncryptionKey, cred.APIKeyEncrypted)
	require.NoError(t, err)
	assert.Equal(t, "my-api-key", string(plain))
	mockStore.AssertExpectations(t)
}

func TestAddCredential_MissingFields(t *testing.T) {
	svc, _, _ := newTestExchangeService(t)
	_, err := svc.AddCredential(context.Background(), 1, "", "key", "secret")
	assert.Error(t, err)

	_, err = svc.AddCredential(context.Background(), 1, "binance", "", "secret")
	assert.Error(t, err)
}

func TestGetCredentials_StripEncryptedBytes(t *testing.T) {
	svc, mockStore, _ := newTestExchangeService(t)
	stored := []model.ExchangeCredential{
		{ID: 1, UserID: 1, Exchange: "binance", APIKeyEncrypted: []byte("secret"), APISecretEncrypted: []byte("secret")},
	}
	mockStore.On("GetCredentialsByUserID", uint64(1)).Return(stored, nil)

	creds, err := svc.GetCredentials(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, creds, 1)
	assert.Nil(t, creds[0].APIKeyEncrypted, "encrypted bytes must be stripped from response")
	assert.Nil(t, creds[0].APISecretEncrypted)
}

func TestDeleteCredential_Success(t *testing.T) {
	svc, mockStore, _ := newTestExchangeService(t)
	mockStore.On("DeleteCredential", uint64(1), uint64(1)).Return(nil)

	err := svc.DeleteCredential(context.Background(), 1, 1)
	assert.NoError(t, err)
}

func TestDeleteCredential_NotFound(t *testing.T) {
	svc, mockStore, _ := newTestExchangeService(t)
	mockStore.On("DeleteCredential", uint64(99), uint64(1)).Return(store.ErrNotFound)

	err := svc.DeleteCredential(context.Background(), 99, 1)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestSyncBalances_Success(t *testing.T) {
	svc, mockStore, mockClient := newTestExchangeService(t)

	encKey, _ := appCrypto.Encrypt(testEncryptionKey, []byte("api-key"))
	encSecret, _ := appCrypto.Encrypt(testEncryptionKey, []byte("api-secret"))
	cred := &model.ExchangeCredential{
		ID: 1, UserID: 1, Exchange: "binance",
		APIKeyEncrypted: encKey, APISecretEncrypted: encSecret,
	}
	mockStore.On("GetCredentialByID", uint64(1)).Return(cred, nil)
	mockClient.On("GetBalances", mock.Anything).Return([]exchange.Balance{
		{Symbol: "BTC", Free: "0.5", Locked: "0"},
		{Symbol: "ETH", Free: "10", Locked: "2"},
	}, nil)
	mockStore.On("UpsertBalances", uint64(1), uint64(1), mock.Anything).Return(nil)

	balances, err := svc.SyncBalances(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, balances, 2)
	assert.Equal(t, "BTC", balances[0].Symbol)
}

func TestSyncBalances_ClientError(t *testing.T) {
	svc, mockStore, mockClient := newTestExchangeService(t)

	encKey, _ := appCrypto.Encrypt(testEncryptionKey, []byte("k"))
	encSecret, _ := appCrypto.Encrypt(testEncryptionKey, []byte("s"))
	mockStore.On("GetCredentialByID", uint64(1)).Return(&model.ExchangeCredential{
		ID: 1, Exchange: "binance", APIKeyEncrypted: encKey, APISecretEncrypted: encSecret,
	}, nil)
	mockClient.On("GetBalances", mock.Anything).Return(nil, errors.New("API error"))

	_, err := svc.SyncBalances(context.Background(), 1)
	assert.Error(t, err)
}

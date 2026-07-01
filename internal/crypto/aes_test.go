package crypto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testKey is a valid 32-byte (64 hex char) key for tests.
const testKey = "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		plaintext string
	}{
		{"short value", "hello"},
		{"API key", "my-binance-api-key-abc123"},
		{"long secret", strings.Repeat("x", 200)},
		{"empty", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ct, err := Encrypt(testKey, []byte(tt.plaintext))
			require.NoError(t, err)
			assert.Greater(t, len(ct), 0)

			got, err := Decrypt(testKey, ct)
			require.NoError(t, err)
			assert.Equal(t, tt.plaintext, string(got))
		})
	}
}

func TestEncrypt_DifferentEachTime(t *testing.T) {
	ct1, _ := Encrypt(testKey, []byte("same input"))
	ct2, _ := Encrypt(testKey, []byte("same input"))
	assert.NotEqual(t, ct1, ct2, "each encryption should produce different ciphertext due to random nonce")
}

func TestDecrypt_WrongKey(t *testing.T) {
	ct, _ := Encrypt(testKey, []byte("secret"))
	wrongKey := "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	_, err := Decrypt(wrongKey, ct)
	assert.Error(t, err)
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	ct, _ := Encrypt(testKey, []byte("secret"))
	ct[len(ct)-1] ^= 0xff // flip last byte
	_, err := Decrypt(testKey, ct)
	assert.Error(t, err)
}

func TestDecrypt_TooShort(t *testing.T) {
	_, err := Decrypt(testKey, []byte("short"))
	assert.Error(t, err)
}

func TestEncrypt_InvalidKey(t *testing.T) {
	_, err := Encrypt("notahexkey", []byte("text"))
	assert.Error(t, err)

	// Valid hex but wrong length (16 bytes = 32 hex chars, not 32 bytes)
	_, err = Encrypt("0102030405060708090a0b0c0d0e0f10", []byte("text"))
	assert.Error(t, err)
}

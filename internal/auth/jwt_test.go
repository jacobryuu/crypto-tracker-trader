package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key-32-bytes-padding!"

func TestGenerateToken(t *testing.T) {
	t.Run("valid token generated", func(t *testing.T) {
		token, err := GenerateToken(42, testSecret, 24)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("empty secret returns error", func(t *testing.T) {
		_, err := GenerateToken(1, "", 24)
		assert.Error(t, err)
	})
}

func TestValidateToken(t *testing.T) {
	t.Run("valid token round-trips correctly", func(t *testing.T) {
		token, err := GenerateToken(99, testSecret, 24)
		require.NoError(t, err)

		claims, err := ValidateToken(token, testSecret)
		require.NoError(t, err)
		assert.Equal(t, uint64(99), claims.UserID)
	})

	t.Run("wrong secret returns error", func(t *testing.T) {
		token, _ := GenerateToken(1, testSecret, 24)
		_, err := ValidateToken(token, "wrong-secret")
		assert.Error(t, err)
	})

	t.Run("expired token returns error", func(t *testing.T) {
		claims := Claims{
			UserID: 1,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, _ := token.SignedString([]byte(testSecret))

		_, err := ValidateToken(signed, testSecret)
		assert.Error(t, err)
	})

	t.Run("tampered token returns error", func(t *testing.T) {
		_, err := ValidateToken("eyJhbGciOiJIUzI1NiJ9.tampered.sig", testSecret)
		assert.Error(t, err)
	})

	t.Run("empty string returns error", func(t *testing.T) {
		_, err := ValidateToken("", testSecret)
		assert.Error(t, err)
	})
}

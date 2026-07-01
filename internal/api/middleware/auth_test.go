package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"crypto-tracker-trader/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
)

const testSecret = "test-secret-key-32-bytes-padding!"

func newTestRouter(secret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", Auth(secret), func(c *gin.Context) {
		id, _ := GetUserID(c)
		c.JSON(http.StatusOK, gin.H{"user_id": id})
	})
	return r
}

func TestAuth_ValidToken(t *testing.T) {
	token, err := auth.GenerateToken(42, testSecret, 24)
	assert.NoError(t, err)

	r := newTestRouter(testSecret)
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"user_id":42`)
}

func TestAuth_MissingHeader(t *testing.T) {
	r := newTestRouter(testSecret)
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "authorization header required")
}

func TestAuth_MalformedHeader(t *testing.T) {
	r := newTestRouter(testSecret)
	for _, h := range []string{"token-only", "Basic somebase64", "Bearer"} {
		req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", h)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code, "header: %s", h)
	}
}

func TestAuth_WrongSecret(t *testing.T) {
	token, _ := auth.GenerateToken(1, "other-secret-key-32-bytes-pad!!", 24)
	r := newTestRouter(testSecret)
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_ExpiredToken(t *testing.T) {
	claims := auth.Claims{
		UserID: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(testSecret))

	r := newTestRouter(testSecret)
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetUserID_NotSet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	id, ok := GetUserID(c)
	assert.False(t, ok)
	assert.Equal(t, uint64(0), id)
}

package middleware

import (
	"net/http"
	"strings"

	"crypto-tracker-trader/internal/auth"

	"github.com/gin-gonic/gin"
)

// UserIDKey is the key used to store the authenticated user ID in Gin's context.
const UserIDKey = "userID"

// Auth returns a Gin middleware that validates JWT Bearer tokens.
// Requests without a valid token are rejected with 401 Unauthorized.
func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header format must be: Bearer <token>"})
			return
		}

		claims, err := auth.ValidateToken(parts[1], jwtSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Next()
	}
}

// GetUserID retrieves the authenticated user ID from Gin's context.
// Returns 0, false if not set (i.e., middleware was not applied).
func GetUserID(c *gin.Context) (uint64, bool) {
	val, exists := c.Get(UserIDKey)
	if !exists {
		return 0, false
	}
	userID, ok := val.(uint64)
	return userID, ok
}

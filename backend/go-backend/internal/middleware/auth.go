package middleware

import (
	"net/http"
	"strings"

	"classroom-attendance/go-backend/internal/security"

	"github.com/gin-gonic/gin"
)

const ClaimsKey = "claims"

func AuthRequired(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		claims, err := security.ParseToken(secret, strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.Set(ClaimsKey, claims)
		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, role := range roles {
		allowed[role] = true
	}
	return func(c *gin.Context) {
		claims, ok := CurrentClaims(c)
		if !ok || !allowed[claims.Role] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}
		c.Next()
	}
}

func CurrentClaims(c *gin.Context) (security.Claims, bool) {
	value, exists := c.Get(ClaimsKey)
	if !exists {
		return security.Claims{}, false
	}
	claims, ok := value.(security.Claims)
	return claims, ok
}

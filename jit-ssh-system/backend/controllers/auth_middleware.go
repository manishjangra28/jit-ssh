package controllers

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/golang-jwt/jwt/v5"
)

const sessionDuration = 8 * time.Hour

type SessionClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	jwt.RegisteredClaims
}

func sessionSecret() []byte {
	secret := os.Getenv("JIT_SESSION_SECRET")
	if secret == "" {
		secret = "jit-dev-session-secret-change-me"
	}
	return []byte(secret)
}

func issueSessionToken(userID uuid.UUID, role, email, name string) (string, error) {
	now := time.Now()
	claims := SessionClaims{
		UserID: userID.String(),
		Role:   role,
		Email:  email,
		Name:   name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(sessionDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(sessionSecret())
}

func parseSessionToken(raw string) (*SessionClaims, error) {
	token, err := jwt.ParseWithClaims(raw, &SessionClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return sessionSecret(), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*SessionClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid session token")
	}
	return claims, nil
}

func extractSessionToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}

	if cookie, err := c.Cookie("jit_auth_token"); err == nil {
		return cookie
	}

	return ""
}

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := extractSessionToken(c)
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			return
		}

		claims, err := parseSessionToken(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired session"})
			return
		}

		c.Set("auth.user_id", claims.UserID)
		c.Set("auth.role", claims.Role)
		c.Set("auth.email", claims.Email)
		c.Set("auth.name", claims.Name)
		c.Next()
	}
}

func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(c *gin.Context) {
		role := currentUserRole(c)
		if _, ok := allowed[role]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
			return
		}
		c.Next()
	}
}

func currentUserID(c *gin.Context) string {
	if v, ok := c.Get("auth.user_id"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func currentUserRole(c *gin.Context) string {
	if v, ok := c.Get("auth.role"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func currentUserEmail(c *gin.Context) string {
	if v, ok := c.Get("auth.email"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func isAdminOrApprover(c *gin.Context) bool {
	role := currentUserRole(c)
	return role == "admin" || role == "approver"
}

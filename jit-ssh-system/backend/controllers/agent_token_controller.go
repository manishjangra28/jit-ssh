package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/manishjangra/jit-ssh-system/backend/db"
	"github.com/manishjangra/jit-ssh-system/backend/models"
)

// generateSecureToken creates a cryptographically-random 32-byte hex token.
func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// -----------------------------------------------------------------
// AgentAuthMiddleware validates Bearer tokens on /agent/* routes.
// The token must exist in the agent_tokens table.
// -----------------------------------------------------------------
func AgentAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Registration does NOT need a pre-existing token —
		// it validates a one-time registration token instead.
		// Other routes require a valid stored token.
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || len(authHeader) < 8 || authHeader[:7] != "Bearer " {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Missing or invalid Authorization header. Use: Authorization: Bearer <agent_token>",
			})
			return
		}

		rawToken := authHeader[7:]

		var token models.AgentToken
		if err := db.DB.Where("token = ?", rawToken).First(&token).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid agent token",
			})
			return
		}

		// Update last_used_at asynchronously (fire-and-forget)
		go func(id uuid.UUID) {
			now := time.Now()
			db.DB.Model(&models.AgentToken{}).Where("id = ?", id).Update("last_used_at", now)
		}(token.ID)

		// Attach token info to context for downstream use
		c.Set("agent_token_id", token.ID)
		c.Set("agent_token_server_id", token.ServerID)
		c.Next()
	}
}

// -----------------------------------------------------------------
// Token Management (Admin)
// -----------------------------------------------------------------

type CreateTokenPayload struct {
	Label string `json:"label" binding:"required"`
}

// CreateAgentToken generates a new agent token and returns the raw secret ONCE.
func CreateAgentToken(c *gin.Context) {
	var payload CreateTokenPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	raw, err := generateSecureToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	token := models.AgentToken{
		Token: raw,
		Label: payload.Label,
	}

	if err := db.DB.Create(&token).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save token"})
		return
	}

	// Return the raw token ONE TIME — it cannot be retrieved again
	c.JSON(http.StatusCreated, gin.H{
		"id":         token.ID,
		"label":      token.Label,
		"token":      raw, // shown ONCE
		"created_at": token.CreatedAt,
		"note":       "Copy this token into your agent config. It will NOT be shown again.",
	})
}

// ListAgentTokens returns all tokens (without the raw secret).
func ListAgentTokens(c *gin.Context) {
	var tokens []models.AgentToken
	if err := db.DB.Order("created_at desc").Find(&tokens).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tokens"})
		return
	}

	// We never expose the raw token — only metadata
	type TokenView struct {
		ID          uuid.UUID  `json:"id"`
		Label       string     `json:"label"`
		ServerID    *uuid.UUID `json:"server_id,omitempty"`
		CreatedAt   time.Time  `json:"created_at"`
		LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	}

	var views = []TokenView{}
	for _, t := range tokens {
		views = append(views, TokenView{
			ID:         t.ID,
			Label:      t.Label,
			ServerID:   t.ServerID,
			CreatedAt:  t.CreatedAt,
			LastUsedAt: t.LastUsedAt,
		})
	}

	c.JSON(http.StatusOK, views)
}

// RevokeAgentToken permanently deletes a token, disconnecting the agent.
func RevokeAgentToken(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Delete(&models.AgentToken{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Token revoked. The agent using this token can no longer connect."})
}

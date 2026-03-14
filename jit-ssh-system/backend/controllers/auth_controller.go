package controllers

import (
	"crypto/rand"
	"math/big"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/manishjangra/jit-ssh-system/backend/db"
	"github.com/manishjangra/jit-ssh-system/backend/models"
	"golang.org/x/crypto/bcrypt"
)

const passwordChars = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789!@#&"

func generateRandomPassword(length int) string {
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(passwordChars))))
		b[i] = passwordChars[n.Int64()]
	}
	return string(b)
}

type LoginPayload struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login authenticates a user by email + password
func Login(c *gin.Context) {
	var payload LoginPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := db.DB.Preload("Team").Where("email = ?", payload.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	if user.PasswordHash == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "password_not_set", "message": "No password set. Please ask an admin to set your password."})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(payload.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	token, err := issueSessionToken(user.ID, user.Role, user.Email, user.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	c.SetCookie("jit_auth_token", token, int(sessionDuration.Seconds()), "/", "", false, false)
	c.SetCookie("jit_auth_role", user.Role, int(sessionDuration.Seconds()), "/", "", false, false)
	c.SetCookie("jit_auth_name", user.Name, int(sessionDuration.Seconds()), "/", "", false, false)
	c.SetCookie("jit_auth_id", user.ID.String(), int(sessionDuration.Seconds()), "/", "", false, false)
	c.SetCookie("jit_auth_email", user.Email, int(sessionDuration.Seconds()), "/", "", false, false)

	// Return user info (frontend will set cookies)
	c.JSON(http.StatusOK, gin.H{
		"id":         user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"role":       user.Role,
		"team":       user.Team,
		"token":      token,
		"expires_at": time.Now().Add(sessionDuration),
	})
}

type SetPasswordPayload struct {
	UserID   string `json:"user_id" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

// SetPassword allows an admin to set a user's password
func SetPassword(c *gin.Context) {
	var payload SetPasswordPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	callerID := currentUserID(c)
	callerRole := currentUserRole(c)
	if callerRole != "admin" && callerID != payload.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	if err := db.DB.Model(&models.User{}).Where("id = ?", payload.UserID).Update("password_hash", string(hash)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password set successfully"})
}

// ResetPassword generates a fresh random password and returns it to the admin
func ResetPassword(c *gin.Context) {
	if currentUserRole(c) != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	userID := c.Param("id")

	plain := generateRandomPassword(12)

	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	if err := db.DB.Model(&models.User{}).Where("id = ?", userID).Update("password_hash", string(hash)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}

	// Return the plaintext password ONCE so admin can share it (never stored in plaintext)
	c.JSON(http.StatusOK, gin.H{"temp_password": plain})
}

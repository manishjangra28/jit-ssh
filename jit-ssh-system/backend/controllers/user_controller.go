package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/manishjangra/jit-ssh-system/backend/db"
	"github.com/manishjangra/jit-ssh-system/backend/models"
	"golang.org/x/crypto/bcrypt"
)

// GetUsers lists all users
func GetUsers(c *gin.Context) {
	var users []models.User
	if err := db.DB.Preload("Team").Order("created_at desc").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	c.JSON(http.StatusOK, users)
}

type CreateUserPayload struct {
	Name   string `json:"name" binding:"required"`
	Email  string `json:"email" binding:"required,email"`
	Role   string `json:"role" binding:"required"`
	TeamID string `json:"team_id"`
}

// CreateUser creates a new user
func CreateUser(c *gin.Context) {
	var payload CreateUserPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := models.User{
		Name:  payload.Name,
		Email: payload.Email,
		Role:  payload.Role, // Admin, Approver, Developer
	}

	if payload.TeamID != "" {
		if tid, err := uuid.Parse(payload.TeamID); err == nil {
			user.TeamID = &tid
		}
	}

	// Auto-generate a random one-time password for the new user
	tempPwd := generateRandomPassword(12)
	hash, err := bcrypt.GenerateFromPassword([]byte(tempPwd), bcrypt.DefaultCost)
	if err == nil {
		user.PasswordHash = string(hash)
	}

	if err := db.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Return the temp password ONCE so the admin can share it with the new user
	c.JSON(http.StatusCreated, gin.H{
		"user":         user,
		"temp_password": tempPwd,
	})
}

type UpdateUserPayload struct {
	Role   *string `json:"role"`
	TeamID *string `json:"team_id"` // Optional
	Name   *string `json:"name"`
}

// UpdateUser updates an existing user's details
func UpdateUser(c *gin.Context) {
	id := c.Param("id")

	var payload UpdateUserPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if payload.Role != nil {
		updates["role"] = *payload.Role
	}
	if payload.Name != nil {
		updates["name"] = *payload.Name
	}
	if payload.TeamID != nil {
		if *payload.TeamID == "" {
			updates["team_id"] = nil
		} else if tid, err := uuid.Parse(*payload.TeamID); err == nil {
			updates["team_id"] = &tid
		}
	}

	if err := db.DB.Model(&models.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated"})
}

// DeleteUser permanently removes a user
func DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Delete(&models.User{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

// ToggleUserStatus switches a user between active and inactive
func ToggleUserStatus(c *gin.Context) {
	id := c.Param("id")
	var user models.User
	if err := db.DB.First(&user, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	newStatus := "active"
	if user.Status == "active" {
		newStatus = "inactive"
	}
	if err := db.DB.Model(&models.User{}).Where("id = ?", id).Update("status", newStatus).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle status"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": newStatus})
}

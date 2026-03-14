package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/manishjangra/jit-ssh-system/backend/db"
	"github.com/manishjangra/jit-ssh-system/backend/models"
)

type AddProtectedUserRequest struct {
	Username string `json:"username" binding:"required"`
	Reason   string `json:"reason"`
}

// GetProtectedUsers returns the list of usernames that are globally protected from being locked or deleted by the JIT agent.
func GetProtectedUsers(c *gin.Context) {
	var protectedUsers []models.ProtectedUser
	if err := db.DB.Find(&protectedUsers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch protected users"})
		return
	}

	c.JSON(http.StatusOK, protectedUsers)
}

// AddProtectedUser adds a username to the protection list.
func AddProtectedUser(c *gin.Context) {
	var req AddProtectedUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	protected := models.ProtectedUser{
		Username: req.Username,
		Reason:   req.Reason,
	}

	if err := db.DB.Create(&protected).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to protect user. It may already be in the list."})
		return
	}

	c.JSON(http.StatusCreated, protected)
}

// DeleteProtectedUser removes a username from the protection list.
func DeleteProtectedUser(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Unscoped().Delete(&models.ProtectedUser{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove user protection"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User protection removed successfully"})
}

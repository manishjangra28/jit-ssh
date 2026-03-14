package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/manishjangra/jit-ssh-system/backend/db"
	"github.com/manishjangra/jit-ssh-system/backend/models"
	"github.com/manishjangra/jit-ssh-system/backend/pkg/cloud"
	"github.com/manishjangra/jit-ssh-system/backend/pkg/crypto"
)

type CreateCloudIntegrationRequest struct {
	Name        string `json:"name" binding:"required"`
	Provider    string `json:"provider" binding:"required"`    // aws, gcp, azure
	Credentials string `json:"credentials" binding:"required"` // Raw JSON string of credentials
	Metadata    string `json:"metadata" binding:"required"`    // JSON string for non-secret configs
}

type UpdateCloudIntegrationRequest struct {
	Name        string `json:"name"`
	Credentials string `json:"credentials"` // Optional: only provide if changing
	Metadata    string `json:"metadata"`
}

// CreateCloudIntegration securely stores a new cloud integration
func CreateCloudIntegration(c *gin.Context) {
	var req CreateCloudIntegrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key, err := crypto.GetMasterKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Encryption key is not configured on the server"})
		return
	}

	encryptedCreds, err := crypto.EncryptString(req.Credentials, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt credentials securely"})
		return
	}

	integration := models.CloudIntegration{
		Name:                 req.Name,
		Provider:             models.CloudProviderType(req.Provider),
		EncryptedCredentials: []byte(encryptedCreds),
		Metadata:             req.Metadata,
		Status:               "active",
	}

	if err := db.DB.Create(&integration).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save cloud integration"})
		return
	}

	c.JSON(http.StatusCreated, integration)
}

// GetCloudIntegrations lists all configured cloud integrations (without exposing credentials)
func GetCloudIntegrations(c *gin.Context) {
	var integrations []models.CloudIntegration
	if err := db.DB.Find(&integrations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cloud integrations"})
		return
	}

	c.JSON(http.StatusOK, integrations)
}

// DeleteCloudIntegration removes an integration
func DeleteCloudIntegration(c *gin.Context) {
	id := c.Param("id")

	// Check if it exists first
	var integration models.CloudIntegration
	if err := db.DB.First(&integration, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}

	if err := db.DB.Delete(&integration).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete integration"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cloud integration deleted successfully"})
}

// TestCloudIntegration verifies the stored credentials can authenticate with the target cloud provider
func TestCloudIntegration(c *gin.Context) {
	id := c.Param("id")

	var integration models.CloudIntegration
	if err := db.DB.First(&integration, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}

	provider, err := cloud.NewProvider(&integration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize provider: " + err.Error()})
		return
	}

	err = provider.TestConnection(c.Request.Context())
	if err != nil {
		// Mark as error in DB
		db.DB.Model(&integration).Update("status", "error")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Connection test failed",
			"details": err.Error(),
		})
		return
	}

	// Mark as active in DB if successful
	db.DB.Model(&integration).Update("status", "active")
	c.JSON(http.StatusOK, gin.H{"message": "Connection to cloud provider successful"})
}

// UpdateCloudIntegration updates an existing cloud integration, re-encrypting credentials if they are provided
func UpdateCloudIntegration(c *gin.Context) {
	id := c.Param("id")

	var integration models.CloudIntegration
	if err := db.DB.First(&integration, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}

	var req UpdateCloudIntegrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" {
		integration.Name = req.Name
	}
	if req.Metadata != "" {
		integration.Metadata = req.Metadata
	}

	// If new credentials are provided, re-encrypt them
	if req.Credentials != "" {
		key, err := crypto.GetMasterKey()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Encryption key is not configured"})
			return
		}
		encryptedCreds, err := crypto.EncryptString(req.Credentials, key)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to re-encrypt credentials"})
			return
		}
		integration.EncryptedCredentials = []byte(encryptedCreds)
	}

	if err := db.DB.Save(&integration).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update integration"})
		return
	}

	c.JSON(http.StatusOK, integration)
}

// GetCloudIntegrationGroups fetches the groups dynamically from the cloud provider
func GetCloudIntegrationGroups(c *gin.Context) {
	id := c.Param("id")

	var integration models.CloudIntegration
	if err := db.DB.First(&integration, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}

	provider, err := cloud.NewProvider(&integration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize provider: " + err.Error()})
		return
	}

	groups, err := provider.ListGroups(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch groups from cloud provider",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, groups)
}

package controllers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/manishjangra/jit-ssh-system/backend/db"
	"github.com/manishjangra/jit-ssh-system/backend/models"
	"github.com/manishjangra/jit-ssh-system/backend/pkg/cloud"
)

type CreateCloudRequestPayload struct {
	IntegrationID    string `json:"integration_id" binding:"required"`
	TargetGroupID    string `json:"target_group_id" binding:"required"`
	TargetGroupName  string `json:"target_group_name" binding:"required"`
	DurationHours    int    `json:"duration_hours" binding:"required"`
	Reason           string `json:"reason" binding:"required"`
	RequiresPassword bool   `json:"requires_password"`
	RequiresKeys     bool   `json:"requires_keys"`
}

func CreateCloudRequest(c *gin.Context) {
	var payload CreateCloudRequestPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req := models.CloudAccessRequest{
		UserID:           currentUserID(c),
		IntegrationID:    payload.IntegrationID,
		TargetGroupID:    payload.TargetGroupID,
		TargetGroupName:  payload.TargetGroupName,
		DurationHours:    payload.DurationHours,
		Reason:           payload.Reason,
		RequiresPassword: payload.RequiresPassword,
		RequiresKeys:     payload.RequiresKeys,
		Status:           "pending",
	}

	if err := db.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create cloud request"})
		return
	}

	c.JSON(http.StatusCreated, req)
}

func GetCloudRequests(c *gin.Context) {
	var requests []models.CloudAccessRequest
	query := db.DB.Preload("User").Preload("Integration")
	if !isAdminOrApprover(c) {
		query = query.Where("user_id = ?", currentUserID(c))
	}
	if err := query.Find(&requests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cloud requests"})
		return
	}
	c.JSON(http.StatusOK, requests)
}

func ApproveCloudRequest(c *gin.Context) {
	id := c.Param("id")

	var payload struct {
		TargetGroupID   string `json:"target_group_id"`
		TargetGroupName string `json:"target_group_name"`
	}
	_ = c.ShouldBindJSON(&payload)

	var req models.CloudAccessRequest
	if err := db.DB.Preload("User").Preload("Integration").First(&req, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cloud request not found"})
		return
	}

	if req.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Request is not in pending state"})
		return
	}

	if payload.TargetGroupID != "" {
		req.TargetGroupID = payload.TargetGroupID
	}
	if payload.TargetGroupName != "" {
		req.TargetGroupName = payload.TargetGroupName
	}

	// 1. Instantiate the Cloud Provider
	provider, err := cloud.NewProvider(&req.Integration)
	if err != nil {
		log.Printf("Failed to initialize cloud provider: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize cloud integration"})
		return
	}

	// 2. Grant Access via Provider
	accessReq := cloud.AccessRequest{
		TargetGroupID:     req.TargetGroupID,
		TargetGroupName:   req.TargetGroupName,
		UserEmail:         req.User.Email,
		GeneratePassword:  req.RequiresPassword,
		GenerateAccessKey: req.RequiresKeys,
	}

	result, err := provider.GrantAccess(c.Request.Context(), accessReq)
	if err != nil {
		log.Printf("Failed to grant cloud access: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to grant access on cloud provider"})
		return
	}

	// 3. Update Request Status and ExpiresAt
	now := time.Now()
	expiresAt := now.Add(time.Duration(req.DurationHours) * time.Hour)

	req.Status = "active"
	req.ApprovedAt = &now
	req.ExpiresAt = &expiresAt

	if result != nil {
		req.TempPassword = result.Password
		if result.ConsoleURL != "" {
			req.ConsoleURL = result.ConsoleURL
		}
		req.TempAccessKey = result.AccessKeyID
		req.TempSecretKey = result.SecretAccessKey
	}

	if err := db.DB.Save(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update request status in database"})
		return
	}

	c.JSON(http.StatusOK, req)
}

func RevokeCloudRequest(c *gin.Context) {
	id := c.Param("id")

	var req models.CloudAccessRequest
	if err := db.DB.Preload("User").Preload("Integration").First(&req, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cloud request not found"})
		return
	}

	if req.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Request is not currently active"})
		return
	}

	// 1. Instantiate the Cloud Provider
	provider, err := cloud.NewProvider(&req.Integration)
	if err != nil {
		log.Printf("Failed to initialize cloud provider: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize cloud integration"})
		return
	}

	// 2. Revoke Access via Provider
	accessReq := cloud.AccessRequest{
		TargetGroupID:   req.TargetGroupID,
		TargetGroupName: req.TargetGroupName,
		UserEmail:       req.User.Email,
	}

	if err := provider.RevokeAccess(c.Request.Context(), accessReq); err != nil {
		log.Printf("Failed to revoke cloud access: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke access on cloud provider"})
		return
	}

	// 3. Update Request Status
	req.Status = "revoked"
	if err := db.DB.Save(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update request status in database"})
		return
	}

	c.JSON(http.StatusOK, req)
}

// RejectCloudRequest deletes a pending cloud access request
func RejectCloudRequest(c *gin.Context) {
	id := c.Param("id")

	var req models.CloudAccessRequest
	if err := db.DB.First(&req, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cloud request not found"})
		return
	}

	if req.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only pending requests can be rejected or deleted"})
		return
	}

	if !isAdminOrApprover(c) && req.UserID != currentUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	if err := db.DB.Delete(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Request deleted successfully"})
}

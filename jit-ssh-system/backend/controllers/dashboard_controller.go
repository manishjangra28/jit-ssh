package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/manishjangra/jit-ssh-system/backend/db"
	"github.com/manishjangra/jit-ssh-system/backend/models"
)

// List all servers (online and offline) and clusters
func GetServers(c *gin.Context) {
	var servers []models.Server
	db.DB.Preload("Tags").Preload("Team").Find(&servers)

	// Determine offline status dynamically based on last_seen (e.g. > 30s)
	// For DB side we return raw, the frontend can do logic or backend rewrites it.
	for i := range servers {
		if time.Since(servers[i].LastSeen) > 30*time.Second {
			servers[i].Status = "offline"
			db.DB.Model(&servers[i]).Update("status", "offline")
		}
	}

	c.JSON(http.StatusOK, servers)
}

type UpdateServerTeamPayload struct {
	TeamID *string `json:"team_id"`
}

// UpdateServerTeam assigns or removes a server from a team
func UpdateServerTeam(c *gin.Context) {
	id := c.Param("id")

	var payload UpdateServerTeamPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if payload.TeamID != nil {
		if *payload.TeamID == "" {
			updates["team_id"] = nil
		} else {
			updates["team_id"] = *payload.TeamID
		}
	}

	if err := db.DB.Model(&models.Server{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update server team"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Server team updated"})
}

func GetRequests(c *gin.Context) {
	var requests []models.AccessRequest
	query := db.DB.Preload("User").Preload("Server")
	if !isAdminOrApprover(c) {
		query = query.Where("user_id = ?", currentUserID(c))
	}
	query.Find(&requests)
	c.JSON(http.StatusOK, requests)
}


type CreateAccessRequestPayload struct {
	ServerID          string `json:"server_id" binding:"required"`
	PubKey            string `json:"pub_key" binding:"required"`
	Duration          string `json:"duration" binding:"required"`
	Sudo              bool   `json:"sudo"`
	RequestedPath     string `json:"requested_path"`
	RequestedServices string `json:"requested_services"`
	Reason            string `json:"reason"`
}

func CreateRequest(c *gin.Context) {
	var req CreateAccessRequestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	request := models.AccessRequest{
		ServerID:          db.ParseUUID(req.ServerID),
		UserID:            db.ParseUUID(currentUserID(c)),
		PubKey:            req.PubKey,
		Duration:          req.Duration,
		Sudo:              req.Sudo,
		RequestedPath:     req.RequestedPath,
		RequestedServices: req.RequestedServices,
		Status:            "pending",
	}

	if err := db.DB.Create(&request).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create request"})
		return
	}

	// Notify all admins
	var admins []models.User
	db.DB.Where("role = ?", "admin").Find(&admins)
	for _, admin := range admins {
		createNotification(admin.ID, "New Access Request", "A new request is pending for " + request.Duration + " access.", "info")
	}

	c.JSON(http.StatusCreated, request)
}

type ApproveRequestPayload struct {
	Duration   string `json:"duration"` // Optional: "5m", "1h", etc.
}

// Approvers approve the request
func ApproveRequest(c *gin.Context) {
	reqID := c.Param("id")

	var payload ApproveRequestPayload
	c.ShouldBindJSON(&payload)

	var request models.AccessRequest
	if err := db.DB.Preload("Server").First(&request, "id = ?", reqID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}

	if request.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Request is not in pending state"})
		return
	}

	// RBAC Verification
	approverID := currentUserID(c)
	if approverID != "" {
		var approver models.User
		if err := db.DB.First(&approver, "id = ?", approverID).Error; err == nil {
			if approver.Role == "approver" {
				if request.Server.TeamID == nil || approver.TeamID == nil || *request.Server.TeamID != *approver.TeamID {
					c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: You can only approve requests for servers assigned to your team."})
					return
				}
			}
		}
	}

	// Determine final duration (override vs user requested)
	durationStr := request.Duration
	if payload.Duration != "" {
		durationStr = payload.Duration
	}

	d, err := time.ParseDuration(durationStr)
	if err != nil {
		// Try manual mapping if ParseDuration fails
		durationMap := map[string]time.Duration{
			"5m":  5 * time.Minute,
			"15m": 15 * time.Minute,
			"30m": 30 * time.Minute,
			"1h":  time.Hour,
			"2h":  2 * time.Hour,
			"24h": 24 * time.Hour,
		}
		d = durationMap[durationStr]
	}

	if d == 0 {
		d = time.Hour // default
	}

	request.Status = "approved"
	request.ExpiresAt = time.Now().Add(d)
	request.Duration = durationStr // Update duration if overridden
	request.ApprovedBy = db.ParseUUID(approverID)

	db.DB.Save(&request)

	// Log audit
	audit := models.AuditLog{
		UserID:   request.UserID,
		ServerID: request.ServerID,
		Action:   "Access Request Approved (Duration: " + durationStr + ")",
	}
	db.DB.Create(&audit)

	// Notify asking user
	createNotification(request.UserID, "Access Approved", "Your request for " + request.Server.Hostname + " has been approved.", "success")

	c.JSON(http.StatusOK, gin.H{"status": "approved", "request": request})
}

// RejectRequest denies a pending request
func RejectRequest(c *gin.Context) {
	id := c.Param("id")
	
	var request models.AccessRequest
	if err := db.DB.Preload("Server").First(&request, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}

	if request.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only pending requests can be rejected"})
		return
	}

	request.Status = "rejected"
	db.DB.Save(&request)

	// Audit Log
	audit := models.AuditLog{
		UserID:   request.UserID,
		ServerID: request.ServerID,
		Action:   "Access Request Rejected by Admin",
	}
	db.DB.Create(&audit)

	// Notify asking user
	createNotification(request.UserID, "Access Rejected", "Your access request for " + request.Server.Hostname + " has been rejected.", "error")

	c.JSON(http.StatusOK, gin.H{"message": "Request rejected"})
}

// RevokeRequest manually expires an approved request
func RevokeRequest(c *gin.Context) {
	id := c.Param("id")
	
	var request models.AccessRequest
	if err := db.DB.Preload("Server").First(&request, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}

	if request.Status != "approved" && request.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only active/approved requests can be revoked"})
		return
	}

	request.Status = "expired"
	request.ExpiresAt = time.Now() // Expire immediately
	db.DB.Save(&request)

	// Audit Log
	audit := models.AuditLog{
		UserID:   request.UserID,
		ServerID: request.ServerID,
		Action:   "Access Request Revoked by Admin",
	}
	db.DB.Create(&audit)

	// Notify asking user
	createNotification(request.UserID, "Access Revoked", "Your access to " + request.Server.Hostname + " has been manually revoked by an admin.", "warning")

	c.JSON(http.StatusOK, gin.H{"message": "Access revoked. The agent will remove this user in the next polling cycle."})
}

// Get Logs
func GetLogs(c *gin.Context) {
	var logs []models.AuditLog
	db.DB.Order("timestamp desc").Limit(100).Find(&logs)
	c.JSON(http.StatusOK, logs)
}

func GetNotifications(c *gin.Context) {
	userID := db.ParseUUID(currentUserID(c))
	var notifications []models.Notification
	db.DB.Where("user_id = ?", userID).Order("created_at desc").Limit(20).Find(&notifications)
	c.JSON(http.StatusOK, notifications)
}

func MarkNotificationRead(c *gin.Context) {
	id := c.Param("id")
	db.DB.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", id, db.ParseUUID(currentUserID(c))).
		Update("is_read", true)
	c.JSON(http.StatusOK, gin.H{"status": "read"})
}

func GetLoginEvents(c *gin.Context) {
	var events []models.LoginEvent
	db.DB.Preload("Server").Preload("User").Order("login_time desc").Limit(20).Find(&events)
	c.JSON(http.StatusOK, events)
}

func ClearNotifications(c *gin.Context) {
	userID := db.ParseUUID(currentUserID(c))
	if err := db.DB.Where("user_id = ?", userID).Delete(&models.Notification{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "cleared"})
}

func createNotification(userID uuid.UUID, title, message, nType string) {
	notification := models.Notification{
		UserID:  userID,
		Title:   title,
		Message: message,
		Type:    nType,
	}
	db.DB.Create(&notification)
}

package controllers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/manishjangra/jit-ssh-system/backend/db"
	"github.com/manishjangra/jit-ssh-system/backend/models"
	"gorm.io/gorm"
)

type RegisterRequest struct {
	Hostname   string            `json:"hostname" binding:"required"`
	PrivateIP  string            `json:"private_ip" binding:"required"`
	InstanceID string            `json:"instance_id"`
	AgentID    string            `json:"agent_id" binding:"required"` // Added AgentID to payload as per design
	Region     string            `json:"region"`
	OS         string            `json:"os"`
	Tags       map[string]string `json:"tags"`
}

func RegisterAgent(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Process tags
	var tags []models.ServerTag
	for k, v := range req.Tags {
		tags = append(tags, models.ServerTag{TagKey: k, TagValue: v})
	}

	server := models.Server{
		Hostname:   req.Hostname,
		IP:         req.PrivateIP,
		InstanceID: req.InstanceID,
		AgentID:    req.AgentID, // In reality, this might be a generated token and AgentID pair.
		Status:     "online",
		LastSeen:   time.Now(),
		Tags:       tags,
	}

	// Insert or update server using AgentID as unique key
	var existingServer models.Server
	result := db.DB.Where("agent_id = ?", req.AgentID).First(&existingServer)
	
	if result.Error == gorm.ErrRecordNotFound {
		if err := db.DB.Create(&server).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register server"})
			return
		}
	} else {
		// Update existing server
		server.ID = existingServer.ID
		// Clear existing tags to prevent duplicates (cascade delete logic handles it or we manually delete)
		db.DB.Where("server_id = ?", existingServer.ID).Delete(&models.ServerTag{})
		
		if err := db.DB.Save(&server).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update server"})
			return
		}
	}

	// Link the token to this server
	tokenIDRaw, exists := c.Get("agent_token_id")
	if exists {
		tokenID := tokenIDRaw.(uuid.UUID)
		db.DB.Model(&models.AgentToken{}).Where("id = ?", tokenID).Update("server_id", server.ID)
	}

	c.JSON(http.StatusOK, gin.H{"status": "registered", "server_id": server.ID})
}

type HeartbeatRequest struct {
	AgentID  string `json:"agent_id" binding:"required"`
	Hostname string `json:"hostname"`
	Uptime   int64  `json:"uptime"`
}

func HeartbeatAgent(c *gin.Context) {
	var req HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := db.DB.Model(&models.Server{}).Where("agent_id = ?", req.AgentID).Updates(map[string]interface{}{
		"last_seen": time.Now(),
		"status":    "online",
	})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record heartbeat"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not registered"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "heartbeat_received"})
}

// GetTasks fetches all 'approved' requests that haven't expired yet and need to be applied
// Or 'expired' tasks that need to be removed.
// For simplicity we will assume 'approved' tasks should be sent to the agent to create the user,
// and 'expired' needs deletion.
func GetAgentTasks(c *gin.Context) {
	agentID := c.Query("agent_id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent_id query parameter is required"})
		return
	}

	var server models.Server
	if err := db.DB.Where("agent_id = ?", agentID).First(&server).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Server not found"})
		return
	}

	var tasks []models.AccessRequest
	
	// Example logic:
	// Find requests for this server that are conceptually pending application by the agent.
	// For example, 'approved' requests that haven't been completed yet.
	// (In a real system, you'd add a status like 'CREATE_PENDING', 'DELETE_PENDING')
	
	// Let's fetch the list of active/approved access requests that the agent should fulfill.
	db.DB.Where("server_id = ? AND status IN ('approved', 'expired')", server.ID).Find(&tasks)

	// Format response to match design document expectations
	type TaskResponse struct {
		TaskID    string `json:"task_id"`
		TaskType  string `json:"task_type"` // CREATE_USER or DELETE_USER
		Username  string `json:"username"`
		PubKey    string `json:"pubkey"`
		Sudo      bool   `json:"sudo"`
		ExpiresAt string `json:"expires_at"`
	}

	var response []TaskResponse
	for _, t := range tasks {
		taskType := ""
		if t.Status == "approved" {
			taskType = "CREATE_USER"
		} else if t.Status == "expired" {
			taskType = "DELETE_USER"
		}
		
		// Typically we'll get the Username from the User relation. Let's lookup User.
		var user models.User
		db.DB.First(&user, t.UserID)
		username := ""
		if user.Email != "" {
			// Basic email to username parser or a dedicated DB field. Let's assume a basic split.
			username = user.Email // Would need to strip @domain in a real system.
		}

		response = append(response, TaskResponse{
			TaskID:    t.ID.String(),
			TaskType:  taskType,
			Username:  username,
			PubKey:    t.PubKey,
			Sudo:      t.Sudo,
			ExpiresAt: t.ExpiresAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, response)
}

type TaskCompleteRequest struct {
	AgentID string `json:"agent_id" binding:"required"`
	Status  string `json:"status" binding:"required"` // e.g. 'completed', 'deleted'
}

func CompleteAgentTask(c *gin.Context) {
	taskID := c.Param("id")
	var req TaskCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get Task
	var task models.AccessRequest
	if err := db.DB.First(&task, "id = ?", taskID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	var server models.Server
	if err := db.DB.First(&server, task.ServerID).Error; err != nil || server.AgentID != req.AgentID {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized agent ID for task"})
		return
	}

	log.Printf("Agent %s completed task %s with status %s", req.AgentID, taskID, req.Status)

	if req.Status == "completed" {
		task.Status = "active"
	} else if req.Status == "deleted" {
		task.Status = "completed" // Full lifecycle done
	}

	db.DB.Save(&task)

	c.JSON(http.StatusOK, gin.H{"status": "task_updated"})
}

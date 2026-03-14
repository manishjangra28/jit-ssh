package jobs

import (
	"log"
	"time"

	"github.com/manishjangra/jit-ssh-system/backend/db"
	"github.com/manishjangra/jit-ssh-system/backend/models"
)

// StartSSHExpiryWorker starts a background goroutine that periodically checks
// for expired SSH access requests and queues a DELETE_USER task for the agent.
func StartSSHExpiryWorker() {
	log.Println("[SSHExpiryWorker] Starting background expiry worker...")

	// Check every minute
	ticker := time.NewTicker(1 * time.Minute)

	go func() {
		for {
			<-ticker.C
			processExpiredSSHRequests()
		}
	}()
}

func processExpiredSSHRequests() {
	var expiredRequests []models.AccessRequest

	// Find all active/approved SSH requests where the expiry time has passed
	err := db.DB.Preload("User").Preload("Server").
		Where("(status = 'active' OR status = 'approved') AND expires_at <= ?", time.Now()).
		Find(&expiredRequests).Error

	if err != nil {
		log.Printf("[SSHExpiryWorker] Error querying expired requests: %v", err)
		return
	}

	for _, req := range expiredRequests {
		log.Printf("[SSHExpiryWorker] Processing expiry for SSH request %s (User: %s, Server: %s)",
			req.ID, req.User.Email, req.Server.Hostname)

		// 1. We must find the Agent ID associated with this server to assign the task
		if req.Server.AgentID == "" {
			log.Printf("[SSHExpiryWorker] Warning: Server %s has no active AgentID. Marking request as expired anyway.", req.Server.ID)
			markSSHRequestExpired(&req)
			continue
		}

		// 2. Create the DELETE_USER task in Redis (simulated via DB insert for now since Redis isn't fully wired)
		// We can reuse the AccessRequest status to trigger the agent if we don't have a dedicated Tasks table yet,
		// or if we do have a Tasks table (which seems to be implied by agent/tasks API), we should insert into it.
		// Looking at agent_controller.go (which I can't read directly right now but infer from the agent code),
		// the agent polls for pending requests where status='expired'.

		// Since the agent polls GET /agent/tasks and looks for DELETE_USER, let's update the status to 'expired'
		// which signals the backend API to return a DELETE_USER task to the agent.
		markSSHRequestExpired(&req)
	}
}

func markSSHRequestExpired(req *models.AccessRequest) {
	req.Status = "expired"
	if err := db.DB.Save(req).Error; err != nil {
		log.Printf("[SSHExpiryWorker] Failed to update DB status for request %s: %v", req.ID, err)
	} else {
		log.Printf("[SSHExpiryWorker] Successfully marked request %s as expired", req.ID)
	}
}

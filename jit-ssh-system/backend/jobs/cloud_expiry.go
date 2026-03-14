package jobs

import (
	"context"
	"log"
	"time"

	"github.com/manishjangra/jit-ssh-system/backend/db"
	"github.com/manishjangra/jit-ssh-system/backend/models"
	"github.com/manishjangra/jit-ssh-system/backend/pkg/cloud"
)

// StartCloudExpiryWorker starts a background goroutine that periodically checks
// for expired cloud access requests and revokes them directly in the cloud provider.
func StartCloudExpiryWorker() {
	log.Println("[CloudExpiryWorker] Starting background expiry worker...")

	// Check every minute
	ticker := time.NewTicker(1 * time.Minute)

	go func() {
		for {
			<-ticker.C
			processExpiredCloudRequests()
		}
	}()
}

func processExpiredCloudRequests() {
	var expiredRequests []models.CloudAccessRequest

	// Find all active requests where the expiry time has passed
	// We need to preload the Integration (for credentials) and User (for the email)
	err := db.DB.Preload("Integration").Preload("User").
		Where("status = ? AND expires_at <= ?", "active", time.Now()).
		Find(&expiredRequests).Error

	if err != nil {
		log.Printf("[CloudExpiryWorker] Error querying expired requests: %v", err)
		return
	}

	for _, req := range expiredRequests {
		log.Printf("[CloudExpiryWorker] Processing expiry for request %s (User: %s, Cloud: %s, Group: %s)",
			req.ID, req.User.Email, req.Integration.Provider, req.TargetGroupName)

		// 1. Initialize the correct cloud provider client
		provider, err := cloud.NewProvider(&req.Integration)
		if err != nil {
			log.Printf("[CloudExpiryWorker] Failed to initialize provider for request %s: %v", req.ID, err)
			continue // Skip and retry next time
		}

		// 2. Prepare the revocation request
		accessReq := cloud.AccessRequest{
			TargetGroupID:   req.TargetGroupID,
			TargetGroupName: req.TargetGroupName,
			UserEmail:       req.User.Email,
		}

		// 3. Call the cloud provider to revoke access, with a timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err = provider.RevokeAccess(ctx, accessReq)
		cancel()

		if err != nil {
			log.Printf("[CloudExpiryWorker] Cloud API failed to revoke access for request %s: %v", req.ID, err)
			req.Status = "failed"
		} else {
			req.Status = "expired"
		}

		// 4. Update the database status
		if err := db.DB.Save(&req).Error; err != nil {
			log.Printf("[CloudExpiryWorker] Failed to update DB status for request %s: %v", req.ID, err)
		} else {
			if req.Status == "expired" {
				log.Printf("[CloudExpiryWorker] Successfully revoked and expired request %s", req.ID)
			} else {
				log.Printf("[CloudExpiryWorker] Marked request %s as failed due to revoke error", req.ID)
			}
		}
	}
}

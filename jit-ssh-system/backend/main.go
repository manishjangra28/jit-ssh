package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/manishjangra/jit-ssh-system/backend/controllers"
	"github.com/manishjangra/jit-ssh-system/backend/db"
	"github.com/manishjangra/jit-ssh-system/backend/jobs"
	"github.com/manishjangra/jit-ssh-system/backend/models"
)

func main() {
	// Initialize database connection
	db.InitDB()

	// Optionally AutoMigrate models here.
	log.Println("Running AutoMigrate...")
	err := db.DB.AutoMigrate(
		&models.Team{},
		&models.User{},
		&models.Server{},
		&models.ServerTag{},
		&models.Cluster{},
		&models.AccessRequest{},
		&models.AuditLog{},
		&models.AgentToken{},
		&models.LoginEvent{},
		&models.Notification{},
		&models.CloudIntegration{},
		&models.CloudAccessRequest{},
		&models.ProtectedUser{},
	)
	if err != nil {
		log.Fatalf("failed to auto migrate models: %v", err)
	}

	// Seed database with default admin if empty
	db.SeedDB()

	// Start background jobs
	jobs.StartCloudExpiryWorker()
	jobs.StartSSHExpiryWorker()

	// Setup Router
	r := gin.Default()

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	api := r.Group("/api/v1")
	{
		api.POST("/auth/login", controllers.Login)

		// Agent API (Protected by token)
		agentAPI := api.Group("/agent")
		agentAPI.Use(controllers.AgentAuthMiddleware())
		{
			agentAPI.POST("/register", controllers.RegisterAgent)
			agentAPI.POST("/heartbeat", controllers.HeartbeatAgent)
			agentAPI.GET("/tasks", controllers.GetAgentTasks)
			agentAPI.POST("/tasks/:id/complete", controllers.CompleteAgentTask)
			agentAPI.POST("/report-login", controllers.ReportLogin)
		}

		// Agent Deployment API
		api.GET("/agent/deploy/download", controllers.GetAgentBinary)
		api.GET("/agent/deploy/update", controllers.GetAgentUpdateInfo)

		authAPI := api.Group("")
		authAPI.Use(controllers.AuthRequired())
		{
			// Dashboard API
			authAPI.GET("/servers", controllers.GetServers)
			authAPI.GET("/requests", controllers.GetRequests)
			authAPI.POST("/requests", controllers.CreateRequest)
			authAPI.GET("/login-events", controllers.GetLoginEvents)
			authAPI.GET("/notifications", controllers.GetNotifications)
			authAPI.POST("/notifications/:id/read", controllers.MarkNotificationRead)
			authAPI.DELETE("/notifications", controllers.ClearNotifications)
			authAPI.POST("/auth/set-password", controllers.SetPassword)
			authAPI.GET("/cloud-integrations", controllers.GetCloudIntegrations)
			authAPI.GET("/cloud-integrations/:id/groups", controllers.GetCloudIntegrationGroups)
			authAPI.GET("/cloud-requests", controllers.GetCloudRequests)
			authAPI.POST("/cloud-requests", controllers.CreateCloudRequest)
			authAPI.DELETE("/cloud-requests/:id", controllers.RejectCloudRequest)
		}

		reviewerAPI := api.Group("")
		reviewerAPI.Use(controllers.AuthRequired(), controllers.RequireRoles("admin", "approver"))
		{
			reviewerAPI.POST("/requests/:id/approve", controllers.ApproveRequest)
			reviewerAPI.POST("/requests/:id/revoke", controllers.RevokeRequest)
			reviewerAPI.DELETE("/requests/:id", controllers.RejectRequest)
			reviewerAPI.GET("/logs", controllers.GetLogs)
			reviewerAPI.POST("/cloud-requests/:id/approve", controllers.ApproveCloudRequest)
			reviewerAPI.POST("/cloud-requests/:id/revoke", controllers.RevokeCloudRequest)
		}

		adminAPI := api.Group("")
		adminAPI.Use(controllers.AuthRequired(), controllers.RequireRoles("admin"))
		{
			adminAPI.GET("/agent/deploy/script", controllers.GenerateDeploymentScript)
			adminAPI.GET("/agent-tokens", controllers.ListAgentTokens)
			adminAPI.POST("/agent-tokens", controllers.CreateAgentToken)
			adminAPI.DELETE("/agent-tokens/:id", controllers.RevokeAgentToken)
			adminAPI.PUT("/servers/:id/team", controllers.UpdateServerTeam)
			adminAPI.GET("/users", controllers.GetUsers)
			adminAPI.POST("/users", controllers.CreateUser)
			adminAPI.PUT("/users/:id/role", controllers.UpdateUser)
			adminAPI.DELETE("/users/:id", controllers.DeleteUser)
			adminAPI.PUT("/users/:id/status", controllers.ToggleUserStatus)
			adminAPI.POST("/auth/reset-password/:id", controllers.ResetPassword)
			adminAPI.GET("/teams", controllers.GetTeams)
			adminAPI.POST("/teams", controllers.CreateTeam)
			adminAPI.PUT("/teams/:id", controllers.UpdateTeam)
			adminAPI.POST("/cloud-integrations", controllers.CreateCloudIntegration)
			adminAPI.POST("/cloud-integrations/:id/test", controllers.TestCloudIntegration)
			adminAPI.PUT("/cloud-integrations/:id", controllers.UpdateCloudIntegration)
			adminAPI.DELETE("/cloud-integrations/:id", controllers.DeleteCloudIntegration)
			adminAPI.GET("/protected-users", controllers.GetProtectedUsers)
			adminAPI.POST("/protected-users", controllers.AddProtectedUser)
			adminAPI.DELETE("/protected-users/:id", controllers.DeleteProtectedUser)
		}
	}

	log.Printf("Server starting on port %s", port)
	r.Run(":" + port)
}

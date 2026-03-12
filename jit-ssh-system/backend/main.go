package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/manishjangra/jit-ssh-system/backend/controllers"
	"github.com/manishjangra/jit-ssh-system/backend/db"
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
	)
	if err != nil {
		log.Fatalf("failed to auto migrate models: %v", err)
	}

	// Seed database with default admin if empty
	db.SeedDB()

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
		// Agent API (Protected by token)
		agentAPI := api.Group("/agent")
		agentAPI.Use(controllers.AgentAuthMiddleware())
		{
			agentAPI.POST("/register", controllers.RegisterAgent)
			agentAPI.POST("/heartbeat", controllers.HeartbeatAgent)
			agentAPI.GET("/tasks", controllers.GetAgentTasks)
			agentAPI.POST("/tasks/:id/complete", controllers.CompleteAgentTask)
		}

		// Token Management API (Admin)
		api.GET("/agent-tokens", controllers.ListAgentTokens)
		api.POST("/agent-tokens", controllers.CreateAgentToken)
		api.DELETE("/agent-tokens/:id", controllers.RevokeAgentToken)

		// Dashboard API
		api.GET("/servers", controllers.GetServers)
		api.PUT("/servers/:id/team", controllers.UpdateServerTeam)
		api.GET("/requests", controllers.GetRequests)
		api.POST("/requests", controllers.CreateRequest)
		api.POST("/requests/:id/approve", controllers.ApproveRequest)
		api.POST("/requests/:id/revoke", controllers.RevokeRequest)
		api.GET("/logs", controllers.GetLogs)

		// User Management API
		api.GET("/users", controllers.GetUsers)
		api.POST("/users", controllers.CreateUser)
		api.PUT("/users/:id/role", controllers.UpdateUser)
		api.DELETE("/users/:id", controllers.DeleteUser)
		api.PUT("/users/:id/status", controllers.ToggleUserStatus)

		// Auth API
		api.POST("/auth/login", controllers.Login)
		api.POST("/auth/set-password", controllers.SetPassword)
		api.POST("/auth/reset-password/:id", controllers.ResetPassword)

		// Team Management API
		api.GET("/teams", controllers.GetTeams)
		api.POST("/teams", controllers.CreateTeam)
		api.PUT("/teams/:id", controllers.UpdateTeam)
	}

	log.Printf("Server starting on port %s", port)
	r.Run(":" + port)
}

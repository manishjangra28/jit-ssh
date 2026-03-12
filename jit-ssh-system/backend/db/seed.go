package db

import (
	"log"

	"github.com/manishjangra/jit-ssh-system/backend/models"
	"golang.org/x/crypto/bcrypt"
)

// SeedDB ensures that a default team and a default admin user exist if the database is empty.
func SeedDB() {
	log.Println("Checking if database seeding is required...")

	// 1. Ensure a Default Team exists
	var teamCount int64
	DB.Model(&models.Team{}).Count(&teamCount)
	
	var defaultTeam models.Team
	if teamCount == 0 {
		defaultTeam = models.Team{
			Name:        "Default Operations",
			Description: "Automated default team for JIT bootstrapping.",
		}
		if err := DB.Create(&defaultTeam).Error; err != nil {
			log.Printf("Warning: Failed to create default team: %v", err)
		} else {
			log.Println("Created default team: Default Operations")
		}
	} else {
		// Just get any existing team to associate the admin with if needed
		DB.First(&defaultTeam)
	}

	// 2. Ensure at least one Admin user exists
	var userCount int64
	DB.Model(&models.User{}).Count(&userCount)

	if userCount == 0 {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin-password"), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("Critical: Failed to hash default admin password: %v", err)
		}

		admin := models.User{
			Name:         "System Admin",
			Email:        "admin@jit.local",
			PasswordHash: string(hashedPassword),
			Role:         "admin",
			Status:       "active",
			TeamID:       &defaultTeam.ID,
		}

		if err := DB.Create(&admin).Error; err != nil {
			log.Printf("Warning: Failed to create initial admin user: %v", err)
		} else {
			log.Println("==========================================================")
			log.Println(" BOOTSTRAP: Created default admin user!")
			log.Println(" Email:    admin@jit.local")
			log.Println(" Password: admin-password")
			log.Println(" Please change this password after your first login.")
			log.Println("==========================================================")
		}
	} else {
		log.Println("Database already contains users. Skipping admin seeding.")
	}
}

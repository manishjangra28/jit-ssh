package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/manishjangra/jit-ssh-system/backend/db"
	"github.com/manishjangra/jit-ssh-system/backend/models"
)

// GetTeams lists all teams
func GetTeams(c *gin.Context) {
	var teams []models.Team
	if err := db.DB.Order("name asc").Find(&teams).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch teams"})
		return
	}
	c.JSON(http.StatusOK, teams)
}

type CreateTeamPayload struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// CreateTeam creates a new team
func CreateTeam(c *gin.Context) {
	var payload CreateTeamPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	team := models.Team{
		Name:        payload.Name,
		Description: payload.Description,
	}

	if err := db.DB.Create(&team).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create team"})
		return
	}

	c.JSON(http.StatusCreated, team)
}

type UpdateTeamPayload struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

// UpdateTeam updates a team's details
func UpdateTeam(c *gin.Context) {
	id := c.Param("id")

	var payload UpdateTeamPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if payload.Name != nil {
		updates["name"] = *payload.Name
	}
	if payload.Description != nil {
		updates["description"] = *payload.Description
	}

	if err := db.DB.Model(&models.Team{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update team"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Team updated"})
}

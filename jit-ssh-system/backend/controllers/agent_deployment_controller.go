package controllers

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/manishjangra/jit-ssh-system/backend/db"
	"github.com/manishjangra/jit-ssh-system/backend/models"
)

// CurrentAgentVersion defines the latest version of the agent available.
const CurrentAgentVersion = "1.0.3"

// GetAgentBinary allows users and existing agents to download the latest binary.
func GetAgentBinary(c *gin.Context) {
	binaryPath := "./bin/jit-agent"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Agent binary not found on server. Please ensure 'jit-agent' exists in the backend/bin folder.",
		})
		return
	}
	c.FileAttachment(binaryPath, "jit-agent")
}

// GetAgentUpdateInfo is used for Over-The-Air (OTA) updates.
func GetAgentUpdateInfo(c *gin.Context) {
	// The URL must be reachable from the agent. The agent's config already has the correct base URL.
	// To be safe, we construct a full URL, prioritizing an external URL if provided.
	apiURL := "http://jit_backend:8080/api/v1"
	if externalURL := os.Getenv("EXTERNAL_API_URL"); externalURL != "" {
		apiURL = externalURL
	}

	c.JSON(http.StatusOK, gin.H{
		"version":    CurrentAgentVersion,
		"binary_url": apiURL + "/agent/deploy/download",
	})
}

// GenerateDeploymentScript generates a bash script with embedded credentials.
func GenerateDeploymentScript(c *gin.Context) {
	tokenID := c.Query("token_id")
	if tokenID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token_id query parameter is required"})
		return
	}

	var agentToken models.AgentToken
	if err := db.DB.First(&agentToken, "id = ?", tokenID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent token not found."})
		return
	}

	// This is the fix: For the Docker test environment, we MUST use the Docker service name 'jit_backend'.
	// In a real production deployment, an admin would set EXTERNAL_API_URL on the backend container
	// to the public-facing URL of the JIT service.
	apiURL := "http://jit_backend:8080/api/v1"
	if externalURL := os.Getenv("EXTERNAL_API_URL"); externalURL != "" {
		apiURL = externalURL
	}

	scriptContent := fmt.Sprintf(`#!/bin/bash
# JIT SSH Agent - Zero-Touch Installation Script
# Generated for Token: %s
set -e

# --- Configuration ---
API_URL="%s"
AGENT_TOKEN="%s"
BIN_PATH="/usr/local/bin/jit-agent"
CONF_DIR="/etc/jit"
CONF_FILE="$CONF_DIR/jit-agent.conf"

echo "[*] Initializing JIT Agent Deployment..."

# --- Download Binary ---
echo "[*] Fetching latest agent binary..."
curl -sL --fail "$API_URL/agent/deploy/download" -o "$BIN_PATH"
chmod +x "$BIN_PATH"

# --- Create Configuration ---
echo "[*] Creating configuration file at $CONF_FILE..."
mkdir -p "$CONF_DIR"
cat <<EOF > "$CONF_FILE"
control_plane_url: $API_URL
agent_token: $AGENT_TOKEN
heartbeat_interval: 10
poll_interval: 10
EOF

# --- Install Systemd Service ---
echo "[*] Setting up systemd service..."
cat <<EOF > /etc/systemd/system/jit-agent.service
[Unit]
Description=JIT SSH Agent Service
After=network.target
StartLimitIntervalSec=0

[Service]
Type=simple
User=root
ExecStart=$BIN_PATH
Environment=JIT_CONFIG_PATH=$CONF_FILE
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# --- Launch Agent ---
echo "[*] Starting JIT agent..."
systemctl daemon-reload
systemctl enable jit-agent
systemctl restart jit-agent

echo "[+] Success! Agent is now active. Run 'journalctl -u jit-agent -f' to see logs."
`, agentToken.Label, apiURL, agentToken.Token)

	c.Header("Content-Type", "text/x-shellscript")
	c.Header("Content-Disposition", "attachment; filename=\"install-jit-agent.sh\"")
	c.String(http.StatusOK, scriptContent)
}

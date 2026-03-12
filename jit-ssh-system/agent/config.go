package main

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"strings"
)

// AgentConfig holds all configurable settings for the JIT agent.
// It is loaded from a config file (jit-agent.conf) at startup.
type AgentConfig struct {
	// ControlPlaneURL is the full base URL of the JIT backend/control plane.
	// Example: http://10.0.1.50:8080/api/v1
	ControlPlaneURL string

	// AgentID is the unique identifier of this agent.
	// Auto-generated on first run and stored back to the config file.
	AgentID string

	// AgentToken is the pre-shared secret used to authenticate with the control plane.
	AgentToken string

	// HeartbeatIntervalSec is how often (in seconds) the agent sends a heartbeat.
	HeartbeatIntervalSec int

	// PollIntervalSec is how often (in seconds) the agent polls for new tasks.
	PollIntervalSec int

	// LogFile is where the agent writes its logs. Use "stdout" to print to console.
	LogFile string

	// Tags are key=value pairs sent to the control plane during registration.
	// E.g. "environment=production,team=devops"
	Tags map[string]string
}

const defaultConfigPath = "/etc/jit-agent/jit-agent.conf"

// LoadConfig reads the config file and returns an AgentConfig.
// Falls back to environment variables if the file is not found.
func LoadConfig(path string) *AgentConfig {
	cfg := &AgentConfig{
		ControlPlaneURL:      "http://localhost:8080/api/v1",
		AgentID:              "",
		AgentToken:           "",
		HeartbeatIntervalSec: 30,
		PollIntervalSec:      15,
		LogFile:              "stdout",
		Tags:                 make(map[string]string),
	}

	// Override with environment variables first (lowest priority)
	if v := os.Getenv("JIT_CONTROL_PLANE_URL"); v != "" {
		cfg.ControlPlaneURL = v
	}
	if v := os.Getenv("JIT_AGENT_ID"); v != "" {
		cfg.AgentID = v
	}
	if v := os.Getenv("JIT_AGENT_TOKEN"); v != "" {
		cfg.AgentToken = v
	}

	// Try to read the config file
	f, err := os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[config] Warning: could not open config file %s: %v", path, err)
		} else {
			log.Printf("[config] Config file not found at %s, using defaults + env vars", path)
		}
	} else {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			// Skip empty lines and comments
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])

			switch key {
			case "control_plane_url":
				// Sanitize the URL: remove trailing slashes
				u := strings.TrimRight(val, "/")
				// If the user provided the base URL without /api/v1, we'll keep it as is
				// but many users might append it twice if they aren't careful.
				// However, our code appends path segments, so we just need a clean base.
				cfg.ControlPlaneURL = u
			case "agent_id":
				cfg.AgentID = val
			case "agent_token":
				cfg.AgentToken = val
			case "heartbeat_interval_sec":
				fmt.Sscanf(val, "%d", &cfg.HeartbeatIntervalSec)
			case "poll_interval_sec":
				fmt.Sscanf(val, "%d", &cfg.PollIntervalSec)
			case "log_file":
				cfg.LogFile = val
			case "tags":
				// Format: key1=val1,key2=val2
				for _, pair := range strings.Split(val, ",") {
					pair = strings.TrimSpace(pair)
					kv := strings.SplitN(pair, "=", 2)
					if len(kv) == 2 {
						cfg.Tags[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
					}
				}
			}
		}
	}

	// Auto-generate agent_id if not set, and save it back
	if cfg.AgentID == "" {
		cfg.AgentID = generateAgentID()
		log.Printf("[config] Generated new AgentID: %s", cfg.AgentID)
		_ = saveAgentID(path, cfg.AgentID)
	}

	log.Printf("[config] Loaded: control_plane_url=%s agent_id=%s heartbeat=%ds poll=%ds",
		cfg.ControlPlaneURL, cfg.AgentID, cfg.HeartbeatIntervalSec, cfg.PollIntervalSec)

	return cfg
}

// generateAgentID creates a random 16-hex-char agent identifier.
func generateAgentID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("jit-agent-%x", b)
}

// saveAgentID writes the generated agent_id back to the config file so it persists.
func saveAgentID(configPath, agentID string) error {
	// Read existing content
	content := ""
	if data, err := os.ReadFile(configPath); err == nil {
		content = string(data)
	}

	// If agent_id line exists, replace it; otherwise append
	if strings.Contains(content, "agent_id") {
		lines := strings.Split(content, "\n")
		for i, l := range lines {
			if strings.HasPrefix(strings.TrimSpace(l), "agent_id") {
				lines[i] = "agent_id = " + agentID
				break
			}
		}
		content = strings.Join(lines, "\n")
	} else {
		content += "\nagent_id = " + agentID + "\n"
	}

	return os.WriteFile(configPath, []byte(content), 0600)
}

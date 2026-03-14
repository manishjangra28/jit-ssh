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
type AgentConfig struct {
	ControlPlaneURL      string
	AgentID              string
	AgentToken           string
	HeartbeatIntervalSec int
	PollIntervalSec      int
	LogFile              string
	Tags                 map[string]string
}

// LoadConfig reads agent settings from the specified configuration file path.
// It establishes a clear order of precedence:
// 1. Hardcoded defaults.
// 2. Values from the configuration file (which override defaults).
// 3. Environment variables (which override everything else).
func LoadConfig(path string) *AgentConfig {
	// 1. Start with hardcoded defaults
	cfg := &AgentConfig{
		ControlPlaneURL:      "http://localhost:8080/api/v1",
		AgentID:              "",
		AgentToken:           "",
		HeartbeatIntervalSec: 30,
		PollIntervalSec:      15,
		LogFile:              "stdout",
		Tags:                 make(map[string]string),
	}

	// 2. Attempt to read the config file to override defaults
	file, err := os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[config] Warning: could not open config file %s: %v", path, err)
		}
	} else {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			var parts []string
			if strings.Contains(line, "=") {
				parts = strings.SplitN(line, "=", 2)
			} else if strings.Contains(line, ":") {
				parts = strings.SplitN(line, ":", 2)
			} else {
				continue
			}

			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])

			switch key {
			case "control_plane_url":
				cfg.ControlPlaneURL = strings.TrimRight(val, "/")
			case "agent_id":
				cfg.AgentID = val
			case "agent_token":
				cfg.AgentToken = val
			case "heartbeat_interval_sec", "heartbeat_interval":
				fmt.Sscanf(val, "%d", &cfg.HeartbeatIntervalSec)
			case "poll_interval_sec", "poll_interval":
				fmt.Sscanf(val, "%d", &cfg.PollIntervalSec)
			case "log_file":
				cfg.LogFile = val
			case "tags":
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

	// 3. Override with environment variables (highest priority)
	if v := os.Getenv("JIT_CONTROL_PLANE_URL"); v != "" {
		cfg.ControlPlaneURL = v
	}
	if v := os.Getenv("JIT_AGENT_ID"); v != "" {
		cfg.AgentID = v
	}
	if v := os.Getenv("JIT_AGENT_TOKEN"); v != "" {
		cfg.AgentToken = v
	}

	// Auto-generate agent_id if it's still not set, and save it back to the file.
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

// saveAgentID writes the generated agent_id back to the config file so it persists across restarts.
func saveAgentID(configPath, agentID string) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		// If file doesn't exist, create it.
		content = []byte{}
	}

	lines := strings.Split(string(content), "\n")
	found := false
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "agent_id") {
			lines[i] = "agent_id: " + agentID
			found = true
			break
		}
	}

	var newContent string
	if !found {
		// Append if not found
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			newContent = string(content) + "\nagent_id: " + agentID + "\n"
		} else {
			newContent = string(content) + "agent_id: " + agentID + "\n"
		}
	} else {
		newContent = strings.Join(lines, "\n")
	}

	return os.WriteFile(configPath, []byte(newContent), 0600)
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type Agent struct {
	Config        *AgentConfig
	Hostname      string
	PrivateIP     string
	SystemHandler *SystemHandler
}

// Represents task from Control Plane
type Task struct {
	TaskID    string `json:"task_id"`
	TaskType  string `json:"task_type"` // CREATE_USER or DELETE_USER
	Username  string `json:"username"`
	PubKey    string `json:"pubkey"`
	Sudo      bool   `json:"sudo"`
	ExpiresAt string `json:"expires_at"`
}

func NewAgent(cfg *AgentConfig) *Agent {
	hostname, _ := os.Hostname()
	privateIP := getPrivateIP()

	return &Agent{
		Config:        cfg,
		Hostname:      hostname,
		PrivateIP:     privateIP,
		SystemHandler: NewSystemHandler(),
	}
}

func (a *Agent) Start() {
	log.Printf("Starting JIT Agent [%s] on %s (%s) → Control Plane: %s",
		a.Config.AgentID, a.Hostname, a.PrivateIP, a.Config.ControlPlaneURL)

	if err := a.Register(); err != nil {
		log.Fatalf("Failed to register with Control Plane: %v", err)
	}
	log.Println("Registered with Control Plane successfully.")

	heartbeatInterval := time.Duration(a.Config.HeartbeatIntervalSec) * time.Second
	pollInterval := time.Duration(a.Config.PollIntervalSec) * time.Second

	// Start Heartbeat routine
	go func() {
		for {
			a.SendHeartbeat()
			time.Sleep(heartbeatInterval)
		}
	}()

	// Start Task Poller routine
	go func() {
		for {
			a.PollTasks()
			time.Sleep(pollInterval)
		}
	}()

	// Block forever
	select {}
}

func (a *Agent) authenticatedRequest(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if a.Config.AgentToken != "" {
		req.Header.Set("Authorization", "Bearer "+a.Config.AgentToken)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	return client.Do(req)
}

func (a *Agent) Register() error {
	payload := map[string]interface{}{
		"hostname":    a.Hostname,
		"private_ip":  a.PrivateIP,
		"agent_id":    a.Config.AgentID,
		"instance_id": getEnv("INSTANCE_ID", a.Hostname),
		"region":      getEnv("AWS_REGION", "unknown"),
		"os":          getEnv("OS_TYPE", "linux"),
		"tags":        a.Config.Tags,
	}

	data, _ := json.Marshal(payload)
	resp, err := a.authenticatedRequest("POST", a.Config.ControlPlaneURL+"/agent/register", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (a *Agent) SendHeartbeat() {
	payload := map[string]interface{}{
		"agent_id": a.Config.AgentID,
		"hostname": a.Hostname,
		"uptime":   getUptime(),
	}

	data, _ := json.Marshal(payload)
	resp, err := a.authenticatedRequest("POST", a.Config.ControlPlaneURL+"/agent/heartbeat", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("[heartbeat] Error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[heartbeat] Warning: abnormal response %d", resp.StatusCode)
	}
}

func (a *Agent) PollTasks() {
	url := fmt.Sprintf("%s/agent/tasks?agent_id=%s", a.Config.ControlPlaneURL, a.Config.AgentID)
	resp, err := a.authenticatedRequest("GET", url, nil)
	if err != nil {
		log.Printf("[tasks] Error polling: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	body, _ := io.ReadAll(resp.Body)
	var tasks []Task
	if err := json.Unmarshal(body, &tasks); err != nil {
		log.Printf("[tasks] Failed to unmarshal: %v", err)
		return
	}

	for _, task := range tasks {
		a.handleTask(task)
	}
}

func (a *Agent) handleTask(task Task) {
	log.Printf("[task] %s → %s user=%s", task.TaskID, task.TaskType, task.Username)

	var err error
	var status string

	if task.TaskType == "CREATE_USER" {
		err = a.SystemHandler.CreateUser(task.Username, task.PubKey, task.Sudo)
		status = "completed"
	} else if task.TaskType == "DELETE_USER" {
		if a.SystemHandler.IsProtectedUser(task.Username) {
			log.Printf("[task] Skipping protected user: %s", task.Username)
			status = "deleted"
		} else {
			err = a.SystemHandler.DeleteUser(task.Username)
			status = "deleted"
		}
	} else {
		log.Printf("[task] Unknown type: %s", task.TaskType)
		return
	}

	if err != nil {
		log.Printf("[task] %s failed: %v", task.TaskID, err)
		return
	}

	a.reportTaskComplete(task.TaskID, status)
}

func (a *Agent) reportTaskComplete(taskID, status string) {
	payload := map[string]string{
		"agent_id": a.Config.AgentID,
		"status":   status,
	}
	data, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/agent/tasks/%s/complete", a.Config.ControlPlaneURL, taskID)

	resp, err := a.authenticatedRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		log.Printf("[task] Failed to report %s complete: %v", taskID, err)
		return
	}
	defer resp.Body.Close()
	log.Printf("[task] Marked %s as %s", taskID, status)
}

// getPrivateIP returns the machine's private IP, preferring the env var PRIVATE_IP.
func getPrivateIP() string {
	if ip := os.Getenv("PRIVATE_IP"); ip != "" {
		return ip
	}
	return "127.0.0.1"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getUptime() int64 {
	// In a real agent this would read from /proc/uptime
	return time.Now().Unix()
}

func main() {
	// Determine config file path (flag > env var > default)
	configPath := defaultConfigPath
	if v := os.Getenv("JIT_CONFIG_PATH"); v != "" {
		configPath = v
	}
	// Allow override via positional arg: ./jit-agent /path/to/config
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg := LoadConfig(configPath)
	agent := NewAgent(cfg)
	agent.Start()
}

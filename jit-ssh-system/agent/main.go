package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const AgentVersion = "1.0.3"

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
	Path      string `json:"path"`
	Services  string `json:"services"`
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

	// Start Login Monitor routine
	go a.MonitorLogins()

	// Start OTA Update check routine (every hour)
	go func() {
		// Wait 1 minute after startup before first check
		time.Sleep(1 * time.Minute)
		for {
			a.CheckForUpdates()
			time.Sleep(1 * time.Hour)
		}
	}()

	// Block forever
	select {}
}

func (a *Agent) MonitorLogins() {
	log.Println("[login-monitor] Starting background login monitor...")
	lastCheck := time.Now().Add(-1 * time.Minute)

	for {
		// We use 'last -F' to get full timestamps
		// Example output: manish   pts/0        127.0.0.1        Fri Mar 13 02:45:00 2026   still logged in
		cmd := "last -F -n 10"
		out, err := exec.Command("sh", "-c", cmd).Output()
		if err == nil {
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				if line == "" || strings.HasPrefix(line, "wtmp begins") {
					continue
				}

				// Very basic parsing of 'last' output
				fields := strings.Fields(line)
				if len(fields) < 10 {
					continue
				}

				username := fields[0]
				// Skip system users we don't care about
				if username == "reboot" || username == "runlevel" {
					continue
				}

				// Parse date: Fri Mar 13 02:45:00 2026
				dateStr := fmt.Sprintf("%s %s %s %s %s", fields[3], fields[4], fields[5], fields[6], fields[7])
				loginTime, err := time.Parse("Mon Jan 02 15:04:05 2006", dateStr)

				if err == nil && loginTime.After(lastCheck) {
					log.Printf("[login-monitor] Detected new login: %s at %s", username, loginTime)
					a.ReportLogin(username, fields[2], "login", loginTime)
					lastCheck = loginTime
				}
			}
		}
		time.Sleep(30 * time.Second)
	}
}

func (a *Agent) ReportLogin(username, remoteIP, eventType string, t time.Time) {
	payload := map[string]interface{}{
		"agent_id":  a.Config.AgentID,
		"username":  username,
		"remote_ip": remoteIP,
		"type":      eventType,
		"timestamp": t.Format(time.RFC3339),
	}

	data, _ := json.Marshal(payload)
	resp, err := a.authenticatedRequest("POST", a.Config.ControlPlaneURL+"/agent/report-login", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("[login-monitor] Failed to report: %v", err)
		return
	}
	defer resp.Body.Close()
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
		err = a.SystemHandler.CreateUser(task.Username, task.PubKey, task.Sudo, task.Path, task.Services)
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
		a.reportTaskComplete(task.TaskID, "failed")
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

func (a *Agent) CheckForUpdates() {
	url := a.Config.ControlPlaneURL + "/agent/deploy/update"
	resp, err := a.authenticatedRequest("GET", url, nil)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var info struct {
		Version   string `json:"version"`
		BinaryURL string `json:"binary_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return
	}

	if info.Version != "" && info.Version != AgentVersion {
		log.Printf("[ota] Update found: %s -> %s", AgentVersion, info.Version)
		a.applyUpdate(info.BinaryURL)
	}
}

func (a *Agent) applyUpdate(updateURL string) {
	// Download new binary
	resp, err := http.Get(updateURL)
	if err != nil {
		log.Printf("[ota] Failed to download update: %v", err)
		return
	}
	defer resp.Body.Close()

	exePath, err := os.Executable()
	if err != nil {
		return
	}

	tempPath := exePath + ".tmp"
	// Create with execution permissions
	f, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		log.Printf("[ota] Failed to create temp file: %v", err)
		return
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		log.Printf("[ota] Failed to save binary: %v", err)
		f.Close()
		return
	}
	f.Close()

	// Atomically replace binary
	if err := os.Rename(tempPath, exePath); err != nil {
		log.Printf("[ota] Replace failed: %v", err)
		return
	}

	log.Println("[ota] Update successfully applied. Restarting agent...")
	os.Exit(0) // Rely on systemd to restart the process
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
	// Determine config file path (env var > positional arg > default)
	configPath := "/etc/jit/jit-agent.conf" // Default path
	if v := os.Getenv("JIT_CONFIG_PATH"); v != "" {
		configPath = v
	}
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg := LoadConfig(configPath)
	agent := NewAgent(cfg)
	agent.Start()
}

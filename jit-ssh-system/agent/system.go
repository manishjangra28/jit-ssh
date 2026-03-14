package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// SystemHandler interfaces with the OS for user management
type SystemHandler struct {
	IsMock bool
}

func NewSystemHandler() *SystemHandler {
	// If not running on Linux or not root, default to mock mode for safety during dev
	isLinux := runtime.GOOS == "linux"
	isRoot := os.Geteuid() == 0

	mock := !isLinux || !isRoot
	if mock {
		log.Println("WARNING: SystemHandler running in MOCK mode (either not Linux or not root).")
	}

	return &SystemHandler{IsMock: mock}
}

func (s *SystemHandler) CreateUser(username, pubKey string, sudo bool, path, services string) error {
	if s.IsMock {
		log.Printf("[MOCK] Created/Unlocked user %s with sudo=%v path=%s services=%s\n", username, sudo, path, services)
		log.Printf("[MOCK] Added pubKey to %s authorized_keys\n", username)
		return nil
	}

	// 1. Check if user already exists
	checkCmd := exec.Command("id", username)
	err := checkCmd.Run()

	if err != nil {
		// User does not exist, create fresh
		log.Printf("Creating new user %s", username)
		cmd := exec.Command("useradd", "-m", "-s", "/bin/bash", username)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create user %s: %v", username, err)
		}
	} else {
		// User exists, unlock account and ensure shell is restored
		log.Printf("User %s already exists, unlocking account", username)
		exec.Command("usermod", "-U", "-s", "/bin/bash", username).Run()
	}

	// 2. Setup SSH directory
	sshDir := fmt.Sprintf("/home/%s/.ssh", username)
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create .ssh dir: %v", err)
	}

	// 3. Add PubKey (Overwrite to ensure current key is the only one)
	authKeysPath := fmt.Sprintf("%s/authorized_keys", sshDir)
	if err := os.WriteFile(authKeysPath, []byte(pubKey+"\n"), 0600); err != nil {
		return fmt.Errorf("failed to write authorized_keys: %v", err)
	}

	// Fix Ownership
	cmd := exec.Command("chown", "-R", fmt.Sprintf("%s:%s", username, username), sshDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to chown .ssh dir: %v", err)
	}

	// 4. Setup Sudo if required
	sudoerFile := fmt.Sprintf("/etc/sudoers.d/%s", username)
	if sudo {
		sudoContent := fmt.Sprintf("%s ALL=(ALL) NOPASSWD:ALL\n", username)
		if err := os.WriteFile(sudoerFile, []byte(sudoContent), 0440); err != nil {
			return fmt.Errorf("failed to create sudoers file: %v", err)
		}
	} else {
		// Remove sudoers if it was there from a previous request
		os.Remove(sudoerFile)
	}

	// 5. Add to Services Groups (e.g. docker)
	if services != "" {
		for _, svc := range strings.Split(services, ",") {
			svc = strings.TrimSpace(svc)
			if svc == "" {
				continue
			}
			log.Printf("Adding user %s to group %s", username, svc)
			exec.Command("usermod", "-aG", svc, username).Run()
		}
	}

	// 6. Path Permissions (ACL)
	if path != "" {
		log.Printf("Granting user %s access to path %s", username, path)
		exec.Command("setfacl", "-R", "-m", fmt.Sprintf("u:%s:rx", username), path).Run()
	}

	log.Printf("Successfully provisioned access for user %s", username)
	return nil
}

// DeleteUser in this implementation actually just LOCKS the user to preserve data
func (s *SystemHandler) DeleteUser(username string) error {
	// 1. Force kill all user processes (kicks them out of SSH)
	_ = exec.Command("pkill", "-KILL", "-u", username).Run()

	if s.IsMock {
		log.Printf("[MOCK] Locked user %s and removed sudoers\n", username)
		return nil
	}

	// 2. Lock the user account (-L) and set shell to nologin to prevent any type of access
	// We keep the files and home directory intact.
	cmd := exec.Command("usermod", "-L", "-s", "/usr/sbin/nologin", username)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to lock user %s: %v", username, err)
	}

	// 3. Clear authorized_keys to be extra safe
	sshDir := fmt.Sprintf("/home/%s/.ssh", username)
	authKeysPath := fmt.Sprintf("%s/authorized_keys", sshDir)
	if _, err := os.Stat(authKeysPath); err == nil {
		os.WriteFile(authKeysPath, []byte("# Access Expired\n"), 0600)
	}

	// 4. Remove sudoers file for security
	sudoerFile := fmt.Sprintf("/etc/sudoers.d/%s", username)
	if _, err := os.Stat(sudoerFile); err == nil {
		if err := os.Remove(sudoerFile); err != nil {
			log.Printf("Warning: failed to remove sudoers file for %s: %v", username, err)
		}
	}

	log.Printf("Successfully locked access for user %s (data preserved)", username)
	return nil
}

// AllowedUsers prevents locking of critical system users
var AllowedUsers = map[string]bool{
	"root":     true,
	"ubuntu":   true,
	"ec2-user": true,
	"admin":    true,
}

func (s *SystemHandler) IsProtectedUser(username string) bool {
	return AllowedUsers[username]
}

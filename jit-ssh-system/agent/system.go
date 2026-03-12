package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
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

func (s *SystemHandler) CreateUser(username, pubKey string, sudo bool) error {
	if s.IsMock {
		log.Printf("[MOCK] Created user %s with sudo=%v\n", username, sudo)
		log.Printf("[MOCK] Added pubKey to %s authorized_keys\n", username)
		return nil
	}

	// 1. Create User
	// useradd -m <username>
	cmd := exec.Command("useradd", "-m", username)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create user %s: %v", username, err)
	}

	// 2. Setup SSH directory
	sshDir := fmt.Sprintf("/home/%s/.ssh", username)
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create .ssh dir: %v", err)
	}

	// 3. Add PubKey
	authKeysPath := fmt.Sprintf("%s/authorized_keys", sshDir)
	if err := os.WriteFile(authKeysPath, []byte(pubKey+"\n"), 0600); err != nil {
		return fmt.Errorf("failed to write authorized_keys: %v", err)
	}

	// Fix Ownership
	cmd = exec.Command("chown", "-R", fmt.Sprintf("%s:%s", username, username), sshDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to chown .ssh dir: %v", err)
	}

	// 4. Setup Sudo if required
	if sudo {
		sudoerFile := fmt.Sprintf("/etc/sudoers.d/%s", username)
		sudoContent := fmt.Sprintf("%s ALL=(ALL) NOPASSWD:ALL\n", username)
		if err := os.WriteFile(sudoerFile, []byte(sudoContent), 0440); err != nil {
			return fmt.Errorf("failed to create sudoers file: %v", err)
		}
	}

	log.Printf("Successfully created user %s", username)
	return nil
}

func (s *SystemHandler) DeleteUser(username string) error {
	if s.IsMock {
		log.Printf("[MOCK] Deleted user %s and removed sudoers\n", username)
		return nil
	}

	// 1. Delete user
	cmd := exec.Command("userdel", "-r", username)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete user %s: %v", username, err)
	}

	// 2. Remove sudoers file
	sudoerFile := fmt.Sprintf("/etc/sudoers.d/%s", username)
	if _, err := os.Stat(sudoerFile); err == nil {
		if err := os.Remove(sudoerFile); err != nil {
			log.Printf("Warning: failed to remove sudoers file for %s: %v", username, err)
		}
	}

	log.Printf("Successfully deleted user %s", username)
	return nil
}

// AllowedUsers prevents deletion of critical system users
var AllowedUsers = map[string]bool{
	"root":     true,
	"ubuntu":   true,
	"ec2-user": true,
	"admin":    true,
}

func (s *SystemHandler) IsProtectedUser(username string) bool {
	return AllowedUsers[username]
}

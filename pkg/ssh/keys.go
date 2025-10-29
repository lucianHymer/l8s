package ssh

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"l8s/pkg/config"
)

// ReadPublicKey reads an SSH public key from a file
func ReadPublicKey(path string) (string, error) {
	// Expand tilde if present
	if strings.HasPrefix(path, "~/") {
		path = filepath.Join(GetHomeDir(), path[2:])
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read SSH public key: %w", err)
	}

	key := strings.TrimSpace(string(data))
	if key == "" {
		return "", fmt.Errorf("SSH public key file is empty")
	}

	return key, nil
}

// ValidatePublicKey validates an SSH public key format
func ValidatePublicKey(key string) error {
	if key == "" {
		return fmt.Errorf("SSH public key cannot be empty")
	}

	// Check for common SSH key prefixes
	validPrefixes := []string{"ssh-rsa", "ssh-ed25519", "ecdsa-sha2-nistp256", "ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521"}
	hasValidPrefix := false
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(key, prefix+" ") {
			hasValidPrefix = true
			break
		}
	}

	if !hasValidPrefix {
		return fmt.Errorf("invalid SSH public key format")
	}

	// Basic validation: should have at least 2 parts (type and key)
	parts := strings.Fields(key)
	if len(parts) < 2 {
		return fmt.Errorf("invalid SSH public key format: missing key data")
	}

	return nil
}

// GenerateAuthorizedKeys generates authorized_keys content from a public key
func GenerateAuthorizedKeys(publicKey string) string {
	return "# Managed by l8s\n" + publicKey + "\n"
}

// GenerateSSHConfigEntry generates an SSH config entry for a container
func GenerateSSHConfigEntry(containerName string, sshPort int, containerUser, prefix, remoteHost string, knownHostsPath string) string {
	// Remove prefix from container name for the host alias
	hostAlias := containerName
	if strings.HasPrefix(containerName, prefix+"-") {
		hostAlias = containerName
	}

	// If knownHostsPath is provided, use strict checking with CA
	if knownHostsPath != "" {
		return fmt.Sprintf(`Host %s
    HostName %s
    Port %d
    User %s
    StrictHostKeyChecking yes
    UserKnownHostsFile %s
    ControlMaster auto
    ControlPath ~/.ssh/control-%%r@%%h:%%p
    ControlPersist 1h
    ServerAliveInterval 30
    ServerAliveCountMax 6
    ConnectTimeout 10
    TCPKeepAlive yes
`, hostAlias, remoteHost, sshPort, containerUser, knownHostsPath)
	}

	// Fallback to insecure mode if no CA configured
	return fmt.Sprintf(`Host %s
    HostName %s
    Port %d
    User %s
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
    ControlMaster auto
    ControlPath ~/.ssh/control-%%r@%%h:%%p
    ControlPersist 1h
    ServerAliveInterval 30
    ServerAliveCountMax 6
    ConnectTimeout 10
    TCPKeepAlive yes
`, hostAlias, remoteHost, sshPort, containerUser)
}

// AddSSHConfigEntry adds an SSH config entry to the SSH config file
func AddSSHConfigEntry(configPath, entry string) error {
	// Ensure .ssh directory exists
	sshDir := filepath.Dir(configPath)
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	// Check if config file exists, create if not
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, []byte(""), 0600); err != nil {
			return fmt.Errorf("failed to create SSH config file: %w", err)
		}
	}

	// Read existing content
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read SSH config file: %w", err)
	}

	// Check if entry already exists
	if strings.Contains(string(content), "Host "+strings.Split(entry, "\n")[0][5:]) {
		// Entry already exists, update it
		return updateSSHConfigEntry(configPath, entry)
	}

	// Append the new entry
	file, err := os.OpenFile(configPath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open SSH config file: %w", err)
	}
	defer file.Close()

	// Add newline if file doesn't end with one
	if len(content) > 0 && content[len(content)-1] != '\n' {
		if _, err := file.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to write to SSH config file: %w", err)
		}
	}

	if _, err := file.WriteString(entry); err != nil {
		return fmt.Errorf("failed to write to SSH config file: %w", err)
	}

	return nil
}

// RemoveSSHConfigEntry removes an SSH config entry from the SSH config file
func RemoveSSHConfigEntry(configPath, containerName string) error {
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// No config file, nothing to remove
		return nil
	}

	// Read the file
	file, err := os.Open(configPath)
	if err != nil {
		return fmt.Errorf("failed to open SSH config file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	inTargetHost := false
	
	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)
		
		// Check if this is the start of our target host block
		if strings.HasPrefix(trimmedLine, "Host ") {
			hostName := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "Host"))
			if hostName == containerName {
				inTargetHost = true
				continue // Skip this line
			} else {
				inTargetHost = false
			}
		}
		
		// If we're in the target host block and hit another Host line or empty line after content
		if inTargetHost && (strings.HasPrefix(trimmedLine, "Host ") || (trimmedLine == "" && len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "")) {
			inTargetHost = false
		}
		
		// Only add lines that are not part of the target host block
		if !inTargetHost {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read SSH config file: %w", err)
	}

	// Write the modified content back
	content := strings.Join(lines, "\n")
	if len(lines) > 0 && lines[len(lines)-1] != "" {
		content += "\n"
	}

	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write SSH config file: %w", err)
	}

	return nil
}

// FindSSHPublicKey finds an SSH public key in standard locations
func FindSSHPublicKey() (string, error) {
	homeDir := GetHomeDir()
	sshDir := filepath.Join(homeDir, ".ssh")

	// Common SSH public key filenames (alphabetical order)
	keyFiles := []string{
		"id_dsa.pub",
		"id_ecdsa.pub",
		"id_ed25519.pub",
		"id_rsa.pub",
	}

	for _, keyFile := range keyFiles {
		keyPath := filepath.Join(sshDir, keyFile)
		if _, err := os.Stat(keyPath); err == nil {
			key, err := ReadPublicKey(keyPath)
			if err == nil && ValidatePublicKey(key) == nil {
				return key, nil
			}
		}
	}

	return "", fmt.Errorf("no SSH public key found in %s", sshDir)
}

// IsPortAvailable checks if a port is available for use
func IsPortAvailable(port int) bool {
	// Try to listen on the port
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	listener.Close()
	
	// Also check if we can dial it (in case something is listening but not accepting)
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 100*time.Millisecond)
	if err == nil {
		conn.Close()
		return false
	}
	
	return true
}

// GetHomeDir returns the user's home directory
func GetHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to HOME environment variable
		home = os.Getenv("HOME")
		if home == "" {
			home = "/"
		}
	}
	return home
}

// updateSSHConfigEntry updates an existing SSH config entry
func updateSSHConfigEntry(configPath, entry string) error {
	// Extract the host name from the entry
	lines := strings.Split(entry, "\n")
	if len(lines) == 0 {
		return fmt.Errorf("invalid SSH config entry")
	}
	
	hostLine := lines[0]
	if !strings.HasPrefix(hostLine, "Host ") {
		return fmt.Errorf("invalid SSH config entry: missing Host line")
	}
	
	hostName := strings.TrimSpace(strings.TrimPrefix(hostLine, "Host"))
	
	// Remove the old entry
	if err := RemoveSSHConfigEntry(configPath, hostName); err != nil {
		return err
	}
	
	// Add the new entry
	return AddSSHConfigEntry(configPath, entry)
}

// AddSSHConfig adds an SSH config entry for a container
func AddSSHConfig(name, hostname string, port int, user string) error {
	cfg, err := config.Load(config.GetConfigPath())
	if err != nil {
		return err
	}
	
	address, err := cfg.GetActiveAddress()
	if err != nil {
		return err
	}
	
	sshConfigPath := filepath.Join(GetHomeDir(), ".ssh", "config")
	entry := GenerateSSHConfigEntry(
		fmt.Sprintf("dev-%s", name), 
		port, 
		user, 
		"dev",
		address, // Use connection address
		cfg.KnownHostsPath, // Pass known hosts path for CA trust
	)
	return AddSSHConfigEntry(sshConfigPath, entry)
}

// RemoveSSHConfig removes an SSH config entry for a container
func RemoveSSHConfig(name string) error {
	sshConfigPath := filepath.Join(GetHomeDir(), ".ssh", "config")
	return RemoveSSHConfigEntry(sshConfigPath, fmt.Sprintf("dev-%s", name))
}
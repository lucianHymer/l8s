package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"l8s/pkg/config"
	"l8s/pkg/ssh"
)

type ConnectionListCommand struct {
	config *config.Config
	err    error
}

func (c *ConnectionListCommand) Execute(ctx context.Context) error {
	if c.err != nil {
		return c.err
	}
	
	connections := c.config.ListConnections()
	
	for name, conn := range connections {
		marker := " "
		active := ""
		if name == c.config.ActiveConnection {
			marker = "*"
			active = " [active]"
		}
		
		desc := conn.Description
		if desc == "" {
			desc = "No description"
		}
		
		fmt.Printf("%s %s - %s (%s)%s\n", 
			marker, name, desc, conn.Address, active)
	}
	
	return nil
}

type ConnectionShowCommand struct {
	config *config.Config
	err    error
}

func (c *ConnectionShowCommand) Execute(ctx context.Context) error {
	if c.err != nil {
		return c.err
	}
	
	conn, err := c.config.GetActiveConnection()
	if err != nil {
		return err
	}
	
	fmt.Printf("Active Podman connection: %s\n", c.config.ActiveConnection)
	fmt.Printf("  Address: %s\n", conn.Address)
	fmt.Printf("  User: %s\n", c.config.RemoteUser)
	fmt.Printf("  Socket: %s\n", c.config.RemoteSocket)
	if c.config.SSHKeyPath != "" {
		fmt.Printf("  SSH Key: %s\n", c.config.SSHKeyPath)
	}
	if conn.Description != "" {
		fmt.Printf("  Description: %s\n", conn.Description)
	}
	
	return nil
}

type ConnectionSwitchCommand struct {
	config           *config.Config
	targetConnection string
	dryRun           bool
	err              error
}

func (c *ConnectionSwitchCommand) Execute(ctx context.Context) error {
	if c.err != nil {
		return c.err
	}
	
	// Validate target connection exists
	newConn, exists := c.config.Connections[c.targetConnection]
	if !exists {
		return fmt.Errorf("connection '%s' not found in configuration", c.targetConnection)
	}
	
	// Get current connection address for comparison
	currentAddress, err := c.config.GetActiveAddress()
	if err != nil {
		return err
	}
	
	if c.config.ActiveConnection == c.targetConnection {
		fmt.Printf("Already using Podman connection: %s\n", c.targetConnection)
		return nil
	}
	
	fmt.Printf("Switching Podman connection from '%s' to '%s'...\n", 
		c.config.ActiveConnection, c.targetConnection)
	
	// Find and update all SSH configs
	sshConfigPath := filepath.Join(ssh.GetHomeDir(), ".ssh", "config")
	updates, err := c.findSSHConfigUpdates(sshConfigPath, currentAddress, newConn.Address)
	if err != nil {
		return fmt.Errorf("failed to read SSH config: %w", err)
	}
	
	if len(updates) > 0 {
		fmt.Printf("Updating SSH configurations for %d containers:\n", len(updates))
		
		if !c.dryRun {
			for _, container := range updates {
				err := c.updateSSHConfigEntry(sshConfigPath, container, newConn.Address)
				if err != nil {
					fmt.Printf("  ✗ %s: %v\n", container, err)
				} else {
					fmt.Printf("  ✓ %s: %s → %s\n", 
						container, currentAddress, newConn.Address)
				}
			}
		} else {
			for _, container := range updates {
				fmt.Printf("  Would update %s: %s → %s\n", 
					container, currentAddress, newConn.Address)
			}
		}
	}
	
	if !c.dryRun {
		// Update active connection in config
		if err := c.config.SetActiveConnection(c.targetConnection); err != nil {
			return fmt.Errorf("failed to update config: %w", err)
		}
		
		fmt.Printf("Switched to Podman connection: %s\n", c.targetConnection)
	} else {
		fmt.Printf("Would switch to Podman connection: %s\n", c.targetConnection)
	}
	
	return nil
}

// findSSHConfigUpdates finds all l8s-managed SSH config entries that need updating
func (c *ConnectionSwitchCommand) findSSHConfigUpdates(configPath, oldHost, newHost string) ([]string, error) {
	// Read the SSH config file
	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil // No SSH config, nothing to update
		}
		return nil, err
	}
	
	var containers []string
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	
	var currentHost string
	var currentHostName string
	inHostBlock := false
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Check for Host directive
		if strings.HasPrefix(line, "Host ") {
			// If we were in a host block, check if it needs updating
			if inHostBlock && currentHost != "" && currentHostName == oldHost {
				// This is an l8s-managed container that needs updating
				if strings.HasPrefix(currentHost, "dev-") {
					containers = append(containers, currentHost)
				}
			}
			
			// Start new host block
			currentHost = strings.TrimPrefix(line, "Host ")
			currentHost = strings.TrimSpace(currentHost)
			currentHostName = ""
			inHostBlock = true
		} else if inHostBlock && strings.HasPrefix(line, "HostName ") {
			// Extract the hostname
			currentHostName = strings.TrimPrefix(line, "HostName ")
			currentHostName = strings.TrimSpace(currentHostName)
		}
	}
	
	// Check the last host block
	if inHostBlock && currentHost != "" && currentHostName == oldHost {
		if strings.HasPrefix(currentHost, "dev-") {
			containers = append(containers, currentHost)
		}
	}
	
	return containers, nil
}

// updateSSHConfigEntry updates the HostName field for a specific SSH config entry
func (c *ConnectionSwitchCommand) updateSSHConfigEntry(configPath, container, newHost string) error {
	// Read the entire SSH config
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	
	lines := strings.Split(string(content), "\n")
	var updatedLines []string
	inTargetBlock := false
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Check if we're entering the target host block
		if strings.HasPrefix(trimmed, "Host ") {
			host := strings.TrimSpace(strings.TrimPrefix(trimmed, "Host "))
			inTargetBlock = (host == container)
			updatedLines = append(updatedLines, line)
		} else if inTargetBlock && strings.HasPrefix(trimmed, "HostName ") {
			// Replace the HostName line
			// Preserve original indentation
			indent := strings.TrimSuffix(line, trimmed)
			updatedLines = append(updatedLines, indent+"HostName "+newHost)
		} else {
			updatedLines = append(updatedLines, line)
		}
	}
	
	// Write back the updated config
	updatedContent := strings.Join(updatedLines, "\n")
	return os.WriteFile(configPath, []byte(updatedContent), 0600)
}

// ParseSSHConfig parses SSH config and returns all l8s-managed entries
func ParseSSHConfig(configPath string) (map[string]string, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}
	
	entries := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	
	var currentHost string
	var currentHostName string
	inHostBlock := false
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if strings.HasPrefix(line, "Host ") {
			// Save previous block if it was an l8s container
			if inHostBlock && currentHost != "" && strings.HasPrefix(currentHost, "dev-") {
				entries[currentHost] = currentHostName
			}
			
			// Start new block
			currentHost = strings.TrimSpace(strings.TrimPrefix(line, "Host "))
			currentHostName = ""
			inHostBlock = true
		} else if inHostBlock && strings.HasPrefix(line, "HostName ") {
			currentHostName = strings.TrimSpace(strings.TrimPrefix(line, "HostName "))
		}
	}
	
	// Save last block if it was an l8s container
	if inHostBlock && currentHost != "" && strings.HasPrefix(currentHost, "dev-") {
		entries[currentHost] = currentHostName
	}
	
	return entries, nil
}

// getHomeDir is a wrapper to allow testing
var getHomeDirFunc = ssh.GetHomeDir

// ValidateSSHConfigsMatchConnection validates that all l8s SSH configs point to the active connection
func ValidateSSHConfigsMatchConnection(activeAddress string) error {
	sshConfigPath := filepath.Join(getHomeDirFunc(), ".ssh", "config")
	entries, err := ParseSSHConfig(sshConfigPath)
	if err != nil {
		return fmt.Errorf("failed to parse SSH config: %w", err)
	}
	
	var mismatched []string
	for container, hostname := range entries {
		if hostname != activeAddress {
			mismatched = append(mismatched, fmt.Sprintf("%s (points to %s)", container, hostname))
		}
	}
	
	if len(mismatched) > 0 {
		return fmt.Errorf("SSH configs out of sync: %s", strings.Join(mismatched, ", "))
	}
	
	return nil
}
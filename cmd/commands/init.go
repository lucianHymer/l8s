package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/l8s/l8s/pkg/config"
)

// InitCmd creates the init command
func InitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize l8s configuration",
		Long: `Initialize l8s configuration by setting up remote server connection details.
	
l8s ONLY supports remote container management for security isolation.
You'll need:
- A remote server with Podman installed
- SSH access to the server
- SSH key authentication configured`,
		RunE: runInit,
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println("=== L8s Configuration Setup ===")
	fmt.Println()
	fmt.Println("l8s ONLY supports remote container management for security isolation.")
	fmt.Println("This setup will configure your connection to a remote Podman server.")
	fmt.Println()

	// Create config with defaults
	cfg := config.DefaultConfig()

	// Prompt for remote server configuration
	fmt.Println("=== Remote Server Configuration ===")
	
	remoteHost, err := promptWithDefault("Remote server hostname/IP", "")
	if err != nil {
		return err
	}
	if remoteHost == "" {
		return fmt.Errorf("remote server hostname is required")
	}
	cfg.RemoteHost = remoteHost
	
	remoteUser, err := promptWithDefault("Remote server username", "podman")
	if err != nil {
		return err
	}
	cfg.RemoteUser = remoteUser
	
	// Show sudo setup instructions for non-root users
	if remoteUser != "root" {
		fmt.Printf("\n📝 Note: Using non-root user '%s'. You'll need to set up sudo access:\n", remoteUser)
		fmt.Printf("   On the remote server, run:\n")
		fmt.Printf("   echo \"%s ALL=(ALL) NOPASSWD: /usr/bin/podman\" | sudo tee /etc/sudoers.d/podman\n\n", remoteUser)
	}
	
	remoteSocket, err := promptWithDefault("Remote Podman socket path", cfg.RemoteSocket)
	if err != nil {
		return err
	}
	cfg.RemoteSocket = remoteSocket
	
	// Test SSH connectivity
	fmt.Printf("\nTesting SSH connection to %s@%s...\n", cfg.RemoteUser, cfg.RemoteHost)
	testCmd := exec.Command("ssh", "-o", "ConnectTimeout=5", 
		fmt.Sprintf("%s@%s", cfg.RemoteUser, cfg.RemoteHost), "echo", "OK")
	output, err := testCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to connect via SSH: %v\n", err)
		fmt.Printf("Output: %s\n", string(output))
		fmt.Printf("\nPlease ensure:\n")
		fmt.Printf("1. SSH key is configured: ssh-copy-id %s@%s\n", cfg.RemoteUser, cfg.RemoteHost)
		fmt.Printf("2. Server is accessible\n")
		if cfg.RemoteUser != "root" {
			fmt.Printf("3. User has sudo access to Podman (see instructions above)\n")
		} else {
			fmt.Printf("3. User has Podman access\n")
		}
		return fmt.Errorf("SSH connection test failed")
	}
	fmt.Println("✓ SSH connection successful")
	
	// Prompt for other configuration
	fmt.Println("\n=== Container Configuration ===")
	
	sshKeyPath, err := promptWithDefault("SSH private key path", cfg.SSHKeyPath)
	if err != nil {
		return err
	}
	cfg.SSHKeyPath = sshKeyPath
	
	baseImage, err := promptWithDefault("Base container image", cfg.BaseImage)
	if err != nil {
		return err
	}
	cfg.BaseImage = baseImage
	
	containerPrefix, err := promptWithDefault("Container name prefix", cfg.ContainerPrefix)
	if err != nil {
		return err
	}
	cfg.ContainerPrefix = containerPrefix
	
	containerUser, err := promptWithDefault("Container user", cfg.ContainerUser)
	if err != nil {
		return err
	}
	cfg.ContainerUser = containerUser
	
	sshPortStart, err := promptWithDefault("SSH port range start", fmt.Sprintf("%d", cfg.SSHPortStart))
	if err != nil {
		return err
	}
	if _, err := fmt.Sscanf(sshPortStart, "%d", &cfg.SSHPortStart); err != nil {
		return fmt.Errorf("invalid port number: %w", err)
	}
	
	// Auto-detect SSH public key if not specified
	if cfg.SSHPublicKey == "" {
		fmt.Println("\nDetecting SSH public key...")
		// Try common locations
		possibleKeys := []string{
			cfg.SSHKeyPath + ".pub",
			"~/.ssh/id_ed25519.pub",
			"~/.ssh/id_rsa.pub",
			"~/.ssh/id_ecdsa.pub",
		}
		
		for _, keyPath := range possibleKeys {
			expandedPath := expandPath(keyPath)
			if _, err := os.Stat(expandedPath); err == nil {
				cfg.SSHPublicKey = keyPath
				fmt.Printf("✓ Found SSH public key at %s\n", keyPath)
				break
			}
		}
		
		if cfg.SSHPublicKey == "" {
			pubKeyPath, err := promptWithDefault("SSH public key path", "~/.ssh/id_ed25519.pub")
			if err != nil {
				return err
			}
			cfg.SSHPublicKey = pubKeyPath
		}
	}
	
	// Save configuration
	configPath := config.GetConfigPath()
	fmt.Printf("\nSaving configuration to %s...\n", configPath)
	
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	
	fmt.Println("\n=== Configuration Complete ===")
	fmt.Printf("Configuration saved to: %s\n", configPath)
	fmt.Println("\nNext steps:")
	fmt.Printf("1. Ensure Podman is running on %s\n", cfg.RemoteHost)
	if cfg.RemoteUser != "root" {
		fmt.Printf("   - Set up sudo access: echo \"%s ALL=(ALL) NOPASSWD: /usr/bin/podman\" | sudo tee /etc/sudoers.d/podman\n", cfg.RemoteUser)
	}
	fmt.Printf("2. Run 'l8s create <name> <git-url>' to create your first container\n")
	fmt.Printf("3. Use 'l8s list' to see all containers\n")
	
	return nil
}

func promptWithDefault(prompt, defaultValue string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	
	input = strings.TrimSpace(input)
	if input == "" && defaultValue != "" {
		return defaultValue, nil
	}
	
	return input, nil
}

// expandPath expands tilde in paths
func expandPath(path string) string {
	if path == "" {
		return path
	}

	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			home = os.Getenv("HOME")
			if home == "" {
				return path
			}
		}
		return filepath.Join(home, path[2:])
	}

	return path
}
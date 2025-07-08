package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the l8s application configuration
type Config struct {
	// Remote connection settings (required)
	RemoteHost   string `yaml:"remote_host"`
	RemoteUser   string `yaml:"remote_user"`
	RemoteSocket string `yaml:"remote_socket,omitempty"`
	
	// SSH authentication (ssh-agent required)
	SSHKeyPath string `yaml:"ssh_key_path,omitempty"`
	
	// Existing fields
	SSHPortStart    int    `yaml:"ssh_port_start"`
	BaseImage       string `yaml:"base_image"`
	ContainerPrefix string `yaml:"container_prefix"`
	SSHPublicKey    string `yaml:"ssh_public_key"`
	ContainerUser   string `yaml:"container_user"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		// Remote settings (must be configured)
		RemoteHost:   "",
		RemoteUser:   "",
		RemoteSocket: "/run/podman/podman.sock",
		SSHKeyPath:   filepath.Join(home, ".ssh", "id_ed25519"),
		
		// Existing defaults
		SSHPortStart:    2200,
		BaseImage:       "localhost/l8s-fedora:latest",
		ContainerPrefix: "dev",
		SSHPublicKey:    "", // Empty means auto-detect
		ContainerUser:   "dev",
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate remote configuration (required)
	if c.RemoteHost == "" {
		return fmt.Errorf("remote_host is required - l8s ONLY supports remote container management")
	}
	if c.RemoteUser == "" {
		return fmt.Errorf("remote_user is required - l8s ONLY supports remote container management")
	}
	
	// Validate SSH port start
	if c.SSHPortStart < 1024 || c.SSHPortStart > 65000 {
		return fmt.Errorf("ssh_port_start must be between 1024 and 65000")
	}

	// Validate base image
	if c.BaseImage == "" {
		return fmt.Errorf("base_image cannot be empty")
	}

	// Validate container prefix
	if c.ContainerPrefix == "" {
		return fmt.Errorf("container_prefix cannot be empty")
	}
	if len(c.ContainerPrefix) > 10 {
		return fmt.Errorf("container_prefix must be 10 characters or less")
	}
	// Check that prefix is valid for container names
	if !isValidContainerPrefix(c.ContainerPrefix) {
		return fmt.Errorf("container_prefix must consist of lowercase letters, numbers, and hyphens")
	}

	// Validate container user
	if c.ContainerUser == "" {
		return fmt.Errorf("container_user cannot be empty")
	}
	if !isValidUsername(c.ContainerUser) {
		return fmt.Errorf("container_user must be a valid Linux username")
	}

	return nil
}

// GetConfigPath returns the default config file path
func GetConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "l8s", "config.yaml")
}

// Load loads configuration from the specified path
func Load(path string) (*Config, error) {
	// Expand tilde if present
	path = expandPath(path)

	// Start with defaults
	config := DefaultConfig()

	// Check if config file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// No config file, validate defaults
		if err := config.Validate(); err != nil {
			return nil, fmt.Errorf("invalid configuration: %w", err)
		}
		return config, nil
	}

	// Read config file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Expand paths in config
	config.SSHPublicKey = expandPath(config.SSHPublicKey)
	config.SSHKeyPath = expandPath(config.SSHKeyPath)

	// Set defaults for optional fields if not provided
	if config.RemoteSocket == "" {
		config.RemoteSocket = "/run/podman/podman.sock"
	}
	if config.SSHKeyPath == "" {
		home, _ := os.UserHomeDir()
		config.SSHKeyPath = filepath.Join(home, ".ssh", "id_ed25519")
	}

	return config, nil
}

// Save saves configuration to the specified path
func (c *Config) Save(path string) error {
	// Expand tilde if present
	path = expandPath(path)

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
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

// isValidContainerPrefix checks if a string is valid as a container name prefix
func isValidContainerPrefix(prefix string) bool {
	// Must start with lowercase letter
	if len(prefix) == 0 || !isLowerLetter(rune(prefix[0])) {
		return false
	}

	// Check all characters
	for i, ch := range prefix {
		if !isLowerLetter(ch) && !isDigit(ch) && ch != '-' {
			return false
		}
		// No consecutive hyphens
		if ch == '-' && i > 0 && prefix[i-1] == '-' {
			return false
		}
	}

	// Must not end with hyphen
	if prefix[len(prefix)-1] == '-' {
		return false
	}

	return true
}

// isValidUsername checks if a string is a valid Linux username
func isValidUsername(username string) bool {
	if len(username) == 0 || len(username) > 32 {
		return false
	}

	// Must start with lowercase letter or underscore
	if !isLowerLetter(rune(username[0])) && username[0] != '_' {
		return false
	}

	// Check all characters
	for _, ch := range username {
		if !isLowerLetter(ch) && !isDigit(ch) && ch != '_' && ch != '-' {
			return false
		}
	}

	return true
}

// isLowerLetter checks if a rune is a lowercase letter
func isLowerLetter(ch rune) bool {
	return ch >= 'a' && ch <= 'z'
}

// isDigit checks if a rune is a digit
func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}
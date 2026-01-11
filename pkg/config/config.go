package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConnectionConfig holds configuration for a network connection to the Podman host
type ConnectionConfig struct {
	Address     string `yaml:"address"`     // IP address or hostname
	Description string `yaml:"description,omitempty"`
	// Future fields can be added here as needed
}

// Config holds the l8s application configuration
type Config struct {
	// Active connection selector
	ActiveConnection string                      `yaml:"active_connection"`
	
	// Connection configurations
	Connections map[string]ConnectionConfig `yaml:"connections"`
	
	// Host settings (same for all connections)
	RemoteUser   string `yaml:"remote_user"`
	RemoteSocket string `yaml:"remote_socket,omitempty"`
	SSHKeyPath   string `yaml:"ssh_key_path,omitempty"`
	
	// SSH CA settings
	CAPrivateKeyPath string `yaml:"ca_private_key_path,omitempty"`
	CAPublicKeyPath  string `yaml:"ca_public_key_path,omitempty"`
	KnownHostsPath   string `yaml:"known_hosts_path,omitempty"`
	
	// Shared settings
	SSHPortStart    int    `yaml:"ssh_port_start"`
	WebPortStart    int    `yaml:"web_port_start"`
	AudioEnabled    bool   `yaml:"audio_enabled"`       // Whether audio support is enabled
	AudioPort       int    `yaml:"audio_port"`          // PulseAudio TCP port (default 4713)
	AudioSocketPath string `yaml:"audio_socket_path"`   // Path to PulseAudio socket on host
	BaseImage       string `yaml:"base_image"`
	ContainerPrefix string `yaml:"container_prefix"`
	ContainerUser   string `yaml:"container_user"`
	SSHPublicKey    string `yaml:"ssh_public_key"`
	DotfilesPath    string `yaml:"dotfiles_path,omitempty"`
	GitHubToken     string `yaml:"github_token,omitempty"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		ActiveConnection: "",
		Connections:      make(map[string]ConnectionConfig),
		RemoteUser:       "",
		RemoteSocket:     "/run/podman/podman.sock",
		SSHKeyPath:       "",
		
		// Shared defaults
		SSHPortStart:    2200,
		WebPortStart:    3000,
		AudioEnabled:    true,
		AudioPort:       4713,
		AudioSocketPath: "/run/user/1000/pulse",
		BaseImage:       "localhost/l8s-fedora:latest",
		ContainerPrefix: "dev",
		SSHPublicKey:    "", // Empty means auto-detect
		ContainerUser:   "dev",
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate connections configuration
	if len(c.Connections) == 0 {
		return fmt.Errorf("at least one connection must be configured")
	}
	
	if c.ActiveConnection == "" {
		return fmt.Errorf("active_connection must be specified")
	}
	
	activeConn, err := c.GetActiveConnection()
	if err != nil {
		return err
	}
	
	// Validate active connection configuration
	if activeConn.Address == "" {
		return fmt.Errorf("address is required for connection '%s'", c.ActiveConnection)
	}
	
	// Validate host settings (same for all connections)
	if c.RemoteUser == "" {
		return fmt.Errorf("remote_user is required")
	}
	
	// Validate SSH port start
	if c.SSHPortStart < 1024 || c.SSHPortStart > 65000 {
		return fmt.Errorf("ssh_port_start must be between 1024 and 65000")
	}

	// Validate Web port start
	if c.WebPortStart < 1024 || c.WebPortStart > 65000 {
		return fmt.Errorf("web_port_start must be between 1024 and 65000")
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
	config.DotfilesPath = expandPath(config.DotfilesPath)
	config.SSHKeyPath = expandPath(config.SSHKeyPath)
	config.CAPrivateKeyPath = expandPath(config.CAPrivateKeyPath)
	config.CAPublicKeyPath = expandPath(config.CAPublicKeyPath)
	config.KnownHostsPath = expandPath(config.KnownHostsPath)
	
	// Set defaults
	if config.RemoteSocket == "" {
		config.RemoteSocket = "/run/podman/podman.sock"
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

// GetActiveConnection returns the active connection configuration
func (c *Config) GetActiveConnection() (*ConnectionConfig, error) {
	if c.ActiveConnection == "" {
		return nil, fmt.Errorf("no active connection configured")
	}
	
	conn, exists := c.Connections[c.ActiveConnection]
	if !exists {
		return nil, fmt.Errorf("active connection '%s' not found in configuration", c.ActiveConnection)
	}
	
	return &conn, nil
}

// GetActiveAddress returns just the address of the active connection
func (c *Config) GetActiveAddress() (string, error) {
	conn, err := c.GetActiveConnection()
	if err != nil {
		return "", err
	}
	return conn.Address, nil
}

// SetActiveConnection updates the active connection
func (c *Config) SetActiveConnection(name string) error {
	if _, exists := c.Connections[name]; !exists {
		return fmt.Errorf("connection '%s' not found in configuration", name)
	}
	
	c.ActiveConnection = name
	return c.Save(GetConfigPath())
}

// SetActiveConnectionWithPath updates the active connection and saves to a specific path (useful for testing)
func (c *Config) SetActiveConnectionWithPath(name string, path string) error {
	if _, exists := c.Connections[name]; !exists {
		return fmt.Errorf("connection '%s' not found in configuration", name)
	}
	
	c.ActiveConnection = name
	return c.Save(path)
}

// ListConnections returns all configured connections
func (c *Config) ListConnections() map[string]ConnectionConfig {
	return c.Connections
}
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds the l8s application configuration
type Config struct {
	SSHPortStart    int    `mapstructure:"ssh_port_start"`
	BaseImage       string `mapstructure:"base_image"`
	ContainerPrefix string `mapstructure:"container_prefix"`
	SSHPublicKey    string `mapstructure:"ssh_public_key"`
	ContainerUser   string `mapstructure:"container_user"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		SSHPortStart:    2200,
		BaseImage:       "localhost/l8s-fedora:latest",
		ContainerPrefix: "dev",
		SSHPublicKey:    "", // Empty means auto-detect
		ContainerUser:   "dev",
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.SSHPortStart < 1024 {
		return fmt.Errorf("SSH port must be >= 1024")
	}
	if c.BaseImage == "" {
		return fmt.Errorf("base image cannot be empty")
	}
	if c.ContainerPrefix == "" {
		return fmt.Errorf("container prefix cannot be empty")
	}
	if c.ContainerUser == "" {
		return fmt.Errorf("container user cannot be empty")
	}
	return nil
}

// GetConfigPath returns the default config file path
func GetConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "l8s", "config.yaml")
}

// ExpandPath expands ~ to home directory
func ExpandPath(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// LoadConfig loads configuration from file using Viper
// This is a placeholder - actual implementation will use Viper
func LoadConfig() (*Config, error) {
	// In actual implementation, this will:
	// 1. Set defaults using DefaultConfig()
	// 2. Set config name and paths
	// 3. Read config file
	// 4. Unmarshal into Config struct
	// 5. Validate the config
	return DefaultConfig(), nil
}
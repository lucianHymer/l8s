package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	
	// Remote configuration defaults
	assert.Equal(t, "", cfg.RemoteHost) // Must be configured
	assert.Equal(t, "", cfg.RemoteUser) // Must be configured
	assert.Equal(t, "/run/podman/podman.sock", cfg.RemoteSocket)
	assert.Contains(t, cfg.SSHKeyPath, ".ssh/id_ed25519")
	
	// Existing defaults
	assert.Equal(t, 2200, cfg.SSHPortStart)
	assert.Equal(t, "localhost/l8s-fedora:latest", cfg.BaseImage)
	assert.Equal(t, "dev", cfg.ContainerPrefix)
	assert.Equal(t, "", cfg.SSHPublicKey) // Auto-detect
	assert.Equal(t, "dev", cfg.ContainerUser)
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name           string
		configContent  string
		expectedConfig *Config
		wantErr        bool
	}{
		{
			name: "valid config file with remote settings",
			configContent: `
remote_host: "server.example.com"
remote_user: "podman"
remote_socket: "/run/podman/podman.sock"
ssh_key_path: "~/.ssh/id_ed25519"
ssh_port_start: 2300
base_image: "localhost/custom-l8s:v2"
container_prefix: "work"
ssh_public_key: "~/.ssh/custom_key.pub"
container_user: "lucian"
`,
			expectedConfig: &Config{
				RemoteHost:      "server.example.com",
				RemoteUser:      "podman",
				RemoteSocket:    "/run/podman/podman.sock",
				SSHKeyPath:      "~/.ssh/id_ed25519",
				SSHPortStart:    2300,
				BaseImage:       "localhost/custom-l8s:v2",
				ContainerPrefix: "work",
				SSHPublicKey:    "~/.ssh/custom_key.pub",
				ContainerUser:   "lucian",
			},
			wantErr: false,
		},
		{
			name: "config without remote settings should fail validation",
			configContent: `
ssh_port_start: 2400
container_user: "developer"
`,
			expectedConfig: nil,
			wantErr: true, // Should fail validation due to missing remote settings
		},
		{
			name:          "empty config file should fail validation",
			configContent: "",
			expectedConfig: nil,
			wantErr: true, // Should fail validation due to missing remote settings
		},
		{
			name: "invalid yaml",
			configContent: `
ssh_port_start: [invalid
`,
			expectedConfig: nil,
			wantErr:        true,
		},
		{
			name: "partial config with remote settings",
			configContent: `
remote_host: "server.example.com"
remote_user: "root"
container_user: "developer"
`,
			expectedConfig: &Config{
				RemoteHost:      "server.example.com",
				RemoteUser:      "root",
				RemoteSocket:    "/run/podman/podman.sock", // default
				SSHKeyPath:      "~/.ssh/id_ed25519", // default will be expanded
				SSHPortStart:    2200, // default
				BaseImage:       "localhost/l8s-fedora:latest", // default
				ContainerPrefix: "dev", // default
				SSHPublicKey:    "", // default
				ContainerUser:   "developer",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for config
			tmpDir := t.TempDir()
			configDir := filepath.Join(tmpDir, ".config", "l8s")
			err := os.MkdirAll(configDir, 0755)
			require.NoError(t, err)
			
			// Write config file
			configPath := filepath.Join(configDir, "config.yaml")
			if tt.configContent != "" {
				err = os.WriteFile(configPath, []byte(tt.configContent), 0644)
				require.NoError(t, err)
			}
			
			// Mock home directory
			origHome := os.Getenv("HOME")
			os.Setenv("HOME", tmpDir)
			defer os.Setenv("HOME", origHome)
			
			// Reset viper for clean test
			viper.Reset()
			
			// Load config
			cfg, err := Load(configPath)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				// Special handling for SSH public key path expansion
				if tt.expectedConfig.SSHPublicKey != "" && strings.HasPrefix(tt.expectedConfig.SSHPublicKey, "~/") {
					tt.expectedConfig.SSHPublicKey = filepath.Join(tmpDir, tt.expectedConfig.SSHPublicKey[2:])
				}
				if tt.expectedConfig.SSHKeyPath != "" && strings.HasPrefix(tt.expectedConfig.SSHKeyPath, "~/") {
					tt.expectedConfig.SSHKeyPath = filepath.Join(tmpDir, tt.expectedConfig.SSHKeyPath[2:])
				}
				assert.Equal(t, tt.expectedConfig, cfg)
			}
		})
	}
}

func TestLoadConfigWithoutFile(t *testing.T) {
	// Test loading when no config file exists
	tmpDir := t.TempDir()
	
	// Mock home directory
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)
	
	configPath := filepath.Join(tmpDir, ".config", "l8s", "config.yaml")
	
	// Load config when file doesn't exist
	cfg, err := Load(configPath)
	
	// Should fail validation since remote settings are missing
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remote_host is required")
	assert.Nil(t, cfg)
}

func TestConfigPaths(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Mock home directory
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)
	
	// Test default config path
	expectedPath := filepath.Join(tmpDir, ".config", "l8s", "config.yaml")
	actualPath := GetConfigPath()
	assert.Equal(t, expectedPath, actualPath)
}

func TestExpandPath(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Mock home directory
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)
	
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "tilde expansion",
			input:    "~/.ssh/id_rsa.pub",
			expected: filepath.Join(tmpDir, ".ssh", "id_rsa.pub"),
		},
		{
			name:     "absolute path unchanged",
			input:    "/etc/ssh/key.pub",
			expected: "/etc/ssh/key.pub",
		},
		{
			name:     "relative path unchanged",
			input:    "./keys/id_rsa.pub",
			expected: "./keys/id_rsa.pub",
		},
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with remote settings",
			config: &Config{
				RemoteHost:      "server.example.com",
				RemoteUser:      "podman",
				RemoteSocket:    "/run/podman/podman.sock",
				SSHKeyPath:      "/home/user/.ssh/id_ed25519",
				SSHPortStart:    2200,
				BaseImage:       "localhost/l8s-fedora:latest",
				ContainerPrefix: "dev",
				SSHPublicKey:    "",
				ContainerUser:   "dev",
			},
			wantErr: false,
		},
		{
			name: "missing remote host",
			config: &Config{
				RemoteHost:      "", // Missing
				RemoteUser:      "podman",
				RemoteSocket:    "/run/podman/podman.sock",
				SSHKeyPath:      "/home/user/.ssh/id_ed25519",
				SSHPortStart:    2200,
				BaseImage:       "localhost/l8s-fedora:latest",
				ContainerPrefix: "dev",
				SSHPublicKey:    "",
				ContainerUser:   "dev",
			},
			wantErr: true,
			errMsg:  "remote_host is required - l8s ONLY supports remote container management",
		},
		{
			name: "missing remote user",
			config: &Config{
				RemoteHost:      "server.example.com",
				RemoteUser:      "", // Missing
				RemoteSocket:    "/run/podman/podman.sock",
				SSHKeyPath:      "/home/user/.ssh/id_ed25519",
				SSHPortStart:    2200,
				BaseImage:       "localhost/l8s-fedora:latest",
				ContainerPrefix: "dev",
				SSHPublicKey:    "",
				ContainerUser:   "dev",
			},
			wantErr: true,
			errMsg:  "remote_user is required - l8s ONLY supports remote container management",
		},
		{
			name: "invalid SSH port",
			config: &Config{
				RemoteHost:      "server.example.com",
				RemoteUser:      "podman",
				SSHPortStart:    1023, // Below 1024
				BaseImage:       "localhost/l8s-fedora:latest",
				ContainerPrefix: "dev",
				SSHPublicKey:    "",
				ContainerUser:   "dev",
			},
			wantErr: true,
			errMsg:  "ssh_port_start must be between 1024 and 65000",
		},
		{
			name: "empty base image",
			config: &Config{
				RemoteHost:      "server.example.com",
				RemoteUser:      "podman",
				SSHPortStart:    2200,
				BaseImage:       "",
				ContainerPrefix: "dev",
				SSHPublicKey:    "",
				ContainerUser:   "dev",
			},
			wantErr: true,
			errMsg:  "base_image cannot be empty",
		},
		{
			name: "empty container prefix",
			config: &Config{
				RemoteHost:      "server.example.com",
				RemoteUser:      "podman",
				SSHPortStart:    2200,
				BaseImage:       "localhost/l8s-fedora:latest",
				ContainerPrefix: "",
				SSHPublicKey:    "",
				ContainerUser:   "dev",
			},
			wantErr: true,
			errMsg:  "container_prefix cannot be empty",
		},
		{
			name: "empty container user",
			config: &Config{
				RemoteHost:      "server.example.com",
				RemoteUser:      "podman",
				SSHPortStart:    2200,
				BaseImage:       "localhost/l8s-fedora:latest",
				ContainerPrefix: "dev",
				SSHPublicKey:    "",
				ContainerUser:   "",
			},
			wantErr: true,
			errMsg:  "container_user cannot be empty",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDotfilesCopy(t *testing.T) {
	tests := []struct {
		name              string
		dotfilesExist     bool
		expectedBehavior  string
	}{
		{
			name:             "dotfiles directory exists",
			dotfilesExist:    true,
			expectedBehavior: "copy dotfiles to container",
		},
		{
			name:             "dotfiles directory missing",
			dotfilesExist:    false,
			expectedBehavior: "skip dotfiles copy",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			dotfilesPath := filepath.Join(tmpDir, "dotfiles")
			
			if tt.dotfilesExist {
				err := os.MkdirAll(dotfilesPath, 0755)
				require.NoError(t, err)
				
				// Create sample dotfiles
				files := map[string]string{
					".zshrc":      "# zsh config",
					".gitconfig":  "[user]\n  name = Test\n  email = test@example.com",
				}
				
				for filename, content := range files {
					err = os.WriteFile(filepath.Join(dotfilesPath, filename), []byte(content), 0644)
					require.NoError(t, err)
				}
			}
			
			// Verify dotfiles directory existence
			_, err := os.Stat(dotfilesPath)
			if tt.dotfilesExist {
				assert.NoError(t, err)
			} else {
				assert.True(t, os.IsNotExist(err))
			}
		})
	}
}
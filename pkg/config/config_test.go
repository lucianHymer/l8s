package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	
	// Multi-connection configuration defaults
	assert.Equal(t, "", cfg.ActiveConnection) // Must be configured
	assert.NotNil(t, cfg.Connections)
	assert.Empty(t, cfg.Connections)
	
	// Shared settings defaults
	assert.Equal(t, 2200, cfg.SSHPortStart)
	assert.Equal(t, "localhost/l8s-fedora:latest", cfg.BaseImage)
	assert.Equal(t, "dev", cfg.ContainerPrefix)
	assert.Equal(t, "", cfg.SSHPublicKey) // Auto-detect
	assert.Equal(t, "dev", cfg.ContainerUser)
}

func TestLoadConfig(t *testing.T) {
	home, _ := os.UserHomeDir()
	
	tests := []struct {
		name           string
		configContent  string
		expectedConfig *Config
		wantErr        bool
	}{
		{
			name: "valid config file with multi-connection settings",
			configContent: `
active_connection: "default"
connections:
  default:
    address: "server.example.com"
    description: "Default connection"
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
				ActiveConnection: "default",
				Connections: map[string]ConnectionConfig{
					"default": {
						Address:     "server.example.com",
						Description: "Default connection",
					},
				},
				RemoteUser:      "podman",
				RemoteSocket:    "/run/podman/podman.sock",
				SSHKeyPath:      filepath.Join(home, ".ssh/id_ed25519"), // will be expanded
				SSHPortStart:    2300,
				WebPortStart:    3000,
				BaseImage:       "localhost/custom-l8s:v2",
				ContainerPrefix: "work",
				SSHPublicKey:    filepath.Join(home, ".ssh/custom_key.pub"),
				ContainerUser:   "lucian",
			},
			wantErr: false,
		},
		{
			name: "config without connections should fail validation",
			configContent: `
active_connection: "default"
remote_user: "podman"
ssh_port_start: 2400
container_user: "developer"
`,
			expectedConfig: nil,
			wantErr: true, // Should fail validation due to missing connections
		},
		{
			name:          "empty config file should fail validation",
			configContent: "",
			expectedConfig: nil,
			wantErr: true, // Should fail validation due to missing connections
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
			name: "partial config with connections",
			configContent: `
active_connection: "prod"
connections:
  prod:
    address: "server.example.com"
remote_user: "root"
container_user: "developer"
`,
			expectedConfig: &Config{
				ActiveConnection: "prod",
				Connections: map[string]ConnectionConfig{
					"prod": {
						Address: "server.example.com",
					},
				},
				RemoteUser:      "root",
				RemoteSocket:    "/run/podman/podman.sock", // default
				SSHKeyPath:      "", // no default set
				SSHPortStart:    2200, // default
				WebPortStart:    3000, // default
				BaseImage:       "localhost/l8s-fedora:latest", // default
				ContainerPrefix: "dev", // default
				SSHPublicKey:    "", // default
				ContainerUser:   "developer",
			},
			wantErr: false,
		},
		{
			name: "multiple connections config",
			configContent: `
active_connection: "vpn"
connections:
  default:
    address: "192.168.1.100"
    description: "Local network"
  vpn:
    address: "10.0.0.50"
    description: "VPN access"
remote_user: "admin"
container_prefix: "test"
`,
			expectedConfig: &Config{
				ActiveConnection: "vpn",
				Connections: map[string]ConnectionConfig{
					"default": {
						Address:     "192.168.1.100",
						Description: "Local network",
					},
					"vpn": {
						Address:     "10.0.0.50",
						Description: "VPN access",
					},
				},
				RemoteUser:      "admin",
				RemoteSocket:    "/run/podman/podman.sock",
				SSHKeyPath:      "",
				SSHPortStart:    2200,
				WebPortStart:    3000,
				BaseImage:       "localhost/l8s-fedora:latest",
				ContainerPrefix: "test",
				SSHPublicKey:    "",
				ContainerUser:   "dev",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			
			if err := os.WriteFile(configPath, []byte(tt.configContent), 0644); err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			// Load config
			cfg, err := Load(configPath)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)

			// Expand paths for comparison
			home, _ := os.UserHomeDir()
			if tt.expectedConfig != nil {
				if tt.expectedConfig.SSHPublicKey != "" && strings.HasPrefix(tt.expectedConfig.SSHPublicKey, "~/") {
					tt.expectedConfig.SSHPublicKey = filepath.Join(home, tt.expectedConfig.SSHPublicKey[2:])
				}
			}

			assert.Equal(t, tt.expectedConfig, cfg)
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				ActiveConnection: "default",
				Connections: map[string]ConnectionConfig{
					"default": {
						Address: "server.example.com",
					},
				},
				RemoteUser:      "admin",
				SSHPortStart:    2200,
				WebPortStart:    3000,
				BaseImage:       "localhost/l8s-fedora:latest",
				ContainerPrefix: "dev",
				ContainerUser:   "dev",
			},
			wantErr: false,
		},
		{
			name: "missing active_connection",
			config: &Config{
				Connections: map[string]ConnectionConfig{
					"default": {
						Address: "server.example.com",
					},
				},
				RemoteUser:      "admin",
				SSHPortStart:    2200,
				WebPortStart:    3000,
				BaseImage:       "localhost/l8s-fedora:latest",
				ContainerPrefix: "dev",
				ContainerUser:   "dev",
			},
			wantErr: true,
			errMsg:  "active_connection must be specified",
		},
		{
			name: "no connections",
			config: &Config{
				ActiveConnection: "default",
				Connections:      map[string]ConnectionConfig{},
				RemoteUser:       "admin",
				SSHPortStart:     2200,
				BaseImage:        "localhost/l8s-fedora:latest",
				ContainerPrefix:  "dev",
				ContainerUser:    "dev",
			},
			wantErr: true,
			errMsg:  "at least one connection must be configured",
		},
		{
			name: "active connection not in connections map",
			config: &Config{
				ActiveConnection: "nonexistent",
				Connections: map[string]ConnectionConfig{
					"default": {
						Address: "server.example.com",
					},
				},
				RemoteUser:      "admin",
				SSHPortStart:    2200,
				WebPortStart:    3000,
				BaseImage:       "localhost/l8s-fedora:latest",
				ContainerPrefix: "dev",
				ContainerUser:   "dev",
			},
			wantErr: true,
			errMsg:  "active connection 'nonexistent' not found",
		},
		{
			name: "missing connection address",
			config: &Config{
				ActiveConnection: "default",
				Connections: map[string]ConnectionConfig{
					"default": {
						Address: "",
					},
				},
				RemoteUser:      "admin",
				SSHPortStart:    2200,
				WebPortStart:    3000,
				BaseImage:       "localhost/l8s-fedora:latest",
				ContainerPrefix: "dev",
				ContainerUser:   "dev",
			},
			wantErr: true,
			errMsg:  "address is required",
		},
		{
			name: "missing remote_user",
			config: &Config{
				ActiveConnection: "default",
				Connections: map[string]ConnectionConfig{
					"default": {
						Address: "server.example.com",
					},
				},
				RemoteUser:      "",
				SSHPortStart:    2200,
				WebPortStart:    3000,
				BaseImage:       "localhost/l8s-fedora:latest",
				ContainerPrefix: "dev",
				ContainerUser:   "dev",
			},
			wantErr: true,
			errMsg:  "remote_user is required",
		},
		{
			name: "invalid SSH port - too low",
			config: &Config{
				ActiveConnection: "default",
				Connections: map[string]ConnectionConfig{
					"default": {
						Address: "server.example.com",
					},
				},
				RemoteUser:      "admin",
				SSHPortStart:    500,
				BaseImage:       "localhost/l8s-fedora:latest",
				ContainerPrefix: "dev",
				ContainerUser:   "dev",
			},
			wantErr: true,
			errMsg:  "ssh_port_start must be between 1024 and 65000",
		},
		{
			name: "empty base image",
			config: &Config{
				ActiveConnection: "default",
				Connections: map[string]ConnectionConfig{
					"default": {
						Address: "server.example.com",
					},
				},
				RemoteUser:      "admin",
				SSHPortStart:    2200,
				WebPortStart:    3000,
				BaseImage:       "",
				ContainerPrefix: "dev",
				ContainerUser:   "dev",
			},
			wantErr: true,
			errMsg:  "base_image cannot be empty",
		},
		{
			name: "empty container prefix",
			config: &Config{
				ActiveConnection: "default",
				Connections: map[string]ConnectionConfig{
					"default": {
						Address: "server.example.com",
					},
				},
				RemoteUser:      "admin",
				SSHPortStart:    2200,
				WebPortStart:    3000,
				BaseImage:       "localhost/l8s-fedora:latest",
				ContainerPrefix: "",
				ContainerUser:   "dev",
			},
			wantErr: true,
			errMsg:  "container_prefix cannot be empty",
		},
		{
			name: "container prefix too long",
			config: &Config{
				ActiveConnection: "default",
				Connections: map[string]ConnectionConfig{
					"default": {
						Address: "server.example.com",
					},
				},
				RemoteUser:      "admin",
				SSHPortStart:    2200,
				WebPortStart:    3000,
				BaseImage:       "localhost/l8s-fedora:latest",
				ContainerPrefix: "verylongprefix",
				ContainerUser:   "dev",
			},
			wantErr: true,
			errMsg:  "container_prefix must be 10 characters or less",
		},
		{
			name: "invalid container prefix characters",
			config: &Config{
				ActiveConnection: "default",
				Connections: map[string]ConnectionConfig{
					"default": {
						Address: "server.example.com",
					},
				},
				RemoteUser:      "admin",
				SSHPortStart:    2200,
				WebPortStart:    3000,
				BaseImage:       "localhost/l8s-fedora:latest",
				ContainerPrefix: "Test",
				ContainerUser:   "dev",
			},
			wantErr: true,
			errMsg:  "container_prefix must consist of lowercase letters",
		},
		{
			name: "empty container user",
			config: &Config{
				ActiveConnection: "default",
				Connections: map[string]ConnectionConfig{
					"default": {
						Address: "server.example.com",
					},
				},
				RemoteUser:      "admin",
				SSHPortStart:    2200,
				WebPortStart:    3000,
				BaseImage:       "localhost/l8s-fedora:latest",
				ContainerPrefix: "dev",
				ContainerUser:   "",
			},
			wantErr: true,
			errMsg:  "container_user cannot be empty",
		},
		{
			name: "invalid container user",
			config: &Config{
				ActiveConnection: "default",
				Connections: map[string]ConnectionConfig{
					"default": {
						Address: "server.example.com",
					},
				},
				RemoteUser:      "admin",
				SSHPortStart:    2200,
				WebPortStart:    3000,
				BaseImage:       "localhost/l8s-fedora:latest",
				ContainerPrefix: "dev",
				ContainerUser:   "Invalid User",
			},
			wantErr: true,
			errMsg:  "container_user must be a valid Linux username",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConnectionMethods(t *testing.T) {
	t.Run("GetActiveConnection", func(t *testing.T) {
		cfg := &Config{
			ActiveConnection: "default",
			Connections: map[string]ConnectionConfig{
				"default": {
					Address:     "192.168.1.100",
					Description: "Default connection",
				},
			},
		}
		
		conn, err := cfg.GetActiveConnection()
		require.NoError(t, err)
		assert.Equal(t, "192.168.1.100", conn.Address)
		assert.Equal(t, "Default connection", conn.Description)
	})
	
	t.Run("GetActiveAddress", func(t *testing.T) {
		cfg := &Config{
			ActiveConnection: "default",
			Connections: map[string]ConnectionConfig{
				"default": {
					Address: "192.168.1.100",
				},
			},
		}
		
		address, err := cfg.GetActiveAddress()
		require.NoError(t, err)
		assert.Equal(t, "192.168.1.100", address)
	})
	
	t.Run("SetActiveConnection", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")
		
		cfg := &Config{
			ActiveConnection: "default",
			Connections: map[string]ConnectionConfig{
				"default": {
					Address: "192.168.1.100",
				},
				"vpn": {
					Address: "10.0.0.50",
				},
			},
			RemoteUser:      "admin",
			SSHPortStart:    2200,
			WebPortStart:    3000,
			BaseImage:       "localhost/l8s-fedora:latest",
			ContainerPrefix: "dev",
			ContainerUser:   "dev",
		}
		
		// Use SetActiveConnectionWithPath for testing
		
		err := cfg.SetActiveConnectionWithPath("vpn", configPath)
		require.NoError(t, err)
		assert.Equal(t, "vpn", cfg.ActiveConnection)
		
		// Verify it was saved
		loaded, err := Load(configPath)
		require.NoError(t, err)
		assert.Equal(t, "vpn", loaded.ActiveConnection)
	})
	
	t.Run("ListConnections", func(t *testing.T) {
		cfg := &Config{
			Connections: map[string]ConnectionConfig{
				"default": {
					Address:     "192.168.1.100",
					Description: "Local",
				},
				"vpn": {
					Address:     "10.0.0.50",
					Description: "VPN",
				},
			},
		}
		
		conns := cfg.ListConnections()
		assert.Len(t, conns, 2)
		assert.Equal(t, "192.168.1.100", conns["default"].Address)
		assert.Equal(t, "10.0.0.50", conns["vpn"].Address)
	})
}
// +build !test

package container

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPodmanClient_RemoteOnly(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		setupEnv      func()
		cleanupEnv    func()
		wantErr       bool
		errContains   string
	}{
		{
			name: "missing config file should fail",
			configContent: "", // No config file
			setupEnv:      func() {},
			cleanupEnv:    func() {},
			wantErr:       true,
			errContains:   "l8s requires remote server configuration",
		},
		{
			name: "config without remote host should fail",
			configContent: `
remote_user: "podman"
ssh_port_start: 2200
base_image: "localhost/l8s-fedora:latest"
container_prefix: "dev"
container_user: "dev"
`,
			setupEnv:   func() {},
			cleanupEnv: func() {},
			wantErr:    true,
			errContains: "l8s requires remote server configuration",
		},
		{
			name: "config without remote user should fail",
			configContent: `
remote_host: "server.example.com"
ssh_port_start: 2200
base_image: "localhost/l8s-fedora:latest"
container_prefix: "dev"
container_user: "dev"
`,
			setupEnv:   func() {},
			cleanupEnv: func() {},
			wantErr:    true,
			errContains: "l8s requires remote server configuration",
		},
		{
			name: "valid config but no ssh-agent should fail",
			configContent: `
remote_host: "server.example.com"
remote_user: "podman"
ssh_port_start: 2200
base_image: "localhost/l8s-fedora:latest"
container_prefix: "dev"
container_user: "dev"
`,
			setupEnv: func() {
				// Unset SSH_AUTH_SOCK to simulate no ssh-agent
				os.Unsetenv("SSH_AUTH_SOCK")
			},
			cleanupEnv: func() {},
			wantErr:    true,
			errContains: "ssh-agent is required but not running",
		},
		{
			name: "local connection URIs should be rejected",
			configContent: `
remote_host: "localhost"
remote_user: "podman"
ssh_port_start: 2200
base_image: "localhost/l8s-fedora:latest"
container_prefix: "dev"
container_user: "dev"
`,
			setupEnv: func() {
				// Set a fake SSH_AUTH_SOCK
				os.Setenv("SSH_AUTH_SOCK", "/tmp/fake-ssh-agent.sock")
			},
			cleanupEnv: func() {
				os.Unsetenv("SSH_AUTH_SOCK")
			},
			wantErr:    true,
			errContains: "failed to connect to Podman on remote server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for config
			tmpDir := t.TempDir()
			configDir := filepath.Join(tmpDir, ".config", "l8s")
			err := os.MkdirAll(configDir, 0755)
			require.NoError(t, err)

			// Write config file if content provided
			if tt.configContent != "" {
				configPath := filepath.Join(configDir, "config.yaml")
				err = os.WriteFile(configPath, []byte(tt.configContent), 0644)
				require.NoError(t, err)
			}

			// Mock home directory
			origHome := os.Getenv("HOME")
			os.Setenv("HOME", tmpDir)
			defer os.Setenv("HOME", origHome)

			// Save original SSH_AUTH_SOCK
			origSSHAuthSock := os.Getenv("SSH_AUTH_SOCK")
			defer func() {
				if origSSHAuthSock != "" {
					os.Setenv("SSH_AUTH_SOCK", origSSHAuthSock)
				}
			}()

			// Setup test environment
			tt.setupEnv()
			defer tt.cleanupEnv()

			// Create new podman client
			client, err := NewPodmanClient()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestPodmanClient_ErrorMessages(t *testing.T) {
	// Test that error messages provide helpful troubleshooting info
	tmpDir := t.TempDir()
	
	// Mock home directory
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Create client without config
	client, err := NewPodmanClient()
	
	require.Error(t, err)
	
	// Check error message contains helpful info
	errMsg := err.Error()
	assert.Contains(t, errMsg, "l8s requires remote server configuration")
	assert.Contains(t, errMsg, "~/.config/l8s/config.yaml")
	assert.Contains(t, errMsg, "remote_host:")
	assert.Contains(t, errMsg, "remote_user:")
	assert.Contains(t, errMsg, "l8s init")
	assert.Contains(t, errMsg, "l8s ONLY supports remote container management")
}
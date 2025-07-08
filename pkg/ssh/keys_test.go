package ssh

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadPublicKey(t *testing.T) {
	tests := []struct {
		name        string
		keyContent  string
		setupFile   bool
		wantErr     bool
		errContains string
	}{
		{
			name:       "valid ed25519 key",
			keyContent: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9pKe4 user@example.com",
			setupFile:  true,
			wantErr:    false,
		},
		{
			name:       "valid rsa key",
			keyContent: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7... user@example.com",
			setupFile:  true,
			wantErr:    false,
		},
		{
			name:        "invalid key format",
			keyContent:  "invalid-key-format",
			setupFile:   true,
			wantErr:     false, // ReadPublicKey doesn't validate, just reads
		},
		{
			name:        "empty file",
			keyContent:  "",
			setupFile:   true,
			wantErr:     true,
			errContains: "SSH public key file is empty",
		},
		{
			name:        "file not found",
			setupFile:   false,
			wantErr:     true,
			errContains: "no such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			keyPath := filepath.Join(tmpDir, "test_key.pub")

			if tt.setupFile {
				err := os.WriteFile(keyPath, []byte(tt.keyContent), 0644)
				require.NoError(t, err)
			}

			key, err := ReadPublicKey(keyPath)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Empty(t, key)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.keyContent, key)
			}
		})
	}
}

func TestValidatePublicKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "valid ed25519 key",
			key:     "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9pKe4 user@example.com",
			wantErr: false,
		},
		{
			name:    "valid rsa key",
			key:     "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7... user@example.com",
			wantErr: false,
		},
		{
			name:    "valid ecdsa key",
			key:     "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEmKSENjQEezOmxkZMy7opKgwFB9nkt5YRrYMjNuG5N87uRgg6CLrbo5wAdT/y6v0mKV0U2w0WZ2YB/++Tpockg= user@example.com",
			wantErr: false,
		},
		{
			name:    "empty key",
			key:     "",
			wantErr: true,
		},
		{
			name:    "invalid format",
			key:     "invalid-key-format",
			wantErr: true,
		},
		{
			name:    "missing key type",
			key:     "AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9pKe4 user@example.com",
			wantErr: true,
		},
		{
			name:    "unsupported key type",
			key:     "ssh-dss AAAAB3NzaC1kc3MAAACBAP... user@example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePublicKey(tt.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerateAuthorizedKeys(t *testing.T) {
	publicKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9pKe4 user@example.com"
	
	content := GenerateAuthorizedKeys(publicKey)
	
	// Should contain the public key
	assert.Contains(t, content, publicKey)
	
	// Should have proper permissions comment
	assert.Contains(t, content, "# Managed by l8s")
	
	// Should end with newline
	assert.True(t, content[len(content)-1] == '\n')
}

func TestSSHConfigEntry(t *testing.T) {
	tests := []struct {
		name          string
		containerName string
		sshPort       int
		containerUser string
		remoteHost    string
		want          string
	}{
		{
			name:          "standard config with remote host",
			containerName: "dev-myproject",
			sshPort:       2200,
			containerUser: "dev",
			remoteHost:    "server.example.com",
			want: `Host dev-myproject
    HostName server.example.com
    Port 2200
    User dev
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
    ForwardAgent yes
    ControlMaster auto
    ControlPath ~/.ssh/control-%r@%h:%p
    ControlPersist 10m`,
		},
		{
			name:          "custom user",
			containerName: "dev-test",
			sshPort:       2201,
			containerUser: "lucian",
			remoteHost:    "remote.server.io",
			want: `Host dev-test
    HostName remote.server.io
    Port 2201
    User lucian
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
    ForwardAgent yes
    ControlMaster auto
    ControlPath ~/.ssh/control-%r@%h:%p
    ControlPersist 10m`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := GenerateSSHConfigEntry(tt.containerName, tt.sshPort, tt.containerUser, "dev", tt.remoteHost)
			assert.Equal(t, tt.want+"\n", entry)
		})
	}
}

func TestManageSSHConfig(t *testing.T) {
	t.Run("add new entry to empty config", func(t *testing.T) {
		tmpDir := t.TempDir()
		sshDir := filepath.Join(tmpDir, ".ssh")
		err := os.MkdirAll(sshDir, 0700)
		require.NoError(t, err)
		
		configPath := filepath.Join(sshDir, "config")
		
		entry := GenerateSSHConfigEntry("dev-myproject", 2200, "dev", "dev", "localhost")
		err = AddSSHConfigEntry(configPath, entry)
		require.NoError(t, err)
		
		// Verify the entry was added
		content, err := os.ReadFile(configPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "Host dev-myproject")
		assert.Contains(t, string(content), "Port 2200")
		
		// Verify file permissions
		info, err := os.Stat(configPath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	})

	t.Run("add entry to existing config", func(t *testing.T) {
		tmpDir := t.TempDir()
		sshDir := filepath.Join(tmpDir, ".ssh")
		err := os.MkdirAll(sshDir, 0700)
		require.NoError(t, err)
		
		configPath := filepath.Join(sshDir, "config")
		
		// Create existing config
		existingConfig := `Host existing
    HostName example.com
    User admin

`
		err = os.WriteFile(configPath, []byte(existingConfig), 0600)
		require.NoError(t, err)
		
		entry := GenerateSSHConfigEntry("dev-myproject", 2200, "dev", "dev", "localhost")
		err = AddSSHConfigEntry(configPath, entry)
		require.NoError(t, err)
		
		// Verify both entries exist
		content, err := os.ReadFile(configPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "Host existing")
		assert.Contains(t, string(content), "Host dev-myproject")
	})

	t.Run("remove entry from config", func(t *testing.T) {
		tmpDir := t.TempDir()
		sshDir := filepath.Join(tmpDir, ".ssh")
		err := os.MkdirAll(sshDir, 0700)
		require.NoError(t, err)
		
		configPath := filepath.Join(sshDir, "config")
		
		// Create config with l8s entry
		config := `Host existing
    HostName example.com
    User admin

# BEGIN l8s managed block: dev-myproject
Host dev-myproject
    HostName localhost
    Port 2200
    User dev
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
    LogLevel ERROR
# END l8s managed block: dev-myproject
`
		err = os.WriteFile(configPath, []byte(config), 0600)
		require.NoError(t, err)
		
		err = RemoveSSHConfigEntry(configPath, "dev-myproject")
		require.NoError(t, err)
		
		// Verify entry was removed but existing entry remains
		content, err := os.ReadFile(configPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "Host existing")
		assert.NotContains(t, string(content), "Host dev-myproject")
	})
}

func TestFindSSHPublicKey(t *testing.T) {
	t.Run("find existing keys", func(t *testing.T) {
		tmpDir := t.TempDir()
		sshDir := filepath.Join(tmpDir, ".ssh")
		err := os.MkdirAll(sshDir, 0700)
		require.NoError(t, err)
		
		// Create test keys
		keys := map[string]string{
			"id_ed25519.pub": "ssh-ed25519 AAAAC3... user@example.com",
			"id_rsa.pub":     "ssh-rsa AAAAB3... user@example.com",
			"id_ecdsa.pub":   "ecdsa-sha2-nistp256 AAAAE2... user@example.com",
		}
		
		for filename, content := range keys {
			err = os.WriteFile(filepath.Join(sshDir, filename), []byte(content), 0644)
			require.NoError(t, err)
		}
		
		// Mock home directory
		origHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpDir)
		defer os.Setenv("HOME", origHome)
		
		key, err := FindSSHPublicKey()
		require.NoError(t, err)
		// Should find one of the keys (alphabetical order)
		assert.True(t, strings.HasPrefix(key, "ssh-ed25519") || 
			strings.HasPrefix(key, "ssh-rsa") || 
			strings.HasPrefix(key, "ecdsa-sha2-nistp256"))
	})

	t.Run("no keys found", func(t *testing.T) {
		tmpDir := t.TempDir()
		sshDir := filepath.Join(tmpDir, ".ssh")
		err := os.MkdirAll(sshDir, 0700)
		require.NoError(t, err)
		
		// Mock home directory
		origHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpDir)
		defer os.Setenv("HOME", origHome)
		
		_, err = FindSSHPublicKey()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no SSH public key found")
	})
}

func TestIsPortAvailable(t *testing.T) {
	// Test with a likely available high port
	available := IsPortAvailable(55555)
	assert.True(t, available, "Port 55555 should be available")
	
	// Note: Testing unavailable ports is tricky in unit tests
	// as it requires actually binding to a port
}

func TestCopySSHKeyToContainer(t *testing.T) {
	// This is more of an integration test that would require
	// a mock container runtime or actual container
	t.Skip("Requires container runtime mock")
}
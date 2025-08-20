package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSSHConfig(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected map[string]string
	}{
		{
			name: "single l8s container",
			content: `Host dev-myproject
    HostName 192.168.1.100
    Port 2201
    User dev
    StrictHostKeyChecking no`,
			expected: map[string]string{
				"dev-myproject": "192.168.1.100",
			},
		},
		{
			name: "multiple l8s containers",
			content: `Host dev-project1
    HostName 192.168.1.100
    Port 2201
    User dev

Host dev-project2
    HostName 192.168.1.100
    Port 2202
    User dev

Host other-host
    HostName example.com
    Port 22
    User admin`,
			expected: map[string]string{
				"dev-project1": "192.168.1.100",
				"dev-project2": "192.168.1.100",
			},
		},
		{
			name: "mixed with non-l8s hosts",
			content: `Host github.com
    HostName github.com
    User git

Host dev-webapp
    HostName 10.0.0.50
    Port 2203
    User dev

Host production
    HostName prod.example.com
    Port 22
    User deploy`,
			expected: map[string]string{
				"dev-webapp": "10.0.0.50",
			},
		},
		{
			name:     "empty config",
			content:  "",
			expected: map[string]string{},
		},
		{
			name: "no hostname in block",
			content: `Host dev-broken
    Port 2204
    User dev`,
			expected: map[string]string{
				"dev-broken": "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config")
			
			err := os.WriteFile(configPath, []byte(tt.content), 0600)
			require.NoError(t, err)
			
			entries, err := ParseSSHConfig(configPath)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, entries)
		})
	}

	t.Run("file not exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "nonexistent")
		
		entries, err := ParseSSHConfig(configPath)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}

func TestValidateSSHConfigsMatchConnection(t *testing.T) {
	// Create a temporary SSH config
	tmpDir := t.TempDir()
	sshDir := filepath.Join(tmpDir, ".ssh")
	err := os.MkdirAll(sshDir, 0700)
	require.NoError(t, err)
	
	configPath := filepath.Join(sshDir, "config")
	
	// Override GetHomeDir for this test
	oldGetHomeDir := getHomeDirFunc
	defer func() { getHomeDirFunc = oldGetHomeDir }()
	getHomeDirFunc = func() string { return tmpDir }

	tests := []struct {
		name       string
		content    string
		activeHost string
		wantErr    bool
		errMsg     string
	}{
		{
			name: "all configs match",
			content: `Host dev-project1
    HostName 192.168.1.100
    Port 2201

Host dev-project2
    HostName 192.168.1.100
    Port 2202`,
			activeHost: "192.168.1.100",
			wantErr:    false,
		},
		{
			name: "one config mismatched",
			content: `Host dev-project1
    HostName 192.168.1.100
    Port 2201

Host dev-project2
    HostName 10.0.0.50
    Port 2202`,
			activeHost: "192.168.1.100",
			wantErr:    true,
			errMsg:     "dev-project2 (points to 10.0.0.50)",
		},
		{
			name: "all configs mismatched",
			content: `Host dev-project1
    HostName 10.0.0.50
    Port 2201

Host dev-project2
    HostName 10.0.0.50
    Port 2202`,
			activeHost: "192.168.1.100",
			wantErr:    true,
			errMsg:     "SSH configs out of sync",
		},
		{
			name:       "no l8s configs",
			content:    `Host github.com
    HostName github.com
    User git`,
			activeHost: "192.168.1.100",
			wantErr:    false,
		},
		{
			name:       "empty config",
			content:    "",
			activeHost: "192.168.1.100",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := os.WriteFile(configPath, []byte(tt.content), 0600)
			require.NoError(t, err)
			
			err = ValidateSSHConfigsMatchConnection(tt.activeHost)
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

func TestConnectionSwitchCommand_UpdateSSHConfigEntry(t *testing.T) {
	tests := []struct {
		name        string
		original    string
		container   string
		newHost     string
		expected    string
	}{
		{
			name: "update single entry",
			original: `Host dev-myproject
    HostName 192.168.1.100
    Port 2201
    User dev
    StrictHostKeyChecking no

Host github.com
    HostName github.com
    User git`,
			container: "dev-myproject",
			newHost:   "10.0.0.50",
			expected: `Host dev-myproject
    HostName 10.0.0.50
    Port 2201
    User dev
    StrictHostKeyChecking no

Host github.com
    HostName github.com
    User git`,
		},
		{
			name: "update with different indentation",
			original: `Host dev-webapp
  HostName 192.168.1.100
  Port 2202
  User dev`,
			container: "dev-webapp",
			newHost:   "vpn.example.com",
			expected: `Host dev-webapp
  HostName vpn.example.com
  Port 2202
  User dev`,
		},
		{
			name: "no change for non-matching container",
			original: `Host dev-project1
    HostName 192.168.1.100
    Port 2201

Host dev-project2
    HostName 192.168.1.100
    Port 2202`,
			container: "dev-project3",
			newHost:   "10.0.0.50",
			expected: `Host dev-project1
    HostName 192.168.1.100
    Port 2201

Host dev-project2
    HostName 192.168.1.100
    Port 2202`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config")
			
			err := os.WriteFile(configPath, []byte(tt.original), 0600)
			require.NoError(t, err)
			
			cmd := &ConnectionSwitchCommand{}
			err = cmd.updateSSHConfigEntry(configPath, tt.container, tt.newHost)
			require.NoError(t, err)
			
			content, err := os.ReadFile(configPath)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(content))
		})
	}
}

func TestConnectionSwitchCommand_FindSSHConfigUpdates(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		oldHost  string
		newHost  string
		expected []string
	}{
		{
			name: "find matching containers",
			content: `Host dev-project1
    HostName 192.168.1.100
    Port 2201

Host dev-project2
    HostName 192.168.1.100
    Port 2202

Host dev-project3
    HostName 10.0.0.50
    Port 2203`,
			oldHost: "192.168.1.100",
			newHost: "10.0.0.50",
			expected: []string{"dev-project1", "dev-project2"},
		},
		{
			name: "no matching containers",
			content: `Host dev-project1
    HostName 10.0.0.50
    Port 2201

Host github.com
    HostName github.com
    User git`,
			oldHost:  "192.168.1.100",
			newHost:  "10.0.0.50",
			expected: []string{},
		},
		{
			name: "ignore non-l8s hosts",
			content: `Host production
    HostName 192.168.1.100
    Port 22

Host dev-webapp
    HostName 192.168.1.100
    Port 2201`,
			oldHost:  "192.168.1.100",
			newHost:  "10.0.0.50",
			expected: []string{"dev-webapp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config")
			
			err := os.WriteFile(configPath, []byte(tt.content), 0600)
			require.NoError(t, err)
			
			cmd := &ConnectionSwitchCommand{}
			updates, err := cmd.findSSHConfigUpdates(configPath, tt.oldHost, tt.newHost)
			require.NoError(t, err)
			assert.ElementsMatch(t, tt.expected, updates)
		})
	}

	t.Run("file not exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "nonexistent")
		
		cmd := &ConnectionSwitchCommand{}
		updates, err := cmd.findSSHConfigUpdates(configPath, "192.168.1.100", "10.0.0.50")
		require.NoError(t, err)
		assert.Empty(t, updates)
	})
}


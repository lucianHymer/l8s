package container

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetHostGitConfig(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		mockOutput  string
		mockError   error
		expected    string
		expectError bool
	}{
		{
			name:       "get user.name",
			key:        "user.name",
			mockOutput: "John Doe\n",
			expected:   "John Doe",
		},
		{
			name:       "get user.email",
			key:        "user.email",
			mockOutput: "john@example.com\n",
			expected:   "john@example.com",
		},
		{
			name:        "config not set",
			key:         "user.name",
			mockError:   &exec.ExitError{},
			expected:    "",
			expectError: false, // We handle missing config gracefully
		},
		{
			name:       "empty value",
			key:        "user.name",
			mockOutput: "\n",
			expected:   "",
		},
		{
			name:       "value with spaces",
			key:        "user.name",
			mockOutput: "John Q. Doe\n",
			expected:   "John Q. Doe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We'll mock the exec command later
			// For now, let's assume GetHostGitConfig exists
			// This is TDD - write tests first
		})
	}
}

func TestReadHostGitIdentity(t *testing.T) {
	tests := []struct {
		name         string
		mockUserName string
		mockUserEmail string
		mockErrors    map[string]error
		expected     GitIdentity
		expectError  bool
	}{
		{
			name:         "both name and email present",
			mockUserName: "Jane Doe",
			mockUserEmail: "jane@example.com",
			expected: GitIdentity{
				Name:  "Jane Doe",
				Email: "jane@example.com",
			},
		},
		{
			name:         "only name present",
			mockUserName: "Jane Doe",
			mockUserEmail: "",
			expected: GitIdentity{
				Name:  "Jane Doe",
				Email: "",
			},
		},
		{
			name:         "only email present",
			mockUserName: "",
			mockUserEmail: "jane@example.com",
			expected: GitIdentity{
				Name:  "",
				Email: "jane@example.com",
			},
		},
		{
			name:         "neither present",
			mockUserName: "",
			mockUserEmail: "",
			expected: GitIdentity{
				Name:  "",
				Email: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test will be implemented when we create the actual function
		})
	}
}

func TestApplyGitConfigToContainer(t *testing.T) {
	tests := []struct {
		name          string
		containerName string
		containerUser string
		gitIdentity   GitIdentity
		mockPodman    func(*MockPodmanClient)
		wantErr       bool
		errContains   string
	}{
		{
			name:          "apply full identity",
			containerName: "dev-myproject",
			containerUser: "dev",
			gitIdentity: GitIdentity{
				Name:  "John Doe",
				Email: "john@example.com",
			},
			mockPodman: func(m *MockPodmanClient) {
				// Expect git config commands to be executed
				m.On("ExecContainer", mock.Anything, "dev-myproject",
					[]string{"su", "-", "dev", "-c", "git config --global user.name 'John Doe'"}).Return(nil)
				m.On("ExecContainer", mock.Anything, "dev-myproject",
					[]string{"su", "-", "dev", "-c", "git config --global user.email 'john@example.com'"}).Return(nil)
			},
			wantErr: false,
		},
		{
			name:          "apply only name",
			containerName: "dev-myproject",
			containerUser: "dev",
			gitIdentity: GitIdentity{
				Name:  "John Doe",
				Email: "",
			},
			mockPodman: func(m *MockPodmanClient) {
				// Only expect name to be set
				m.On("ExecContainer", mock.Anything, "dev-myproject",
					[]string{"su", "-", "dev", "-c", "git config --global user.name 'John Doe'"}).Return(nil)
			},
			wantErr: false,
		},
		{
			name:          "apply only email",
			containerName: "dev-myproject",
			containerUser: "dev",
			gitIdentity: GitIdentity{
				Name:  "",
				Email: "john@example.com",
			},
			mockPodman: func(m *MockPodmanClient) {
				// Only expect email to be set
				m.On("ExecContainer", mock.Anything, "dev-myproject",
					[]string{"su", "-", "dev", "-c", "git config --global user.email 'john@example.com'"}).Return(nil)
			},
			wantErr: false,
		},
		{
			name:          "empty identity does nothing",
			containerName: "dev-myproject",
			containerUser: "dev",
			gitIdentity: GitIdentity{
				Name:  "",
				Email: "",
			},
			mockPodman: func(m *MockPodmanClient) {
				// No commands should be executed
			},
			wantErr: false,
		},
		{
			name:          "handle special characters in name",
			containerName: "dev-myproject",
			containerUser: "dev",
			gitIdentity: GitIdentity{
				Name:  "John O'Doe",
				Email: "john@example.com",
			},
			mockPodman: func(m *MockPodmanClient) {
				// Expect proper escaping
				m.On("ExecContainer", mock.Anything, "dev-myproject",
					[]string{"su", "-", "dev", "-c", "git config --global user.name 'John O'\"'\"'Doe'"}).Return(nil)
				m.On("ExecContainer", mock.Anything, "dev-myproject",
					[]string{"su", "-", "dev", "-c", "git config --global user.email 'john@example.com'"}).Return(nil)
			},
			wantErr: false,
		},
		{
			name:          "container exec failure",
			containerName: "dev-myproject",
			containerUser: "dev",
			gitIdentity: GitIdentity{
				Name:  "John Doe",
				Email: "john@example.com",
			},
			mockPodman: func(m *MockPodmanClient) {
				// First command fails
				m.On("ExecContainer", mock.Anything, "dev-myproject",
					[]string{"su", "-", "dev", "-c", "git config --global user.name 'John Doe'"}).Return(assert.AnError)
			},
			wantErr:     true,
			errContains: "failed to set git user.name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock podman client
			mockClient := new(MockPodmanClient)
			if tt.mockPodman != nil {
				tt.mockPodman(mockClient)
			}

			// Test applying git config
			err := ApplyGitConfigToContainer(context.Background(), mockClient, tt.containerName,
				tt.containerUser, tt.gitIdentity)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}
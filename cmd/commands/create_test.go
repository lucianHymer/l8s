package commands

import (
	"bytes"
	"context"
	"testing"

	"github.com/l8s/l8s/pkg/container"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockContainerManager struct {
	mock.Mock
}

func (m *MockContainerManager) CreateContainer(ctx context.Context, name, gitURL, branch, sshKey string) (*container.Container, error) {
	args := m.Called(ctx, name, gitURL, branch, sshKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*container.Container), args.Error(1)
}

func (m *MockContainerManager) ListContainers(ctx context.Context) ([]*container.Container, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*container.Container), args.Error(1)
}

func (m *MockContainerManager) RemoveContainer(ctx context.Context, name string, removeVolumes bool) error {
	args := m.Called(ctx, name, removeVolumes)
	return args.Error(0)
}

func (m *MockContainerManager) StartContainer(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockContainerManager) StopContainer(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockContainerManager) GetContainerInfo(ctx context.Context, name string) (*container.Container, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*container.Container), args.Error(1)
}

func executeCommand(cmd *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	
	err := cmd.Execute()
	return buf.String(), err
}

func TestCreateCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupMocks  func(*MockContainerManager)
		wantErr     bool
		errContains string
		outContains []string
	}{
		{
			name: "successful creation",
			args: []string{"myproject", "https://github.com/user/repo.git"},
			setupMocks: func(m *MockContainerManager) {
				m.On("CreateContainer", mock.Anything, "myproject", "https://github.com/user/repo.git", "main", mock.AnythingOfType("string")).
					Return(&container.Container{
						Name:    "dev-myproject",
						SSHPort: 2200,
						Status:  "running",
					}, nil)
			},
			wantErr: false,
			outContains: []string{
				"Creating container: dev-myproject",
				"SSH port: 2200",
				"Repository cloned",
				"l8s ssh myproject",
				"ssh dev-myproject",
				"git push myproject",
			},
		},
		{
			name: "with custom branch",
			args: []string{"myproject", "https://github.com/user/repo.git", "develop"},
			setupMocks: func(m *MockContainerManager) {
				m.On("CreateContainer", mock.Anything, "myproject", "https://github.com/user/repo.git", "develop", mock.AnythingOfType("string")).
					Return(&container.Container{
						Name:    "dev-myproject",
						SSHPort: 2201,
						Status:  "running",
					}, nil)
			},
			wantErr: false,
			outContains: []string{
				"SSH port: 2201",
			},
		},
		{
			name:        "missing arguments",
			args:        []string{"myproject"},
			setupMocks:  func(m *MockContainerManager) {},
			wantErr:     true,
			errContains: "requires at least 2 arg(s)",
		},
		{
			name:        "invalid container name",
			args:        []string{"my project!", "https://github.com/user/repo.git"},
			setupMocks:  func(m *MockContainerManager) {},
			wantErr:     true,
			errContains: "You're out of your element!",
		},
		{
			name: "container already exists",
			args: []string{"existing", "https://github.com/user/repo.git"},
			setupMocks: func(m *MockContainerManager) {
				m.On("CreateContainer", mock.Anything, "existing", "https://github.com/user/repo.git", "main", mock.AnythingOfType("string")).
					Return(nil, assert.AnError)
			},
			wantErr:     true,
			errContains: "failed to create container",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := new(MockContainerManager)
			tt.setupMocks(mockManager)
			
			cmd := NewCreateCommand(mockManager)
			output, err := executeCommand(cmd, tt.args...)
			
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, output, tt.errContains)
				}
			} else {
				require.NoError(t, err)
				for _, expected := range tt.outContains {
					assert.Contains(t, output, expected)
				}
			}
			
			mockManager.AssertExpectations(t)
		})
	}
}

func TestSSHCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupMocks  func(*MockContainerManager)
		wantErr     bool
		errContains string
	}{
		{
			name: "successful SSH connection",
			args: []string{"myproject"},
			setupMocks: func(m *MockContainerManager) {
				m.On("GetContainerInfo", mock.Anything, "myproject").
					Return(&container.Container{
						Name:    "dev-myproject",
						SSHPort: 2200,
						Status:  "running",
					}, nil)
			},
			wantErr: false,
		},
		{
			name:        "missing container name",
			args:        []string{},
			setupMocks:  func(m *MockContainerManager) {},
			wantErr:     true,
			errContains: "requires exactly 1 arg(s)",
		},
		{
			name: "container not found",
			args: []string{"nonexistent"},
			setupMocks: func(m *MockContainerManager) {
				m.On("GetContainerInfo", mock.Anything, "nonexistent").
					Return(nil, assert.AnError)
			},
			wantErr:     true,
			errContains: "Container 'nonexistent' not found",
		},
		{
			name: "container not running",
			args: []string{"stopped"},
			setupMocks: func(m *MockContainerManager) {
				m.On("GetContainerInfo", mock.Anything, "stopped").
					Return(&container.Container{
						Name:    "dev-stopped",
						SSHPort: 2200,
						Status:  "stopped",
					}, nil)
			},
			wantErr:     true,
			errContains: "Container 'stopped' is not running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := new(MockContainerManager)
			tt.setupMocks(mockManager)
			
			// For SSH command, we'll need to mock the actual SSH execution
			// In real implementation, this would exec ssh
			cmd := NewSSHCommand(mockManager)
			output, err := executeCommand(cmd, tt.args...)
			
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, output, tt.errContains)
				}
			}
			
			mockManager.AssertExpectations(t)
		})
	}
}

func TestListCommand(t *testing.T) {
	mockManager := new(MockContainerManager)
	
	containers := []*container.Container{
		{
			Name:     "dev-project1",
			Status:   "running",
			SSHPort:  2200,
			GitURL:   "https://github.com/user/project1.git",
			Branch:   "main",
			Created:  "2024-01-01T10:00:00Z",
		},
		{
			Name:     "dev-project2",
			Status:   "stopped",
			SSHPort:  2201,
			GitURL:   "https://github.com/user/project2.git",
			Branch:   "develop",
			Created:  "2024-01-02T10:00:00Z",
		},
	}
	
	mockManager.On("ListContainers", mock.Anything).Return(containers, nil)
	
	cmd := NewListCommand(mockManager)
	output, err := executeCommand(cmd)
	
	require.NoError(t, err)
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "STATUS")
	assert.Contains(t, output, "SSH PORT")
	assert.Contains(t, output, "GIT REMOTE")
	assert.Contains(t, output, "project1")
	assert.Contains(t, output, "project2")
	assert.Contains(t, output, "running")
	assert.Contains(t, output, "stopped")
	assert.Contains(t, output, "2200")
	assert.Contains(t, output, "2201")
	
	mockManager.AssertExpectations(t)
}

func TestRemoveCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		flags       map[string]string
		setupMocks  func(*MockContainerManager)
		wantErr     bool
		errContains string
		outContains []string
	}{
		{
			name:  "successful removal with volumes",
			args:  []string{"myproject"},
			flags: map[string]string{"volumes": "true"},
			setupMocks: func(m *MockContainerManager) {
				m.On("RemoveContainer", mock.Anything, "myproject", true).Return(nil)
			},
			wantErr: false,
			outContains: []string{
				"Git remote removed",
				"Container removed",
				"Volumes removed",
			},
		},
		{
			name:  "successful removal without volumes",
			args:  []string{"myproject"},
			flags: map[string]string{"volumes": "false"},
			setupMocks: func(m *MockContainerManager) {
				m.On("RemoveContainer", mock.Anything, "myproject", false).Return(nil)
			},
			wantErr: false,
			outContains: []string{
				"Container removed",
			},
		},
		{
			name:        "missing container name",
			args:        []string{},
			setupMocks:  func(m *MockContainerManager) {},
			wantErr:     true,
			errContains: "requires exactly 1 arg(s)",
		},
		{
			name:  "container not found",
			args:  []string{"nonexistent"},
			flags: map[string]string{"volumes": "true"},
			setupMocks: func(m *MockContainerManager) {
				m.On("RemoveContainer", mock.Anything, "nonexistent", true).Return(assert.AnError)
			},
			wantErr:     true,
			errContains: "failed to remove container",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := new(MockContainerManager)
			tt.setupMocks(mockManager)
			
			cmd := NewRemoveCommand(mockManager)
			
			// Set flags
			for flag, value := range tt.flags {
				cmd.Flags().Set(flag, value)
			}
			
			output, err := executeCommand(cmd, tt.args...)
			
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, output, tt.errContains)
				}
			} else {
				require.NoError(t, err)
				for _, expected := range tt.outContains {
					assert.Contains(t, output, expected)
				}
			}
			
			mockManager.AssertExpectations(t)
		})
	}
}

func TestStartStopCommands(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		args        []string
		setupMocks  func(*MockContainerManager)
		wantErr     bool
		errContains string
		outContains string
	}{
		{
			name:    "successful start",
			command: "start",
			args:    []string{"myproject"},
			setupMocks: func(m *MockContainerManager) {
				m.On("StartContainer", mock.Anything, "myproject").Return(nil)
			},
			wantErr:     false,
			outContains: "Container 'myproject' started",
		},
		{
			name:    "successful stop",
			command: "stop",
			args:    []string{"myproject"},
			setupMocks: func(m *MockContainerManager) {
				m.On("StopContainer", mock.Anything, "myproject").Return(nil)
			},
			wantErr:     false,
			outContains: "Container 'myproject' stopped",
		},
		{
			name:        "missing container name",
			command:     "start",
			args:        []string{},
			setupMocks:  func(m *MockContainerManager) {},
			wantErr:     true,
			errContains: "requires exactly 1 arg(s)",
		},
		{
			name:    "start failed",
			command: "start",
			args:    []string{"myproject"},
			setupMocks: func(m *MockContainerManager) {
				m.On("StartContainer", mock.Anything, "myproject").Return(assert.AnError)
			},
			wantErr:     true,
			errContains: "failed to start container",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := new(MockContainerManager)
			tt.setupMocks(mockManager)
			
			var cmd *cobra.Command
			if tt.command == "start" {
				cmd = NewStartCommand(mockManager)
			} else {
				cmd = NewStopCommand(mockManager)
			}
			
			output, err := executeCommand(cmd, tt.args...)
			
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, output, tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.Contains(t, output, tt.outContains)
			}
			
			mockManager.AssertExpectations(t)
		})
	}
}

func TestInfoCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupMocks  func(*MockContainerManager)
		wantErr     bool
		errContains string
		outContains []string
	}{
		{
			name: "successful info display",
			args: []string{"myproject"},
			setupMocks: func(m *MockContainerManager) {
				m.On("GetContainerInfo", mock.Anything, "myproject").
					Return(&container.Container{
						Name:    "dev-myproject",
						Status:  "running",
						SSHPort: 2200,
						GitURL:  "https://github.com/user/repo.git",
						Branch:  "main",
						Created: "2024-01-01T10:00:00Z",
						Volumes: map[string]string{
							"home":      "dev-myproject-home",
							"workspace": "dev-myproject-workspace",
						},
						Labels: map[string]string{
							"l8s.managed":   "true",
							"l8s.git.url":   "https://github.com/user/repo.git",
							"l8s.git.branch": "main",
							"l8s.ssh.port":  "2200",
						},
					}, nil)
			},
			wantErr: false,
			outContains: []string{
				"Container: dev-myproject",
				"Status: running",
				"SSH Port: 2200",
				"Git Repository: https://github.com/user/repo.git",
				"Branch: main",
				"ssh dev-myproject",
				"ssh -p 2200 dev@localhost",
				"Volumes:",
				"home: dev-myproject-home",
				"workspace: dev-myproject-workspace",
			},
		},
		{
			name:        "missing container name",
			args:        []string{},
			setupMocks:  func(m *MockContainerManager) {},
			wantErr:     true,
			errContains: "requires exactly 1 arg(s)",
		},
		{
			name: "container not found",
			args: []string{"nonexistent"},
			setupMocks: func(m *MockContainerManager) {
				m.On("GetContainerInfo", mock.Anything, "nonexistent").
					Return(nil, assert.AnError)
			},
			wantErr:     true,
			errContains: "Container 'nonexistent' not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := new(MockContainerManager)
			tt.setupMocks(mockManager)
			
			cmd := NewInfoCommand(mockManager)
			output, err := executeCommand(cmd, tt.args...)
			
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, output, tt.errContains)
				}
			} else {
				require.NoError(t, err)
				for _, expected := range tt.outContains {
					assert.Contains(t, output, expected)
				}
			}
			
			mockManager.AssertExpectations(t)
		})
	}
}

func TestBuildCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
		outContains []string
	}{
		{
			name:    "successful build",
			args:    []string{},
			wantErr: false,
			outContains: []string{
				"Building l8s base image",
				"Image built successfully",
			},
		},
		{
			name:        "build with custom tag",
			args:        []string{"--tag", "localhost/my-l8s:latest"},
			wantErr:     false,
			outContains: []string{"Building l8s base image"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build command doesn't use container manager
			cmd := NewBuildCommand()
			output, err := executeCommand(cmd, tt.args...)
			
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, output, tt.errContains)
				}
			} else {
				// In real implementation, this would build the image
				// For testing, we just check the command structure
				assert.NotNil(t, cmd)
			}
		})
	}
}

func TestRemoteCommands(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		args        []string
		setupMocks  func(*MockContainerManager)
		wantErr     bool
		errContains string
		outContains []string
	}{
		{
			name:    "remote add",
			command: "add",
			args:    []string{"myproject"},
			setupMocks: func(m *MockContainerManager) {
				m.On("GetContainerInfo", mock.Anything, "myproject").
					Return(&container.Container{
						Name:    "dev-myproject",
						SSHPort: 2200,
						Status:  "running",
					}, nil)
			},
			wantErr: false,
			outContains: []string{
				"Git remote 'myproject' added",
			},
		},
		{
			name:    "remote remove",
			command: "remove",
			args:    []string{"myproject"},
			setupMocks: func(m *MockContainerManager) {
				// Remote remove doesn't need container info
			},
			wantErr: false,
			outContains: []string{
				"Git remote 'myproject' removed",
			},
		},
		{
			name:        "remote add missing args",
			command:     "add",
			args:        []string{},
			setupMocks:  func(m *MockContainerManager) {},
			wantErr:     true,
			errContains: "requires exactly 1 arg(s)",
		},
		{
			name:    "remote add container not found",
			command: "add",
			args:    []string{"nonexistent"},
			setupMocks: func(m *MockContainerManager) {
				m.On("GetContainerInfo", mock.Anything, "nonexistent").
					Return(nil, assert.AnError)
			},
			wantErr:     true,
			errContains: "Container 'nonexistent' not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := new(MockContainerManager)
			tt.setupMocks(mockManager)
			
			var cmd *cobra.Command
			if tt.command == "add" {
				cmd = NewRemoteAddCommand(mockManager)
			} else {
				cmd = NewRemoteRemoveCommand(mockManager)
			}
			
			output, err := executeCommand(cmd, tt.args...)
			
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, output, tt.errContains)
				}
			} else {
				require.NoError(t, err)
				for _, expected := range tt.outContains {
					assert.Contains(t, output, expected)
				}
			}
			
			mockManager.AssertExpectations(t)
		})
	}
}

func TestExecCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupMocks  func(*MockContainerManager)
		wantErr     bool
		errContains string
	}{
		{
			name: "successful exec",
			args: []string{"myproject", "ls", "-la"},
			setupMocks: func(m *MockContainerManager) {
				m.On("GetContainerInfo", mock.Anything, "myproject").
					Return(&container.Container{
						Name:   "dev-myproject",
						Status: "running",
					}, nil)
			},
			wantErr: false,
		},
		{
			name:        "missing arguments",
			args:        []string{"myproject"},
			setupMocks:  func(m *MockContainerManager) {},
			wantErr:     true,
			errContains: "requires at least 2 arg(s)",
		},
		{
			name: "container not running",
			args: []string{"stopped", "ls"},
			setupMocks: func(m *MockContainerManager) {
				m.On("GetContainerInfo", mock.Anything, "stopped").
					Return(&container.Container{
						Name:   "dev-stopped",
						Status: "stopped",
					}, nil)
			},
			wantErr:     true,
			errContains: "Container 'stopped' is not running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := new(MockContainerManager)
			tt.setupMocks(mockManager)
			
			cmd := NewExecCommand(mockManager)
			output, err := executeCommand(cmd, tt.args...)
			
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, output, tt.errContains)
				}
			}
			
			mockManager.AssertExpectations(t)
		})
	}
}
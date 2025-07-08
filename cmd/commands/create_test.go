package commands_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/l8s/l8s/pkg/cli"
	"github.com/l8s/l8s/pkg/config"
	"github.com/l8s/l8s/pkg/container"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockContainerManager is a mock implementation of the ContainerManager interface
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

func (m *MockContainerManager) ExecContainer(ctx context.Context, name string, cmd []string) error {
	args := m.Called(ctx, name, cmd)
	return args.Error(0)
}

func (m *MockContainerManager) SSHIntoContainer(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockContainerManager) BuildImage(ctx context.Context, containerfile string) error {
	args := m.Called(ctx, containerfile)
	return args.Error(0)
}

// MockGitClient is a mock implementation of the GitClient interface
type MockGitClient struct {
	mock.Mock
}

func (m *MockGitClient) CloneRepository(repoPath, gitURL, branch string) error {
	args := m.Called(repoPath, gitURL, branch)
	return args.Error(0)
}

func (m *MockGitClient) AddRemote(repoPath, remoteName, remoteURL string) error {
	args := m.Called(repoPath, remoteName, remoteURL)
	return args.Error(0)
}

func (m *MockGitClient) RemoveRemote(repoPath, remoteName string) error {
	args := m.Called(repoPath, remoteName)
	return args.Error(0)
}

func (m *MockGitClient) ListRemotes(repoPath string) (map[string]string, error) {
	args := m.Called(repoPath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockGitClient) SetUpstream(repoPath, remoteName, branch string) error {
	args := m.Called(repoPath, remoteName, branch)
	return args.Error(0)
}

func (m *MockGitClient) CurrentBranch(repoPath string) (string, error) {
	args := m.Called(repoPath)
	return args.String(0), args.Error(1)
}

func (m *MockGitClient) ValidateGitURL(gitURL string) error {
	args := m.Called(gitURL)
	return args.Error(0)
}

// MockSSHClient is a mock implementation of the SSHClient interface
type MockSSHClient struct {
	mock.Mock
}

func (m *MockSSHClient) ReadPublicKey(keyPath string) (string, error) {
	args := m.Called(keyPath)
	return args.String(0), args.Error(1)
}

func (m *MockSSHClient) FindSSHPublicKey() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockSSHClient) AddSSHConfig(name, hostname string, port int, user string) error {
	args := m.Called(name, hostname, port, user)
	return args.Error(0)
}

func (m *MockSSHClient) RemoveSSHConfig(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockSSHClient) GenerateAuthorizedKeys(publicKey string) string {
	args := m.Called(publicKey)
	return args.String(0)
}

func (m *MockSSHClient) IsPortAvailable(port int) bool {
	args := m.Called(port)
	return args.Bool(0)
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
		setupMocks  func(*MockContainerManager, *MockSSHClient)
		wantErr     bool
		errContains string
		outContains []string
	}{
		{
			name: "successful creation",
			args: []string{"myproject", "https://github.com/user/repo.git"},
			setupMocks: func(m *MockContainerManager, s *MockSSHClient) {
				s.On("FindSSHPublicKey").Return("ssh-rsa AAAAB3NzaC1yc2E...", nil)
				m.On("CreateContainer", mock.Anything, "myproject", "https://github.com/user/repo.git", "main", "ssh-rsa AAAAB3NzaC1yc2E...").
					Return(&container.Container{
						Name:      "dev-myproject",
						SSHPort:   2200,
						Status:    "running",
						CreatedAt: time.Now(),
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
			setupMocks: func(m *MockContainerManager, s *MockSSHClient) {
				s.On("FindSSHPublicKey").Return("ssh-rsa AAAAB3NzaC1yc2E...", nil)
				m.On("CreateContainer", mock.Anything, "myproject", "https://github.com/user/repo.git", "develop", "ssh-rsa AAAAB3NzaC1yc2E...").
					Return(&container.Container{
						Name:      "dev-myproject",
						SSHPort:   2201,
						Status:    "running",
						CreatedAt: time.Now(),
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
			setupMocks:  func(m *MockContainerManager, s *MockSSHClient) {},
			wantErr:     true,
			errContains: "accepts between 2 and 3 arg(s)",
		},
		{
			name: "no ssh key found",
			args: []string{"myproject", "https://github.com/user/repo.git"},
			setupMocks: func(m *MockContainerManager, s *MockSSHClient) {
				s.On("FindSSHPublicKey").Return("", fmt.Errorf("no key found"))
			},
			wantErr:     true,
			errContains: "no SSH public key found",
		},
		{
			name: "container already exists",
			args: []string{"existing", "https://github.com/user/repo.git"},
			setupMocks: func(m *MockContainerManager, s *MockSSHClient) {
				s.On("FindSSHPublicKey").Return("ssh-rsa AAAAB3NzaC1yc2E...", nil)
				m.On("CreateContainer", mock.Anything, "existing", "https://github.com/user/repo.git", "main", "ssh-rsa AAAAB3NzaC1yc2E...").
					Return(nil, fmt.Errorf("container 'existing' already exists"))
			},
			wantErr:     true,
			errContains: "already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := new(MockContainerManager)
			mockSSH := new(MockSSHClient)
			mockGit := new(MockGitClient)
			tt.setupMocks(mockManager, mockSSH)
			
			cfg := config.DefaultConfig()
			factory := cli.NewTestCommandFactory(cfg, mockManager, mockGit, mockSSH)
			cmd := factory.CreateCmd()
			output, err := executeCommand(cmd, tt.args...)
			
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				for _, expected := range tt.outContains {
					assert.Contains(t, output, expected)
				}
			}
			
			mockManager.AssertExpectations(t)
			mockSSH.AssertExpectations(t)
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
				m.On("SSHIntoContainer", mock.Anything, "myproject").Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "missing container name",
			args:        []string{},
			setupMocks:  func(m *MockContainerManager) {},
			wantErr:     true,
			errContains: "accepts 1 arg(s)",
		},
		{
			name: "container not found",
			args: []string{"nonexistent"},
			setupMocks: func(m *MockContainerManager) {
				m.On("SSHIntoContainer", mock.Anything, "nonexistent").
					Return(fmt.Errorf("container 'nonexistent' not found"))
			},
			wantErr:     true,
			errContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := new(MockContainerManager)
			mockSSH := new(MockSSHClient)
			mockGit := new(MockGitClient)
			tt.setupMocks(mockManager)
			
			cfg := config.DefaultConfig()
			factory := cli.NewTestCommandFactory(cfg, mockManager, mockGit, mockSSH)
			cmd := factory.SSHCmd()
			_, err := executeCommand(cmd, tt.args...)
			
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
			
			mockManager.AssertExpectations(t)
		})
	}
}

func TestListCommand(t *testing.T) {
	mockManager := new(MockContainerManager)
	mockSSH := new(MockSSHClient)
	mockGit := new(MockGitClient)
	
	containers := []*container.Container{
		{
			Name:      "dev-project1",
			Status:    "running",
			SSHPort:   2200,
			GitURL:    "https://github.com/user/project1.git",
			GitBranch: "main",
			CreatedAt: time.Now().Add(-2 * time.Hour),
		},
		{
			Name:      "dev-project2",
			Status:    "stopped",
			SSHPort:   2201,
			GitURL:    "https://github.com/user/project2.git",
			GitBranch: "develop",
			CreatedAt: time.Now().Add(-72 * time.Hour),
		},
	}
	
	mockManager.On("ListContainers", mock.Anything).Return(containers, nil)
	
	cfg := config.DefaultConfig()
	factory := cli.NewTestCommandFactory(cfg, mockManager, mockGit, mockSSH)
	cmd := factory.ListCmd()
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
			errContains: "accepts 1 arg(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := new(MockContainerManager)
			mockSSH := new(MockSSHClient)
			mockGit := new(MockGitClient)
			tt.setupMocks(mockManager)
			
			cfg := config.DefaultConfig()
			factory := cli.NewTestCommandFactory(cfg, mockManager, mockGit, mockSSH)
			
			var cmd *cobra.Command
			if tt.command == "start" {
				cmd = factory.StartCmd()
			} else {
				cmd = factory.StopCmd()
			}
			
			output, err := executeCommand(cmd, tt.args...)
			
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				assert.Contains(t, output, tt.outContains)
			}
			
			mockManager.AssertExpectations(t)
		})
	}
}
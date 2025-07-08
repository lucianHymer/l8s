package cli

import (
	"context"
	"errors"
	"testing"

	"l8s/pkg/config"
	"l8s/pkg/container"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Enhanced mock for testing create command
type MockContainerManagerWithGit struct {
	mock.Mock
}

func (m *MockContainerManagerWithGit) CreateContainer(ctx context.Context, name, gitURL, branch, sshKey string) (*container.Container, error) {
	args := m.Called(ctx, name, gitURL, branch, sshKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*container.Container), args.Error(1)
}

func (m *MockContainerManagerWithGit) StartContainer(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockContainerManagerWithGit) StopContainer(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockContainerManagerWithGit) RemoveContainer(ctx context.Context, name string, removeVolumes bool) error {
	args := m.Called(ctx, name, removeVolumes)
	return args.Error(0)
}

func (m *MockContainerManagerWithGit) ListContainers(ctx context.Context) ([]*container.Container, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*container.Container), args.Error(1)
}

func (m *MockContainerManagerWithGit) GetContainerInfo(ctx context.Context, name string) (*container.Container, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*container.Container), args.Error(1)
}

func (m *MockContainerManagerWithGit) BuildImage(ctx context.Context, containerfile string) error {
	args := m.Called(ctx, containerfile)
	return args.Error(0)
}

func (m *MockContainerManagerWithGit) ExecContainer(ctx context.Context, name string, command []string) error {
	args := m.Called(ctx, name, command)
	return args.Error(0)
}

func (m *MockContainerManagerWithGit) SSHIntoContainer(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

// Enhanced mock for git operations
type MockGitClientEnhanced struct {
	mock.Mock
}

func (m *MockGitClientEnhanced) CloneRepository(repoPath, gitURL, branch string) error {
	args := m.Called(repoPath, gitURL, branch)
	return args.Error(0)
}

func (m *MockGitClientEnhanced) AddRemote(repoPath, remoteName, remoteURL string) error {
	args := m.Called(repoPath, remoteName, remoteURL)
	return args.Error(0)
}

func (m *MockGitClientEnhanced) RemoveRemote(repoPath, remoteName string) error {
	args := m.Called(repoPath, remoteName)
	return args.Error(0)
}

func (m *MockGitClientEnhanced) ListRemotes(repoPath string) (map[string]string, error) {
	args := m.Called(repoPath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockGitClientEnhanced) SetUpstream(repoPath, remoteName, branch string) error {
	args := m.Called(repoPath, remoteName, branch)
	return args.Error(0)
}

func (m *MockGitClientEnhanced) GetCurrentBranch(repoPath string) (string, error) {
	args := m.Called(repoPath)
	return args.String(0), args.Error(1)
}

func (m *MockGitClientEnhanced) ValidateGitURL(gitURL string) error {
	args := m.Called(gitURL)
	return args.Error(0)
}

func (m *MockGitClientEnhanced) IsGitRepository(path string) bool {
	args := m.Called(path)
	return args.Bool(0)
}

func (m *MockGitClientEnhanced) PushBranch(repoPath, branch, remoteName string, force bool) error {
	args := m.Called(repoPath, branch, remoteName, force)
	return args.Error(0)
}

func (m *MockGitClientEnhanced) InitRepository(repoPath string, allowPush bool, defaultBranch string) error {
	args := m.Called(repoPath, allowPush, defaultBranch)
	return args.Error(0)
}

func TestCreateCommandNewFlow(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		branch          string
		isGitRepo       bool
		currentBranch   string
		setupMocks      func(*LazyCommandFactory, *MockContainerManagerWithGit, *MockGitClientEnhanced)
		wantErr         bool
		errContains     string
	}{
		{
			name:          "create from git repository without branch flag",
			args:          []string{"mycontainer"},
			isGitRepo:     true,
			currentBranch: "main",
			setupMocks: func(f *LazyCommandFactory, cm *MockContainerManagerWithGit, gc *MockGitClientEnhanced) {
				// Check if current directory is a git repo
				gc.On("IsGitRepository", ".").Return(true)
				gc.On("GetCurrentBranch", ".").Return("main", nil)
				
				// Create container with empty git URL
				cm.On("CreateContainer", mock.Anything, "mycontainer", "test-key").
					Return(&container.Container{
						Name:    "mycontainer",
						SSHPort: 2222,
					}, nil)
				
				// Add git remote
				gc.On("AddRemote", ".", "mycontainer", "dev-mycontainer:/workspace/project").Return(nil)
				
				// Push initial code
				gc.On("PushBranch", ".", "main", "mycontainer", false).Return(nil)
			},
			wantErr: false,
		},
		{
			name:          "create from git repository with branch flag",
			args:          []string{"mycontainer"},
			branch:        "feature",
			isGitRepo:     true,
			currentBranch: "main",
			setupMocks: func(f *LazyCommandFactory, cm *MockContainerManagerWithGit, gc *MockGitClientEnhanced) {
				// Check if current directory is a git repo
				gc.On("IsGitRepository", ".").Return(true)
				// Don't need to get current branch when branch is specified
				
				// Create container
				cm.On("CreateContainer", mock.Anything, "mycontainer", "test-key").
					Return(&container.Container{
						Name:    "mycontainer",
						SSHPort: 2222,
					}, nil)
				
				// Add git remote
				gc.On("AddRemote", ".", "mycontainer", "dev-mycontainer:/workspace/project").Return(nil)
				
				// Push specified branch
				gc.On("PushBranch", ".", "feature", "mycontainer", false).Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "create from non-git directory",
			args:        []string{"mycontainer"},
			isGitRepo:   false,
			setupMocks: func(f *LazyCommandFactory, cm *MockContainerManagerWithGit, gc *MockGitClientEnhanced) {
				// Check if current directory is a git repo
				gc.On("IsGitRepository", ".").Return(false)
			},
			wantErr:     true,
			errContains: "must be run from within a git repository",
		},
		{
			name:          "create with git remote add failure",
			args:          []string{"mycontainer"},
			isGitRepo:     true,
			currentBranch: "main",
			setupMocks: func(f *LazyCommandFactory, cm *MockContainerManagerWithGit, gc *MockGitClientEnhanced) {
				gc.On("IsGitRepository", ".").Return(true)
				gc.On("GetCurrentBranch", ".").Return("main", nil)
				
				cm.On("CreateContainer", mock.Anything, "mycontainer", "test-key").
					Return(&container.Container{
						Name:    "mycontainer",
						SSHPort: 2222,
					}, nil)
				
				// Fail to add git remote
				gc.On("AddRemote", ".", "mycontainer", "dev-mycontainer:/workspace/project").
					Return(errors.New("remote already exists"))
			},
			wantErr:     true,
			errContains: "remote already exists",
		},
		{
			name:          "create with push failure",
			args:          []string{"mycontainer"},
			isGitRepo:     true,
			currentBranch: "main",
			setupMocks: func(f *LazyCommandFactory, cm *MockContainerManagerWithGit, gc *MockGitClientEnhanced) {
				gc.On("IsGitRepository", ".").Return(true)
				gc.On("GetCurrentBranch", ".").Return("main", nil)
				
				cm.On("CreateContainer", mock.Anything, "mycontainer", "test-key").
					Return(&container.Container{
						Name:    "mycontainer",
						SSHPort: 2222,
					}, nil)
				
				gc.On("AddRemote", ".", "mycontainer", "dev-mycontainer:/workspace/project").Return(nil)
				
				// Push fails
				gc.On("PushBranch", ".", "main", "mycontainer", false).
					Return(errors.New("failed to push"))
			},
			wantErr:     true,
			errContains: "failed to push",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up mocks
			containerMgr := new(MockContainerManagerWithGit)
			gitClient := new(MockGitClientEnhanced)
			
			factory := &LazyCommandFactory{
				Config:       &config.Config{SSHPublicKey: "test-key"},
				ContainerMgr: containerMgr,
				GitClient:    gitClient,
				SSHClient:    &MockSSHClient{},
				initialized:  true, // Mark as already initialized
			}
			
			// Set up test-specific mocks
			tt.setupMocks(factory, containerMgr, gitClient)
			
			// Create command and set branch flag if needed
			cmd := factory.CreateCmd()
			if tt.branch != "" {
				cmd.Flags().Set("branch", tt.branch)
			}
			
			// Execute command
			err := cmd.RunE(cmd, tt.args)
			
			// Check expectations
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
			
			// Verify all mock expectations were met
			containerMgr.AssertExpectations(t)
			gitClient.AssertExpectations(t)
		})
	}
}

func TestCreateCommandValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
	}{
		{
			name:        "no arguments",
			args:        []string{},
			wantErr:     true,
			errContains: "accepts 1 arg(s), received 0",
		},
		{
			name:        "too many arguments",
			args:        []string{"container1", "container2"},
			wantErr:     true,
			errContains: "accepts 1 arg(s), received 2",
		},
		{
			name:    "valid single argument",
			args:    []string{"mycontainer"},
			wantErr: false, // Will fail later in git check, but args validation passes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := &LazyCommandFactory{
				Config:       &config.Config{SSHPublicKey: "test-key"},
				ContainerMgr: &MockContainerManager{},
				GitClient:    &MockGitClientEnhanced{},
				SSHClient:    &MockSSHClient{},
				initialized:  true,
			}
			
			cmd := factory.CreateCmd()
			
			// First validate args
			err := cmd.Args(cmd, tt.args)
			
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
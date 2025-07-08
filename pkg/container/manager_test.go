package container

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockPodmanClient mocks the Podman client interface
type MockPodmanClient struct {
	mock.Mock
}

func (m *MockPodmanClient) ContainerExists(ctx context.Context, name string) (bool, error) {
	args := m.Called(ctx, name)
	return args.Bool(0), args.Error(1)
}

func (m *MockPodmanClient) CreateContainer(ctx context.Context, config ContainerConfig) (*Container, error) {
	args := m.Called(ctx, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Container), args.Error(1)
}

func (m *MockPodmanClient) StartContainer(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockPodmanClient) StopContainer(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockPodmanClient) RemoveContainer(ctx context.Context, name string, removeVolumes bool) error {
	args := m.Called(ctx, name, removeVolumes)
	return args.Error(0)
}

func (m *MockPodmanClient) ListContainers(ctx context.Context) ([]*Container, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Container), args.Error(1)
}

func (m *MockPodmanClient) GetContainerInfo(ctx context.Context, name string) (*Container, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Container), args.Error(1)
}

func (m *MockPodmanClient) FindAvailablePort(startPort int) (int, error) {
	args := m.Called(startPort)
	return args.Int(0), args.Error(1)
}

func (m *MockPodmanClient) ExecContainer(ctx context.Context, name string, cmd []string) error {
	args := m.Called(ctx, name, cmd)
	return args.Error(0)
}

func (m *MockPodmanClient) CopyToContainer(ctx context.Context, name string, src, dst string) error {
	args := m.Called(ctx, name, src, dst)
	return args.Error(0)
}

func TestManager_CreateContainer(t *testing.T) {
	tests := []struct {
		name        string
		containerName string
		gitURL      string
		branch      string
		sshKey      string
		setupMocks  func(*MockPodmanClient)
		wantErr     bool
		errContains string
	}{
		{
			name:          "successful container creation",
			containerName: "myproject",
			gitURL:        "https://github.com/user/repo.git",
			branch:        "main",
			sshKey:        "ssh-ed25519 AAAAC3... user@example.com",
			setupMocks: func(m *MockPodmanClient) {
				m.On("ContainerExists", mock.Anything, "dev-myproject").Return(false, nil)
				m.On("FindAvailablePort", 2200).Return(2200, nil)
				m.On("CreateContainer", mock.Anything, mock.MatchedBy(func(config ContainerConfig) bool {
					// Verify container config including labels
					return config.Name == "dev-myproject" &&
						config.Hostname == "myproject" &&
						config.SSHPort == 2200 &&
						config.Image == "localhost/l8s-fedora:latest" &&
						// Verify labels for metadata tracking
						config.Labels["l8s.managed"] == "true" &&
						config.Labels["l8s.git.url"] == "https://github.com/user/repo.git" &&
						config.Labels["l8s.git.branch"] == "main" &&
						config.Labels["l8s.ssh.port"] == "2200"
				})).Return(&Container{
					Name:     "dev-myproject",
					Status:   "created",
					SSHPort:  2200,
					GitURL:   "https://github.com/user/repo.git",
					Branch:   "main",
					Labels: map[string]string{
						"l8s.managed":   "true",
						"l8s.git.url":   "https://github.com/user/repo.git",
						"l8s.git.branch": "main",
						"l8s.ssh.port":  "2200",
					},
				}, nil)
				m.On("StartContainer", mock.Anything, "dev-myproject").Return(nil)
			},
			wantErr: false,
		},
		{
			name:          "container already exists",
			containerName: "existing",
			gitURL:        "https://github.com/user/repo.git",
			branch:        "main",
			sshKey:        "ssh-ed25519 AAAAC3... user@example.com",
			setupMocks: func(m *MockPodmanClient) {
				m.On("ContainerExists", mock.Anything, "dev-existing").Return(true, nil)
			},
			wantErr:     true,
			errContains: "already exists",
		},
		{
			name:          "invalid container name",
			containerName: "invalid name!",
			gitURL:        "https://github.com/user/repo.git",
			branch:        "main",
			sshKey:        "ssh-ed25519 AAAAC3... user@example.com",
			setupMocks:    func(m *MockPodmanClient) {},
			wantErr:       true,
			errContains:   "invalid container name",
		},
		{
			name:          "empty git URL",
			containerName: "myproject",
			gitURL:        "",
			branch:        "main",
			sshKey:        "ssh-ed25519 AAAAC3... user@example.com",
			setupMocks:    func(m *MockPodmanClient) {},
			wantErr:       true,
			errContains:   "git URL is required",
		},
		{
			name:          "no available ports",
			containerName: "myproject",
			gitURL:        "https://github.com/user/repo.git",
			branch:        "main",
			sshKey:        "ssh-ed25519 AAAAC3... user@example.com",
			setupMocks: func(m *MockPodmanClient) {
				m.On("ContainerExists", mock.Anything, "dev-myproject").Return(false, nil)
				m.On("FindAvailablePort", 2200).Return(0, assert.AnError)
			},
			wantErr:     true,
			errContains: "no available SSH port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockPodmanClient)
			tt.setupMocks(mockClient)

			manager := NewManager(mockClient, Config{
				BaseImage:       "localhost/l8s-fedora:latest",
				ContainerPrefix: "dev",
				SSHPortStart:    2200,
				ContainerUser:   "dev",
			})

			container, err := manager.CreateContainer(context.Background(), tt.containerName, tt.gitURL, tt.branch, tt.sshKey)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, container)
			} else {
				require.NoError(t, err)
				require.NotNil(t, container)
				assert.Equal(t, "dev-myproject", container.Name)
				assert.Equal(t, 2200, container.SSHPort)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestManager_ListContainers(t *testing.T) {
	mockClient := new(MockPodmanClient)
	
	expectedContainers := []*Container{
		{
			Name:     "dev-project1",
			Status:   "running",
			SSHPort:  2200,
			GitURL:   "https://github.com/user/project1.git",
			Branch:   "main",
			Created:  "2024-01-01T10:00:00Z",
			Labels: map[string]string{
				"l8s.managed":   "true",
				"l8s.git.url":   "https://github.com/user/project1.git",
				"l8s.git.branch": "main",
				"l8s.ssh.port":  "2200",
			},
		},
		{
			Name:     "dev-project2",
			Status:   "stopped",
			SSHPort:  2201,
			GitURL:   "https://github.com/user/project2.git",
			Branch:   "develop",
			Created:  "2024-01-02T10:00:00Z",
			Labels: map[string]string{
				"l8s.managed":   "true",
				"l8s.git.url":   "https://github.com/user/project2.git",
				"l8s.git.branch": "develop",
				"l8s.ssh.port":  "2201",
			},
		},
	}
	
	mockClient.On("ListContainers", mock.Anything).Return(expectedContainers, nil)
	
	manager := NewManager(mockClient, Config{
		ContainerPrefix: "dev",
	})
	
	containers, err := manager.ListContainers(context.Background())
	
	require.NoError(t, err)
	assert.Len(t, containers, 2)
	assert.Equal(t, expectedContainers, containers)
	
	mockClient.AssertExpectations(t)
}

func TestManager_RemoveContainer(t *testing.T) {
	tests := []struct {
		name          string
		containerName string
		removeVolumes bool
		setupMocks    func(*MockPodmanClient)
		wantErr       bool
		errContains   string
	}{
		{
			name:          "successful removal with volumes",
			containerName: "myproject",
			removeVolumes: true,
			setupMocks: func(m *MockPodmanClient) {
				m.On("ContainerExists", mock.Anything, "dev-myproject").Return(true, nil)
				m.On("StopContainer", mock.Anything, "dev-myproject").Return(nil)
				m.On("RemoveContainer", mock.Anything, "dev-myproject", true).Return(nil)
			},
			wantErr: false,
		},
		{
			name:          "successful removal without volumes",
			containerName: "myproject",
			removeVolumes: false,
			setupMocks: func(m *MockPodmanClient) {
				m.On("ContainerExists", mock.Anything, "dev-myproject").Return(true, nil)
				m.On("StopContainer", mock.Anything, "dev-myproject").Return(nil)
				m.On("RemoveContainer", mock.Anything, "dev-myproject", false).Return(nil)
			},
			wantErr: false,
		},
		{
			name:          "container not found",
			containerName: "nonexistent",
			removeVolumes: true,
			setupMocks: func(m *MockPodmanClient) {
				m.On("ContainerExists", mock.Anything, "dev-nonexistent").Return(false, nil)
			},
			wantErr:     true,
			errContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockPodmanClient)
			tt.setupMocks(mockClient)

			manager := NewManager(mockClient, Config{
				ContainerPrefix: "dev",
			})

			err := manager.RemoveContainer(context.Background(), tt.containerName, tt.removeVolumes)

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

func TestManager_GetContainerInfo(t *testing.T) {
	mockClient := new(MockPodmanClient)
	
	expectedContainer := &Container{
		Name:     "dev-myproject",
		Status:   "running",
		SSHPort:  2200,
		GitURL:   "https://github.com/user/repo.git",
		Branch:   "main",
		Created:  "2024-01-01T10:00:00Z",
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
	}
	
	mockClient.On("GetContainerInfo", mock.Anything, "dev-myproject").Return(expectedContainer, nil)
	
	manager := NewManager(mockClient, Config{
		ContainerPrefix: "dev",
	})
	
	container, err := manager.GetContainerInfo(context.Background(), "myproject")
	
	require.NoError(t, err)
	assert.Equal(t, expectedContainer, container)
	
	mockClient.AssertExpectations(t)
}

func TestManager_StartStopContainer(t *testing.T) {
	mockClient := new(MockPodmanClient)
	
	// Test Start
	mockClient.On("ContainerExists", mock.Anything, "dev-myproject").Return(true, nil).Once()
	mockClient.On("StartContainer", mock.Anything, "dev-myproject").Return(nil)
	
	// Test Stop
	mockClient.On("ContainerExists", mock.Anything, "dev-myproject").Return(true, nil).Once()
	mockClient.On("StopContainer", mock.Anything, "dev-myproject").Return(nil)
	
	manager := NewManager(mockClient, Config{
		ContainerPrefix: "dev",
	})
	
	// Test Start
	err := manager.StartContainer(context.Background(), "myproject")
	require.NoError(t, err)
	
	// Test Stop
	err = manager.StopContainer(context.Background(), "myproject")
	require.NoError(t, err)
	
	mockClient.AssertExpectations(t)
}

func TestValidateContainerName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple name", "myproject", false},
		{"valid with dash", "my-project", false},
		{"valid with numbers", "project123", false},
		{"empty name", "", true},
		{"spaces in name", "my project", true},
		{"special characters", "my@project", true},
		{"starts with number", "123project", true},
		{"starts with dash", "-project", true},
		{"uppercase letters", "MyProject", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateContainerName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestManager_ListContainersWithLabels(t *testing.T) {
	mockClient := new(MockPodmanClient)
	
	// Simulate mix of l8s and non-l8s containers
	allContainers := []*Container{
		{
			Name:    "dev-project1",
			Status:  "running",
			SSHPort: 2200,
			Labels: map[string]string{
				"l8s.managed":   "true",
				"l8s.git.url":   "https://github.com/user/project1.git",
				"l8s.git.branch": "main",
				"l8s.ssh.port":  "2200",
			},
		},
		{
			Name:   "some-other-container",
			Status: "running",
			Labels: map[string]string{
				"other": "label",
			},
		},
		{
			Name:    "dev-project2",
			Status:  "stopped",
			SSHPort: 2201,
			Labels: map[string]string{
				"l8s.managed":   "true",
				"l8s.git.url":   "https://github.com/user/project2.git",
				"l8s.git.branch": "develop",
				"l8s.ssh.port":  "2201",
			},
		},
	}
	
	mockClient.On("ListContainers", mock.Anything).Return(allContainers, nil)
	
	manager := NewManager(mockClient, Config{
		ContainerPrefix: "dev",
	})
	
	containers, err := manager.ListContainers(context.Background())
	
	require.NoError(t, err)
	// Should filter to only l8s managed containers
	assert.Len(t, containers, 2)
	for _, c := range containers {
		assert.Equal(t, "true", c.Labels["l8s.managed"])
	}
	
	mockClient.AssertExpectations(t)
}
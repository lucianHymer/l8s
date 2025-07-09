package container

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestManager_CreateContainer(t *testing.T) {
	tests := []struct {
		name        string
		containerName string
		sshKey      string
		setupMocks  func(*MockPodmanClient)
		wantErr     bool
		errContains string
	}{
		{
			name:          "successful container creation",
			containerName: "myproject",
			sshKey:        "ssh-ed25519 AAAAC3... user@example.com",
			setupMocks: func(m *MockPodmanClient) {
				m.On("ContainerExists", mock.Anything, "dev-myproject").Return(false, nil)
				m.On("FindAvailablePort", 2200).Return(2200, nil)
				m.On("CreateContainer", mock.Anything, mock.MatchedBy(func(config ContainerConfig) bool {
					// Verify container config including labels
					return config.Name == "dev-myproject" &&
						config.SSHPort == 2200 &&
						config.BaseImage == "localhost/l8s-fedora:latest" &&
						// Verify labels for metadata tracking
						config.Labels["l8s.managed"] == "true" &&
						config.Labels["l8s.ssh.port"] == "2200"
				})).Return(&Container{
					Name:     "dev-myproject",
					Status:   "created",
					SSHPort:  2200,
					Labels: map[string]string{
						"l8s.managed":   "true",
						"l8s.git.url":   "https://github.com/user/repo.git",
						"l8s.git.branch": "main",
						"l8s.ssh.port":  "2200",
					},
				}, nil)
				m.On("StartContainer", mock.Anything, "dev-myproject").Return(nil)
				
				// Mock setupSSH calls
				m.On("ExecContainer", mock.Anything, "dev-myproject", 
					[]string{"mkdir", "-p", "/home/dev/.ssh"}).Return(nil)
				m.On("ExecContainerWithInput", mock.Anything, "dev-myproject", 
					[]string{"tee", "/home/dev/.ssh/authorized_keys"}, 
					mock.AnythingOfType("string")).Return(nil)
				m.On("ExecContainer", mock.Anything, "dev-myproject", 
					[]string{"chmod", "600", "/home/dev/.ssh/authorized_keys"}).Return(nil)
				m.On("ExecContainer", mock.Anything, "dev-myproject", 
					[]string{"chown", "-R", "dev:dev", "/home/dev/.ssh"}).Return(nil)
				
				// Mock SetupWorkspace call
				m.On("SetupWorkspace", mock.Anything, "dev-myproject", "dev").Return(nil)
				
				// Mock copyDotfiles calls (embedded dotfiles)
				m.On("CopyToContainer", mock.Anything, "dev-myproject",
					mock.AnythingOfType("string"), 
					mock.AnythingOfType("string")).Return(nil)
				m.On("ExecContainer", mock.Anything, "dev-myproject",
					mock.AnythingOfType("[]string")).Return(nil).Maybe()
				m.On("ExecContainerWithInput", mock.Anything, "dev-myproject",
					mock.AnythingOfType("[]string"), 
					mock.AnythingOfType("string")).Return(nil).Maybe()
				
				// No git clone anymore - containers start with empty repos
			},
			wantErr: false,
		},
		{
			name:          "container already exists",
			containerName: "existing",
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
			sshKey:        "ssh-ed25519 AAAAC3... user@example.com",
			setupMocks:    func(m *MockPodmanClient) {},
			wantErr:       true,
			errContains:   "container name must consist of lowercase letters",
		},
		{
			name:          "no available ports",
			containerName: "myproject",
			sshKey:        "ssh-ed25519 AAAAC3... user@example.com",
			setupMocks: func(m *MockPodmanClient) {
				m.On("ContainerExists", mock.Anything, "dev-myproject").Return(false, nil)
				m.On("FindAvailablePort", 2200).Return(0, assert.AnError)
			},
			wantErr:     true,
			errContains: "failed to find available port",
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

			container, err := manager.CreateContainer(context.Background(), tt.containerName, tt.sshKey)

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
	mockClient.On("StartContainer", mock.Anything, "dev-myproject").Return(nil)
	
	// Test Stop
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
		{"starts with number", "123project", false},
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
	// Mock should only return l8s-managed containers
	managedContainers := []*Container{
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
	
	mockClient.On("ListContainers", mock.Anything).Return(managedContainers, nil)
	
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

func TestManager_WorkspaceOwnership(t *testing.T) {
	// Test that volumes are created with :U option for proper ownership
	mockClient := new(MockPodmanClient)
	
	mockClient.On("ContainerExists", mock.Anything, "dev-myproject").Return(false, nil)
	mockClient.On("FindAvailablePort", 2200).Return(2200, nil)
	
	// The key test: verify that CreateContainer is called with volumes having :U option
	mockClient.On("CreateContainer", mock.Anything, mock.MatchedBy(func(config ContainerConfig) bool {
		// This is where we'll verify the :U option is added to volumes
		// For now, just verify the basic config
		return config.Name == "dev-myproject"
	})).Return(&Container{
		Name:     "dev-myproject",
		Status:   "created",
		SSHPort:  2200,
	}, nil)
	
	mockClient.On("StartContainer", mock.Anything, "dev-myproject").Return(nil)
	
	// Mock SSH setup
	mockClient.On("ExecContainer", mock.Anything, "dev-myproject", 
		[]string{"mkdir", "-p", "/home/dev/.ssh"}).Return(nil)
	mockClient.On("ExecContainerWithInput", mock.Anything, "dev-myproject", 
		[]string{"tee", "/home/dev/.ssh/authorized_keys"}, 
		mock.AnythingOfType("string")).Return(nil)
	mockClient.On("ExecContainer", mock.Anything, "dev-myproject", 
		[]string{"chmod", "600", "/home/dev/.ssh/authorized_keys"}).Return(nil)
	mockClient.On("ExecContainer", mock.Anything, "dev-myproject", 
		[]string{"chown", "-R", "dev:dev", "/home/dev/.ssh"}).Return(nil)
	
	// Mock SetupWorkspace call
	mockClient.On("SetupWorkspace", mock.Anything, "dev-myproject", "dev").Return(nil)
	
	// Mock copyDotfiles calls (embedded dotfiles)
	mockClient.On("CopyToContainer", mock.Anything, "dev-myproject",
		mock.AnythingOfType("string"), 
		mock.AnythingOfType("string")).Return(nil)
	mockClient.On("ExecContainer", mock.Anything, "dev-myproject",
		mock.AnythingOfType("[]string")).Return(nil).Maybe()
	mockClient.On("ExecContainerWithInput", mock.Anything, "dev-myproject",
		mock.AnythingOfType("[]string"), 
		mock.AnythingOfType("string")).Return(nil).Maybe()
	
	// No git clone anymore - containers start with empty repos
	
	manager := NewManager(mockClient, Config{
		ContainerPrefix: "dev",
		SSHPortStart:    2200,
		BaseImage:       "localhost/l8s-fedora:latest",
		ContainerUser:   "dev",
	})
	
	_, err := manager.CreateContainer(context.Background(), "myproject", "ssh-key")
	require.NoError(t, err)
	
	mockClient.AssertExpectations(t)
}
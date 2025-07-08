// +build test

package container

import (
	"context"
	"fmt"

	"github.com/l8s/l8s/pkg/ssh"
	"github.com/stretchr/testify/mock"
)

// MockPodmanClient is a mock implementation of PodmanClient for testing
type MockPodmanClient struct {
	mock.Mock
}

// ContainerExists mocks the ContainerExists method
func (m *MockPodmanClient) ContainerExists(ctx context.Context, name string) (bool, error) {
	args := m.Called(ctx, name)
	return args.Bool(0), args.Error(1)
}

// CreateContainer mocks the CreateContainer method
func (m *MockPodmanClient) CreateContainer(ctx context.Context, config ContainerConfig) (*Container, error) {
	args := m.Called(ctx, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Container), args.Error(1)
}

// StartContainer mocks the StartContainer method
func (m *MockPodmanClient) StartContainer(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

// StopContainer mocks the StopContainer method
func (m *MockPodmanClient) StopContainer(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

// RemoveContainer mocks the RemoveContainer method
func (m *MockPodmanClient) RemoveContainer(ctx context.Context, name string, removeVolumes bool) error {
	args := m.Called(ctx, name, removeVolumes)
	return args.Error(0)
}

// ListContainers mocks the ListContainers method
func (m *MockPodmanClient) ListContainers(ctx context.Context) ([]*Container, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Container), args.Error(1)
}

// GetContainerInfo mocks the GetContainerInfo method
func (m *MockPodmanClient) GetContainerInfo(ctx context.Context, name string) (*Container, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Container), args.Error(1)
}

// FindAvailablePort mocks the FindAvailablePort method
func (m *MockPodmanClient) FindAvailablePort(startPort int) (int, error) {
	args := m.Called(startPort)
	return args.Int(0), args.Error(1)
}

// ExecContainer mocks the ExecContainer method
func (m *MockPodmanClient) ExecContainer(ctx context.Context, name string, cmd []string) error {
	args := m.Called(ctx, name, cmd)
	return args.Error(0)
}

// CopyToContainer mocks the CopyToContainer method
func (m *MockPodmanClient) CopyToContainer(ctx context.Context, name string, src, dst string) error {
	args := m.Called(ctx, name, src, dst)
	return args.Error(0)
}

// RealPodmanClient is a stub for test builds
type RealPodmanClient struct {
	conn context.Context
}

// NewPodmanClient creates a mock client for testing
func NewPodmanClient() (*RealPodmanClient, error) {
	return &RealPodmanClient{conn: context.Background()}, nil
}

// All methods below are stubs for test builds

func (c *RealPodmanClient) ContainerExists(ctx context.Context, name string) (bool, error) {
	return false, fmt.Errorf("not implemented in test build")
}

func (c *RealPodmanClient) CreateContainer(ctx context.Context, config ContainerConfig) (*Container, error) {
	return nil, fmt.Errorf("not implemented in test build")
}

func (c *RealPodmanClient) StartContainer(ctx context.Context, name string) error {
	return fmt.Errorf("not implemented in test build")
}

func (c *RealPodmanClient) StopContainer(ctx context.Context, name string) error {
	return fmt.Errorf("not implemented in test build")
}

func (c *RealPodmanClient) RemoveContainer(ctx context.Context, name string, removeVolumes bool) error {
	return fmt.Errorf("not implemented in test build")
}

func (c *RealPodmanClient) ListContainers(ctx context.Context) ([]*Container, error) {
	return nil, fmt.Errorf("not implemented in test build")
}

func (c *RealPodmanClient) GetContainerInfo(ctx context.Context, name string) (*Container, error) {
	return nil, fmt.Errorf("not implemented in test build")
}

func (c *RealPodmanClient) FindAvailablePort(startPort int) (int, error) {
	// For tests, just check if port is available
	for port := startPort; port < startPort+100; port++ {
		if ssh.IsPortAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports found")
}

func (c *RealPodmanClient) ExecContainer(ctx context.Context, name string, cmd []string) error {
	return fmt.Errorf("not implemented in test build")
}

func (c *RealPodmanClient) CopyToContainer(ctx context.Context, name string, src, dst string) error {
	return fmt.Errorf("not implemented in test build")
}

// BuildImage is a stub for test builds
func BuildImage(ctx context.Context, containerfilePath, imageName string) error {
	return fmt.Errorf("not implemented in test build")
}
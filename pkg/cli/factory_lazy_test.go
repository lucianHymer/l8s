package cli

import (
	"context"
	"errors"
	"testing"

	"l8s/pkg/config"
	"l8s/pkg/container"
	"github.com/stretchr/testify/assert"
)

func TestLazyCommandFactory(t *testing.T) {
	t.Run("command definitions available without initialization", func(t *testing.T) {
		// This should work even without config
		factory := NewLazyCommandFactory()
		
		// All command definitions should be available
		assert.NotNil(t, factory.CreateCmd())
		assert.NotNil(t, factory.SSHCmd())
		assert.NotNil(t, factory.ListCmd())
		assert.NotNil(t, factory.StartCmd())
		assert.NotNil(t, factory.StopCmd())
		assert.NotNil(t, factory.RemoveCmd())
		assert.NotNil(t, factory.InfoCmd())
		assert.NotNil(t, factory.BuildCmd())
		assert.NotNil(t, factory.RemoteCmd())
		assert.NotNil(t, factory.ExecCmd())
	})

	t.Run("help text available without initialization", func(t *testing.T) {
		factory := NewLazyCommandFactory()
		
		createCmd := factory.CreateCmd()
		assert.Equal(t, "create <name>", createCmd.Use)
		assert.Equal(t, "Create a new development container", createCmd.Short)
		
		sshCmd := factory.SSHCmd()
		assert.Equal(t, "ssh <name>", sshCmd.Use)
		assert.Equal(t, "SSH into a container", sshCmd.Short)
	})

	t.Run("command execution triggers lazy initialization", func(t *testing.T) {
		factory := &LazyCommandFactory{
			// We'll add an initializer function that can be mocked
			initializer: func() error {
				return errors.New("config not found")
			},
		}
		
		createCmd := factory.CreateCmd()
		err := createCmd.RunE(createCmd, []string{"test", "https://github.com/test/repo"})
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config not found")
	})

	t.Run("initialization happens only once", func(t *testing.T) {
		initCount := 0
		factory := &LazyCommandFactory{}
		factory.initializer = func() error {
			initCount++
			// Set up minimal required dependencies
			factory.Config = &config.Config{SSHPublicKey: "test-key"}
			factory.ContainerMgr = &MockContainerManager{}
			factory.GitClient = &MockGitClient{}
			factory.SSHClient = &MockSSHClient{}
			return nil
		}
		
		createCmd := factory.CreateCmd()
		sshCmd := factory.SSHCmd()
		
		// First execution
		_ = createCmd.RunE(createCmd, []string{"test", "https://github.com/test/repo"})
		assert.Equal(t, 1, initCount)
		
		// Second execution should not re-initialize
		_ = sshCmd.RunE(sshCmd, []string{"test"})
		assert.Equal(t, 1, initCount)
	})
}

// Mock implementations for testing
type MockContainerManager struct{}

func (m *MockContainerManager) CreateContainer(ctx context.Context, name, gitURL, branch, sshKey string) (*container.Container, error) {
	return &container.Container{Name: name}, nil
}

func (m *MockContainerManager) StartContainer(ctx context.Context, name string) error {
	return nil
}

func (m *MockContainerManager) StopContainer(ctx context.Context, name string) error {
	return nil
}

func (m *MockContainerManager) RemoveContainer(ctx context.Context, name string, removeVolumes bool) error {
	return nil
}

func (m *MockContainerManager) ListContainers(ctx context.Context) ([]*container.Container, error) {
	return nil, nil
}

func (m *MockContainerManager) GetContainerInfo(ctx context.Context, name string) (*container.Container, error) {
	return nil, nil
}

func (m *MockContainerManager) BuildImage(ctx context.Context, containerfile string) error {
	return nil
}

func (m *MockContainerManager) ExecContainer(ctx context.Context, name string, command []string) error {
	return nil
}

func (m *MockContainerManager) SSHIntoContainer(ctx context.Context, name string) error {
	return nil
}

type MockGitClient struct{}

func (m *MockGitClient) CloneRepository(repoPath, gitURL, branch string) error {
	return nil
}

func (m *MockGitClient) AddRemote(repoPath, remoteName, remoteURL string) error {
	return nil
}

func (m *MockGitClient) RemoveRemote(repoPath, remoteName string) error {
	return nil
}

func (m *MockGitClient) ListRemotes(repoPath string) (map[string]string, error) {
	return nil, nil
}

func (m *MockGitClient) SetUpstream(repoPath, remoteName, branch string) error {
	return nil
}

func (m *MockGitClient) GetCurrentBranch(repoPath string) (string, error) {
	return "", nil
}

func (m *MockGitClient) IsGitRepository(path string) bool {
	return false
}

func (m *MockGitClient) PushBranch(repoPath, branch, remoteName string, force bool) error {
	return nil
}

func (m *MockGitClient) InitRepository(repoPath string, allowPush bool, defaultBranch string) error {
	return nil
}

func (m *MockGitClient) ValidateGitURL(gitURL string) error {
	return nil
}

type MockSSHClient struct{}

func (m *MockSSHClient) ReadPublicKey(keyPath string) (string, error) {
	return "", nil
}

func (m *MockSSHClient) FindSSHPublicKey() (string, error) {
	return "mock-ssh-key", nil
}

func (m *MockSSHClient) AddSSHConfig(name, hostname string, port int, user string) error {
	return nil
}

func (m *MockSSHClient) RemoveSSHConfig(name string) error {
	return nil
}

func (m *MockSSHClient) GenerateAuthorizedKeys(publicKey string) string {
	return ""
}

func (m *MockSSHClient) IsPortAvailable(port int) bool {
	return true
}
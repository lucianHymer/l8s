package cli

import (
	"context"

	"l8s/pkg/container"
)

// ContainerManager defines the interface for container management operations
type ContainerManager interface {
	CreateContainer(ctx context.Context, name, gitURL, branch, sshKey string) (*container.Container, error)
	ListContainers(ctx context.Context) ([]*container.Container, error)
	RemoveContainer(ctx context.Context, name string, removeVolumes bool) error
	StartContainer(ctx context.Context, name string) error
	StopContainer(ctx context.Context, name string) error
	GetContainerInfo(ctx context.Context, name string) (*container.Container, error)
	ExecContainer(ctx context.Context, name string, cmd []string) error
	SSHIntoContainer(ctx context.Context, name string) error
	BuildImage(ctx context.Context, containerfile string) error
}

// GitClient defines the interface for git operations
type GitClient interface {
	CloneRepository(repoPath, gitURL, branch string) error
	AddRemote(repoPath, remoteName, remoteURL string) error
	RemoveRemote(repoPath, remoteName string) error
	ListRemotes(repoPath string) (map[string]string, error)
	SetUpstream(repoPath, remoteName, branch string) error
	CurrentBranch(repoPath string) (string, error)
	ValidateGitURL(gitURL string) error
}

// SSHClient defines the interface for SSH operations
type SSHClient interface {
	ReadPublicKey(keyPath string) (string, error)
	FindSSHPublicKey() (string, error)
	AddSSHConfig(name, hostname string, port int, user string) error
	RemoveSSHConfig(name string) error
	GenerateAuthorizedKeys(publicKey string) string
	IsPortAvailable(port int) bool
	ValidatePublicKey(key string) error
}
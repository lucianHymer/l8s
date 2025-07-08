package container

import (
	"context"
	"time"
)

// Container represents a development container
type Container struct {
	Name      string
	Status    string
	SSHPort   int
	GitURL    string
	GitBranch string
	CreatedAt time.Time
	Labels    map[string]string
}

// ContainerConfig holds configuration for creating a container
type ContainerConfig struct {
	Name          string
	GitURL        string
	GitBranch     string
	SSHPort       int
	SSHPublicKey  string
	BaseImage     string
	ContainerUser string
	Labels        map[string]string
}

// PodmanClient defines the interface for Podman operations
type PodmanClient interface {
	ContainerExists(ctx context.Context, name string) (bool, error)
	CreateContainer(ctx context.Context, config ContainerConfig) (*Container, error)
	StartContainer(ctx context.Context, name string) error
	StopContainer(ctx context.Context, name string) error
	RemoveContainer(ctx context.Context, name string, removeVolumes bool) error
	ListContainers(ctx context.Context) ([]*Container, error)
	GetContainerInfo(ctx context.Context, name string) (*Container, error)
	FindAvailablePort(startPort int) (int, error)
	ExecContainer(ctx context.Context, name string, cmd []string) error
	CopyToContainer(ctx context.Context, name string, src, dst string) error
}

// Config holds configuration for the container manager
type Config struct {
	SSHPortStart    int
	BaseImage       string
	ContainerPrefix string
	ContainerUser   string
}

// Labels used for container metadata
const (
	LabelManaged  = "l8s.managed"
	LabelGitURL   = "l8s.git.url"
	LabelGitBranch = "l8s.git.branch"
	LabelSSHPort  = "l8s.ssh.port"
)
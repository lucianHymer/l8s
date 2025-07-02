package container

import (
	"context"
	"fmt"
)

// CopyDotfiles copies dotfiles from source directory to target directory
func CopyDotfiles(sourceDir, targetDir, containerUser string) error {
	// Implementation will copy dotfiles preserving structure and permissions
	return fmt.Errorf("not implemented")
}

// shouldCopyFile determines if a file should be copied based on its name
func shouldCopyFile(filename string) bool {
	// Implementation will check if file should be copied
	return false
}

// CopyDotfilesToContainer copies dotfiles to a container via Podman
func CopyDotfilesToContainer(client PodmanClient, containerName, dotfilesDir, containerUser string) error {
	// Implementation will use podman exec/cp to copy files
	return fmt.Errorf("not implemented")
}

// PodmanClient interface extended for dotfiles operations
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
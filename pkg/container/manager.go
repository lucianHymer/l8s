package container

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/l8s/l8s/pkg/git"
	"github.com/l8s/l8s/pkg/ssh"
)

// Manager handles container operations
type Manager struct {
	client PodmanClient
	config Config
}

// NewManager creates a new container manager
func NewManager(client PodmanClient, config Config) *Manager {
	return &Manager{
		client: client,
		config: config,
	}
}

// CreateContainer creates a new development container
func (m *Manager) CreateContainer(ctx context.Context, name, gitURL, branch, sshKey string) (*Container, error) {
	// Validate container name
	if err := validateContainerName(name); err != nil {
		return nil, err
	}

	// Check if container already exists
	containerName := m.config.ContainerPrefix + "-" + name
	exists, err := m.client.ContainerExists(ctx, containerName)
	if err != nil {
		return nil, fmt.Errorf("failed to check container existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("container '%s' already exists", name)
	}

	// Validate git URL
	if err := git.ValidateGitURL(gitURL); err != nil {
		return nil, err
	}

	// Use provided branch or default to main
	if branch == "" {
		branch = "main"
	}

	// Find available SSH port
	sshPort, err := m.client.FindAvailablePort(m.config.SSHPortStart)
	if err != nil {
		return nil, fmt.Errorf("failed to find available port: %w", err)
	}

	// Create container configuration
	config := ContainerConfig{
		Name:          containerName,
		GitURL:        gitURL,
		GitBranch:     branch,
		SSHPort:       sshPort,
		SSHPublicKey:  sshKey,
		BaseImage:     m.config.BaseImage,
		ContainerUser: m.config.ContainerUser,
		Labels: map[string]string{
			LabelManaged:   "true",
			LabelGitURL:    gitURL,
			LabelGitBranch: branch,
			LabelSSHPort:   fmt.Sprintf("%d", sshPort),
		},
	}

	// Create the container
	container, err := m.client.CreateContainer(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Start the container
	if err := m.client.StartContainer(ctx, containerName); err != nil {
		// Clean up on failure
		_ = m.client.RemoveContainer(ctx, containerName, true)
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Set up SSH
	if err := m.setupSSH(ctx, containerName, sshKey); err != nil {
		// Clean up on failure
		_ = m.client.RemoveContainer(ctx, containerName, true)
		return nil, fmt.Errorf("failed to setup SSH: %w", err)
	}

	// Copy dotfiles
	if err := m.copyDotfiles(ctx, containerName); err != nil {
		// Log error but don't fail container creation
		fmt.Printf("Warning: failed to copy dotfiles: %v\n", err)
	}

	// Clone git repository
	if err := m.cloneRepository(ctx, containerName, gitURL, branch); err != nil {
		// Clean up on failure
		_ = m.client.RemoveContainer(ctx, containerName, true)
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Add SSH config entry
	sshConfigEntry := ssh.GenerateSSHConfigEntry(containerName, sshPort, m.config.ContainerUser, m.config.ContainerPrefix)
	sshConfigPath := filepath.Join(ssh.GetHomeDir(), ".ssh", "config")
	if err := ssh.AddSSHConfigEntry(sshConfigPath, sshConfigEntry); err != nil {
		// Log error but don't fail container creation
		fmt.Printf("Warning: failed to add SSH config entry: %v\n", err)
	}

	// Add git remote on host
	if err := m.addGitRemote(name, containerName, sshPort); err != nil {
		// Log error but don't fail container creation
		fmt.Printf("Warning: failed to add git remote: %v\n", err)
	}

	return container, nil
}

// ListContainers lists all l8s-managed containers
func (m *Manager) ListContainers(ctx context.Context) ([]*Container, error) {
	return m.client.ListContainers(ctx)
}

// RemoveContainer removes a container and optionally its volumes
func (m *Manager) RemoveContainer(ctx context.Context, name string, removeVolumes bool) error {
	containerName := m.config.ContainerPrefix + "-" + name

	// Check if container exists
	exists, err := m.client.ContainerExists(ctx, containerName)
	if err != nil {
		return fmt.Errorf("failed to check container existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("container '%s' not found", name)
	}

	// Remove git remote
	if err := m.removeGitRemote(name); err != nil {
		// Log error but continue with removal
		fmt.Printf("Warning: failed to remove git remote: %v\n", err)
	}

	// Remove SSH config entry
	sshConfigPath := filepath.Join(ssh.GetHomeDir(), ".ssh", "config")
	if err := ssh.RemoveSSHConfigEntry(sshConfigPath, containerName); err != nil {
		// Log error but continue with removal
		fmt.Printf("Warning: failed to remove SSH config entry: %v\n", err)
	}

	// Remove the container
	return m.client.RemoveContainer(ctx, containerName, removeVolumes)
}

// StartContainer starts a stopped container
func (m *Manager) StartContainer(ctx context.Context, name string) error {
	containerName := m.config.ContainerPrefix + "-" + name
	return m.client.StartContainer(ctx, containerName)
}

// StopContainer stops a running container
func (m *Manager) StopContainer(ctx context.Context, name string) error {
	containerName := m.config.ContainerPrefix + "-" + name
	return m.client.StopContainer(ctx, containerName)
}

// GetContainerInfo returns information about a specific container
func (m *Manager) GetContainerInfo(ctx context.Context, name string) (*Container, error) {
	containerName := m.config.ContainerPrefix + "-" + name
	return m.client.GetContainerInfo(ctx, containerName)
}

// ExecContainer executes a command in the container
func (m *Manager) ExecContainer(ctx context.Context, name string, cmd []string) error {
	containerName := m.config.ContainerPrefix + "-" + name
	return m.client.ExecContainer(ctx, containerName, cmd)
}

// setupSSH sets up SSH access in the container
func (m *Manager) setupSSH(ctx context.Context, containerName, publicKey string) error {
	// Create .ssh directory
	mkdirCmd := []string{"mkdir", "-p", fmt.Sprintf("/home/%s/.ssh", m.config.ContainerUser)}
	if err := m.client.ExecContainer(ctx, containerName, mkdirCmd); err != nil {
		return err
	}

	// Generate authorized_keys content
	authorizedKeys := ssh.GenerateAuthorizedKeys(publicKey)

	// Write authorized_keys file
	writeCmd := []string{"sh", "-c", fmt.Sprintf("echo '%s' > /home/%s/.ssh/authorized_keys", authorizedKeys, m.config.ContainerUser)}
	if err := m.client.ExecContainer(ctx, containerName, writeCmd); err != nil {
		return err
	}

	// Set permissions
	chmodCmd := []string{"chmod", "600", fmt.Sprintf("/home/%s/.ssh/authorized_keys", m.config.ContainerUser)}
	if err := m.client.ExecContainer(ctx, containerName, chmodCmd); err != nil {
		return err
	}

	// Set ownership
	chownCmd := []string{"chown", "-R", fmt.Sprintf("%s:%s", m.config.ContainerUser, m.config.ContainerUser), fmt.Sprintf("/home/%s/.ssh", m.config.ContainerUser)}
	return m.client.ExecContainer(ctx, containerName, chownCmd)
}

// copyDotfiles copies dotfiles to the container
func (m *Manager) copyDotfiles(ctx context.Context, containerName string) error {
	// Get dotfiles directory
	dotfilesDir := filepath.Join(filepath.Dir(os.Args[0]), "..", "dotfiles")
	
	// Check if dotfiles directory exists
	if _, err := os.Stat(dotfilesDir); os.IsNotExist(err) {
		// Try relative to current directory
		dotfilesDir = "dotfiles"
		if _, err := os.Stat(dotfilesDir); os.IsNotExist(err) {
			// No dotfiles directory, skip
			return nil
		}
	}
	
	// Copy dotfiles to container
	return CopyDotfilesToContainer(ctx, m.client, containerName, dotfilesDir, m.config.ContainerUser)
}

// cloneRepository clones the git repository in the container
func (m *Manager) cloneRepository(ctx context.Context, containerName, gitURL, branch string) error {
	cloneCmd := []string{"git", "clone", "-b", branch, gitURL, "/workspace/project"}
	return m.client.ExecContainer(ctx, containerName, cloneCmd)
}

// addGitRemote adds a git remote for the container
func (m *Manager) addGitRemote(name, containerName string, sshPort int) error {
	// This will interact with the git package
	return nil
}

// removeGitRemote removes the git remote for the container
func (m *Manager) removeGitRemote(name string) error {
	// This will interact with the git package
	return nil
}

// validateContainerName validates the container name
func validateContainerName(name string) error {
	if name == "" {
		return fmt.Errorf("container name cannot be empty")
	}

	// Check length
	if len(name) > 63 {
		return fmt.Errorf("container name must be 63 characters or less")
	}

	// Check format: lowercase letters, numbers, and hyphens
	validName := regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("container name must consist of lowercase letters, numbers, and hyphens, and must start and end with a letter or number")
	}

	// Check for consecutive hyphens
	if strings.Contains(name, "--") {
		return fmt.Errorf("container name cannot contain consecutive hyphens")
	}

	return nil
}
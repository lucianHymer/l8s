package container

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/l8s/l8s/pkg/cleanup"
	"github.com/l8s/l8s/pkg/git"
	"github.com/l8s/l8s/pkg/logging"
	"github.com/l8s/l8s/pkg/ssh"
)

// Manager handles container operations
type Manager struct {
	client PodmanClient
	config Config
	logger *slog.Logger
}

// NewManager creates a new container manager
func NewManager(client PodmanClient, config Config) *Manager {
	return &Manager{
		client: client,
		config: config,
		logger: logging.Default(),
	}
}

// CreateContainer creates a new development container
func (m *Manager) CreateContainer(ctx context.Context, name, gitURL, branch, sshKey string) (*Container, error) {
	// Create cleanup handler
	cleaner := cleanup.New(m.logger)
	defer func() {
		if err := recover(); err != nil {
			m.logger.Error("panic during container creation",
				logging.WithField("panic", err),
				logging.WithField("container", name))
			cleaner.Cleanup(ctx)
			panic(err)
		}
	}()

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

	m.logger.Info("creating container",
		logging.WithField("name", name),
		logging.WithField("git_url", gitURL),
		logging.WithField("branch", branch),
		logging.WithField("ssh_port", sshPort))

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

	// Add cleanup handler for container
	cleaner.Add("remove_container", func(ctx context.Context) error {
		m.logger.Debug("removing container", logging.WithField("container", containerName))
		return m.client.RemoveContainer(ctx, containerName, true)
	})

	// Start the container
	if err := m.client.StartContainer(ctx, containerName); err != nil {
		cleaner.Cleanup(ctx)
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Set up SSH
	if err := m.setupSSH(ctx, containerName, sshKey); err != nil {
		cleaner.Cleanup(ctx)
		return nil, fmt.Errorf("failed to setup SSH: %w", err)
	}

	// Copy dotfiles
	if err := m.copyDotfiles(ctx, containerName); err != nil {
		// Log error but don't fail container creation
		m.logger.Warn("failed to copy dotfiles",
			logging.WithError(err),
			logging.WithField("container", containerName))
	}

	// Clone git repository
	if err := m.cloneRepository(ctx, containerName, gitURL, branch); err != nil {
		cleaner.Cleanup(ctx)
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Add SSH config entry
	// Note: AddSSHConfig will load remote host from config
	if err := ssh.AddSSHConfig(name, "", sshPort, m.config.ContainerUser); err != nil {
		// Log error but don't fail container creation
		m.logger.Warn("failed to add SSH config entry",
			logging.WithError(err),
			logging.WithField("container", containerName))
	}

	// Add cleanup handler for SSH config
	cleaner.Add("remove_ssh_config", func(ctx context.Context) error {
		m.logger.Debug("removing SSH config entry", logging.WithField("container", containerName))
		return ssh.RemoveSSHConfig(name)
	})

	// Add git remote on host
	if err := m.addGitRemote(name, containerName, sshPort); err != nil {
		// Log error but don't fail container creation
		m.logger.Warn("failed to add git remote",
			logging.WithError(err),
			logging.WithField("container", containerName))
	}

	// Success - clear cleanup handlers
	cleaner = cleanup.New(m.logger)

	m.logger.Info("container created successfully",
		logging.WithField("name", name),
		logging.WithField("container", containerName),
		logging.WithField("ssh_port", sshPort))

	return container, nil
}

// ListContainers lists all l8s-managed containers
func (m *Manager) ListContainers(ctx context.Context) ([]*Container, error) {
	return m.client.ListContainers(ctx)
}

// RemoveContainer removes a container and optionally its volumes
func (m *Manager) RemoveContainer(ctx context.Context, name string, removeVolumes bool) error {
	containerName := m.config.ContainerPrefix + "-" + name

	m.logger.Info("removing container",
		logging.WithField("name", name),
		logging.WithField("container", containerName),
		logging.WithField("remove_volumes", removeVolumes))

	// Check if container exists
	exists, err := m.client.ContainerExists(ctx, containerName)
	if err != nil {
		return fmt.Errorf("failed to check container existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("container '%s' not found", name)
	}

	// Create cleanup handler for partial failures
	var errors []error

	// Remove git remote
	if err := m.removeGitRemote(name); err != nil {
		// Log error but continue with removal
		m.logger.Warn("failed to remove git remote",
			logging.WithError(err),
			logging.WithField("container", containerName))
		errors = append(errors, fmt.Errorf("git remote: %w", err))
	}

	// Remove SSH config entry
	if err := ssh.RemoveSSHConfig(name); err != nil {
		// Log error but continue with removal
		m.logger.Warn("failed to remove SSH config entry",
			logging.WithError(err),
			logging.WithField("container", containerName))
		errors = append(errors, fmt.Errorf("SSH config: %w", err))
	}

	// Remove the container
	if err := m.client.RemoveContainer(ctx, containerName, removeVolumes); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	m.logger.Info("container removed successfully",
		logging.WithField("name", name),
		logging.WithField("container", containerName))

	// Report non-critical errors
	if len(errors) > 0 {
		m.logger.Warn("removal completed with warnings",
			logging.WithField("warnings", errors))
	}

	return nil
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
	m.logger.Debug("setting up SSH",
		logging.WithField("container", containerName))

	// Create cleanup handler for partial SSH setup
	cleaner := cleanup.New(m.logger)

	// Create .ssh directory
	sshDir := fmt.Sprintf("/home/%s/.ssh", m.config.ContainerUser)
	mkdirCmd := []string{"mkdir", "-p", sshDir}
	if err := m.client.ExecContainer(ctx, containerName, mkdirCmd); err != nil {
		return fmt.Errorf("failed to create SSH directory: %w", err)
	}

	// Add cleanup to remove directory on failure
	cleaner.Add("remove_ssh_dir", func(ctx context.Context) error {
		return m.client.ExecContainer(ctx, containerName, []string{"rm", "-rf", sshDir})
	})

	// Generate authorized_keys content
	authorizedKeys := ssh.GenerateAuthorizedKeys(publicKey)

	// Write authorized_keys file using tee to avoid shell injection
	authorizedKeysPath := fmt.Sprintf("%s/authorized_keys", sshDir)
	writeCmd := []string{"tee", authorizedKeysPath}
	if err := m.client.ExecContainerWithInput(ctx, containerName, writeCmd, authorizedKeys); err != nil {
		cleaner.Cleanup(ctx)
		return fmt.Errorf("failed to write authorized_keys: %w", err)
	}

	// Set permissions
	chmodCmd := []string{"chmod", "600", authorizedKeysPath}
	if err := m.client.ExecContainer(ctx, containerName, chmodCmd); err != nil {
		cleaner.Cleanup(ctx)
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Set ownership
	chownCmd := []string{"chown", "-R", fmt.Sprintf("%s:%s", m.config.ContainerUser, m.config.ContainerUser), sshDir}
	if err := m.client.ExecContainer(ctx, containerName, chownCmd); err != nil {
		cleaner.Cleanup(ctx)
		return fmt.Errorf("failed to set ownership: %w", err)
	}

	m.logger.Debug("SSH setup completed",
		logging.WithField("container", containerName))

	return nil
}

// copyDotfiles copies dotfiles to the container
func (m *Manager) copyDotfiles(ctx context.Context, containerName string) error {
	// Use repository's dotfiles directory for consistent dev environment
	// This provides isolation and ensures all developers get the same experience
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	
	// Look for dotfiles in the repository
	dotfilesDir := filepath.Join(workDir, "dotfiles")
	
	// Check if dotfiles directory exists
	if _, err := os.Stat(dotfilesDir); os.IsNotExist(err) {
		// No repository dotfiles, skip
		m.logger.Debug("no repository dotfiles found, skipping", 
			logging.WithField("path", dotfilesDir))
		return nil
	}
	
	m.logger.Info("copying repository dotfiles to container",
		logging.WithField("source", dotfilesDir),
		logging.WithField("container", containerName))
	
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

// SSHIntoContainer executes SSH into the container
func (m *Manager) SSHIntoContainer(ctx context.Context, name string) error {
	containerName := m.config.ContainerPrefix + "-" + name
	
	// Get container info
	cont, err := m.client.GetContainerInfo(ctx, containerName)
	if err != nil {
		return fmt.Errorf("failed to get container info: %w", err)
	}
	
	// Check if container is running
	if cont.Status != "running" {
		return fmt.Errorf("container '%s' is not running", name)
	}
	
	// Execute SSH command
	sshCmd := exec.Command("ssh", fmt.Sprintf("%s-%s", m.config.ContainerPrefix, name))
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr
	
	return sshCmd.Run()
}

// BuildImage builds the container image
func (m *Manager) BuildImage(ctx context.Context, containerfile string) error {
	// Check if containerfile exists
	if _, err := os.Stat(containerfile); err != nil {
		return fmt.Errorf("containerfile not found: %w", err)
	}
	
	// Build the image using podman build
	buildCmd := exec.Command("podman", "build", "-f", containerfile, "-t", m.config.BaseImage, ".")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	
	return buildCmd.Run()
}
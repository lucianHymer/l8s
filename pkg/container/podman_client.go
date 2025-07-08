// +build !test

package container

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/containers"
	"github.com/containers/podman/v5/pkg/bindings/images"
	"github.com/containers/podman/v5/pkg/bindings/system"
	"github.com/containers/podman/v5/pkg/specgen"
	"github.com/containers/podman/v5/pkg/api/handlers"
	dockerContainer "github.com/docker/docker/api/types/container"
	"github.com/l8s/l8s/pkg/config"
	"github.com/l8s/l8s/pkg/ssh"
)

// RealPodmanClient implements PodmanClient using actual Podman bindings
type RealPodmanClient struct {
	conn context.Context
}

// NewPodmanClient creates a new Podman client
func NewPodmanClient() (*RealPodmanClient, error) {
	// Load configuration
	cfg, err := config.Load(config.GetConfigPath())
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	
	// Validate remote configuration
	if cfg.RemoteHost == "" || cfg.RemoteUser == "" {
		return nil, fmt.Errorf(`l8s requires remote server configuration.

Please configure your remote server in ~/.config/l8s/config.yaml:

  remote_host: your-server.example.com
  remote_user: your-username

Or run 'l8s init' to set up your configuration.

Note: l8s ONLY supports remote container management for security isolation.`)
	}
	
	// Build connection string
	connectionURI := fmt.Sprintf("ssh://%s@%s%s",
		cfg.RemoteUser,
		cfg.RemoteHost,
		cfg.RemoteSocket,
	)
	
	// Verify ssh-agent is running
	if _, exists := os.LookupEnv("SSH_AUTH_SOCK"); !exists {
		return nil, fmt.Errorf(`ssh-agent is required but not running.

Please start ssh-agent and add your key:
  eval $(ssh-agent)
  ssh-add %s

l8s requires ssh-agent for secure remote connections.`, cfg.SSHKeyPath)
	}
	
	// Set SSH key if specified
	if cfg.SSHKeyPath != "" {
		os.Setenv("CONTAINER_SSHKEY", cfg.SSHKeyPath)
	}
	
	// Create connection
	ctx := context.Background()
	conn, err := bindings.NewConnection(ctx, connectionURI)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote Podman at %s: %w", cfg.RemoteHost, err)
	}
	
	// Test connection
	_, err = system.Info(conn, nil)
	if err != nil {
		return nil, fmt.Errorf(`failed to connect to Podman on remote server.

Connection details:
  Host: %s
  User: %s
  Socket: %s

Error: %w

Troubleshooting:
1. Verify SSH access: ssh %s@%s
2. Check Podman socket is running: systemctl --user status podman.socket
3. Ensure user has Podman permissions
4. Check ~/.config/l8s/config.yaml settings

For setup instructions, see: https://github.com/l8s/l8s/docs/REMOTE_SETUP.md`,
			cfg.RemoteHost, cfg.RemoteUser, cfg.RemoteSocket, err,
			cfg.RemoteUser, cfg.RemoteHost)
	}
	
	return &RealPodmanClient{conn: conn}, nil
}

// ContainerExists checks if a container exists
func (c *RealPodmanClient) ContainerExists(ctx context.Context, name string) (bool, error) {
	exists, err := containers.Exists(c.conn, name, nil)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// CreateContainer creates a new container
func (c *RealPodmanClient) CreateContainer(ctx context.Context, config ContainerConfig) (*Container, error) {
	// Create spec for container
	s := specgen.NewSpecGenerator(config.BaseImage, false)
	s.Name = config.Name
	s.Hostname = config.Name
	s.Labels = config.Labels

	// Set up port mapping for SSH
	// In Podman v5, we need to expose the port and publish it
	s.Expose = map[uint16]string{
		22: "tcp",
	}
	// Publish all exposed ports (this will map them to available host ports)
	// For specific port mapping, we'll need to handle this after container creation
	publishPorts := true
	s.PublishExposedPorts = &publishPorts

	// Create volumes
	homeVolume := config.Name + "-home"
	workspaceVolume := config.Name + "-workspace"
	
	s.Volumes = []*specgen.NamedVolume{
		&specgen.NamedVolume{
			Name: homeVolume,
			Dest: fmt.Sprintf("/home/%s", config.ContainerUser),
		},
		&specgen.NamedVolume{
			Name: workspaceVolume,
			Dest: "/workspace",
		},
	}

	// Set environment variables
	s.Env = map[string]string{
		"USER": config.ContainerUser,
		"HOME": fmt.Sprintf("/home/%s", config.ContainerUser),
	}

	// Set command to run SSH daemon
	s.Command = []string{"/usr/sbin/sshd", "-D"}

	// Create the container
	createResponse, err := containers.CreateWithSpec(c.conn, s, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Get container info
	inspect, err := containers.Inspect(c.conn, createResponse.ID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	// Convert to our Container type
	container := &Container{
		Name:      config.Name,
		Status:    inspect.State.Status,
		SSHPort:   config.SSHPort,
		GitURL:    config.GitURL,
		GitBranch: config.GitBranch,
		CreatedAt: inspect.Created,
		Labels:    inspect.Config.Labels,
	}

	return container, nil
}

// StartContainer starts a container
func (c *RealPodmanClient) StartContainer(ctx context.Context, name string) error {
	return containers.Start(c.conn, name, nil)
}

// StopContainer stops a container
func (c *RealPodmanClient) StopContainer(ctx context.Context, name string) error {
	timeout := uint(10)
	return containers.Stop(c.conn, name, &containers.StopOptions{
		Timeout: &timeout,
	})
}

// RemoveContainer removes a container
func (c *RealPodmanClient) RemoveContainer(ctx context.Context, name string, removeVolumes bool) error {
	force := true
	_, err := containers.Remove(c.conn, name, &containers.RemoveOptions{
		Force:   &force,
		Volumes: &removeVolumes,
	})
	return err
}

// ListContainers lists all l8s-managed containers
func (c *RealPodmanClient) ListContainers(ctx context.Context) ([]*Container, error) {
	// List containers with l8s.managed label
	filters := map[string][]string{
		"label": {LabelManaged + "=true"},
	}
	
	listOpts := &containers.ListOptions{
		All:     new(bool),
		Filters: filters,
	}
	*listOpts.All = true

	containerList, err := containers.List(c.conn, listOpts)
	if err != nil {
		return nil, err
	}

	// Convert to our Container type
	var result []*Container
	for _, c := range containerList {
		// Parse SSH port from labels
		sshPort := 0
		if portStr, ok := c.Labels[LabelSSHPort]; ok {
			if p, err := strconv.Atoi(portStr); err == nil {
				sshPort = p
			}
		}

		container := &Container{
			Name:      c.Names[0],
			Status:    c.State,
			SSHPort:   sshPort,
			GitURL:    c.Labels[LabelGitURL],
			GitBranch: c.Labels[LabelGitBranch],
			CreatedAt: c.Created,
			Labels:    c.Labels,
		}
		result = append(result, container)
	}

	return result, nil
}

// GetContainerInfo gets information about a specific container
func (c *RealPodmanClient) GetContainerInfo(ctx context.Context, name string) (*Container, error) {
	// Inspect the container
	inspect, err := containers.Inspect(c.conn, name, nil)
	if err != nil {
		return nil, err
	}

	// Check if it's an l8s-managed container
	if managed, ok := inspect.Config.Labels[LabelManaged]; !ok || managed != "true" {
		return nil, fmt.Errorf("container '%s' is not managed by l8s", name)
	}

	// Parse SSH port from labels
	sshPort := 0
	if portStr, ok := inspect.Config.Labels[LabelSSHPort]; ok {
		if p, err := strconv.Atoi(portStr); err == nil {
			sshPort = p
		}
	}

	container := &Container{
		Name:      name,
		Status:    inspect.State.Status,
		SSHPort:   sshPort,
		GitURL:    inspect.Config.Labels[LabelGitURL],
		GitBranch: inspect.Config.Labels[LabelGitBranch],
		CreatedAt: inspect.Created,
		Labels:    inspect.Config.Labels,
	}

	return container, nil
}

// FindAvailablePort finds an available port starting from the given port
func (c *RealPodmanClient) FindAvailablePort(startPort int) (int, error) {
	for port := startPort; port < startPort+100; port++ {
		if ssh.IsPortAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports found in range %d-%d", startPort, startPort+100)
}

// ExecContainer executes a command in a container
func (c *RealPodmanClient) ExecContainer(ctx context.Context, name string, cmd []string) error {
	// Create exec session
	// In Podman v5, ExecCreate takes ExecCreateConfig from handlers
	attachStderr := true
	attachStdout := true
	execConfig := &handlers.ExecCreateConfig{
		ExecOptions: dockerContainer.ExecOptions{
			Cmd:          cmd,
			AttachStderr: attachStderr,
			AttachStdout: attachStdout,
		},
	}

	execID, err := containers.ExecCreate(c.conn, name, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec session: %w", err)
	}

	// Start the exec session
	if err := containers.ExecStart(c.conn, execID, nil); err != nil {
		return fmt.Errorf("failed to start exec session: %w", err)
	}

	// Wait for completion
	for {
		inspect, err := containers.ExecInspect(c.conn, execID, nil)
		if err != nil {
			return fmt.Errorf("failed to inspect exec session: %w", err)
		}

		if !inspect.Running {
			if inspect.ExitCode != 0 {
				return fmt.Errorf("command exited with code %d", inspect.ExitCode)
			}
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// ExecContainerWithInput executes a command in a container with stdin input
func (c *RealPodmanClient) ExecContainerWithInput(ctx context.Context, name string, cmd []string, input string) error {
	// Create exec configuration with stdin attached
	attachStderr := true
	attachStdout := true
	attachStdin := true
	execConfig := &handlers.ExecCreateConfig{
		ExecOptions: dockerContainer.ExecOptions{
			Cmd:          cmd,
			AttachStderr: attachStderr,
			AttachStdout: attachStdout,
			AttachStdin:  attachStdin,
		},
	}

	execID, err := containers.ExecCreate(c.conn, name, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec session: %w", err)
	}

	// Create a buffer reader from the input
	inputReader := bufio.NewReader(strings.NewReader(input))
	
	// Create output buffer to capture stdout/stderr
	var outputBuffer bytes.Buffer
	outputWriter := io.Writer(&outputBuffer)
	
	// Start and attach to the exec session
	attachInput := true
	attachOutput := false
	attachError := true
	execOptions := &containers.ExecStartAndAttachOptions{
		InputStream:  inputReader,
		OutputStream: &outputWriter,
		ErrorStream:  &outputWriter,
		AttachInput:  &attachInput,
		AttachOutput: &attachOutput,
		AttachError:  &attachError,
	}
	
	if err := containers.ExecStartAndAttach(c.conn, execID, execOptions); err != nil {
		return fmt.Errorf("failed to start exec session: %w", err)
	}

	// Check exit status
	inspect, err := containers.ExecInspect(c.conn, execID, nil)
	if err != nil {
		return fmt.Errorf("failed to inspect exec session: %w", err)
	}

	if inspect.ExitCode != 0 {
		return fmt.Errorf("command exited with code %d: %s", inspect.ExitCode, outputBuffer.String())
	}

	return nil
}

// CopyToContainer copies a file or directory to a container
func (c *RealPodmanClient) CopyToContainer(ctx context.Context, name string, src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Get file info
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	// Create tar archive
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Create tar header
	header := &tar.Header{
		Name:    dst,
		Size:    srcInfo.Size(),
		Mode:    int64(srcInfo.Mode()),
		ModTime: srcInfo.ModTime(),
	}

	// Write header
	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write tar header: %w", err)
	}

	// Copy file content
	if _, err := io.Copy(tw, srcFile); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Close tar writer
	if err := tw.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}

	// Copy to container
	reader := bytes.NewReader(buf.Bytes())
	// In Podman v5, CopyFromArchive returns a function and a cancel channel
	copyFunc, _ := containers.CopyFromArchive(c.conn, name, "/", reader)

	if err := copyFunc(); err != nil {
		return fmt.Errorf("failed to copy to container: %w", err)
	}

	return nil
}

// BuildImage builds the container image
func BuildImage(ctx context.Context, containerfilePath, imageName string) error {
	client, err := NewPodmanClient()
	if err != nil {
		return err
	}

	// Open Containerfile
	containerfile, err := os.Open(containerfilePath)
	if err != nil {
		return fmt.Errorf("failed to open Containerfile: %w", err)
	}
	defer containerfile.Close()

	// Create build options
	// In Podman v5, BuildOptions is complex - for now, just use empty options
	// The image name will be derived from the Containerfile
	buildOptions := images.BuildOptions{}

	// Build the image
	_, err = images.Build(client.conn, []string{containerfilePath}, buildOptions)
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}

	return nil
}
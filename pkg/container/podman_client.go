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
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/containers/common/libnetwork/types"
	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/containers"
	"github.com/containers/podman/v5/pkg/bindings/system"
	"github.com/containers/podman/v5/pkg/specgen"
	"github.com/containers/podman/v5/pkg/api/handlers"
	dockerContainer "github.com/docker/docker/api/types/container"
	"l8s/pkg/config"
)

// RealPodmanClient implements PodmanClient using actual Podman bindings
// 
// Architecture: SSH (non-root) -> System Podman (root via passwordless sudo)
// - SSH connection is made as non-root user for security
// - Podman runs as root via passwordless sudo (required because rootless 
//   Podman doesn't work properly inside unprivileged LXC containers)
// - Uses system socket at /run/podman/podman.sock
type RealPodmanClient struct {
	conn       context.Context
	remoteHost string
	remoteUser string
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
	
	// Build connection string for SSH access to remote Podman
	// 
	// IMPORTANT: This uses the system (root) Podman socket at /run/podman/podman.sock
	// 
	// For this to work, the remote server must be configured with:
	// 1. A 'podman' group that has access to the socket
	// 2. The user must be a member of the 'podman' group
	// 3. The socket must have group read/write permissions (660)
	// 4. The /run/podman directory must be accessible (755)
	//
	// See docs/REMOTE_SERVER_SETUP.md for detailed setup instructions
	connectionURI := fmt.Sprintf("ssh://%s@%s/run/podman/podman.sock",
		cfg.RemoteUser,
		cfg.RemoteHost,
	)
	
	// Verify ssh-agent is running
	if _, exists := os.LookupEnv("SSH_AUTH_SOCK"); !exists {
		return nil, fmt.Errorf(`ssh-agent is required but not running.

Please start ssh-agent and add your key:
  eval $(ssh-agent)
  ssh-add %s

l8s requires ssh-agent for secure remote connections.`, cfg.SSHKeyPath)
	}
	
	// Create connection using ssh-agent for authentication
	ctx := context.Background()
	conn, err := bindings.NewConnection(ctx, connectionURI)
	if err != nil {
		// Check if this is an SSH authentication error
		errStr := err.Error()
		if strings.Contains(errStr, "handshake failed") || strings.Contains(errStr, "unable to authenticate") {
			return nil, fmt.Errorf(`SSH authentication failed.

Connection details:
  Host: %s
  User: %s

Error: %w

This error typically occurs when:
1. ssh-agent is not running (already checked - it's running)
2. Your SSH key is not added to ssh-agent
3. The SSH key doesn't match the server's authorized_keys

Please ensure your SSH key is added to ssh-agent:
  ssh-add -l  # List keys in agent
  ssh-add %s  # Add your key if not listed

Also verify you can SSH directly to the server:
  ssh %s@%s`, cfg.RemoteHost, cfg.RemoteUser, err, cfg.SSHKeyPath, cfg.RemoteUser, cfg.RemoteHost)
		}
		return nil, fmt.Errorf("failed to connect to remote Podman at %s: %w", cfg.RemoteHost, err)
	}
	
	// Test connection
	_, err = system.Info(conn, nil)
	if err != nil {
		// Check if this is also an SSH authentication error
		errStr := err.Error()
		if strings.Contains(errStr, "handshake failed") || strings.Contains(errStr, "unable to authenticate") {
			return nil, fmt.Errorf(`SSH authentication failed during connection test.

Connection details:
  Host: %s
  User: %s

Error: %w

This error typically occurs when:
1. ssh-agent is not running (already checked - it's running)
2. Your SSH key is not added to ssh-agent
3. The SSH key doesn't match the server's authorized_keys

Please ensure your SSH key is added to ssh-agent:
  ssh-add -l  # List keys in agent
  ssh-add %s  # Add your key if not listed

Also verify you can SSH directly to the server:
  ssh %s@%s`, cfg.RemoteHost, cfg.RemoteUser, err, cfg.SSHKeyPath, cfg.RemoteUser, cfg.RemoteHost)
		}
		
		return nil, fmt.Errorf(`failed to connect to Podman on remote server.

Connection details:
  Host: %s
  User: %s
  Socket: %s

Error: %w

Troubleshooting:
1. Verify SSH access: ssh %s@%s
2. Check Podman socket is running: sudo systemctl status podman.socket
3. Ensure user is in 'podman' group: ssh %s@%s "groups"
4. Check socket permissions: ssh %s@%s "ls -la /run/podman/podman.sock"
5. Verify ssh-agent has your key: ssh-add -l

For detailed setup instructions, see: docs/REMOTE_SERVER_SETUP.md`,
			cfg.RemoteHost, cfg.RemoteUser, cfg.RemoteSocket, err,
			cfg.RemoteUser, cfg.RemoteHost,
			cfg.RemoteUser, cfg.RemoteHost,
			cfg.RemoteUser, cfg.RemoteHost)
	}
	
	return &RealPodmanClient{
		conn:       conn,
		remoteHost: cfg.RemoteHost,
		remoteUser: cfg.RemoteUser,
	}, nil
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

	// Set up specific port mapping for SSH
	s.PortMappings = []types.PortMapping{
		{
			HostPort:      uint16(config.SSHPort),
			ContainerPort: 22,
			Protocol:      "tcp",
		},
	}

	// Create volumes
	homeVolume := config.Name + "-home"
	workspaceVolume := config.Name + "-workspace"
	
	s.Volumes = []*specgen.NamedVolume{
		&specgen.NamedVolume{
			Name: homeVolume,
			Dest: fmt.Sprintf("/home/%s", config.ContainerUser),
			Options: []string{"U"},
		},
		&specgen.NamedVolume{
			Name: workspaceVolume,
			Dest: "/workspace",
			Options: []string{"U"},
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
	if err != nil {
		return err
	}
	
	// If removeVolumes is true, explicitly remove named volumes
	// This is a workaround for cases where Podman doesn't remove named volumes
	if removeVolumes {
		// Try to remove the named volumes explicitly
		// We don't fail if this errors as the volumes might have been removed already
		homeVolume := name + "-home"
		workspaceVolume := name + "-workspace"
		
		// Use exec to run podman volume rm commands
		// We ignore errors as volumes might not exist or might have been removed
		exec.Command("ssh", fmt.Sprintf("%s@%s", c.remoteUser, c.remoteHost), 
			"sudo", "podman", "volume", "rm", "-f", homeVolume).Run()
		exec.Command("ssh", fmt.Sprintf("%s@%s", c.remoteUser, c.remoteHost), 
			"sudo", "podman", "volume", "rm", "-f", workspaceVolume).Run()
	}
	
	return nil
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
		// Get SSH port from port mappings if available
		sshPort := 0
		if c.Ports != nil {
			for _, port := range c.Ports {
				if port.ContainerPort == 22 && port.Protocol == "tcp" {
					sshPort = int(port.HostPort)
					break
				}
			}
		}
		
		// Fallback to label if no port mapping found
		if sshPort == 0 {
			if portStr, ok := c.Labels[LabelSSHPort]; ok {
				if p, err := strconv.Atoi(portStr); err == nil {
					sshPort = p
				}
			}
		}

		container := &Container{
			Name:      c.Names[0],
			Status:    c.State,
			SSHPort:   sshPort,
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

	// Get SSH port from actual port mappings
	sshPort := 0
	if inspect.NetworkSettings != nil && len(inspect.NetworkSettings.Ports) > 0 {
		// Look for port 22/tcp mapping
		if portBindings, ok := inspect.NetworkSettings.Ports["22/tcp"]; ok && len(portBindings) > 0 {
			if p, err := strconv.Atoi(portBindings[0].HostPort); err == nil {
				sshPort = p
			}
		}
	}
	// Fallback to label if no port mapping found
	if sshPort == 0 {
		if portStr, ok := inspect.Config.Labels[LabelSSHPort]; ok {
			if p, err := strconv.Atoi(portStr); err == nil {
				sshPort = p
			}
		}
	}

	container := &Container{
		Name:      name,
		Status:    inspect.State.Status,
		SSHPort:   sshPort,
		CreatedAt: inspect.Created,
		Labels:    inspect.Config.Labels,
	}

	return container, nil
}

// FindAvailablePort finds an available port starting from the given port
func (c *RealPodmanClient) FindAvailablePort(startPort int) (int, error) {
	// Get all running containers to check which ports are in use on the remote
	containers, err := c.ListContainers(context.Background())
	if err != nil {
		return 0, fmt.Errorf("failed to list containers: %w", err)
	}
	
	// Build a map of ports in use by running containers
	portsInUse := make(map[int]bool)
	for _, container := range containers {
		// Only consider running containers
		if container.Status == "running" && container.SSHPort > 0 {
			portsInUse[container.SSHPort] = true
		}
	}
	
	// Find the first available port
	for port := startPort; port < startPort+100; port++ {
		if !portsInUse[port] {
			return port, nil
		}
	}
	
	return 0, fmt.Errorf("no available ports found in range %d-%d", startPort, startPort+100)
}

// ExecContainer executes a command in a container
func (c *RealPodmanClient) ExecContainer(ctx context.Context, name string, cmd []string) error {
	// Create output buffers
	var stdout, stderr bytes.Buffer
	
	// Create exec session with output capture
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

	// Start and attach to capture output
	outWriter := io.Writer(&stdout)
	errWriter := io.Writer(&stderr)
	attachOptions := &containers.ExecStartAndAttachOptions{
		OutputStream: &outWriter,
		ErrorStream:  &errWriter,
		AttachOutput: &attachStdout,
		AttachError:  &attachStderr,
	}
	
	if err := containers.ExecStartAndAttach(c.conn, execID, attachOptions); err != nil {
		// If attach fails, try regular start
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
	}
	
	// Check exit status
	inspect, err := containers.ExecInspect(c.conn, execID, nil)
	if err != nil {
		return fmt.Errorf("failed to inspect exec session: %w", err)
	}

	if inspect.ExitCode != 0 {
		errOutput := strings.TrimSpace(stderr.String())
		if errOutput != "" {
			return fmt.Errorf("command exited with code %d: %s", inspect.ExitCode, errOutput)
		}
		return fmt.Errorf("command exited with code %d", inspect.ExitCode)
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

// SetupWorkspace ensures the workspace directory exists with proper permissions
func (c *RealPodmanClient) SetupWorkspace(ctx context.Context, name string, containerUser string) error {
	// Create workspace directory
	mkdirCmd := []string{"mkdir", "-p", "/workspace"}
	if err := c.ExecContainer(ctx, name, mkdirCmd); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// Set ownership of workspace directory
	chownCmd := []string{"chown", fmt.Sprintf("%s:%s", containerUser, containerUser), "/workspace"}
	if err := c.ExecContainer(ctx, name, chownCmd); err != nil {
		return fmt.Errorf("failed to set workspace ownership: %w", err)
	}

	return nil
}

// BuildImage builds the container image on the remote server
func BuildImage(ctx context.Context, containerfilePath, imageName string) error {
	// Load configuration to get remote details
	cfg, err := config.Load(config.GetConfigPath())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create a temporary directory on the remote server
	tempDir := fmt.Sprintf("/tmp/l8s-build-%d", time.Now().Unix())
	mkdirCmd := fmt.Sprintf("ssh %s@%s 'mkdir -p %s'", cfg.RemoteUser, cfg.RemoteHost, tempDir)
	if err := runCommand(mkdirCmd); err != nil {
		return fmt.Errorf("failed to create temp directory on remote: %w", err)
	}
	
	// Copy the Containerfile to the remote server
	remotePath := filepath.Join(tempDir, "Containerfile")
	scpCmd := fmt.Sprintf("scp %s %s@%s:%s", containerfilePath, cfg.RemoteUser, cfg.RemoteHost, remotePath)
	if err := runCommand(scpCmd); err != nil {
		return fmt.Errorf("failed to copy Containerfile to remote: %w", err)
	}
	
	// Build the image on the remote server using sudo podman with container user
	buildCmd := fmt.Sprintf("ssh %s@%s 'sudo podman build --build-arg CONTAINER_USER=%s -t %s %s && rm -rf %s'", 
		cfg.RemoteUser, cfg.RemoteHost, cfg.ContainerUser, imageName, tempDir, tempDir)
	
	if err := runCommand(buildCmd); err != nil {
		return fmt.Errorf("failed to build image on remote: %w", err)
	}

	return nil
}

// runCommand executes a shell command and returns any error
func runCommand(cmd string) error {
	execCmd := exec.Command("sh", "-c", cmd)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	return execCmd.Run()
}
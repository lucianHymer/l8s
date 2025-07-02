// +build !test

package container

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/containers/podman/v4/pkg/bindings"
	"github.com/containers/podman/v4/pkg/bindings/containers"
	"github.com/containers/podman/v4/pkg/bindings/images"
	"github.com/containers/podman/v4/pkg/specgen"
	"github.com/l8s/l8s/pkg/ssh"
)

// RealPodmanClient implements PodmanClient using actual Podman bindings
type RealPodmanClient struct {
	conn context.Context
}

// NewPodmanClient creates a new Podman client
func NewPodmanClient() (*RealPodmanClient, error) {
	// Get connection to Podman socket
	conn, err := bindings.NewConnection(context.Background(), "unix://run/podman/podman.sock")
	if err != nil {
		// Try user socket
		home := os.Getenv("HOME")
		if home == "" {
			home = "/tmp"
		}
		userSocket := fmt.Sprintf("unix://%s/.local/share/containers/podman/machine/podman.sock", home)
		conn, err = bindings.NewConnection(context.Background(), userSocket)
		if err != nil {
			// Try XDG runtime dir
			xdgRuntime := os.Getenv("XDG_RUNTIME_DIR")
			if xdgRuntime != "" {
				xdgSocket := fmt.Sprintf("unix://%s/podman/podman.sock", xdgRuntime)
				conn, err = bindings.NewConnection(context.Background(), xdgSocket)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to connect to Podman: %w", err)
			}
		}
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
	portMapping := specgen.PortMapping{
		ContainerPort: 22,
		HostPort:      uint16(config.SSHPort),
		Protocol:      "tcp",
	}
	s.PortMappings = []specgen.PortMapping{portMapping}

	// Create volumes
	homeVolume := config.Name + "-home"
	workspaceVolume := config.Name + "-workspace"
	
	s.Volumes = []specgen.NamedVolume{
		{
			Name: homeVolume,
			Dest: fmt.Sprintf("/home/%s", config.ContainerUser),
		},
		{
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
	return containers.Remove(c.conn, name, &containers.RemoveOptions{
		Force:   &force,
		Volumes: &removeVolumes,
	})
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
			CreatedAt: time.Unix(c.Created, 0),
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
	execConfig := &containers.ExecOptions{
		Cmd:          cmd,
		AttachStderr: new(bool),
		AttachStdout: new(bool),
	}
	*execConfig.AttachStderr = true
	*execConfig.AttachStdout = true

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
	copyFunc, cancelFunc := containers.CopyFromArchive(c.conn, name, "/", reader)
	defer cancelFunc()

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
	buildOptions := &images.BuildOptions{
		Output: imageName,
	}

	// Build the image
	_, err = images.Build(client.conn, []string{containerfilePath}, *buildOptions)
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}

	return nil
}
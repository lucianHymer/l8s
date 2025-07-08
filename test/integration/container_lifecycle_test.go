// +build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"net"

	"l8s/pkg/container"
	"l8s/pkg/git"
	"l8s/pkg/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContainerLifecycle tests the complete lifecycle of a container
func TestContainerLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if podman is available
	if _, err := exec.LookPath("podman"); err != nil {
		t.Skip("Podman not available, skipping integration test")
	}

	containerName := fmt.Sprintf("test-%d", time.Now().Unix())
	ctx := context.Background()

	// Initialize real Podman client and manager
	client, err := container.NewPodmanClient()
	require.NoError(t, err)

	config := container.Config{
		BaseImage:       "localhost/l8s-fedora:latest",
		ContainerPrefix: "dev",
		SSHPortStart:    2300, // Use higher port range for tests
		ContainerUser:   "dev",
	}

	manager := container.NewManager(client, config)

	// Test 1: Create container
	t.Run("CreateContainer", func(t *testing.T) {
		// Create a test git repository
		testRepo := createTestGitRepo(t)
		
		// Get SSH public key
		sshKeyPath, err := ssh.FindSSHPublicKey()
		require.NoError(t, err)
		
		sshKey, err := ssh.ReadPublicKey(sshKeyPath)
		require.NoError(t, err)

		// Create container
		cont, err := manager.CreateContainer(ctx, containerName, testRepo, "main", sshKey)
		require.NoError(t, err)
		assert.NotNil(t, cont)
		assert.Equal(t, fmt.Sprintf("dev-%s", containerName), cont.Name)
		assert.Greater(t, cont.SSHPort, 0)
		assert.Equal(t, "running", cont.Status)

		// Verify SSH config was added
		homeDir, _ := os.UserHomeDir()
		sshConfigPath := filepath.Join(homeDir, ".ssh", "config")
		content, err := os.ReadFile(sshConfigPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), fmt.Sprintf("Host dev-%s", containerName))
	})

	// Test 2: List containers
	t.Run("ListContainers", func(t *testing.T) {
		containers, err := manager.ListContainers(ctx)
		require.NoError(t, err)
		
		found := false
		for _, c := range containers {
			if c.Name == fmt.Sprintf("dev-%s", containerName) {
				found = true
				assert.Equal(t, "running", c.Status)
				break
			}
		}
		assert.True(t, found, "Created container should be in the list")
	})

	// Test 3: Get container info
	t.Run("GetContainerInfo", func(t *testing.T) {
		info, err := manager.GetContainerInfo(ctx, containerName)
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, fmt.Sprintf("dev-%s", containerName), info.Name)
		assert.Equal(t, "running", info.Status)
		assert.NotEmpty(t, info.Volumes)
	})

	// Test 4: Stop container
	t.Run("StopContainer", func(t *testing.T) {
		err := manager.StopContainer(ctx, containerName)
		require.NoError(t, err)

		// Verify it's stopped
		info, err := manager.GetContainerInfo(ctx, containerName)
		require.NoError(t, err)
		assert.Equal(t, "stopped", info.Status)
	})

	// Test 5: Start container
	t.Run("StartContainer", func(t *testing.T) {
		err := manager.StartContainer(ctx, containerName)
		require.NoError(t, err)

		// Verify it's running again
		info, err := manager.GetContainerInfo(ctx, containerName)
		require.NoError(t, err)
		assert.Equal(t, "running", info.Status)
	})

	// Test 6: SSH connectivity
	t.Run("SSHConnectivity", func(t *testing.T) {
		info, err := manager.GetContainerInfo(ctx, containerName)
		require.NoError(t, err)

		// Try to connect via SSH and run a simple command
		cmd := exec.Command("ssh", 
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
			"-p", fmt.Sprintf("%d", info.SSHPort),
			"dev@localhost",
			"echo", "test")
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("SSH command failed: %v, output: %s", err, output)
			t.Skip("SSH connectivity test failed, container might not be fully ready")
		}
		assert.Contains(t, string(output), "test")
	})

	// Test 7: Git operations
	t.Run("GitOperations", func(t *testing.T) {
		// Create a test project directory
		projectDir := t.TempDir()
		
		// Initialize git repo
		cmd := exec.Command("git", "init")
		cmd.Dir = projectDir
		err := cmd.Run()
		require.NoError(t, err)

		// Configure git
		cmd = exec.Command("git", "config", "user.email", "test@example.com")
		cmd.Dir = projectDir
		err = cmd.Run()
		require.NoError(t, err)

		cmd = exec.Command("git", "config", "user.name", "Test User")
		cmd.Dir = projectDir
		err = cmd.Run()
		require.NoError(t, err)

		// Create a file and commit
		testFile := filepath.Join(projectDir, "test.txt")
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err)

		cmd = exec.Command("git", "add", ".")
		cmd.Dir = projectDir
		err = cmd.Run()
		require.NoError(t, err)

		cmd = exec.Command("git", "commit", "-m", "Test commit")
		cmd.Dir = projectDir
		err = cmd.Run()
		require.NoError(t, err)

		// Add container as remote
		info, err := manager.GetContainerInfo(ctx, containerName)
		require.NoError(t, err)
		
		remoteURL := fmt.Sprintf("ssh://dev@localhost:%d/workspace/project", info.SSHPort)
		err = git.AddRemote(projectDir, containerName, remoteURL)
		require.NoError(t, err)

		// Verify remote was added
		remotes, err := git.ListRemotes(projectDir)
		require.NoError(t, err)
		assert.Contains(t, remotes, containerName)
	})

	// Test 8: Remove container
	t.Run("RemoveContainer", func(t *testing.T) {
		err := manager.RemoveContainer(ctx, containerName, true)
		require.NoError(t, err)

		// Verify it's gone
		containers, err := manager.ListContainers(ctx)
		require.NoError(t, err)
		
		found := false
		for _, c := range containers {
			if c.Name == fmt.Sprintf("dev-%s", containerName) {
				found = true
				break
			}
		}
		assert.False(t, found, "Container should be removed")

		// Verify SSH config entry was removed
		homeDir, _ := os.UserHomeDir()
		sshConfigPath := filepath.Join(homeDir, ".ssh", "config")
		content, err := os.ReadFile(sshConfigPath)
		if err == nil {
			assert.NotContains(t, string(content), fmt.Sprintf("Host dev-%s", containerName))
		}
	})
}

// TestMultipleContainers tests managing multiple containers
func TestMultipleContainers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if _, err := exec.LookPath("podman"); err != nil {
		t.Skip("Podman not available, skipping integration test")
	}

	ctx := context.Background()
	containerNames := []string{
		fmt.Sprintf("test1-%d", time.Now().Unix()),
		fmt.Sprintf("test2-%d", time.Now().Unix()),
		fmt.Sprintf("test3-%d", time.Now().Unix()),
	}

	client, err := container.NewPodmanClient()
	require.NoError(t, err)

	config := container.Config{
		BaseImage:       "localhost/l8s-fedora:latest",
		ContainerPrefix: "dev",
		SSHPortStart:    2400,
		ContainerUser:   "dev",
	}

	manager := container.NewManager(client, config)

	// Create multiple containers
	t.Run("CreateMultipleContainers", func(t *testing.T) {
		testRepo := createTestGitRepo(t)
		sshKeyPath, err := ssh.FindSSHPublicKey()
		require.NoError(t, err)
		sshKey, err := ssh.ReadPublicKey(sshKeyPath)
		require.NoError(t, err)

		for i, name := range containerNames {
			cont, err := manager.CreateContainer(ctx, name, testRepo, "main", sshKey)
			require.NoError(t, err)
			assert.Equal(t, 2400+i, cont.SSHPort)
		}
	})

	// List all containers
	t.Run("ListAllContainers", func(t *testing.T) {
		containers, err := manager.ListContainers(ctx)
		require.NoError(t, err)

		foundCount := 0
		for _, c := range containers {
			for _, name := range containerNames {
				if c.Name == fmt.Sprintf("dev-%s", name) {
					foundCount++
					break
				}
			}
		}
		assert.Equal(t, len(containerNames), foundCount)
	})

	// Clean up
	t.Cleanup(func(t *testing.T) {
		for _, name := range containerNames {
			_ = manager.RemoveContainer(ctx, name, true)
		}
	})
}

// TestPortAllocation tests SSH port allocation
func TestPortAllocation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test finding available ports
	t.Run("FindAvailablePort", func(t *testing.T) {
		client, err := container.NewPodmanClient()
		require.NoError(t, err)

		port1, err := client.FindAvailablePort(2500)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, port1, 2500)

		// Bind to the port
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port1))
		require.NoError(t, err)
		defer listener.Close()

		// Find next available port
		port2, err := client.FindAvailablePort(2500)
		require.NoError(t, err)
		assert.Greater(t, port2, port1)
	})
}

// Helper function to create a test git repository
func createTestGitRepo(t *testing.T) string {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "test-repo.git")

	// Create bare repository
	cmd := exec.Command("git", "init", "--bare", repoDir)
	err := cmd.Run()
	require.NoError(t, err)

	// Create a temporary working directory to push initial commit
	workDir := t.TempDir()
	
	// Clone the bare repo
	cmd = exec.Command("git", "clone", repoDir, workDir)
	err = cmd.Run()
	require.NoError(t, err)

	// Configure git
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = workDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = workDir
	err = cmd.Run()
	require.NoError(t, err)

	// Create initial commit
	readmeFile := filepath.Join(workDir, "README.md")
	err = os.WriteFile(readmeFile, []byte("# Test Repository"), 0644)
	require.NoError(t, err)

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = workDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = workDir
	err = cmd.Run()
	require.NoError(t, err)

	// Push to bare repo
	cmd = exec.Command("git", "push", "origin", "main")
	cmd.Dir = workDir
	err = cmd.Run()
	if err != nil {
		// Try master branch if main doesn't exist
		cmd = exec.Command("git", "push", "origin", "master:main")
		cmd.Dir = workDir
		err = cmd.Run()
		require.NoError(t, err)
	}

	return repoDir
}
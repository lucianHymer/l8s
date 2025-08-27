# L8s Container Rebuild Command - Implementation Guide

## Overview

Implement a `rebuild` command that recreates an existing container with a potentially updated base image while preserving all persistent data (volumes) and configuration (SSH port, container name).

## Command Specification

```bash
l8s rebuild <container> [flags]
```

### Flags
- `--build` - Build a new image before rebuilding
- `--skip-build` - Skip building and use existing image

If neither flag is provided, the command will prompt interactively.

## Implementation Details

### 1. CLI Command Structure

#### File: `pkg/cli/factory_lazy.go`

Add a new method to create the rebuild command:

```go
func (f *LazyCommandFactory) RebuildCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "rebuild <name>",
        Short: "Rebuild container with updated image while preserving data",
        Long: `Rebuild recreates a container with the latest base image while preserving:
- All workspace and home directory data (volumes)
- SSH port assignment
- Container name and configuration`,
        Args: cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            // Initialize dependencies
            if err := f.Initialize(); err != nil {
                return err
            }
            
            // Get flags
            build, _ := cmd.Flags().GetBool("build")
            skipBuild, _ := cmd.Flags().GetBool("skip-build")
            
            // Validate mutually exclusive flags
            if build && skipBuild {
                return fmt.Errorf("--build and --skip-build are mutually exclusive")
            }
            
            return f.lazyFactory.HandleRebuild(args[0], build, skipBuild)
        },
    }
    
    cmd.Flags().Bool("build", false, "Build image before rebuilding")
    cmd.Flags().Bool("skip-build", false, "Skip build and use existing image")
    
    return cmd
}
```

#### File: `cmd/l8s/root.go`

Register the command in the root command:

```go
rootCmd.AddCommand(factory.RebuildCmd())
```

### 2. Handler Implementation

#### File: `pkg/cli/handlers.go`

Add the rebuild handler:

```go
func (f *CommandFactory) HandleRebuild(name string, build, skipBuild bool) error {
    ctx := context.Background()
    
    // Step 1: Get current container info
    containerInfo, err := f.ContainerMgr.GetContainerInfo(ctx, name)
    if err != nil {
        return fmt.Errorf("container '%s' not found: %w", name, err)
    }
    
    // Step 2: Handle image build decision
    var shouldBuild bool
    if !build && !skipBuild {
        // Interactive prompt when no flags specified
        fmt.Printf("Would you like to rebuild the base image first? [Y/n]: ")
        response := getUserInput()
        shouldBuild = (response == "" || strings.ToLower(response) == "y")
    } else {
        shouldBuild = build
    }
    
    // Step 4: Build image if requested
    if shouldBuild {
        spinner := NewSpinner("Building image")
        spinner.Start()
        if err := f.ContainerMgr.BuildImage(ctx); err != nil {
            spinner.Stop(false)
            return fmt.Errorf("failed to build image: %w", err)
        }
        spinner.Stop(true)
    }
    
    // Step 5: Execute rebuild
    spinner := NewSpinner("Rebuilding container")
    spinner.Start()
    
    opts := container.RebuildOptions{
        PreserveSSHPort: true,
    }
    
    if err := f.ContainerMgr.RebuildContainer(ctx, name, opts); err != nil {
        spinner.Stop(false)
        return fmt.Errorf("failed to rebuild container: %w", err)
    }
    spinner.Stop(true)
    
    // Step 6: Display success information
    color.Printf("{green}✓{reset} Container '%s' rebuilt successfully!\n", name)
    fmt.Printf("\nConnect with:\n")
    fmt.Printf("  ssh %s-%s\n", f.Config.ContainerPrefix, name)
    
    return nil
}
```

### 3. Container Manager Implementation

#### File: `pkg/container/types.go`

Add rebuild options struct:

```go
type RebuildOptions struct {
    PreserveSSHPort bool
}
```

Update the `ContainerManager` interface:

```go
type ContainerManager interface {
    // ... existing methods ...
    RebuildContainer(ctx context.Context, name string, opts RebuildOptions) error
}
```

#### File: `pkg/container/manager.go`

Implement the rebuild logic:

```go
func (m *Manager) RebuildContainer(ctx context.Context, name string, opts RebuildOptions) error {
    containerName := m.config.ContainerPrefix + "-" + name
    
    // Step 1: Get current container configuration
    containerInfo, err := m.client.GetContainerInfo(ctx, containerName)
    if err != nil {
        return fmt.Errorf("failed to get container info: %w", err)
    }
    
    // Extract SSH port to preserve
    sshPort := containerInfo.SSHPort
    if sshPort == 0 {
        return fmt.Errorf("container has no SSH port configured")
    }
    
    // Step 2: Stop the container
    m.logger.Debug("stopping container for rebuild",
        logging.WithField("container", containerName))
    
    if err := m.client.StopContainer(ctx, containerName); err != nil {
        // Container might already be stopped
        m.logger.Debug("container stop failed (may already be stopped)",
            logging.WithError(err))
    }
    
    // Step 3: Remove container (preserves named volumes automatically)
    m.logger.Debug("removing container",
        logging.WithField("container", containerName))
    
    if err := m.client.RemoveContainer(ctx, containerName, false); err != nil {
        return fmt.Errorf("failed to remove container: %w", err)
    }
    
    // Step 4: Create new container with same configuration
    m.logger.Debug("creating new container",
        logging.WithField("container", containerName),
        logging.WithField("ssh_port", sshPort))
    
    // Generate new SSH key for the container
    sshKey, err := m.generateSSHKey(name)
    if err != nil {
        return fmt.Errorf("failed to generate SSH key: %w", err)
    }
    
    config := &ContainerConfig{
        Name:          containerName,
        SSHPort:       sshPort,  // Preserve the same SSH port
        SSHPublicKey:  sshKey,
        BaseImage:     m.config.BaseImage,  // Use current configured image
        ContainerUser: m.config.ContainerUser,
        Labels: map[string]string{
            LabelManaged:  "true",
            LabelSSHPort:  fmt.Sprintf("%d", sshPort),
            LabelPurpose:  name,
        },
    }
    
    if err := m.client.CreateContainer(ctx, config); err != nil {
        return fmt.Errorf("failed to create container: %w", err)
    }
    
    // Step 5: Start the new container
    if err := m.client.StartContainer(ctx, containerName); err != nil {
        // Try to clean up if start fails
        _ = m.client.RemoveContainer(ctx, containerName, false)
        return fmt.Errorf("failed to start container: %w", err)
    }
    
    // Step 6: Wait for SSH to be ready
    if err := m.waitForSSH(sshPort); err != nil {
        m.logger.Warn("SSH readiness check failed",
            logging.WithError(err))
    }
    
    // Step 7: Deploy dotfiles to the new container
    dotfilesPath := m.config.DotfilesPath
    if dotfilesPath == "" {
        dotfilesPath = embed.GetDotfilesPath()
    }
    
    if err := m.deployDotfiles(containerName, dotfilesPath); err != nil {
        m.logger.Warn("failed to deploy dotfiles",
            logging.WithError(err))
    }
    
    // Step 8: Setup SSH config
    if err := m.updateSSHConfig(name, sshPort); err != nil {
        m.logger.Warn("failed to update SSH config",
            logging.WithError(err))
    }
    
    m.logger.Info("container rebuilt successfully",
        logging.WithField("container", containerName),
        logging.WithField("ssh_port", sshPort))
    
    return nil
}
```

### 4. Mock Implementation for Testing

#### File: `pkg/container/mock_client.go`

Add rebuild method to mock:

```go
func (m *MockPodmanClient) RebuildContainer(ctx context.Context, name string, opts RebuildOptions) error {
    args := m.Called(ctx, name, opts)
    return args.Error(0)
}
```

### 5. Tests

#### File: `pkg/cli/handlers_test.go`

Add tests for the rebuild handler:

```go
func TestHandleRebuild(t *testing.T) {
    tests := []struct {
        name          string
        containerName string
        build         bool
        skipBuild     bool
        setupMocks    func(*MockContainerManager)
        expectError   bool
        errorContains string
    }{
        {
            name:          "successful rebuild with build",
            containerName: "myproject",
            build:         true,
            setupMocks: func(m *MockContainerManager) {
                // Get container info
                m.On("GetContainerInfo", mock.Anything, "myproject").Return(&container.Container{
                    Name:    "dev-myproject",
                    SSHPort: 2201,
                    Status:  "running",
                }, nil)
                
                // Build image
                m.On("BuildImage", mock.Anything).Return(nil)
                
                // Rebuild container
                m.On("RebuildContainer", mock.Anything, "myproject", 
                    container.RebuildOptions{PreserveSSHPort: true}).Return(nil)
            },
            expectError: false,
        },
        {
            name:          "successful rebuild without build",
            containerName: "myproject",
            skipBuild:     true,
            setupMocks: func(m *MockContainerManager) {
                m.On("GetContainerInfo", mock.Anything, "myproject").Return(&container.Container{
                    Name:    "dev-myproject",
                    SSHPort: 2201,
                    Status:  "running",
                }, nil)
                
                m.On("RebuildContainer", mock.Anything, "myproject",
                    container.RebuildOptions{PreserveSSHPort: true}).Return(nil)
            },
            expectError: false,
        },
        {
            name:          "container not found",
            containerName: "nonexistent",
            skipBuild:     true,
            setupMocks: func(m *MockContainerManager) {
                m.On("GetContainerInfo", mock.Anything, "nonexistent").
                    Return(nil, fmt.Errorf("container not found"))
            },
            expectError:   true,
            errorContains: "not found",
        },
    }
    
    // Run tests...
}
```

### 6. Integration Tests

#### File: `test/integration/rebuild_test.go`

```go
func TestContainerRebuild(t *testing.T) {
    // Skip if not in integration test mode
    if !isIntegrationTest() {
        t.Skip("Skipping integration test")
    }
    
    manager := setupTestManager(t)
    ctx := context.Background()
    
    // Create a container
    err := manager.CreateContainer(ctx, "rebuild-test", container.CreateOptions{})
    require.NoError(t, err)
    defer manager.RemoveContainer(ctx, "rebuild-test", true)
    
    // Add a file to the workspace to verify persistence
    err = manager.ExecContainer(ctx, "rebuild-test", 
        []string{"touch", "/workspace/test-file.txt"})
    require.NoError(t, err)
    
    // Get original SSH port
    info, err := manager.GetContainerInfo(ctx, "rebuild-test")
    require.NoError(t, err)
    originalPort := info.SSHPort
    
    // Rebuild the container
    err = manager.RebuildContainer(ctx, "rebuild-test", 
        container.RebuildOptions{PreserveSSHPort: true})
    require.NoError(t, err)
    
    // Verify container is running
    info, err = manager.GetContainerInfo(ctx, "rebuild-test")
    require.NoError(t, err)
    assert.Equal(t, "running", info.Status)
    assert.Equal(t, originalPort, info.SSHPort)
    
    // Verify file persisted
    err = manager.ExecContainer(ctx, "rebuild-test",
        []string{"test", "-f", "/workspace/test-file.txt"})
    assert.NoError(t, err, "workspace file should persist after rebuild")
}
```

## Technical Notes

### Volume Persistence

Podman named volumes automatically persist when a container is removed (unless explicitly deleted with `--volumes` flag). The rebuild process relies on this behavior:

1. Volumes are named: `<container-name>-home` and `<container-name>-workspace`
2. When removing the container with `RemoveContainer(ctx, name, false)`, volumes persist
3. When creating the new container, specify the same volume names to reattach them

### SSH Port Preservation

The SSH port must be preserved to maintain:
- SSH config entries (`~/.ssh/config`)
- Git remote URLs (though we use SSH aliases like `dev-myproject`)
- User muscle memory

### Image Source

The rebuild uses `m.config.BaseImage` from the configuration file. This ensures consistency with the create command and respects any configuration changes the user has made.

### Error Handling

Critical failure points:
1. **Container doesn't exist**: Fail immediately with clear error
2. **Build fails**: Stop and report error before removing container
3. **Remove fails**: Stop to avoid orphaned resources
4. **Create fails**: Attempt cleanup of partially created container
5. **Start fails**: Remove the new container and report error

### User Experience

Progress indicators using the same spinner pattern as create:
```
Building image... ✓
Rebuilding container... ✓
```

Success output shows SSH connection command:
```
✓ Container 'myproject' rebuilt successfully!

Connect with:
  ssh dev-myproject
```

## Future Enhancements

1. **Backup option**: Create volume snapshots before rebuild
2. **Image versioning**: Tag images with versions for rollback capability
3. **Parallel rebuilds**: Support rebuilding multiple containers
4. **Diff detection**: Only rebuild if image actually changed
5. **Custom image override**: Allow specifying a different image for rebuild
package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/l8s/l8s/pkg/config"
	"github.com/l8s/l8s/pkg/container"
	"github.com/l8s/l8s/pkg/git"
	"github.com/spf13/cobra"
)

// RemoteCmd creates the remote command
func RemoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remote",
		Short: "Manage git remotes for containers",
		Long:  `Manage git remotes for containers.`,
	}

	cmd.AddCommand(remoteAddCmd())
	cmd.AddCommand(remoteRemoveCmd())

	return cmd
}

func remoteAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <name>",
		Short: "Add git remote for existing container",
		Long:  `Add a git remote for an existing container to the current repository.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runRemoteAdd,
	}
}

func remoteRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove git remote for container",
		Long:  `Remove the git remote for a container from the current repository.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runRemoteRemove,
		Aliases: []string{"rm"},
	}
}

func runRemoteAdd(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if it's a git repository
	if _, err := os.Stat(filepath.Join(cwd, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not in a git repository")
	}

	// Load configuration
	cfg, err := config.Load(config.GetConfigPath())
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create Podman client
	podmanClient, err := container.NewPodmanClient()
	if err != nil {
		return fmt.Errorf("failed to create Podman client: %w", err)
	}

	// Create container manager
	managerConfig := container.Config{
		SSHPortStart:    cfg.SSHPortStart,
		BaseImage:       cfg.BaseImage,
		ContainerPrefix: cfg.ContainerPrefix,
		ContainerUser:   cfg.ContainerUser,
	}
	manager := container.NewManager(podmanClient, managerConfig)

	// Get container info
	ctx := context.Background()
	cont, err := manager.GetContainerInfo(ctx, name)
	if err != nil {
		return err
	}

	// Generate remote URL
	containerName := cfg.ContainerPrefix + "-" + name
	remoteURL := git.GenerateSSHRemoteURL(containerName, cont.SSHPort, cfg.ContainerUser, "/workspace/project")

	// Add remote
	if err := git.AddRemote(cwd, name, remoteURL); err != nil {
		return err
	}

	fmt.Printf("✓ Added remote '%s' -> %s\n", name, remoteURL)
	return nil
}

func runRemoteRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if it's a git repository
	if _, err := os.Stat(filepath.Join(cwd, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not in a git repository")
	}

	// Remove remote
	if err := git.RemoveRemote(cwd, name); err != nil {
		return err
	}

	fmt.Printf("✓ Removed remote '%s'\n", name)
	return nil
}
package commands

import (
	"context"
	"fmt"

	"github.com/l8s/l8s/pkg/config"
	"github.com/l8s/l8s/pkg/container"
	"github.com/spf13/cobra"
)

// ExecCmd creates the exec command
func ExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec <name> <command> [args...]",
		Short: "Execute command in container",
		Long:  `Execute a command in a container (wrapper around podman exec).`,
		Args:  cobra.MinimumNArgs(2),
		RunE:  runExec,
	}

	return cmd
}

func runExec(cmd *cobra.Command, args []string) error {
	name := args[0]
	command := args[1:]

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

	// Execute command
	ctx := context.Background()
	if err := manager.ExecContainer(ctx, name, command); err != nil {
		return err
	}

	return nil
}
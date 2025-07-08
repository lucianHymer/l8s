package commands

import (
	"context"
	"fmt"

	"github.com/l8s/l8s/pkg/color"
	"github.com/l8s/l8s/pkg/config"
	"github.com/l8s/l8s/pkg/container"
	"github.com/spf13/cobra"
)

// StopCmd creates the stop command
func StopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop <name>",
		Short: "Stop a running container",
		Long:  `Stop a running l8s container.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runStop,
	}

	return cmd
}

func runStop(cmd *cobra.Command, args []string) error {
	name := args[0]

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

	// Stop container
	ctx := context.Background()
	if err := manager.StopContainer(ctx, name); err != nil {
		return err
	}

	color.Printf("{green}✓{reset} Container '{bold}%s{reset}' stopped\n", name)
	return nil
}
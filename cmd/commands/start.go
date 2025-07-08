package commands

import (
	"context"
	"fmt"

	"github.com/l8s/l8s/pkg/color"
	"github.com/l8s/l8s/pkg/config"
	"github.com/l8s/l8s/pkg/container"
	"github.com/spf13/cobra"
)

// StartCmd creates the start command
func StartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start <name>",
		Short: "Start a stopped container",
		Long:  `Start a stopped l8s container.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runStart,
	}

	return cmd
}

func runStart(cmd *cobra.Command, args []string) error {
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

	// Start container
	ctx := context.Background()
	if err := manager.StartContainer(ctx, name); err != nil {
		return err
	}

	color.Printf("{green}✓{reset} Container '{bold}%s{reset}' started\n", name)
	return nil
}
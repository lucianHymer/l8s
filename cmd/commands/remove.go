package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/l8s/l8s/pkg/config"
	"github.com/l8s/l8s/pkg/container"
	"github.com/spf13/cobra"
)

// RemoveCmd creates the remove command
func RemoveCmd() *cobra.Command {
	var force bool
	var keepVolumes bool

	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a container",
		Long:  `Remove a container and optionally its volumes.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(cmd, args, force, keepVolumes)
		},
		Aliases: []string{"rm"},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force removal without confirmation")
	cmd.Flags().BoolVar(&keepVolumes, "keep-volumes", false, "Keep volumes when removing container")

	return cmd
}

func runRemove(cmd *cobra.Command, args []string, force bool, keepVolumes bool) error {
	name := args[0]

	// Load configuration
	cfg, err := config.Load(config.GetConfigPath())
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Confirm removal if not forced
	if !force {
		removeVolumesText := ""
		if !keepVolumes {
			removeVolumesText = " and volumes"
		}
		fmt.Printf("Remove container %s-%s%s? (y/N): ", cfg.ContainerPrefix, name, removeVolumesText)
		
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Removal cancelled.")
			return nil
		}
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

	// Remove container
	ctx := context.Background()
	if err := manager.RemoveContainer(ctx, name, !keepVolumes); err != nil {
		return err
	}

	fmt.Printf("✓ Git remote removed\n")
	fmt.Printf("✓ Container removed\n")
	if !keepVolumes {
		fmt.Printf("✓ Volumes removed\n")
	}

	return nil
}
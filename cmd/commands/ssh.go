package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/l8s/l8s/pkg/config"
	"github.com/l8s/l8s/pkg/container"
	"github.com/spf13/cobra"
)

// SSHCmd creates the ssh command
func SSHCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh <name>",
		Short: "SSH into a container",
		Long:  `Connect to a container via SSH.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runSSH,
	}

	return cmd
}

func runSSH(cmd *cobra.Command, args []string) error {
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

	// Get container info
	ctx := context.Background()
	cont, err := manager.GetContainerInfo(ctx, name)
	if err != nil {
		return fmt.Errorf("container '%s' not found", name)
	}

	// Check if container is running
	if cont.Status != "running" {
		return fmt.Errorf("container '%s' is not running", name)
	}

	// Build SSH command
	containerName := cfg.ContainerPrefix + "-" + name
	fmt.Println("\"The Dude abides... connecting...\"")
	
	// Execute SSH
	sshCmd := exec.Command("ssh", containerName)
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	return sshCmd.Run()
}
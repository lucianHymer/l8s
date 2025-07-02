package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/l8s/l8s/pkg/config"
	"github.com/l8s/l8s/pkg/container"
	"github.com/l8s/l8s/pkg/ssh"
	"github.com/spf13/cobra"
)

// InfoCmd creates the info command
func InfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info <name>",
		Short: "Show detailed container information",
		Long:  `Show detailed information about a specific container.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runInfo,
	}

	return cmd
}

func runInfo(cmd *cobra.Command, args []string) error {
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
		return err
	}

	// Display information
	fmt.Printf("Container: %s\n", cont.Name)
	fmt.Printf("Status: %s\n", cont.Status)
	fmt.Printf("Created: %s (%s ago)\n", cont.CreatedAt.Format(time.RFC3339), formatDuration(time.Since(cont.CreatedAt)))
	fmt.Printf("\nSSH Connection:\n")
	fmt.Printf("  Port: %d\n", cont.SSHPort)
	fmt.Printf("  Command: ssh %s-%s\n", cfg.ContainerPrefix, name)
	fmt.Printf("  Alt command: ssh -p %d %s@localhost\n", cont.SSHPort, cfg.ContainerUser)
	
	if cont.GitURL != "" {
		fmt.Printf("\nGit Repository:\n")
		fmt.Printf("  URL: %s\n", cont.GitURL)
		fmt.Printf("  Branch: %s\n", cont.GitBranch)
		fmt.Printf("  Remote: %s-%s:/workspace/project\n", cfg.ContainerPrefix, name)
	}
	
	fmt.Printf("\nVolumes:\n")
	fmt.Printf("  Home: %s-%s-home -> /home/%s\n", cfg.ContainerPrefix, name, cfg.ContainerUser)
	fmt.Printf("  Workspace: %s-%s-workspace -> /workspace\n", cfg.ContainerPrefix, name)
	
	fmt.Printf("\nSSH Config:\n")
	sshConfigEntry := ssh.GenerateSSHConfigEntry(cont.Name, cont.SSHPort, cfg.ContainerUser, cfg.ContainerPrefix)
	fmt.Print(sshConfigEntry)

	return nil
}
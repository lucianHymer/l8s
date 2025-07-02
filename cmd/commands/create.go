package commands

import (
	"context"
	"fmt"

	"github.com/l8s/l8s/pkg/config"
	"github.com/l8s/l8s/pkg/container"
	"github.com/l8s/l8s/pkg/ssh"
	"github.com/spf13/cobra"
)

// CreateCmd creates the create command
func CreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name> <git-url> [branch]",
		Short: "Create a new development container",
		Long:  `Creates a new development container with SSH access and clones the specified git repository.`,
		Args:  cobra.RangeArgs(2, 3),
		RunE:  runCreate,
	}

	return cmd
}

func runCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	gitURL := args[1]
	branch := "main"
	if len(args) > 2 {
		branch = args[2]
	}

	// Load configuration
	cfg, err := config.Load(config.GetConfigPath())
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Find SSH key
	sshKey := cfg.SSHPublicKey
	if sshKey == "" {
		// Auto-detect SSH key
		key, err := ssh.FindSSHPublicKey()
		if err != nil {
			return fmt.Errorf("no SSH public key found in ~/.ssh/")
		}
		sshKey = key
	} else {
		// Read specified SSH key
		key, err := ssh.ReadPublicKey(sshKey)
		if err != nil {
			return fmt.Errorf("failed to read SSH public key: %w", err)
		}
		sshKey = key
	}

	// Validate SSH key
	if err := ssh.ValidatePublicKey(sshKey); err != nil {
		return fmt.Errorf("invalid SSH public key: %w", err)
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

	// Create container
	fmt.Printf("🎳 Creating container: %s-%s\n", cfg.ContainerPrefix, name)
	
	ctx := context.Background()
	cont, err := manager.CreateContainer(ctx, name, gitURL, branch, sshKey)
	if err != nil {
		return err
	}

	// Display success message
	fmt.Printf("✓ SSH port: %d\n", cont.SSHPort)
	fmt.Printf("✓ Repository cloned\n")
	fmt.Printf("✓ SSH config entry added\n")
	fmt.Printf("✓ Git remote '%s' added (%s-%s:/workspace/project)\n", name, cfg.ContainerPrefix, name)
	fmt.Printf("\nConnection options:\n")
	fmt.Printf("- l8s ssh %s\n", name)
	fmt.Printf("- ssh %s-%s\n", cfg.ContainerPrefix, name)
	fmt.Printf("- git push %s\n", name)

	return nil
}
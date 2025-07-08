package commands

import (
	"context"
	"fmt"

	"github.com/l8s/l8s/pkg/cli"
	"github.com/l8s/l8s/pkg/color"
	"github.com/l8s/l8s/pkg/ssh"
	"github.com/spf13/cobra"
)

// CreateCmd creates the create command with a container manager dependency
func CreateCmd(containerMgr cli.ContainerManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name> <git-url> [branch]",
		Short: "Create a new development container",
		Long:  `Creates a new development container with SSH access and clones the specified git repository.`,
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateWithManager(cmd, args, containerMgr)
		},
	}

	return cmd
}

func runCreateWithManager(cmd *cobra.Command, args []string, containerMgr cli.ContainerManager) error {
	name := args[0]
	gitURL := args[1]
	branch := "main"
	if len(args) > 2 {
		branch = args[2]
	}

	// This function will be called from the factory which already has config loaded
	// For now, we'll use a temporary solution to get the SSH key
	sshKey, err := ssh.FindSSHPublicKey()
	if err != nil {
		return fmt.Errorf("no SSH public key found in ~/.ssh/")
	}

	// Validate SSH key
	if err := ssh.ValidatePublicKey(sshKey); err != nil {
		return fmt.Errorf("invalid SSH public key: %w", err)
	}

	// Create container
	color.Printf("🎳 {cyan}Creating container:{reset} {bold}dev-%s{reset}\n", name)
	
	ctx := context.Background()
	cont, err := containerMgr.CreateContainer(ctx, name, gitURL, branch, sshKey)
	if err != nil {
		return err
	}

	// Display success message
	color.Printf("{green}✓{reset} SSH port: {bold}%d{reset}\n", cont.SSHPort)
	color.Printf("{green}✓{reset} Repository cloned\n")
	color.Printf("{green}✓{reset} SSH config entry added\n")
	color.Printf("{green}✓{reset} Git remote '{bold}%s{reset}' added (dev-%s:/workspace/project)\n", name, name)
	color.Printf("\n{cyan}Connection options:{reset}\n")
	color.Printf("- {bold}l8s ssh %s{reset}\n", name)
	color.Printf("- {bold}ssh dev-%s{reset}\n", name)
	color.Printf("- {bold}git push %s{reset}\n", name)

	return nil
}
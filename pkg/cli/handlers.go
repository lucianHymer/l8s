package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

// runCreate handles the create command
func (f *CommandFactory) runCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	gitURL := args[1]
	branch := "main"
	if len(args) > 2 {
		branch = args[2]
	}

	// Find SSH key
	sshKey := f.Config.SSHPublicKey
	if sshKey == "" {
		key, err := f.SSHClient.FindSSHPublicKey()
		if err != nil {
			return fmt.Errorf("no SSH public key found in ~/.ssh/")
		}
		sshKey = key
	} else {
		key, err := f.SSHClient.ReadPublicKey(sshKey)
		if err != nil {
			return fmt.Errorf("failed to read SSH public key: %w", err)
		}
		sshKey = key
	}

	// Create container
	fmt.Fprintf(cmd.OutOrStdout(), "🎳 Creating container: %s-%s\n", f.Config.ContainerPrefix, name)
	
	ctx := context.Background()
	cont, err := f.ContainerMgr.CreateContainer(ctx, name, gitURL, branch, sshKey)
	if err != nil {
		return err
	}

	// Display success message
	fmt.Fprintf(cmd.OutOrStdout(), "✓ SSH port: %d\n", cont.SSHPort)
	fmt.Fprintf(cmd.OutOrStdout(), "✓ Repository cloned\n")
	fmt.Fprintf(cmd.OutOrStdout(), "✓ SSH config entry added\n")
	fmt.Fprintf(cmd.OutOrStdout(), "✓ Git remote '%s' added (%s-%s:/workspace/project)\n", name, f.Config.ContainerPrefix, name)
	fmt.Fprintf(cmd.OutOrStdout(), "\nConnection options:\n")
	fmt.Fprintf(cmd.OutOrStdout(), "- l8s ssh %s\n", name)
	fmt.Fprintf(cmd.OutOrStdout(), "- ssh %s-%s\n", f.Config.ContainerPrefix, name)
	fmt.Fprintf(cmd.OutOrStdout(), "- git push %s\n", name)

	return nil
}

// runSSH handles the ssh command
func (f *CommandFactory) runSSH(cmd *cobra.Command, args []string) error {
	name := args[0]
	
	fmt.Fprintln(cmd.OutOrStdout(), "\"The Dude abides... connecting...\"")
	
	ctx := context.Background()
	return f.ContainerMgr.SSHIntoContainer(ctx, name)
}

// runList handles the list command
func (f *CommandFactory) runList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	containers, err := f.ContainerMgr.ListContainers(ctx)
	if err != nil {
		return err
	}

	if len(containers) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No l8s containers found")
		return nil
	}

	// Create table writer
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tSSH PORT\tGIT REMOTE\tCREATED")

	for _, c := range containers {
		gitRemote := "✗"
		if c.GitURL != "" {
			gitRemote = "✓"
		}
		
		created := formatDuration(time.Since(c.CreatedAt))
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n", 
			strings.TrimPrefix(c.Name, f.Config.ContainerPrefix+"-"),
			c.Status,
			c.SSHPort,
			gitRemote,
			created,
		)
	}
	
	return w.Flush()
}

// runStart handles the start command
func (f *CommandFactory) runStart(cmd *cobra.Command, args []string) error {
	name := args[0]
	
	ctx := context.Background()
	err := f.ContainerMgr.StartContainer(ctx, name)
	if err != nil {
		return err
	}
	
	fmt.Fprintf(cmd.OutOrStdout(), "✓ Container '%s' started\n", name)
	return nil
}

// runStop handles the stop command
func (f *CommandFactory) runStop(cmd *cobra.Command, args []string) error {
	name := args[0]
	
	ctx := context.Background()
	err := f.ContainerMgr.StopContainer(ctx, name)
	if err != nil {
		return err
	}
	
	fmt.Fprintf(cmd.OutOrStdout(), "✓ Container '%s' stopped\n", name)
	return nil
}

// runRemove handles the remove command
func (f *CommandFactory) runRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	
	// Confirm removal
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Remove container %s-%s and volumes? (y/N): ", f.Config.ContainerPrefix, name)
	response, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	
	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("Aborted")
		return nil
	}
	
	ctx := context.Background()
	
	// Remove git remote
	currentDir, err := os.Getwd()
	if err == nil {
		// Try to remove remote, but don't fail if it doesn't exist
		_ = f.GitClient.RemoveRemote(currentDir, name)
		fmt.Printf("✓ Git remote removed\n")
	}
	
	// Remove container
	err = f.ContainerMgr.RemoveContainer(ctx, name, true)
	if err != nil {
		return err
	}
	
	fmt.Printf("✓ Container removed\n")
	fmt.Printf("✓ Volumes removed\n")
	
	return nil
}

// runInfo handles the info command
func (f *CommandFactory) runInfo(cmd *cobra.Command, args []string) error {
	name := args[0]
	
	ctx := context.Background()
	cont, err := f.ContainerMgr.GetContainerInfo(ctx, name)
	if err != nil {
		return err
	}
	
	fmt.Printf("Container: %s\n", cont.Name)
	fmt.Printf("Status: %s\n", cont.Status)
	fmt.Printf("SSH Port: %d\n", cont.SSHPort)
	fmt.Printf("Git URL: %s\n", cont.GitURL)
	fmt.Printf("Git Branch: %s\n", cont.GitBranch)
	fmt.Printf("Created: %s\n", cont.CreatedAt.Format(time.RFC3339))
	
	fmt.Printf("\nSSH Connection:\n")
	fmt.Printf("- l8s ssh %s\n", strings.TrimPrefix(cont.Name, f.Config.ContainerPrefix+"-"))
	fmt.Printf("- ssh -p %d %s@localhost\n", cont.SSHPort, f.Config.ContainerUser)
	
	fmt.Printf("\nSSH Config:\n")
	fmt.Printf("Host %s\n", cont.Name)
	fmt.Printf("    HostName localhost\n")
	fmt.Printf("    Port %d\n", cont.SSHPort)
	fmt.Printf("    User %s\n", f.Config.ContainerUser)
	fmt.Printf("    StrictHostKeyChecking no\n")
	fmt.Printf("    UserKnownHostsFile /dev/null\n")
	
	return nil
}

// runBuild handles the build command
func (f *CommandFactory) runBuild(cmd *cobra.Command, args []string) error {
	containerfile := "containers/Containerfile"
	if len(args) > 0 {
		containerfile = args[0]
	}
	
	fmt.Println("Building l8s base image...")
	
	ctx := context.Background()
	err := f.ContainerMgr.BuildImage(ctx, containerfile)
	if err != nil {
		return err
	}
	
	fmt.Printf("✓ Image built successfully\n")
	return nil
}

// runRemoteAdd handles the remote add command
func (f *CommandFactory) runRemoteAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	
	ctx := context.Background()
	cont, err := f.ContainerMgr.GetContainerInfo(ctx, name)
	if err != nil {
		return err
	}
	
	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	
	// Add remote
	remoteURL := fmt.Sprintf("ssh://%s@localhost:%d/workspace/project", f.Config.ContainerUser, cont.SSHPort)
	err = f.GitClient.AddRemote(currentDir, name, remoteURL)
	if err != nil {
		return err
	}
	
	fmt.Printf("✓ Git remote '%s' added\n", name)
	return nil
}

// runRemoteRemove handles the remote remove command
func (f *CommandFactory) runRemoteRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	
	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	
	// Remove remote
	err = f.GitClient.RemoveRemote(currentDir, name)
	if err != nil {
		return err
	}
	
	fmt.Printf("✓ Git remote '%s' removed\n", name)
	return nil
}

// runExec handles the exec command
func (f *CommandFactory) runExec(cmd *cobra.Command, args []string) error {
	name := args[0]
	command := args[1:]
	
	ctx := context.Background()
	return f.ContainerMgr.ExecContainer(ctx, name, command)
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}
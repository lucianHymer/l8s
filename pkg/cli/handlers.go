package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/juju/ansiterm"
	"l8s/pkg/color"
	"l8s/pkg/config"
	"github.com/spf13/cobra"
)

// runCreate handles the create command
func (f *CommandFactory) runCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	
	// Check if we're in a git repository
	if !f.GitClient.IsGitRepository(".") {
		return fmt.Errorf("l8s create must be run from within a git repository")
	}
	
	// Get branch from flag or use current branch
	branch, _ := cmd.Flags().GetString("branch")
	if branch == "" {
		currentBranch, err := f.GitClient.GetCurrentBranch(".")
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}
		branch = currentBranch
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
	
	// Validate SSH key
	if err := f.SSHClient.ValidatePublicKey(sshKey); err != nil {
		return fmt.Errorf("invalid SSH public key: %w", err)
	}

	// Create container with empty git URL
	color.Printf("🎳 {cyan}Creating container:{reset} {bold}%s-%s{reset}\n", f.Config.ContainerPrefix, name)
	
	ctx := context.Background()
	cont, err := f.ContainerMgr.CreateContainer(ctx, name, sshKey)
	if err != nil {
		return err
	}

	// Add git remote to local repository
	remoteURL := fmt.Sprintf("%s-%s:/workspace/project", f.Config.ContainerPrefix, name)
	if err := f.GitClient.AddRemote(".", name, remoteURL); err != nil {
		// If we fail to add the remote, try to clean up the container
		color.Printf("{red}✗{reset} Failed to add git remote: %v\n", err)
		color.Printf("{yellow}!{reset} Cleaning up container...\n")
		_ = f.ContainerMgr.RemoveContainer(ctx, name, true)
		return fmt.Errorf("failed to add git remote: %w", err)
	}

	// Push the branch to the container
	color.Printf("{cyan}→{reset} Pushing {bold}%s{reset} branch to container...\n", branch)
	if err := f.GitClient.PushBranch(".", branch, name, false); err != nil {
		// If push fails, clean up remote but keep container (user might want to debug)
		color.Printf("{red}✗{reset} Failed to push code: %v\n", err)
		_ = f.GitClient.RemoveRemote(".", name)
		color.Printf("{yellow}!{reset} Container created but code push failed\n")
		color.Printf("{yellow}!{reset} You may need to manually push or remove the container\n")
		return fmt.Errorf("failed to push initial code: %w", err)
	}

	// Display success message
	color.Printf("{green}✓{reset} SSH port: {bold}%d{reset}\n", cont.SSHPort)
	color.Printf("{green}✓{reset} Git remote '{bold}%s{reset}' added\n", name)
	color.Printf("{green}✓{reset} Pushed {bold}%s{reset} branch (HEAD: %s) to container\n", branch, getShortCommitHash())
	color.Printf("{green}✓{reset} Container ready with your code\n")
	
	color.Printf("\n{cyan}Connection options:{reset}\n")
	color.Printf("- {bold}l8s ssh %s{reset}\n", name)
	color.Printf("- {bold}ssh %s-%s{reset}\n", f.Config.ContainerPrefix, name)
	color.Printf("- {bold}git push %s %s{reset}\n", name, branch)
	color.Printf("\n🎳 Her life is in your hands, dude.\n")

	return nil
}

// getShortCommitHash returns the short commit hash of HEAD
func getShortCommitHash() string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

// runSSH handles the ssh command
func (f *CommandFactory) runSSH(cmd *cobra.Command, args []string) error {
	name := args[0]
	
	
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

	// Create color-aware table writer using juju/ansiterm
	w := ansiterm.NewTabWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
	
	// Print header in bold
	if os.Getenv("NO_COLOR") == "" {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			color.Bold("NAME"),
			color.Bold("STATUS"),
			color.Bold("SSH PORT"),
			color.Bold("GIT REMOTE"),
			color.Bold("CREATED"))
	} else {
		fmt.Fprintln(w, "NAME\tSTATUS\tSSH PORT\tGIT REMOTE\tCREATED")
	}

	for _, c := range containers {
		// Check if git remote exists for this container
		remotes, _ := f.GitClient.ListRemotes(".")
		containerName := strings.TrimPrefix(c.Name, f.Config.ContainerPrefix+"-")
		_, hasRemote := remotes[containerName]
		gitRemote := formatGitStatus(hasRemote)
		
		created := formatDuration(time.Since(c.CreatedAt))
		status := formatStatus(c.Status)
		
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n", 
			strings.TrimPrefix(c.Name, f.Config.ContainerPrefix+"-"),
			status,
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
	
	color.Printf("{green}✓{reset} Container '{bold}%s{reset}' started\n", name)
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
	
	color.Printf("{green}✓{reset} Container '{bold}%s{reset}' stopped\n", name)
	return nil
}

// runRemove handles the remove command
func (f *CommandFactory) runRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	
	// Get flags
	force, _ := cmd.Flags().GetBool("force")
	keepVolumes, _ := cmd.Flags().GetBool("keep-volumes")
	
	// Confirm removal unless --force is specified
	if !force {
		reader := bufio.NewReader(os.Stdin)
		prompt := fmt.Sprintf("Remove container %s-%s", f.Config.ContainerPrefix, name)
		if !keepVolumes {
			prompt += " and volumes"
		}
		prompt += "? (y/N): "
		fmt.Print(prompt)
		
		response, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Aborted")
			return nil
		}
	}
	
	ctx := context.Background()
	
	// Remove git remote
	currentDir, err := os.Getwd()
	if err == nil {
		// Try to remove remote, but don't fail if it doesn't exist
		_ = f.GitClient.RemoveRemote(currentDir, name)
		color.Printf("{green}✓{reset} Git remote removed\n")
	}
	
	// Remove container
	removeVolumes := !keepVolumes
	err = f.ContainerMgr.RemoveContainer(ctx, name, removeVolumes)
	if err != nil {
		return err
	}
	
	color.Printf("{green}✓{reset} Container removed\n")
	if removeVolumes {
		color.Printf("{green}✓{reset} Volumes removed\n")
	} else {
		color.Printf("{yellow}!{reset} Volumes kept\n")
	}
	
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
	// Check if git remote exists
	remotes, _ := f.GitClient.ListRemotes(".")
	containerName := strings.TrimPrefix(cont.Name, f.Config.ContainerPrefix+"-")
	if remoteURL, hasRemote := remotes[containerName]; hasRemote {
		fmt.Printf("Git Remote: %s -> %s\n", containerName, remoteURL)
	} else {
		fmt.Printf("Git Remote: (none)\n")
	}
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
	fmt.Println("Building l8s base image...")
	
	ctx := context.Background()
	err := f.ContainerMgr.BuildImage(ctx, "")  // Empty string since we no longer use containerfile param
	if err != nil {
		return err
	}
	
	color.Printf("{green}✓{reset} Image built successfully\n")
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
	
	color.Printf("{green}✓{reset} Git remote '{bold}%s{reset}' added\n", name)
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
	
	color.Printf("{green}✓{reset} Git remote '{bold}%s{reset}' removed\n", name)
	return nil
}

// runExec handles the exec command
func (f *CommandFactory) runExec(cmd *cobra.Command, args []string) error {
	name := args[0]
	command := args[1:]
	
	ctx := context.Background()
	return f.ContainerMgr.ExecContainer(ctx, name, command)
}

// runPaste handles the paste command
func (f *CommandFactory) runPaste(cmd *cobra.Command, args []string) error {
	containerName := args[0]
	var customName string
	if len(args) > 1 {
		customName = args[1]
	}

	// Check platform - only macOS supported initially
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("paste command is currently only supported on macOS")
	}

	ctx := context.Background()

	// Check if container exists and is running
	cont, err := f.ContainerMgr.GetContainerInfo(ctx, containerName)
	if err != nil {
		return fmt.Errorf("container '%s' not found: %w", containerName, err)
	}
	if cont.Status != "running" {
		return fmt.Errorf("container '%s' is not running (status: %s)", containerName, cont.Status)
	}

	// Detect clipboard content type and extract content
	clipboardType, localPath, err := extractClipboardContent()
	if err != nil {
		return fmt.Errorf("failed to extract clipboard: %w", err)
	}
	defer os.Remove(localPath) // Clean up temp file

	// Determine destination filename
	var destFilename string
	if customName != "" {
		destFilename = fmt.Sprintf("clipboard-%s.%s", customName, clipboardType)
	} else {
		destFilename = fmt.Sprintf("clipboard.%s", clipboardType)
	}
	destPath := fmt.Sprintf("/tmp/claude-clipboard/%s", destFilename)

	// Ensure directory exists in container
	if err := f.ContainerMgr.ExecContainer(ctx, containerName, []string{"mkdir", "-p", "/tmp/claude-clipboard"}); err != nil {
		return fmt.Errorf("failed to create clipboard directory: %w", err)
	}

	// If using default name, remove old default files
	if customName == "" {
		// Remove both default files since we're replacing with new one
		f.ContainerMgr.ExecContainer(ctx, containerName, []string{"rm", "-f", "/tmp/claude-clipboard/clipboard.png"})
		f.ContainerMgr.ExecContainer(ctx, containerName, []string{"rm", "-f", "/tmp/claude-clipboard/clipboard.txt"})
	}

	// Read local file content
	content, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read clipboard content: %w", err)
	}

	// Transfer content to container
	if err := f.ContainerMgr.ExecContainerWithInput(ctx, containerName, []string{"tee", destPath}, content); err != nil {
		return fmt.Errorf("failed to paste to container: %w", err)
	}

	color.Printf("{green}✓{reset} Pasted to %s\n", destPath)
	return nil
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

// formatStatus returns a colored status string
func formatStatus(status string) string {
	if os.Getenv("NO_COLOR") != "" {
		return status
	}
	
	switch status {
	case "running":
		return color.Green + status + color.Reset
	case "stopped", "exited":
		return color.Red + status + color.Reset
	case "paused":
		return color.Yellow + status + color.Reset
	default:
		return status
	}
}

// formatGitStatus returns a colored git status indicator
func formatGitStatus(hasGit bool) string {
	if os.Getenv("NO_COLOR") != "" {
		if hasGit {
			return "✓"
		}
		return "✗"
	}
	
	if hasGit {
		return color.Green + "✓" + color.Reset
	}
	return color.Red + "✗" + color.Reset
}

// runInit handles the init command
func (f *CommandFactory) runInit(cmd *cobra.Command, args []string) error {
	fmt.Println("=== L8s Configuration Setup ===")
	fmt.Println()
	fmt.Println("l8s ONLY supports remote container management for security isolation.")
	fmt.Println("This setup will configure your connection to a remote Podman server.")
	fmt.Println()

	// Create config with defaults
	cfg := config.DefaultConfig()
	
	// Initialize the default connection
	connCfg := config.ConnectionConfig{}

	// Prompt for connection configuration
	fmt.Println("=== Connection Configuration ===")
	
	address, err := promptWithDefault("Server IP address or hostname", "")
	if err != nil {
		return err
	}
	if address == "" {
		return fmt.Errorf("server address is required")
	}
	connCfg.Address = address
	connCfg.Description = "Default connection"
	
	// Prompt for host configuration (same for all connections)
	fmt.Println("\n=== Host Configuration ===")
	
	remoteUser, err := promptWithDefault("Remote server username", "podman")
	if err != nil {
		return err
	}
	cfg.RemoteUser = remoteUser
	
	// Show sudo setup instructions for non-root users
	if remoteUser != "root" {
		fmt.Printf("\n📝 Note: Using non-root user '%s'. You'll need to set up sudo access:\n", remoteUser)
		fmt.Printf("   On the remote server, run:\n")
		fmt.Printf("   echo \"%s ALL=(ALL) NOPASSWD: /usr/bin/podman\" | sudo tee /etc/sudoers.d/podman\n\n", remoteUser)
	}
	
	remoteSocket, err := promptWithDefault("Remote Podman socket path", "/run/podman/podman.sock")
	if err != nil {
		return err
	}
	cfg.RemoteSocket = remoteSocket
	
	// Test SSH connectivity
	fmt.Printf("\nTesting SSH connection to %s@%s...\n", cfg.RemoteUser, connCfg.Address)
	testCmd := exec.Command("ssh", "-o", "ConnectTimeout=5", 
		fmt.Sprintf("%s@%s", cfg.RemoteUser, connCfg.Address), "echo", "OK")
	output, err := testCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to connect via SSH: %v\n", err)
		fmt.Printf("Output: %s\n", string(output))
		fmt.Printf("\nPlease ensure:\n")
		fmt.Printf("1. SSH key is configured: ssh-copy-id %s@%s\n", cfg.RemoteUser, connCfg.Address)
		fmt.Printf("2. Server is accessible\n")
		if cfg.RemoteUser != "root" {
			fmt.Printf("3. User has sudo access to Podman (see instructions above)\n")
		} else {
			fmt.Printf("3. User has Podman access\n")
		}
		return fmt.Errorf("SSH connection test failed")
	}
	color.Printf("{green}✓{reset} SSH connection successful\n")
	
	// Prompt for other configuration
	fmt.Println("\n=== Container Configuration ===")
	
	sshKeyPath, err := promptWithDefault("SSH private key path", "")
	if err != nil {
		return err
	}
	if sshKeyPath != "" {
		cfg.SSHKeyPath = sshKeyPath
	}
	
	// Add the connection configuration
	cfg.Connections["default"] = connCfg
	cfg.ActiveConnection = "default"
	
	baseImage, err := promptWithDefault("Base container image", cfg.BaseImage)
	if err != nil {
		return err
	}
	cfg.BaseImage = baseImage
	
	containerPrefix, err := promptWithDefault("Container name prefix", cfg.ContainerPrefix)
	if err != nil {
		return err
	}
	cfg.ContainerPrefix = containerPrefix
	
	containerUser, err := promptWithDefault("Container user", cfg.ContainerUser)
	if err != nil {
		return err
	}
	cfg.ContainerUser = containerUser
	
	sshPortStart, err := promptWithDefault("SSH port range start", fmt.Sprintf("%d", cfg.SSHPortStart))
	if err != nil {
		return err
	}
	if _, err := fmt.Sscanf(sshPortStart, "%d", &cfg.SSHPortStart); err != nil {
		return fmt.Errorf("invalid port number: %w", err)
	}
	
	// Auto-detect SSH public key if not specified
	if cfg.SSHPublicKey == "" {
		fmt.Println("\nDetecting SSH public key...")
		// Try common locations
		possibleKeys := []string{
			cfg.SSHKeyPath + ".pub",
			"~/.ssh/id_ed25519.pub",
			"~/.ssh/id_rsa.pub",
			"~/.ssh/id_ecdsa.pub",
		}
		
		for _, keyPath := range possibleKeys {
			expandedPath := expandPath(keyPath)
			if _, err := os.Stat(expandedPath); err == nil {
				cfg.SSHPublicKey = keyPath
				color.Printf("{green}✓{reset} Found SSH public key at %s\n", keyPath)
				break
			}
		}
		
		if cfg.SSHPublicKey == "" {
			pubKeyPath, err := promptWithDefault("SSH public key path", "~/.ssh/id_ed25519.pub")
			if err != nil {
				return err
			}
			cfg.SSHPublicKey = pubKeyPath
		}
	}
	
	// Save configuration
	configPath := config.GetConfigPath()
	fmt.Printf("\nSaving configuration to %s...\n", configPath)
	
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	
	fmt.Println("\n=== Configuration Complete ===")
	fmt.Printf("Configuration saved to: %s\n", configPath)
	fmt.Println("\nNext steps:")
	fmt.Printf("1. Ensure Podman is running on %s\n", connCfg.Address)
	if cfg.RemoteUser != "root" {
		fmt.Printf("   - Set up sudo access: echo \"%s ALL=(ALL) NOPASSWD: /usr/bin/podman\" | sudo tee /etc/sudoers.d/podman\n", cfg.RemoteUser)
	}
	fmt.Printf("2. Run 'l8s create <name>' to create your first container (from within a git repository)\n")
	fmt.Printf("3. Use 'l8s list' to see all containers\n")
	
	return nil
}

// promptWithDefault prompts the user for input with a default value
func promptWithDefault(prompt, defaultValue string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	
	input = strings.TrimSpace(input)
	if input == "" && defaultValue != "" {
		return defaultValue, nil
	}
	
	return input, nil
}

// expandPath expands tilde in paths
func expandPath(path string) string {
	if path == "" {
		return path
	}

	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			home = os.Getenv("HOME")
			if home == "" {
				return path
			}
		}
		return filepath.Join(home, path[2:])
	}

	return path
}

// HandleRebuild handles the rebuild command
func (f *CommandFactory) HandleRebuild(name string, build, skipBuild bool) error {
	ctx := context.Background()
	
	// Step 1: Get current container info to verify it exists
	_, err := f.ContainerMgr.GetContainerInfo(ctx, name)
	if err != nil {
		return fmt.Errorf("container '%s' not found: %w", name, err)
	}
	
	// Step 2: Handle image build decision
	var shouldBuild bool
	if !build && !skipBuild {
		// Interactive prompt when no flags specified
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Would you like to rebuild the base image first? [Y/n]: ")
		response, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		response = strings.TrimSpace(strings.ToLower(response))
		shouldBuild = (response == "" || response == "y" || response == "yes")
	} else {
		shouldBuild = build
	}
	
	// Step 3: Build image if requested
	if shouldBuild {
		fmt.Println("Building l8s base image...")
		if err := f.ContainerMgr.BuildImage(ctx, ""); err != nil {
			return fmt.Errorf("failed to build image: %w", err)
		}
		color.Printf("{green}✓{reset} Image built successfully\n")
	}
	
	// Step 4: Execute rebuild
	color.Printf("🎳 {cyan}Rebuilding container:{reset} {bold}%s-%s{reset}\n", f.Config.ContainerPrefix, name)
	
	if err := f.ContainerMgr.RebuildContainer(ctx, name); err != nil {
		return fmt.Errorf("failed to rebuild container: %w", err)
	}
	
	// Step 5: Display success information
	color.Printf("{green}✓{reset} Container rebuilt successfully!\n")
	fmt.Printf("\nConnect with:\n")
	fmt.Printf("  ssh %s-%s\n", f.Config.ContainerPrefix, name)
	
	return nil
}

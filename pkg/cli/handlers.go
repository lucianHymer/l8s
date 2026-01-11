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
	"github.com/spf13/cobra"
	"l8s/pkg/color"
	"l8s/pkg/config"
	"l8s/pkg/embed"
	"l8s/pkg/ssh"
)

// runCreate handles the create command
func (f *CommandFactory) runCreate(cmd *cobra.Command, args []string) error {
	// Get repository root to support running from subdirectories
	repoRoot, err := f.GitClient.GetRepositoryRoot(".")
	if err != nil {
		return fmt.Errorf("l8s create must be run from within a git repository\nThis command requires a git worktree to determine the target container.")
	}

	// Generate container name from worktree
	fullName, err := GetContainerNameFromWorktree(f.Config.ContainerPrefix)
	if err != nil {
		return fmt.Errorf("failed to generate container name: %w", err)
	}
	// Remove prefix for the short name (used in git remotes, etc.)
	// fullName is like "dev-myrepo-a3f2d1", shortName is "myrepo-a3f2d1"
	shortName := fullName[len(f.Config.ContainerPrefix)+1:]

	// Check if container already exists
	ctx := context.Background()
	existingContainer, err := f.ContainerMgr.GetContainerInfo(ctx, shortName)
	if err == nil && existingContainer != nil {
		return fmt.Errorf("container '%s' already exists for this worktree\nUse 'l8s ssh' to connect or 'l8s rm' to remove it first", fullName)
	}

	// Get branch from flag or use current branch
	branch, _ := cmd.Flags().GetString("branch")
	if branch == "" {
		currentBranch, err := f.GitClient.GetCurrentBranch(repoRoot)
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
	color.Printf("🎳 {cyan}Creating container:{reset} {bold}%s{reset}\n", fullName)

	cont, err := f.ContainerMgr.CreateContainer(ctx, shortName, sshKey)
	if err != nil {
		return err
	}

	// Add git remote to local repository
	remoteURL := fmt.Sprintf("%s:/workspace/project", fullName)
	if err := f.GitClient.AddRemote(repoRoot, shortName, remoteURL); err != nil {
		// If we fail to add the remote, try to clean up the container
		color.Printf("{red}✗{reset} Failed to add git remote: %v\n", err)
		color.Printf("{yellow}!{reset} Cleaning up container...\n")
		_ = f.ContainerMgr.RemoveContainer(ctx, shortName, true)
		return fmt.Errorf("failed to add git remote: %w", err)
	}

	// Push the branch to the container
	color.Printf("{cyan}→{reset} Pushing {bold}%s{reset} branch to container...\n", branch)
	if err := f.GitClient.PushBranch(repoRoot, branch, shortName, false); err != nil {
		// If push fails, clean up remote but keep container (user might want to debug)
		color.Printf("{red}✗{reset} Failed to push code: %v\n", err)
		_ = f.GitClient.RemoveRemote(repoRoot, shortName)
		color.Printf("{yellow}!{reset} Container created but code push failed\n")
		color.Printf("{yellow}!{reset} You may need to manually push or remove the container\n")
		return fmt.Errorf("failed to push initial code: %w", err)
	}

	// Replicate origin remote to container if it exists in host repo
	// This enables GitHub CLI (gh) to work automatically
	hostRemotes, err := f.GitClient.ListRemotes(repoRoot)
	if err == nil {
		if originURL, exists := hostRemotes["origin"]; exists {
			color.Printf("{cyan}→{reset} Adding origin remote to container for GitHub CLI support...\n")
			addRemoteCmd := []string{"su", "-", f.Config.ContainerUser, "-c",
				fmt.Sprintf("cd /workspace/project && git remote add origin %s", originURL)}
			if err := f.ContainerMgr.ExecContainer(ctx, shortName, addRemoteCmd); err != nil {
				// Non-fatal - gh CLI just won't work automatically
				color.Printf("{yellow}!{reset} Could not add origin remote to container (gh CLI may require -R flag)\n")
			}
		}
	}

	// Checkout the branch in the container so it matches what we pushed
	color.Printf("{cyan}→{reset} Checking out {bold}%s{reset} branch in container...\n", branch)
	checkoutCmd := []string{"su", "-", f.Config.ContainerUser, "-c",
		fmt.Sprintf("cd /workspace/project && git checkout %s", branch)}
	if err := f.ContainerMgr.ExecContainer(ctx, shortName, checkoutCmd); err != nil {
		// Non-fatal, but warn the user
		color.Printf("{yellow}!{reset} Warning: Failed to checkout branch in container: %v\n", err)
	}

	// Display success message
	color.Printf("{green}✓{reset} SSH port: {bold}%d{reset}\n", cont.SSHPort)
	color.Printf("{green}✓{reset} Git remote '{bold}%s{reset}' added\n", shortName)
	color.Printf("{green}✓{reset} Pushed {bold}%s{reset} branch (HEAD: %s) to container\n", branch, getShortCommitHash())
	color.Printf("{green}✓{reset} Container ready with your code\n")

	color.Printf("\n{cyan}Connection options:{reset}\n")
	color.Printf("- {bold}l8s ssh{reset} (from this worktree)\n")
	color.Printf("- {bold}ssh %s{reset}\n", fullName)
	color.Printf("- {bold}git push %s %s{reset}\n", shortName, branch)
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
	// Check if we're in a git repository
	if !f.GitClient.IsGitRepository(".") {
		return fmt.Errorf("l8s ssh must be run from within a git repository\nThis command requires a git worktree to determine the target container.")
	}

	// Generate container name from worktree
	fullName, err := GetContainerNameFromWorktree(f.Config.ContainerPrefix)
	if err != nil {
		return fmt.Errorf("failed to determine container: %w", err)
	}
	// Remove prefix for the short name
	shortName := fullName[len(f.Config.ContainerPrefix)+1:]

	ctx := context.Background()
	return f.ContainerMgr.SSHIntoContainer(ctx, shortName)
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

	// Check if we're in a git repository and get the expected container name
	expectedContainerName := GetExpectedContainerName(f.Config.ContainerPrefix)

	// Get repository root if we're in a git repo (for git remote checking)
	repoRoot := "."
	if root, err := f.GitClient.GetRepositoryRoot("."); err == nil {
		repoRoot = root
	}

	// Create color-aware table writer using juju/ansiterm
	w := ansiterm.NewTabWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)

	// Print header in bold
	if os.Getenv("NO_COLOR") == "" {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			color.Bold(""),
			color.Bold("NAME"),
			color.Bold("STATUS"),
			color.Bold("SSH PORT"),
			color.Bold("WEB PORT"),
			color.Bold("GIT REMOTE"),
			color.Bold("CREATED"))
	} else {
		fmt.Fprintln(w, "\tNAME\tSTATUS\tSSH PORT\tWEB PORT\tGIT REMOTE\tCREATED")
	}

	for _, c := range containers {
		// Check if git remote exists for this container
		remotes, _ := f.GitClient.ListRemotes(repoRoot)
		containerName := strings.TrimPrefix(c.Name, f.Config.ContainerPrefix+"-")
		_, hasRemote := remotes[containerName]
		gitRemote := formatGitStatus(hasRemote)

		created := formatDuration(time.Since(c.CreatedAt))
		status := formatStatus(c.Status)

		// Mark the current worktree's container with an arrow
		marker := " "
		if expectedContainerName != "" && c.Name == expectedContainerName {
			marker = "→"
		}

		webPort := "-"
		if c.WebPort > 0 {
			webPort = fmt.Sprintf("%d", c.WebPort)
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\t%s\n",
			marker,
			c.Name,
			status,
			c.SSHPort,
			webPort,
			gitRemote,
			created,
		)
	}

	if expectedContainerName != "" {
		w.Flush()
		fmt.Fprintln(cmd.OutOrStdout())
		color.Printf("{cyan}→{reset} Current worktree container\n")
	}

	// Flush the table first
	if err := w.Flush(); err != nil {
		return err
	}

	// Show audio tunnel status
	fmt.Fprintln(cmd.OutOrStdout())
	if isAudioTunnelConnected() {
		color.Printf("{green}Audio tunnel:{reset} ✓ connected\n")
	} else {
		color.Printf("{dim}Audio tunnel:{reset} not connected (run 'l8s audio connect')\n")
	}

	return nil
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
	// Check if we're in a git repository
	if !f.GitClient.IsGitRepository(".") {
		return fmt.Errorf("l8s remove must be run from within a git repository\nThis command requires a git worktree to determine the target container.")
	}

	// Generate container name from worktree
	fullName, err := GetContainerNameFromWorktree(f.Config.ContainerPrefix)
	if err != nil {
		return fmt.Errorf("failed to determine container: %w", err)
	}
	// Remove prefix for the short name
	name := fullName[len(f.Config.ContainerPrefix)+1:]

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
	if cont.WebPort > 0 {
		fmt.Printf("Web Port: %d (container:3000)\n", cont.WebPort)
	}
	// Check if git remote exists
	remotes, _ := f.GitClient.ListRemotes(".")
	containerName := strings.TrimPrefix(cont.Name, f.Config.ContainerPrefix+"-")
	if remoteURL, hasRemote := remotes[containerName]; hasRemote {
		fmt.Printf("Git Remote: %s -> %s\n", containerName, remoteURL)
	} else {
		fmt.Printf("Git Remote: (none)\n")
	}
	fmt.Printf("Created: %s\n", cont.CreatedAt.Format(time.RFC3339))

	// Audio tunnel status (global, not per-container)
	if isAudioTunnelConnected() {
		color.Printf("{green}Audio:{reset} ✓ connected\n")
	} else {
		color.Printf("{dim}Audio:{reset} not connected (run 'l8s audio connect')\n")
	}

	fmt.Printf("\nSSH Connection:\n")
	fmt.Printf("- l8s ssh %s\n", strings.TrimPrefix(cont.Name, f.Config.ContainerPrefix+"-"))
	fmt.Printf("- ssh -p %d %s@localhost\n", cont.SSHPort, f.Config.ContainerUser)

	if cont.WebPort > 0 {
		fmt.Printf("\nWeb Access:\n")
		fmt.Printf("- http://localhost:%d\n", cont.WebPort)
	}

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
	err := f.ContainerMgr.BuildImage(ctx, "") // Empty string since we no longer use containerfile param
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
	// Check if we're in a git repository
	if !f.GitClient.IsGitRepository(".") {
		return fmt.Errorf("l8s exec must be run from within a git repository\nThis command requires a git worktree to determine the target container.")
	}

	// Generate container name from worktree
	fullName, err := GetContainerNameFromWorktree(f.Config.ContainerPrefix)
	if err != nil {
		return fmt.Errorf("failed to determine container: %w", err)
	}
	// Remove prefix for the short name
	name := fullName[len(f.Config.ContainerPrefix)+1:]

	// The command is all the arguments
	command := args

	ctx := context.Background()
	return f.ContainerMgr.ExecContainer(ctx, name, command)
}

// runPaste handles the paste command
func (f *CommandFactory) runPaste(cmd *cobra.Command, args []string) error {
	// Check if we're in a git repository
	if !f.GitClient.IsGitRepository(".") {
		return fmt.Errorf("l8s paste must be run from within a git repository\nThis command requires a git worktree to determine the target container.")
	}

	var customName string
	if len(args) > 0 {
		customName = args[0]
	}

	// Check platform - only macOS supported initially
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("paste command is currently only supported on macOS")
	}

	ctx := context.Background()

	// Generate container name from worktree
	fullName, err := GetContainerNameFromWorktree(f.Config.ContainerPrefix)
	if err != nil {
		return fmt.Errorf("failed to determine container name: %w", err)
	}
	// Remove prefix for the short name
	containerName := fullName[len(f.Config.ContainerPrefix)+1:]

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

// isAudioTunnelConnected checks if the audio SSH tunnel is currently running
func isAudioTunnelConnected() bool {
	controlPath := filepath.Join(os.Getenv("HOME"), ".ssh", "control-*@l8s-audio:*")
	matches, _ := filepath.Glob(controlPath)
	return len(matches) > 0
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
	configDir := filepath.Dir(config.GetConfigPath())

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

	// Generate SSH CA
	fmt.Println("\n=== SSH Certificate Authority Setup ===")
	fmt.Println("Generating SSH CA for secure container connections...")

	ca, err := ssh.NewCA(configDir)
	if err != nil {
		return fmt.Errorf("failed to initialize CA: %w", err)
	}

	if !ca.Exists() {
		if err := ca.Generate(); err != nil {
			return fmt.Errorf("failed to generate CA: %w", err)
		}
		color.Printf("{green}✓{reset} Generated SSH CA keypair\n")
	} else {
		color.Printf("{yellow}!{reset} Using existing SSH CA\n")
	}

	// Store CA paths in config
	cfg.CAPrivateKeyPath = ca.PrivateKeyPath
	cfg.CAPublicKeyPath = ca.PublicKeyPath
	cfg.KnownHostsPath = filepath.Join(configDir, "known_hosts")

	// Create known_hosts file with CA entry
	if err := ca.WriteKnownHostsEntry(cfg.KnownHostsPath, connCfg.Address); err != nil {
		return fmt.Errorf("failed to create known_hosts: %w", err)
	}
	color.Printf("{green}✓{reset} Created CA trust configuration\n")

	// GitHub token configuration
	fmt.Println("\n=== GitHub CLI Configuration (Optional) ===")
	fmt.Println("Configure GitHub access for creating PRs, issues, and viewing code.")
	fmt.Print("Would you like to configure a GitHub token? (y/n) [n]: ")

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "y" || response == "yes" {
		fmt.Println("\nTo create a fine-grained personal access token:")
		fmt.Println("1. Open: https://github.com/settings/personal-access-tokens/new")
		fmt.Println("2. Set an expiration date (recommend: 90 days)")
		fmt.Println("3. Select repository access (specific repos for better security)")
		fmt.Println("4. Set these Repository permissions:")
		fmt.Println("   - Actions: Read")
		fmt.Println("   - Contents: Read")
		fmt.Println("   - Issues: Read and write")
		fmt.Println("   - Pull requests: Read and write")
		fmt.Println("   - Metadata: Read (auto-selected)")
		fmt.Println("5. Generate and copy the token")
		fmt.Println()

		tokenInput, err := promptWithDefault("GitHub token (starts with github_pat_)", "")
		if err != nil {
			return err
		}

		tokenInput = strings.TrimSpace(tokenInput)
		if tokenInput != "" {
			// Basic validation
			if !strings.HasPrefix(tokenInput, "github_pat_") && !strings.HasPrefix(tokenInput, "ghp_") {
				fmt.Println("Warning: Token doesn't start with 'github_pat_' or 'ghp_'")
				fmt.Println("Make sure you've copied the correct token.")
			}
			cfg.GitHubToken = tokenInput
			color.Printf("{green}✓{reset} GitHub token configured\n")
		}
	}

	// Save configuration
	configPath := config.GetConfigPath()
	fmt.Printf("\nSaving configuration to %s...\n", configPath)

	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Setup audio on remote host (non-fatal)
	if cfg.AudioEnabled {
		fmt.Println("\n=== Audio Setup ===")
		fmt.Println("Setting up audio on remote host...")

		// Create a temporary factory to call runAudioSetupHost
		// Use a context for the audio setup
		ctx := context.Background()

		// Build the audio setup script
		setupScript := `
mkdir -p ~/.config/pipewire/pipewire-pulse.conf.d/
cat > ~/.config/pipewire/pipewire-pulse.conf.d/10-network.conf << 'EOF'
context.exec = [
    { path = "pactl" args = "load-module module-native-protocol-tcp auth-anonymous=1" }
]
EOF
systemctl --user restart pipewire-pulse
sleep 1
pactl info > /dev/null 2>&1 && echo "PipeWire is running"
`

		// Execute audio setup on remote host
		color.Printf("{cyan}→{reset} Connecting to {bold}%s@%s{reset}...\n", cfg.RemoteUser, connCfg.Address)
		sshCmd := exec.CommandContext(ctx, "ssh",
			fmt.Sprintf("%s@%s", cfg.RemoteUser, connCfg.Address),
			"bash", "-c", setupScript)
		sshCmd.Stdout = os.Stdout
		sshCmd.Stderr = os.Stderr

		if err := sshCmd.Run(); err != nil {
			color.Printf("{yellow}⚠{reset} Audio setup failed (non-fatal): %v\n", err)
			color.Printf("  You can run {bold}l8s audio setup-host{reset} manually later\n")
		} else {
			color.Printf("{green}✓{reset} Audio configured on {bold}%s{reset} (port %d)\n", connCfg.Address, cfg.AudioPort)
		}

		// Add l8s-audio SSH config entry
		audioConfig := ssh.GenerateAudioSSHConfigEntry(
			connCfg.Address,
			cfg.RemoteUser,
			cfg.AudioPort,
			cfg.KnownHostsPath,
		)

		// Add to SSH config (AddSSHConfigEntry handles duplicates)
		sshConfigPath := filepath.Join(os.Getenv("HOME"), ".ssh", "config")
		if err := ssh.AddSSHConfigEntry(sshConfigPath, audioConfig); err != nil {
			color.Printf("{yellow}⚠{reset} Failed to add l8s-audio SSH config: %v\n", err)
		} else {
			color.Printf("{green}✓{reset} Added l8s-audio SSH config entry\n")
		}
	}

	fmt.Println("\n=== Configuration Complete ===")
	fmt.Printf("Configuration saved to: %s\n", configPath)
	color.Printf("{green}✓{reset} SSH CA configured for secure connections\n")
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

// runRebuild handles the rebuild command from a git context
func (f *CommandFactory) runRebuild(cmd *cobra.Command, args []string) error {
	// Check if we're in a git repository
	if !f.GitClient.IsGitRepository(".") {
		return fmt.Errorf("l8s rebuild must be run from within a git repository\nThis command requires a git worktree to determine the target container.")
	}

	// Generate container name from worktree
	fullName, err := GetContainerNameFromWorktree(f.Config.ContainerPrefix)
	if err != nil {
		return fmt.Errorf("failed to determine container: %w", err)
	}
	// Remove prefix for the short name
	name := fullName[len(f.Config.ContainerPrefix)+1:]

	// Get flags
	build, _ := cmd.Flags().GetBool("build")
	skipBuild, _ := cmd.Flags().GetBool("skip-build")

	// Validate mutually exclusive flags
	if build && skipBuild {
		return fmt.Errorf("--build and --skip-build are mutually exclusive")
	}

	return f.HandleRebuild(name, build, skipBuild)
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

// runPush handles the push command
func (f *CommandFactory) runPush(cmd *cobra.Command, args []string) error {
	// Get repository root to support running from subdirectories
	repoRoot, err := f.GitClient.GetRepositoryRoot(".")
	if err != nil {
		return fmt.Errorf("l8s push must be run from within a git repository\nThis command requires a git worktree to determine the target container.")
	}

	// Get current branch
	branch, err := f.GitClient.GetCurrentBranch(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Generate container name from worktree
	fullName, err := GetContainerNameFromWorktree(f.Config.ContainerPrefix)
	if err != nil {
		return fmt.Errorf("failed to determine container: %w", err)
	}
	// Remove prefix for the short name (used as remote name)
	remoteName := fullName[len(f.Config.ContainerPrefix)+1:]

	// Check if remote exists
	remotes, err := f.GitClient.ListRemotes(repoRoot)
	if err != nil {
		return err
	}
	if _, exists := remotes[remoteName]; !exists {
		return fmt.Errorf("container remote '%s' does not exist\nRun 'l8s create' first to create the container", remoteName)
	}

	// Push with fast-forward only (no force)
	color.Printf("{cyan}→{reset} Pushing {bold}%s{reset} branch to container...\n", branch)
	if err := f.GitClient.PushBranch(repoRoot, branch, remoteName, false); err != nil {
		if strings.Contains(err.Error(), "non-fast-forward") || strings.Contains(err.Error(), "rejected") {
			return fmt.Errorf("Cannot push - remote has diverged\nThe container has changes that would be overwritten.\nRun 'l8s pull' first to merge changes, then push again.")
		}
		return fmt.Errorf("failed to push branch: %w", err)
	}

	// Checkout the branch in the container to update the working directory
	ctx := context.Background()
	shortName := fullName[len(f.Config.ContainerPrefix)+1:]
	color.Printf("{cyan}→{reset} Updating working directory in container...\n")
	checkoutCmd := []string{"su", "-", f.Config.ContainerUser, "-c",
		fmt.Sprintf("cd /workspace/project && git checkout %s && git reset --hard HEAD", branch)}
	if err := f.ContainerMgr.ExecContainer(ctx, shortName, checkoutCmd); err != nil {
		color.Printf("{yellow}!{reset} Warning: Failed to update working directory: %v\n", err)
		color.Printf("{yellow}!{reset} Container may need manual 'git checkout %s' and 'git reset --hard HEAD'\n", branch)
	} else {
		color.Printf("{green}✓{reset} Successfully pushed and updated container\n")
	}

	return nil
}

// runPull handles the pull command
func (f *CommandFactory) runPull(cmd *cobra.Command, args []string) error {
	// Get repository root to support running from subdirectories
	repoRoot, err := f.GitClient.GetRepositoryRoot(".")
	if err != nil {
		return fmt.Errorf("l8s pull must be run from within a git repository\nThis command requires a git worktree to determine the target container.")
	}

	// Get current branch
	branch, err := f.GitClient.GetCurrentBranch(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Generate container name from worktree
	fullName, err := GetContainerNameFromWorktree(f.Config.ContainerPrefix)
	if err != nil {
		return fmt.Errorf("failed to determine container: %w", err)
	}
	// Remove prefix for the short name (used as remote name)
	remoteName := fullName[len(f.Config.ContainerPrefix)+1:]

	// Check if remote exists
	remotes, err := f.GitClient.ListRemotes(repoRoot)
	if err != nil {
		return err
	}
	if _, exists := remotes[remoteName]; !exists {
		return fmt.Errorf("container remote '%s' does not exist\nRun 'l8s create' first to create the container", remoteName)
	}

	// Fetch from the remote
	color.Printf("{cyan}→{reset} Fetching changes from container...\n")
	fetchCmd := exec.Command("git", "fetch", remoteName, branch)
	fetchCmd.Dir = repoRoot
	output, err := fetchCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to fetch from container: %w\nOutput: %s", err, string(output))
	}

	// Merge with fast-forward only
	color.Printf("{cyan}→{reset} Merging changes (fast-forward only)...\n")
	mergeCmd := exec.Command("git", "merge", "--ff-only", fmt.Sprintf("%s/%s", remoteName, branch))
	mergeCmd.Dir = repoRoot
	output, err = mergeCmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "not possible to fast-forward") {
			return fmt.Errorf("Cannot pull - local has diverged\nYour local branch has changes that would conflict.\nResolve the divergence manually or use 'git pull %s %s' for a merge.", remoteName, branch)
		}
		return fmt.Errorf("failed to merge changes: %w\nOutput: %s", err, string(output))
	}

	color.Printf("{green}✓{reset} Successfully pulled changes from container\n")
	return nil
}

// runStatus handles the status command
func (f *CommandFactory) runStatus(cmd *cobra.Command, args []string) error {
	// Get repository root to support running from subdirectories
	repoRoot, err := f.GitClient.GetRepositoryRoot(".")
	if err != nil {
		return fmt.Errorf("l8s status must be run from within a git repository\nThis command requires a git worktree to determine the target container.")
	}

	// Generate container name from worktree
	fullName, err := GetContainerNameFromWorktree(f.Config.ContainerPrefix)
	if err != nil {
		return fmt.Errorf("failed to determine container: %w", err)
	}
	// Remove prefix for the short name
	shortName := fullName[len(f.Config.ContainerPrefix)+1:]

	// Get container info
	ctx := context.Background()
	container, err := f.ContainerMgr.GetContainerInfo(ctx, shortName)
	if err != nil {
		color.Printf("{red}✗{reset} Container '{bold}%s{reset}' does not exist\n", fullName)
		color.Printf("Run 'l8s create' to create it.\n")
		return nil
	}

	// Display container info
	color.Printf("{cyan}Container:{reset} {bold}%s{reset}\n", fullName)
	color.Printf("{cyan}Status:{reset} %s\n", formatStatus(container.Status))
	color.Printf("{cyan}SSH Port:{reset} %d\n", container.SSHPort)
	if container.WebPort > 0 {
		color.Printf("{cyan}Web Port:{reset} %d\n", container.WebPort)
	}
	color.Printf("{cyan}Created:{reset} %s\n", container.CreatedAt.Format("2006-01-02 15:04:05"))

	// Check git remote
	remoteName := shortName
	remotes, _ := f.GitClient.ListRemotes(repoRoot)
	if remoteURL, exists := remotes[remoteName]; exists {
		color.Printf("{cyan}Git Remote:{reset} %s → %s\n", remoteName, remoteURL)

		// Get current branch
		branch, _ := f.GitClient.GetCurrentBranch(repoRoot)
		if branch != "" {
			color.Printf("{cyan}Current Branch:{reset} %s\n", branch)
		}
	} else {
		color.Printf("{yellow}!{reset} Git remote not configured\n")
	}

	return nil
}

// runRebuildAll handles the rebuild-all command
func (f *CommandFactory) runRebuildAll(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get all containers
	containers, err := f.ContainerMgr.ListContainers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) == 0 {
		fmt.Println("No containers to rebuild")
		return nil
	}

	// Get flags
	force, _ := cmd.Flags().GetBool("force")
	build, _ := cmd.Flags().GetBool("build")
	skipBuild, _ := cmd.Flags().GetBool("skip-build")

	// Validate mutually exclusive flags
	if build && skipBuild {
		return fmt.Errorf("--build and --skip-build are mutually exclusive")
	}

	// Confirm if not forced
	if !force {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Rebuild all %d containers? [y/N]: ", len(containers))
		response, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Rebuild cancelled")
			return nil
		}
	}

	// Handle image build decision
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

	// Build image if requested
	if shouldBuild {
		fmt.Println("Building l8s base image...")
		if err := f.ContainerMgr.BuildImage(ctx, ""); err != nil {
			return fmt.Errorf("failed to build image: %w", err)
		}
		color.Printf("{green}✓{reset} Image built successfully\n\n")
	}

	// Rebuild each container
	successCount := 0
	failedContainers := []string{}

	for _, container := range containers {
		containerName := strings.TrimPrefix(container.Name, f.Config.ContainerPrefix+"-")
		color.Printf("Rebuilding {bold}%s{reset}...\n", container.Name)

		if err := f.ContainerMgr.RebuildContainer(ctx, containerName); err != nil {
			color.Printf("{red}✗{reset} Failed to rebuild %s: %v\n", container.Name, err)
			failedContainers = append(failedContainers, container.Name)
		} else {
			color.Printf("{green}✓{reset} Successfully rebuilt %s\n", container.Name)
			successCount++
		}
	}

	// Summary
	fmt.Printf("\n")
	color.Printf("Rebuild complete: {green}%d successful{reset}", successCount)
	if len(failedContainers) > 0 {
		color.Printf(", {red}%d failed{reset}\n", len(failedContainers))
		fmt.Println("Failed containers:")
		for _, name := range failedContainers {
			fmt.Printf("  - %s\n", name)
		}
	} else {
		fmt.Println()
	}

	return nil
}

// runInstallZSHPlugin installs the ZSH completion plugin for Oh My Zsh
func (f *CommandFactory) runInstallZSHPlugin(ctx context.Context) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check if Oh My Zsh is installed
	ohmyzshDir := filepath.Join(homeDir, ".oh-my-zsh")
	if _, err := os.Stat(ohmyzshDir); os.IsNotExist(err) {
		return fmt.Errorf("Oh My Zsh not found at %s. Please install Oh My Zsh first: https://ohmyz.sh/", ohmyzshDir)
	}

	// Create destination directory
	pluginDir := filepath.Join(ohmyzshDir, "custom", "plugins", "l8s")
	fmt.Printf("Installing l8s ZSH plugin to %s...\n", pluginDir)

	// Remove existing plugin directory if it exists
	if _, err := os.Stat(pluginDir); err == nil {
		fmt.Println("Removing existing plugin...")
		if err := os.RemoveAll(pluginDir); err != nil {
			return fmt.Errorf("failed to remove existing plugin: %w", err)
		}
	}

	// Extract the embedded ZSH plugin
	if err := embed.ExtractZSHPlugin(pluginDir); err != nil {
		return fmt.Errorf("failed to extract ZSH plugin: %w", err)
	}

	color.Printf("{green}✓{reset} Plugin files installed\n")

	// Update .zshrc to load the plugin
	zshrcPath := filepath.Join(homeDir, ".zshrc")
	zshrcContent, err := os.ReadFile(zshrcPath)
	if err != nil {
		return fmt.Errorf("failed to read .zshrc: %w", err)
	}

	// Check if plugin is already configured
	if strings.Contains(string(zshrcContent), "plugins+=(l8s)") ||
		strings.Contains(string(zshrcContent), "plugins=(") && strings.Contains(string(zshrcContent), "l8s") {
		color.Printf("{green}✓{reset} Plugin already configured in .zshrc\n")
	} else {
		// Add plugin to .zshrc
		fmt.Println("Updating .zshrc...")
		addition := "\n# l8s plugin auto-load\n" +
			"if [[ -d \"$ZSH_CUSTOM/plugins/l8s\" ]]; then\n" +
			"    plugins+=(l8s)\n" +
			"fi\n"

		if err := os.WriteFile(zshrcPath, append(zshrcContent, []byte(addition)...), 0644); err != nil {
			return fmt.Errorf("failed to update .zshrc: %w", err)
		}
		color.Printf("{green}✓{reset} Added l8s plugin to .zshrc\n")
	}

	color.Printf("\n{green}🎉 Installation complete!{reset}\n")
	fmt.Println("\nTo activate the plugin, restart your shell or run:")
	color.Printf("  {cyan}source ~/.zshrc{reset}\n")

	return nil
}

// runTeamJoin joins or creates a team session in the container for current git repository
func (f *CommandFactory) runTeamJoin(ctx context.Context, sessionName string) error {
	// Check if we're in a git repository
	if !f.GitClient.IsGitRepository(".") {
		return fmt.Errorf("l8s team must be run from within a git repository\nThis command requires a git worktree to determine the target container.")
	}

	// Generate container name from worktree
	fullName, err := GetContainerNameFromWorktree(f.Config.ContainerPrefix)
	if err != nil {
		return fmt.Errorf("failed to generate container name: %w", err)
	}
	// Remove prefix for the short name
	name := fullName[len(f.Config.ContainerPrefix)+1:]

	// Validate container exists and is running
	container, err := f.ContainerMgr.GetContainerInfo(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get container info: %w", err)
	}

	if container.Status != "running" {
		return fmt.Errorf("container '%s' is not running (status: %s)\nRun 'l8s start' to start it first.", fullName, container.Status)
	}

	// Execute SSH with team command
	// Use -t to force TTY allocation for interactive session
	// Use fully qualified path to avoid PATH issues
	sshCmd := exec.Command("ssh", "-t",
		fmt.Sprintf("%s-%s", f.Config.ContainerPrefix, name),
		fmt.Sprintf("~/.local/bin/team %s", sessionName))

	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	return sshCmd.Run()
}

// runTeamList lists active team sessions in the container for current git repository
func (f *CommandFactory) runTeamList(ctx context.Context) error {
	// Check if we're in a git repository
	if !f.GitClient.IsGitRepository(".") {
		return fmt.Errorf("l8s team list must be run from within a git repository\nThis command requires a git worktree to determine the target container.")
	}

	// Generate container name from worktree
	fullName, err := GetContainerNameFromWorktree(f.Config.ContainerPrefix)
	if err != nil {
		return fmt.Errorf("failed to generate container name: %w", err)
	}
	// Remove prefix for the short name
	name := fullName[len(f.Config.ContainerPrefix)+1:]

	// Validate container exists and is running
	container, err := f.ContainerMgr.GetContainerInfo(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get container info: %w", err)
	}

	if container.Status != "running" {
		return fmt.Errorf("container '%s' is not running (status: %s)\nRun 'l8s start' to start it first.", fullName, container.Status)
	}

	// Execute team list command via SSH
	// Use fully qualified path to avoid PATH issues
	sshCmd := exec.Command("ssh",
		fmt.Sprintf("%s-%s", f.Config.ContainerPrefix, name),
		"~/.local/bin/team list")
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	return sshCmd.Run()
}

// runAudioSetupHost configures PipeWire/PulseAudio on the remote host
func (f *CommandFactory) runAudioSetupHost(ctx context.Context) error {
	fmt.Println("Setting up audio on remote host...")

	// Get remote host info from config
	conn, err := f.Config.GetActiveConnection()
	if err != nil {
		return fmt.Errorf("no active connection configured: %w", err)
	}
	remoteHost := conn.Address
	remoteUser := f.Config.RemoteUser
	audioPort := f.Config.AudioPort

	// Build the setup script
	// This creates a PipeWire config that loads the native TCP protocol module
	// allowing PulseAudio clients to connect over the network
	setupScript := `
mkdir -p ~/.config/pipewire/pipewire-pulse.conf.d/
cat > ~/.config/pipewire/pipewire-pulse.conf.d/10-network.conf << 'EOF'
context.exec = [
    { path = "pactl" args = "load-module module-native-protocol-tcp auth-anonymous=1" }
]
EOF
systemctl --user restart pipewire-pulse
sleep 1
pactl info > /dev/null 2>&1 && echo "PipeWire is running"
`

	// Execute via SSH
	color.Printf("{cyan}→{reset} Connecting to {bold}%s@%s{reset}...\n", remoteUser, remoteHost)
	sshCmd := exec.CommandContext(ctx, "ssh",
		fmt.Sprintf("%s@%s", remoteUser, remoteHost),
		"bash", "-c", setupScript)
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	if err := sshCmd.Run(); err != nil {
		return fmt.Errorf("failed to setup audio on host: %w", err)
	}

	color.Printf("{green}✓{reset} Audio configured on {bold}%s{reset} (port %d)\n", remoteHost, audioPort)
	color.Printf("\n{cyan}Next steps:{reset}\n")
	color.Printf("- Run {bold}l8s audio setup-mac{reset} to configure your Mac\n")
	color.Printf("- Run {bold}l8s audio connect{reset} to start the audio tunnel\n")

	return nil
}

// runAudioSetupMac installs and configures PulseAudio on macOS
func (f *CommandFactory) runAudioSetupMac(ctx context.Context) error {
	// Check if running on macOS
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("this command is only supported on macOS")
	}

	fmt.Println("Setting up audio on macOS...")

	// Check if Homebrew is installed
	if _, err := exec.LookPath("brew"); err != nil {
		return fmt.Errorf("Homebrew not found. Please install it from https://brew.sh")
	}
	color.Printf("{green}✓{reset} Homebrew found\n")

	// Check if PulseAudio is already installed
	pulseInstalled := false
	if _, err := exec.LookPath("pulseaudio"); err == nil {
		pulseInstalled = true
		color.Printf("{green}✓{reset} PulseAudio already installed\n")
	}

	// Install PulseAudio if needed
	if !pulseInstalled {
		color.Printf("{cyan}→{reset} Installing PulseAudio via Homebrew...\n")
		brewCmd := exec.CommandContext(ctx, "brew", "install", "pulseaudio")
		brewCmd.Stdout = os.Stdout
		brewCmd.Stderr = os.Stderr
		if err := brewCmd.Run(); err != nil {
			return fmt.Errorf("failed to install PulseAudio: %w", err)
		}
		color.Printf("{green}✓{reset} PulseAudio installed\n")
	}

	// Create PulseAudio config directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	pulseConfigDir := filepath.Join(homeDir, ".config", "pulse")
	if err := os.MkdirAll(pulseConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create pulse config directory: %w", err)
	}

	// Write PulseAudio config for tunnel connection
	// Get audio port from config (default 4713)
	audioPort := 4713
	if f.Config != nil && f.Config.AudioPort > 0 {
		audioPort = f.Config.AudioPort
	}

	pulseConfig := fmt.Sprintf(`# L8s audio configuration
# Connect to remote PulseAudio server via SSH tunnel
load-module module-native-protocol-tcp
set-default-sink tunnel-sink

# Auto-connect to the tunneled audio server
.ifexists module-tunnel-sink.so
load-module module-tunnel-sink server=tcp:localhost:%d sink_name=l8s-remote
.endif
`, audioPort)

	configPath := filepath.Join(pulseConfigDir, "default.pa")
	if err := os.WriteFile(configPath, []byte(pulseConfig), 0644); err != nil {
		return fmt.Errorf("failed to write PulseAudio config: %w", err)
	}
	color.Printf("{green}✓{reset} PulseAudio configured at %s\n", configPath)

	color.Printf("\n{cyan}Next steps:{reset}\n")
	color.Printf("1. Run {bold}l8s audio connect{reset} to start the audio tunnel\n")
	color.Printf("2. Audio from containers will play through your Mac speakers\n")

	return nil
}

// runAudioConnect starts the audio SSH tunnel to the remote host
func (f *CommandFactory) runAudioConnect(ctx context.Context) error {
	fmt.Println("Starting audio tunnel...")

	// Check if tunnel is already running by checking control socket
	controlPath := filepath.Join(os.Getenv("HOME"), ".ssh", "control-*@l8s-audio:*")
	matches, _ := filepath.Glob(controlPath)
	if len(matches) > 0 {
		color.Printf("{yellow}!{reset} Audio tunnel already connected\n")
		return nil
	}

	// Start SSH tunnel in background
	// -N = no remote command, -f = background, l8s-audio = the SSH host config
	sshCmd := exec.Command("ssh", "-N", "-f", "l8s-audio")
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	if err := sshCmd.Run(); err != nil {
		return fmt.Errorf("failed to start audio tunnel: %w", err)
	}

	audioPort := 4713
	if f.Config != nil && f.Config.AudioPort > 0 {
		audioPort = f.Config.AudioPort
	}

	color.Printf("{green}✓{reset} Audio tunnel connected (localhost:%d → remote)\n", audioPort)
	color.Printf("  Audio from containers will now play through your speakers\n")

	return nil
}

// runAudioDisconnect stops the audio SSH tunnel
func (f *CommandFactory) runAudioDisconnect(ctx context.Context) error {
	fmt.Println("Stopping audio tunnel...")

	// Use SSH control master to close the connection
	// -O exit sends exit command to the master process
	sshCmd := exec.Command("ssh", "-O", "exit", "l8s-audio")
	output, err := sshCmd.CombinedOutput()

	if err != nil {
		// Check if it just wasn't running
		if strings.Contains(string(output), "No such file") ||
			strings.Contains(string(output), "not running") ||
			strings.Contains(string(output), "No ControlPath") {
			color.Printf("{yellow}!{reset} Audio tunnel was not running\n")
			return nil
		}
		return fmt.Errorf("failed to stop audio tunnel: %w", err)
	}

	color.Printf("{green}✓{reset} Audio tunnel disconnected\n")
	return nil
}

// runAudioStatus shows the current audio tunnel status
func (f *CommandFactory) runAudioStatus(ctx context.Context) error {
	fmt.Println("Audio Configuration Status")
	fmt.Println("══════════════════════════")
	fmt.Println()

	// Show configuration
	audioPort := 4713
	audioSocketPath := "/run/user/1000/pulse"
	audioEnabled := true

	if f.Config != nil {
		if f.Config.AudioPort > 0 {
			audioPort = f.Config.AudioPort
		}
		if f.Config.AudioSocketPath != "" {
			audioSocketPath = f.Config.AudioSocketPath
		}
		audioEnabled = f.Config.AudioEnabled
	}

	// Configuration section
	color.Printf("{bold}Configuration:{reset}\n")
	if audioEnabled {
		color.Printf("  Enabled:     {green}yes{reset}\n")
	} else {
		color.Printf("  Enabled:     {yellow}no{reset}\n")
	}
	color.Printf("  Audio Port:  %d\n", audioPort)
	color.Printf("  Socket Path: %s\n", audioSocketPath)
	fmt.Println()

	// Tunnel status section
	color.Printf("{bold}SSH Tunnel:{reset}\n")
	if isAudioTunnelConnected() {
		color.Printf("  Status:      {green}✓ connected{reset}\n")
		color.Printf("  Local:       localhost:%d\n", audioPort)
	} else {
		color.Printf("  Status:      {yellow}✗ disconnected{reset}\n")
		color.Printf("  Run:         {bold}l8s audio connect{reset} to start\n")
	}
	fmt.Println()

	// Mac setup check (only on macOS)
	if runtime.GOOS == "darwin" {
		color.Printf("{bold}Mac Setup:{reset}\n")
		if _, err := exec.LookPath("pulseaudio"); err == nil {
			color.Printf("  PulseAudio:  {green}✓ installed{reset}\n")
		} else {
			color.Printf("  PulseAudio:  {yellow}✗ not installed{reset}\n")
			color.Printf("  Run:         {bold}l8s audio setup-mac{reset} to install\n")
		}
		fmt.Println()
	}

	// Quick help
	color.Printf("{bold}Commands:{reset}\n")
	color.Printf("  l8s audio connect      Start audio tunnel\n")
	color.Printf("  l8s audio disconnect   Stop audio tunnel\n")
	color.Printf("  l8s audio setup-host   Configure remote host\n")
	color.Printf("  l8s audio setup-mac    Install PulseAudio on Mac\n")

	return nil
}

package cli

import (
	"fmt"

	"github.com/l8s/l8s/pkg/config"
	"github.com/l8s/l8s/pkg/container"
	"github.com/l8s/l8s/pkg/git"
	"github.com/l8s/l8s/pkg/ssh"
	"github.com/spf13/cobra"
)

// CommandFactory creates CLI commands with injected dependencies
type CommandFactory struct {
	Config       *config.Config
	ContainerMgr ContainerManager
	GitClient    GitClient
	SSHClient    SSHClient
}

// NewCommandFactory creates a factory with real dependencies
func NewCommandFactory() (*CommandFactory, error) {
	cfg, err := config.Load(config.GetConfigPath())
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	podmanClient, err := container.NewPodmanClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create podman client: %w", err)
	}

	containerConfig := container.Config{
		SSHPortStart:    cfg.SSHPortStart,
		BaseImage:       cfg.BaseImage,
		ContainerPrefix: cfg.ContainerPrefix,
		ContainerUser:   cfg.ContainerUser,
	}

	return &CommandFactory{
		Config:       cfg,
		ContainerMgr: container.NewManager(podmanClient, containerConfig),
		GitClient:    &gitClientAdapter{},
		SSHClient:    &sshClientAdapter{},
	}, nil
}

// NewTestCommandFactory creates a factory with mock dependencies
func NewTestCommandFactory(
	cfg *config.Config,
	containerMgr ContainerManager,
	gitClient GitClient,
	sshClient SSHClient,
) *CommandFactory {
	return &CommandFactory{
		Config:       cfg,
		ContainerMgr: containerMgr,
		GitClient:    gitClient,
		SSHClient:    sshClient,
	}
}

// CreateCmd returns the create command with injected dependencies
func (f *CommandFactory) CreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name> <git-url> [branch]",
		Short: "Create a new development container",
		Long:  `Creates a new development container with SSH access and clones the specified git repository.`,
		Args:  cobra.RangeArgs(2, 3),
		RunE:  f.runCreate,
	}
}

// SSHCmd returns the ssh command with injected dependencies
func (f *CommandFactory) SSHCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ssh <name>",
		Short: "SSH into a container",
		Args:  cobra.ExactArgs(1),
		RunE:  f.runSSH,
	}
}

// ListCmd returns the list command with injected dependencies
func (f *CommandFactory) ListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List all l8s containers",
		RunE:    f.runList,
		Aliases: []string{"ls"},
	}
}

// StartCmd returns the start command with injected dependencies
func (f *CommandFactory) StartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <name>",
		Short: "Start a stopped container",
		Args:  cobra.ExactArgs(1),
		RunE:  f.runStart,
	}
}

// StopCmd returns the stop command with injected dependencies
func (f *CommandFactory) StopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <name>",
		Short: "Stop a running container",
		Args:  cobra.ExactArgs(1),
		RunE:  f.runStop,
	}
}

// RemoveCmd returns the remove command with injected dependencies
func (f *CommandFactory) RemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove <name>",
		Short:   "Remove a container",
		Args:    cobra.ExactArgs(1),
		RunE:    f.runRemove,
		Aliases: []string{"rm"},
	}
	
	cmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	cmd.Flags().Bool("keep-volumes", false, "Keep volumes when removing container")
	
	return cmd
}

// InfoCmd returns the info command with injected dependencies
func (f *CommandFactory) InfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <name>",
		Short: "Show detailed container information",
		Args:  cobra.ExactArgs(1),
		RunE:  f.runInfo,
	}
}

// BuildCmd returns the build command with injected dependencies
func (f *CommandFactory) BuildCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build",
		Short: "Build or rebuild the base container image",
		RunE:  f.runBuild,
	}
}

// RemoteCmd returns the remote command with subcommands
func (f *CommandFactory) RemoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remote",
		Short: "Manage git remotes for containers",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "add <name>",
			Short: "Add git remote for existing container",
			Args:  cobra.ExactArgs(1),
			RunE:  f.runRemoteAdd,
		},
		&cobra.Command{
			Use:   "remove <name>",
			Short: "Remove git remote for container",
			Args:  cobra.ExactArgs(1),
			RunE:  f.runRemoteRemove,
		},
	)

	return cmd
}

// ExecCmd returns the exec command with injected dependencies
func (f *CommandFactory) ExecCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "exec <name> <command> [args...]",
		Short: "Execute command in container",
		Args:  cobra.MinimumNArgs(2),
		RunE:  f.runExec,
	}
}

// InitCmd returns the init command with injected dependencies
func (f *CommandFactory) InitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize l8s configuration",
		Long: `Initialize l8s configuration by setting up remote server connection details.
	
l8s ONLY supports remote container management for security isolation.
You'll need:
- A remote server with Podman installed
- SSH access to the server
- SSH key authentication configured`,
		RunE: f.runInit,
	}
}

// gitClientAdapter adapts the git package functions to the GitClient interface
type gitClientAdapter struct{}

func (g *gitClientAdapter) CloneRepository(repoPath, gitURL, branch string) error {
	return git.CloneRepository(repoPath, gitURL, branch)
}

func (g *gitClientAdapter) AddRemote(repoPath, remoteName, remoteURL string) error {
	return git.AddRemote(repoPath, remoteName, remoteURL)
}

func (g *gitClientAdapter) RemoveRemote(repoPath, remoteName string) error {
	return git.RemoveRemote(repoPath, remoteName)
}

func (g *gitClientAdapter) ListRemotes(repoPath string) (map[string]string, error) {
	return git.ListRemotes(repoPath)
}

func (g *gitClientAdapter) SetUpstream(repoPath, remoteName, branch string) error {
	return git.SetUpstream(repoPath, remoteName, branch)
}

func (g *gitClientAdapter) CurrentBranch(repoPath string) (string, error) {
	return git.GetCurrentBranch(repoPath)
}

func (g *gitClientAdapter) ValidateGitURL(gitURL string) error {
	return git.ValidateGitURL(gitURL)
}

// sshClientAdapter adapts the ssh package functions to the SSHClient interface
type sshClientAdapter struct{}

func (s *sshClientAdapter) ReadPublicKey(keyPath string) (string, error) {
	return ssh.ReadPublicKey(keyPath)
}

func (s *sshClientAdapter) FindSSHPublicKey() (string, error) {
	return ssh.FindSSHPublicKey()
}

func (s *sshClientAdapter) AddSSHConfig(name, hostname string, port int, user string) error {
	return ssh.AddSSHConfig(name, hostname, port, user)
}

func (s *sshClientAdapter) RemoveSSHConfig(name string) error {
	return ssh.RemoveSSHConfig(name)
}

func (s *sshClientAdapter) GenerateAuthorizedKeys(publicKey string) string {
	return ssh.GenerateAuthorizedKeys(publicKey)
}

func (s *sshClientAdapter) IsPortAvailable(port int) bool {
	return ssh.IsPortAvailable(port)
}

func (s *sshClientAdapter) ValidatePublicKey(key string) error {
	return ssh.ValidatePublicKey(key)
}
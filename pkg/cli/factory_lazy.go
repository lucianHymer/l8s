package cli

import (
	"fmt"
	"sync"

	"l8s/pkg/config"
	"l8s/pkg/container"
	"github.com/spf13/cobra"
)

// LazyCommandFactory creates CLI commands with lazy dependency initialization
type LazyCommandFactory struct {
	// Dependencies that will be initialized lazily
	Config       *config.Config
	ContainerMgr ContainerManager
	GitClient    GitClient
	SSHClient    SSHClient

	// Initialization tracking
	once        sync.Once
	initError   error
	initializer func() error // For testing
}

// NewLazyCommandFactory creates a factory that delays initialization
func NewLazyCommandFactory() *LazyCommandFactory {
	f := &LazyCommandFactory{}
	f.initializer = f.defaultInitializer
	return f
}

// defaultInitializer performs the actual initialization
func (f *LazyCommandFactory) defaultInitializer() error {
	cfg, err := config.Load(config.GetConfigPath())
	if err != nil {
		return fmt.Errorf("failed to load config: %w\n\nRun 'l8s init' to configure l8s for your remote server", err)
	}
	
	// Validate that SSH configs match the active connection
	address, err := cfg.GetActiveAddress()
	if err != nil {
		return fmt.Errorf("failed to get active connection: %w", err)
	}
	
	if err := ValidateSSHConfigsMatchConnection(address); err != nil {
		return fmt.Errorf("SSH configs don't match active connection '%s': %w\n\nRun 'l8s connection switch %s' to fix this",
			cfg.ActiveConnection, err, cfg.ActiveConnection)
	}

	podmanClient, err := container.NewPodmanClient()
	if err != nil {
		return fmt.Errorf("failed to create podman client: %w", err)
	}

	// Get the remote host from active connection
	remoteHost := ""
	if activeAddr, err := cfg.GetActiveAddress(); err == nil {
		remoteHost = activeAddr
	}
	
	containerConfig := container.Config{
		SSHPortStart:     cfg.SSHPortStart,
		WebPortStart:     cfg.WebPortStart,
		BaseImage:        cfg.BaseImage,
		ContainerPrefix:  cfg.ContainerPrefix,
		ContainerUser:    cfg.ContainerUser,
		DotfilesPath:     cfg.DotfilesPath,
		CAPrivateKeyPath: cfg.CAPrivateKeyPath,
		CAPublicKeyPath:  cfg.CAPublicKeyPath,
		KnownHostsPath:   cfg.KnownHostsPath,
		RemoteHost:       remoteHost,
		GitHubToken:      cfg.GitHubToken,
	}

	f.Config = cfg
	f.ContainerMgr = container.NewManager(podmanClient, containerConfig)
	f.GitClient = &gitClientAdapter{}
	f.SSHClient = &sshClientAdapter{}

	return nil
}

// ensureInitialized performs lazy initialization
func (f *LazyCommandFactory) ensureInitialized() error {
	f.once.Do(func() {
		if f.initializer != nil {
			f.initError = f.initializer()
		}
	})
	return f.initError
}

// CreateCmd returns the create command with lazy initialization
func (f *LazyCommandFactory) CreateCmd() *cobra.Command {
	var dotfilesPath string
	var branch string
	
	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a new development container",
		GroupID: "repo-maintenance",
		Long:  `Creates a new development container for the current git worktree.

The container name is automatically generated from the repository name and worktree path.
The container will be initialized with an empty git repository configured to receive pushes.
The current branch (or specified branch) will be pushed to populate the container.
A git remote will be added to your local repository for easy code synchronization.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := f.ensureInitialized(); err != nil {
				return err
			}
			
			// Set CLI dotfiles path if provided
			if dotfilesPath != "" {
				if cm, ok := f.ContainerMgr.(*container.Manager); ok {
					cm.SetCLIDotfilesPath(dotfilesPath)
				}
			}
			
			// Delegate to the original factory implementation
			origFactory := &CommandFactory{
				Config:       f.Config,
				ContainerMgr: f.ContainerMgr,
				GitClient:    f.GitClient,
				SSHClient:    f.SSHClient,
			}
			return origFactory.runCreate(cmd, args)
		},
	}
	
	cmd.Flags().StringVar(&dotfilesPath, "dotfiles-path", "", "Path to dotfiles directory to copy to the container")
	cmd.Flags().StringVar(&branch, "branch", "", "Git branch to push to the container (defaults to current branch)")
	
	return cmd
}

// SSHCmd returns the ssh command with lazy initialization
func (f *LazyCommandFactory) SSHCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "ssh",
		Short:   "SSH into the container for the current worktree",
		GroupID: "working",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := f.ensureInitialized(); err != nil {
				return err
			}
			origFactory := &CommandFactory{
				Config:       f.Config,
				ContainerMgr: f.ContainerMgr,
				GitClient:    f.GitClient,
				SSHClient:    f.SSHClient,
			}
			return origFactory.runSSH(cmd, args)
		},
	}
}

// ListCmd returns the list command with lazy initialization
func (f *LazyCommandFactory) ListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List all l8s containers",
		GroupID: "container",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := f.ensureInitialized(); err != nil {
				return err
			}
			origFactory := &CommandFactory{
				Config:       f.Config,
				ContainerMgr: f.ContainerMgr,
				GitClient:    f.GitClient,
				SSHClient:    f.SSHClient,
			}
			return origFactory.runList(cmd, args)
		},
	}
}

// StartCmd returns the start command with lazy initialization
func (f *LazyCommandFactory) StartCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "start <name>",
		Short:   "Start a stopped container",
		GroupID: "container",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := f.ensureInitialized(); err != nil {
				return err
			}
			origFactory := &CommandFactory{
				Config:       f.Config,
				ContainerMgr: f.ContainerMgr,
				GitClient:    f.GitClient,
				SSHClient:    f.SSHClient,
			}
			return origFactory.runStart(cmd, args)
		},
	}
}

// StopCmd returns the stop command with lazy initialization
func (f *LazyCommandFactory) StopCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "stop <name>",
		Short:   "Stop a running container",
		GroupID: "container",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := f.ensureInitialized(); err != nil {
				return err
			}
			origFactory := &CommandFactory{
				Config:       f.Config,
				ContainerMgr: f.ContainerMgr,
				GitClient:    f.GitClient,
				SSHClient:    f.SSHClient,
			}
			return origFactory.runStop(cmd, args)
		},
	}
}

// RemoveCmd returns the remove command with lazy initialization
func (f *LazyCommandFactory) RemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove",
		Short:   "Remove the container for the current worktree",
		GroupID: "repo-maintenance",
		Args:    cobra.NoArgs,
		Aliases: []string{"rm"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := f.ensureInitialized(); err != nil {
				return err
			}
			origFactory := &CommandFactory{
				Config:       f.Config,
				ContainerMgr: f.ContainerMgr,
				GitClient:    f.GitClient,
				SSHClient:    f.SSHClient,
			}
			return origFactory.runRemove(cmd, args)
		},
	}
	
	cmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	cmd.Flags().Bool("keep-volumes", false, "Keep volumes when removing container")
	
	return cmd
}

// InfoCmd returns the info command with lazy initialization
func (f *LazyCommandFactory) InfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "info <name>",
		Short:   "Show detailed container information",
		GroupID: "container",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := f.ensureInitialized(); err != nil {
				return err
			}
			origFactory := &CommandFactory{
				Config:       f.Config,
				ContainerMgr: f.ContainerMgr,
				GitClient:    f.GitClient,
				SSHClient:    f.SSHClient,
			}
			return origFactory.runInfo(cmd, args)
		},
	}
}

// BuildCmd returns the build command with lazy initialization
func (f *LazyCommandFactory) BuildCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "build",
		Short:   "Build or rebuild the base container image",
		GroupID: "container",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := f.ensureInitialized(); err != nil {
				return err
			}
			origFactory := &CommandFactory{
				Config:       f.Config,
				ContainerMgr: f.ContainerMgr,
				GitClient:    f.GitClient,
				SSHClient:    f.SSHClient,
			}
			return origFactory.runBuild(cmd, args)
		},
	}
}

// RemoteCmd returns the remote command with subcommands and lazy initialization
func (f *LazyCommandFactory) RemoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remote",
		Short:   "Manage git remotes for containers",
		GroupID: "container",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "add <name>",
			Short: "Add git remote for existing container",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := f.ensureInitialized(); err != nil {
					return err
				}
				origFactory := &CommandFactory{
					Config:       f.Config,
					ContainerMgr: f.ContainerMgr,
					GitClient:    f.GitClient,
					SSHClient:    f.SSHClient,
				}
				return origFactory.runRemoteAdd(cmd, args)
			},
		},
		&cobra.Command{
			Use:   "remove <name>",
			Short: "Remove git remote for container",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := f.ensureInitialized(); err != nil {
					return err
				}
				origFactory := &CommandFactory{
					Config:       f.Config,
					ContainerMgr: f.ContainerMgr,
					GitClient:    f.GitClient,
					SSHClient:    f.SSHClient,
				}
				return origFactory.runRemoteRemove(cmd, args)
			},
		},
	)

	return cmd
}

// ExecCmd returns the exec command with lazy initialization
func (f *LazyCommandFactory) ExecCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "exec <command> [args...]",
		Short:   "Execute command in the container for the current worktree",
		GroupID: "working",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := f.ensureInitialized(); err != nil {
				return err
			}
			origFactory := &CommandFactory{
				Config:       f.Config,
				ContainerMgr: f.ContainerMgr,
				GitClient:    f.GitClient,
				SSHClient:    f.SSHClient,
			}
			return origFactory.runExec(cmd, args)
		},
	}
}

// InitCmd returns the init command without lazy initialization
func (f *LazyCommandFactory) InitCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "init",
		Short:   "Initialize l8s configuration",
		GroupID: "setup",
		Long: `Initialize l8s configuration by setting up remote server connection details.
	
l8s ONLY supports remote container management for security isolation.
You'll need:
- A remote server with Podman installed
- SSH access to the server
- SSH key authentication configured`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Init doesn't need dependencies, create a minimal factory
			origFactory := &CommandFactory{}
			return origFactory.runInit(cmd, args)
		},
	}
}

// RebuildCmd returns the rebuild command with lazy initialization
func (f *LazyCommandFactory) RebuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rebuild",
		Short:   "Rebuild the container for the current worktree while preserving data",
		GroupID: "repo-maintenance",
		Long: `Rebuild recreates a container with the latest base image while preserving:
- All workspace and home directory data (volumes)
- SSH port assignment
- Container name and configuration`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := f.ensureInitialized(); err != nil {
				return err
			}
			
			// Get flags
			build, _ := cmd.Flags().GetBool("build")
			skipBuild, _ := cmd.Flags().GetBool("skip-build")
			
			// Validate mutually exclusive flags
			if build && skipBuild {
				return fmt.Errorf("--build and --skip-build are mutually exclusive")
			}
			
			origFactory := &CommandFactory{
				Config:       f.Config,
				ContainerMgr: f.ContainerMgr,
				GitClient:    f.GitClient,
				SSHClient:    f.SSHClient,
			}
			return origFactory.runRebuild(cmd, args)
		},
	}
	
	cmd.Flags().Bool("build", false, "Build image before rebuilding")
	cmd.Flags().Bool("skip-build", false, "Skip build and use existing image")
	
	return cmd
}

// RebuildAllCmd returns the rebuild-all command with lazy initialization
func (f *LazyCommandFactory) RebuildAllCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rebuild-all",
		Short:   "Rebuild all containers with updated image",
		GroupID: "container",
		Long: `Rebuild all containers with the latest base image while preserving their data.
		
This is useful after updating the base image or when you want to refresh all containers.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := f.ensureInitialized(); err != nil {
				return err
			}
			origFactory := &CommandFactory{
				Config:       f.Config,
				ContainerMgr: f.ContainerMgr,
				GitClient:    f.GitClient,
				SSHClient:    f.SSHClient,
			}
			return origFactory.runRebuildAll(cmd, args)
		},
	}
	
	cmd.Flags().Bool("force", false, "Skip confirmation prompt")
	cmd.Flags().Bool("build", false, "Build image before rebuilding")
	cmd.Flags().Bool("skip-build", false, "Skip build and use existing image")
	
	return cmd
}

// PasteCmd returns the paste command with lazy initialization
func (f *LazyCommandFactory) PasteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "paste [name]",
		Short:   "Paste clipboard content to container",
		GroupID: "working",
		Long: `Paste clipboard content (image or text) from your local machine to the container for the current worktree.
Content is saved to /tmp/claude-clipboard/ in the container.

Without a custom name, files are saved as clipboard.png or clipboard.txt (replacing any existing default files).
With a custom name, files are saved as clipboard-<name>.png or clipboard-<name>.txt (preserving existing files).`,
		Args: cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := f.ensureInitialized(); err != nil {
				return err
			}
			origFactory := &CommandFactory{
				Config:       f.Config,
				ContainerMgr: f.ContainerMgr,
				GitClient:    f.GitClient,
				SSHClient:    f.SSHClient,
			}
			return origFactory.runPaste(cmd, args)
		},
	}
}

// PushCmd returns the push command with lazy initialization
func (f *LazyCommandFactory) PushCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "push",
		Short:   "Push current branch to container (fast-forward only)",
		GroupID: "working",
		Long: `Push the current git branch to the container for this worktree.
		
The push will fail if it would overwrite changes in the container (non-fast-forward).
Use 'l8s pull' first if the container has diverged.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := f.ensureInitialized(); err != nil {
				return err
			}
			origFactory := &CommandFactory{
				Config:       f.Config,
				ContainerMgr: f.ContainerMgr,
				GitClient:    f.GitClient,
				SSHClient:    f.SSHClient,
			}
			return origFactory.runPush(cmd, args)
		},
	}
}

// PullCmd returns the pull command with lazy initialization
func (f *LazyCommandFactory) PullCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "pull",
		Short:   "Pull changes from container (fast-forward only)",
		GroupID: "working",
		Long: `Pull changes from the container to your local worktree.
		
The pull will fail if it would overwrite local changes (non-fast-forward).
Resolve conflicts manually if your local branch has diverged.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := f.ensureInitialized(); err != nil {
				return err
			}
			origFactory := &CommandFactory{
				Config:       f.Config,
				ContainerMgr: f.ContainerMgr,
				GitClient:    f.GitClient,
				SSHClient:    f.SSHClient,
			}
			return origFactory.runPull(cmd, args)
		},
	}
}

// StatusCmd returns the status command with lazy initialization
func (f *LazyCommandFactory) StatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "status",
		Short:   "Show status of container for current worktree",
		GroupID: "working",
		Long:    `Display the status and information about the container associated with the current git worktree.`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := f.ensureInitialized(); err != nil {
				return err
			}
			origFactory := &CommandFactory{
				Config:       f.Config,
				ContainerMgr: f.ContainerMgr,
				GitClient:    f.GitClient,
				SSHClient:    f.SSHClient,
			}
			return origFactory.runStatus(cmd, args)
		},
	}
}

// ConnectionCmd returns the connection command with subcommands
func (f *LazyCommandFactory) ConnectionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "connection",
		Short:   "Manage Podman connection configurations",
		GroupID: "setup",
		Long:    "Manage multiple Podman connection configurations for different network access scenarios",
	}
	
	// List subcommand
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all configured Podman connections",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := f.ensureInitialized(); err != nil {
				return err
			}
			return (&ConnectionListCommand{config: f.Config}).Execute(cmd.Context())
		},
	})
	
	// Show subcommand
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show current Podman connection details",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := f.ensureInitialized(); err != nil {
				return err
			}
			return (&ConnectionShowCommand{config: f.Config}).Execute(cmd.Context())
		},
	})
	
	// Switch subcommand
	var dryRun bool
	switchCmd := &cobra.Command{
		Use:   "switch <name>",
		Short: "Switch to a different Podman connection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := f.ensureInitialized(); err != nil {
				return err
			}
			return (&ConnectionSwitchCommand{
				config:           f.Config,
				targetConnection: args[0],
				dryRun:           dryRun,
			}).Execute(cmd.Context())
		},
	}
	switchCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would change without making changes")
	cmd.AddCommand(switchCmd)
	
	return cmd
}

// InstallZSHPluginCmd creates the install-zsh-plugin command
func (f *LazyCommandFactory) InstallZSHPluginCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "install-zsh-plugin",
		Short:   "Install l8s ZSH completion plugin for Oh My Zsh",
		GroupID: "setup",
		Long: `Install the l8s ZSH completion plugin to enable tab completion for l8s commands.

This command will:
  1. Install the plugin to ~/.oh-my-zsh/custom/plugins/l8s
  2. Update your .zshrc to load the plugin

Prerequisites:
  - Oh My Zsh must be installed (https://ohmyz.sh/)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// InstallZSHPlugin doesn't need dependencies, create a minimal factory
			origFactory := &CommandFactory{}
			return origFactory.runInstallZSHPlugin(cmd.Context())
		},
	}
}

// TeamCmd creates the team command for managing dtach sessions
func (f *LazyCommandFactory) TeamCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "team [session-name]",
		Short:   "Manage persistent team sessions in container",
		GroupID: "working",
		Long: `Manage persistent terminal sessions using dtach in the container for the current git repository.

Team sessions allow you to:
  - Create persistent terminal sessions that survive SSH disconnections
  - Resume work after network interruptions or laptop sleep
  - Share terminal sessions for pair programming
  - Keep long-running tasks alive across connections

With no arguments, lists active sessions.
With a session name, joins or creates that session.

Use Ctrl+\ to detach from a session (it will continue running).`,
		Args: cobra.MaximumNArgs(1),
		Example: `  # List active sessions
  l8s team

  # Join or create a "backend" session
  l8s team backend

  # Join or create a "debugging" session
  l8s team debugging`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := f.ensureInitialized(); err != nil {
				return err
			}
			origFactory := &CommandFactory{
				Config:       f.Config,
				ContainerMgr: f.ContainerMgr,
				GitClient:    f.GitClient,
				SSHClient:    f.SSHClient,
			}

			// No args means list
			if len(args) == 0 {
				return origFactory.runTeamList(cmd.Context())
			}

			// With session name, join/create
			return origFactory.runTeamJoin(cmd.Context(), args[0])
		},
	}

	return cmd
}
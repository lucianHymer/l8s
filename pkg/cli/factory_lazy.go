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

	podmanClient, err := container.NewPodmanClient()
	if err != nil {
		return fmt.Errorf("failed to create podman client: %w", err)
	}

	containerConfig := container.Config{
		SSHPortStart:    cfg.SSHPortStart,
		BaseImage:       cfg.BaseImage,
		ContainerPrefix: cfg.ContainerPrefix,
		ContainerUser:   cfg.ContainerUser,
		DotfilesPath:    cfg.DotfilesPath,
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
	
	cmd := &cobra.Command{
		Use:   "create <name> <git-url> [branch]",
		Short: "Create a new development container",
		Long:  `Creates a new development container with SSH access and clones the specified git repository.`,
		Args:  cobra.RangeArgs(2, 3),
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
	
	return cmd
}

// SSHCmd returns the ssh command with lazy initialization
func (f *LazyCommandFactory) SSHCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ssh <name>",
		Short: "SSH into a container",
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
			return origFactory.runSSH(cmd, args)
		},
	}
}

// ListCmd returns the list command with lazy initialization
func (f *LazyCommandFactory) ListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List all l8s containers",
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
		Use:   "start <name>",
		Short: "Start a stopped container",
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
			return origFactory.runStart(cmd, args)
		},
	}
}

// StopCmd returns the stop command with lazy initialization
func (f *LazyCommandFactory) StopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <name>",
		Short: "Stop a running container",
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
			return origFactory.runStop(cmd, args)
		},
	}
}

// RemoveCmd returns the remove command with lazy initialization
func (f *LazyCommandFactory) RemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove <name>",
		Short:   "Remove a container",
		Args:    cobra.ExactArgs(1),
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
		Use:   "info <name>",
		Short: "Show detailed container information",
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
			return origFactory.runInfo(cmd, args)
		},
	}
}

// BuildCmd returns the build command with lazy initialization
func (f *LazyCommandFactory) BuildCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build",
		Short: "Build or rebuild the base container image",
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
		Use:   "remote",
		Short: "Manage git remotes for containers",
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
		Use:   "exec <name> <command> [args...]",
		Short: "Execute command in container",
		Args:  cobra.MinimumNArgs(2),
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
		Use:   "init",
		Short: "Initialize l8s configuration",
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
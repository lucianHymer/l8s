package main

import (
	"log/slog"
	"os"
	"strings"

	"l8s/pkg/cli"
	"l8s/pkg/errors"
	"l8s/pkg/logging"
	"github.com/spf13/cobra"
)

func main() {
	// Initialize logging
	initLogging()

	// Create root command
	rootCmd := &cobra.Command{
		Use:   "l8s",
		Short: "The container management system that really ties the room together",
		Long: `l8s (Lebowskis) is a Podman-based development container management tool 
that creates isolated, SSH-accessible development environments.

Each container is a fully-featured Linux environment with development tools,
accessible via SSH using key-based authentication.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Define command groups for better organization
	rootCmd.AddGroup(
		&cobra.Group{
			ID:    "working",
			Title: "Working Commands (requires git repo)",
		},
		&cobra.Group{
			ID:    "repo-maintenance",
			Title: "Repository Maintenance (requires git repo)",
		},
		&cobra.Group{
			ID:    "container",
			Title: "Container Management (works anywhere)",
		},
		&cobra.Group{
			ID:    "setup",
			Title: "Setup & Configuration",
		},
	)

	// Create lazy command factory
	factory := cli.NewLazyCommandFactory()

	// Add commands from factory - these are lightweight and don't require config
	rootCmd.AddCommand(
		factory.InitCmd(),    // Init doesn't require config
		factory.CreateCmd(),
		factory.SSHCmd(),
		factory.ListCmd(),
		factory.StartCmd(),
		factory.StopCmd(),
		factory.RemoveCmd(),
		factory.RebuildCmd(),
		factory.RebuildAllCmd(),
		factory.InfoCmd(),
		factory.BuildCmd(),
		factory.RemoteCmd(),
		factory.ExecCmd(),
		factory.PasteCmd(),
		factory.TeamCmd(),
		factory.PushCmd(),
		factory.PullCmd(),
		factory.StatusCmd(),
		factory.ConnectionCmd(),
		factory.InstallZSHPluginCmd(),
		factory.AudioCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		errors.PrintError(err)
		os.Exit(1)
	}
}

func initLogging() {
	// Get log level from environment
	level := slog.LevelInfo
	if envLevel := os.Getenv("L8S_LOG_LEVEL"); envLevel != "" {
		switch strings.ToLower(envLevel) {
		case "debug":
			level = slog.LevelDebug
		case "warn", "warning":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		}
	}

	// Get log format from environment
	format := "text"
	if envFormat := os.Getenv("L8S_LOG_FORMAT"); envFormat != "" {
		format = strings.ToLower(envFormat)
	}

	// Create logger configuration
	cfg := logging.Config{
		Level:  strings.ToLower(level.String()),
		Format: format,
		Output: "stderr",
	}

	// Create and set logger
	logger, _ := logging.NewLogger(cfg)
	logging.SetDefault(logger)
}
package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/l8s/l8s/cmd/commands"
	"github.com/l8s/l8s/pkg/cli"
	"github.com/l8s/l8s/pkg/logging"
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
	}

	// Add init command (doesn't require config)
	rootCmd.AddCommand(commands.InitCmd())

	// Check if this is the init command
	if len(os.Args) > 1 && os.Args[1] == "init" {
		// Execute init command without loading config
		if err := rootCmd.Execute(); err != nil {
			os.Exit(1)
		}
		return
	}

	// Create command factory (requires config)
	factory, err := cli.NewCommandFactory()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nRun 'l8s init' to configure l8s for your remote server.\n")
		os.Exit(1)
	}

	// Add commands from factory
	rootCmd.AddCommand(
		factory.CreateCmd(),
		factory.SSHCmd(),
		factory.ListCmd(),
		factory.StartCmd(),
		factory.StopCmd(),
		factory.RemoveCmd(),
		factory.InfoCmd(),
		factory.BuildCmd(),
		factory.RemoteCmd(),
		factory.ExecCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
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
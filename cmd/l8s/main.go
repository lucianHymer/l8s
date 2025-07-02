package main

import (
	"fmt"
	"os"

	"github.com/l8s/l8s/pkg/cli"
	"github.com/spf13/cobra"
)

func main() {
	// Create command factory
	factory, err := cli.NewCommandFactory()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create root command
	rootCmd := &cobra.Command{
		Use:   "l8s",
		Short: "The container management system that really ties the room together",
		Long: `l8s (Lebowskis) is a Podman-based development container management tool 
that creates isolated, SSH-accessible development environments.

Each container is a fully-featured Linux environment with development tools,
accessible via SSH using key-based authentication.`,
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
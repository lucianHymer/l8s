package main

import (
	"fmt"
	"os"

	"github.com/l8s/l8s/cmd/commands"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "l8s",
	Short: "The container management system that really ties the room together",
	Long: `l8s (Lebowskis) is a Podman-based development container management tool 
that creates isolated, SSH-accessible development environments.

Each container is a fully-featured Linux environment with development tools,
accessible via SSH using key-based authentication.`,
}

func main() {
	// Add commands
	rootCmd.AddCommand(commands.CreateCmd())
	rootCmd.AddCommand(commands.SSHCmd())
	rootCmd.AddCommand(commands.ListCmd())
	rootCmd.AddCommand(commands.StopCmd())
	rootCmd.AddCommand(commands.StartCmd())
	rootCmd.AddCommand(commands.RemoveCmd())
	rootCmd.AddCommand(commands.InfoCmd())
	rootCmd.AddCommand(commands.BuildCmd())
	rootCmd.AddCommand(commands.RemoteCmd())
	rootCmd.AddCommand(commands.ExecCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
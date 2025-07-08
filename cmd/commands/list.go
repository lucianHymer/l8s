package commands

import (
	"context"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/l8s/l8s/pkg/config"
	"github.com/l8s/l8s/pkg/container"
	"github.com/spf13/cobra"
)

// ListCmd creates the list command
func ListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all l8s containers",
		Long:  `List all l8s containers with their status.`,
		RunE:  runList,
		Aliases: []string{"ls"},
	}

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load(config.GetConfigPath())
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create Podman client
	podmanClient, err := container.NewPodmanClient()
	if err != nil {
		return fmt.Errorf("failed to create Podman client: %w", err)
	}

	// Create container manager
	managerConfig := container.Config{
		SSHPortStart:    cfg.SSHPortStart,
		BaseImage:       cfg.BaseImage,
		ContainerPrefix: cfg.ContainerPrefix,
		ContainerUser:   cfg.ContainerUser,
	}
	manager := container.NewManager(podmanClient, managerConfig)

	// List containers
	ctx := context.Background()
	containers, err := manager.ListContainers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) == 0 {
		fmt.Println("No l8s containers found.")
		return nil
	}

	// Print table
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tSSH PORT\tGIT REMOTE\tCREATED")
	
	for _, cont := range containers {
		// Remove prefix from name for display
		displayName := cont.Name
		if strings.HasPrefix(displayName, cfg.ContainerPrefix+"-") {
			displayName = strings.TrimPrefix(displayName, cfg.ContainerPrefix+"-")
		}

		// Format git remote status
		gitRemote := "✗"
		if cont.GitURL != "" {
			gitRemote = "✓"
		}

		// Format creation time
		created := formatDuration(time.Since(cont.CreatedAt))

		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
			displayName,
			cont.Status,
			cont.SSHPort,
			gitRemote,
			created,
		)
	}
	
	return w.Flush()
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	} else if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	} else {
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
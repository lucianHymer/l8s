package container

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// GitIdentity represents git user configuration
type GitIdentity struct {
	Name  string
	Email string
}

// GetHostGitConfig reads a git config value from the host system
func GetHostGitConfig(key string) (string, error) {
	cmd := exec.Command("git", "config", "--get", key)
	output, err := cmd.Output()
	if err != nil {
		// If the command fails (e.g., config not set), return empty string
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", nil
		}
		return "", fmt.Errorf("failed to get git config %s: %w", key, err)
	}
	
	// Trim whitespace and newlines
	return strings.TrimSpace(string(output)), nil
}

// ReadHostGitIdentity reads the git user.name and user.email from the host
func ReadHostGitIdentity() (GitIdentity, error) {
	name, err := GetHostGitConfig("user.name")
	if err != nil {
		return GitIdentity{}, fmt.Errorf("failed to read git user.name: %w", err)
	}
	
	email, err := GetHostGitConfig("user.email")
	if err != nil {
		return GitIdentity{}, fmt.Errorf("failed to read git user.email: %w", err)
	}
	
	return GitIdentity{
		Name:  name,
		Email: email,
	}, nil
}

// escapeShellArg escapes a string for use in shell commands
func escapeShellArg(s string) string {
	// Replace single quotes with '\''
	return strings.ReplaceAll(s, "'", "'\"'\"'")
}

// ApplyGitConfigToContainer sets git configuration in the container
func ApplyGitConfigToContainer(ctx context.Context, client PodmanClient, containerName, containerUser string, identity GitIdentity) error {
	// Set user.name if provided
	if identity.Name != "" {
		cmd := []string{"su", "-", containerUser, "-c", fmt.Sprintf("git config --global user.name '%s'", escapeShellArg(identity.Name))}
		if err := client.ExecContainer(ctx, containerName, cmd); err != nil {
			return fmt.Errorf("failed to set git user.name: %w", err)
		}
	}
	
	// Set user.email if provided
	if identity.Email != "" {
		cmd := []string{"su", "-", containerUser, "-c", fmt.Sprintf("git config --global user.email '%s'", escapeShellArg(identity.Email))}
		if err := client.ExecContainer(ctx, containerName, cmd); err != nil {
			return fmt.Errorf("failed to set git user.email: %w", err)
		}
	}
	
	return nil
}
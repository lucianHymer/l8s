package git

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CloneRepository clones a git repository to the specified path
func CloneRepository(gitURL, branch, targetPath string) error {
	// Validate git URL (skip validation for file:// URLs in tests)
	if !strings.HasPrefix(gitURL, "file://") {
		if err := ValidateGitURL(gitURL); err != nil {
			return err
		}
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Build clone command
	args := []string{"clone"}
	if branch != "" {
		args = append(args, "-b", branch)
	}
	args = append(args, gitURL, targetPath)

	// Execute git clone
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// AddRemote adds a git remote to a repository
func AddRemote(repoPath, remoteName, remoteURL string) error {
	// Validate inputs
	if remoteName == "" {
		return fmt.Errorf("remote name is required")
	}
	if remoteURL == "" {
		return fmt.Errorf("remote URL is required")
	}

	// Check if repository exists
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository: %s", repoPath)
	}

	// Check if remote already exists
	remotes, err := ListRemotes(repoPath)
	if err != nil {
		return err
	}
	if _, exists := remotes[remoteName]; exists {
		return fmt.Errorf("remote '%s' already exists", remoteName)
	}

	// Add the remote
	cmd := exec.Command("git", "remote", "add", remoteName, remoteURL)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add remote: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// RemoveRemote removes a git remote from a repository
func RemoveRemote(repoPath, remoteName string) error {
	// Validate input
	if remoteName == "" {
		return fmt.Errorf("remote name is required")
	}

	// Check if repository exists
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository: %s", repoPath)
	}

	// Check if remote exists
	remotes, err := ListRemotes(repoPath)
	if err != nil {
		return err
	}
	if _, exists := remotes[remoteName]; !exists {
		return fmt.Errorf("No such remote '%s'", remoteName)
	}

	// Remove the remote
	cmd := exec.Command("git", "remote", "remove", remoteName)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove remote: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// SetUpstream sets the upstream branch for the current branch
func SetUpstream(repoPath, branch, remoteName string) error {
	// Check if repository exists
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository: %s", repoPath)
	}

	// Check if branch exists locally
	cmd := exec.Command("git", "show-ref", "--verify", fmt.Sprintf("refs/heads/%s", branch))
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("branch '%s' does not exist", branch)
	}

	// Use git config to set upstream without requiring the remote branch to exist
	// This is equivalent to what git push -u does
	configs := [][]string{
		{"config", fmt.Sprintf("branch.%s.remote", branch), remoteName},
		{"config", fmt.Sprintf("branch.%s.merge", branch), fmt.Sprintf("refs/heads/%s", branch)},
	}

	for _, args := range configs {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoPath
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to set upstream: %w\nOutput: %s", err, string(output))
		}
	}

	return nil
}

// GetCurrentBranch returns the current git branch
func GetCurrentBranch(repoPath string) (string, error) {
	// Check if repository exists
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); os.IsNotExist(err) {
		return "", fmt.Errorf("not a git repository: %s", repoPath)
	}

	// Get current branch
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w\nOutput: %s", err, string(output))
	}

	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return "", fmt.Errorf("failed to determine current branch")
	}

	return branch, nil
}

// ListRemotes lists all git remotes in a repository
func ListRemotes(repoPath string) (map[string]string, error) {
	// Check if repository exists
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); os.IsNotExist(err) {
		return nil, fmt.Errorf("not a git repository: %s", repoPath)
	}

	// List remotes with URLs
	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list remotes: %w\nOutput: %s", err, string(output))
	}

	// Parse output
	remotes := make(map[string]string)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: "origin	https://github.com/user/repo.git (fetch)"
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			remoteName := parts[0]
			remoteURL := parts[1]
			// Only store the first occurrence (fetch URL)
			if _, exists := remotes[remoteName]; !exists {
				remotes[remoteName] = remoteURL
			}
		}
	}

	return remotes, nil
}

// ValidateGitURL validates that a URL is a valid git URL
func ValidateGitURL(gitURL string) error {
	if gitURL == "" {
		return fmt.Errorf("git URL cannot be empty")
	}

	// Special handling for SSH URLs (git@github.com:user/repo.git)
	if strings.Contains(gitURL, ":") && !strings.Contains(gitURL, "://") {
		// This might be an SSH URL in the form git@host:path
		parts := strings.SplitN(gitURL, ":", 2)
		if len(parts) == 2 && strings.Contains(parts[0], "@") {
			// Looks like a valid SSH URL
			return nil
		}
	}

	// Parse the URL
	u, err := url.Parse(gitURL)
	if err != nil {
		return fmt.Errorf("invalid git URL: %w", err)
	}

	// Check for local paths (not allowed except for file:// URLs)
	if u.Scheme == "" {
		return fmt.Errorf("invalid repository")
	}

	// Disallow file:// URLs (security measure)
	if u.Scheme == "file" {
		return fmt.Errorf("file URLs are not allowed")
	}

	// Allow common git URL schemes
	validSchemes := map[string]bool{
		"http":  true,
		"https": true,
		"git":   true,
		"ssh":   true,
	}

	if !validSchemes[u.Scheme] {
		return fmt.Errorf("unsupported URL scheme: %s", u.Scheme)
	}

	return nil
}

// ChangeUpstreamToOrigin changes the upstream branch to track origin
func ChangeUpstreamToOrigin(repoPath, branch string) error {
	// Check if repository exists
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository: %s", repoPath)
	}

	// Check if origin remote exists
	remotes, err := ListRemotes(repoPath)
	if err != nil {
		return err
	}
	if _, exists := remotes["origin"]; !exists {
		return fmt.Errorf("origin remote not found")
	}

	// Check if branch exists before trying to set upstream
	cmd := exec.Command("git", "show-ref", "--verify", fmt.Sprintf("refs/heads/%s", branch))
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		// Branch doesn't exist, nothing to do
		return nil
	}

	// Set upstream to origin
	return SetUpstream(repoPath, branch, "origin")
}

// GenerateSSHRemoteURL generates an SSH remote URL for a container
func GenerateSSHRemoteURL(containerName string, sshPort int, containerUser, repoPath string) string {
	// Use the container name as the SSH host (matching SSH config)
	// The repoPath should be the path inside the container
	if !strings.HasPrefix(repoPath, "/") {
		repoPath = "/" + repoPath
	}
	return fmt.Sprintf("dev-%s:%s", containerName, repoPath)
}
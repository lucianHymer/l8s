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
	// Validate git URL
	if err := ValidateGitURL(gitURL); err != nil {
		return err
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
		return fmt.Errorf("remote '%s' does not exist", remoteName)
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

	// Set upstream
	cmd := exec.Command("git", "branch", "--set-upstream-to", fmt.Sprintf("%s/%s", remoteName, branch))
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set upstream: %w\nOutput: %s", err, string(output))
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

	// Parse the URL
	u, err := url.Parse(gitURL)
	if err != nil {
		return fmt.Errorf("invalid git URL: %w", err)
	}

	// Check for local paths (not allowed except for file:// URLs)
	if u.Scheme == "" {
		return fmt.Errorf("local file paths are not allowed")
	}

	// Allow common git URL schemes
	validSchemes := map[string]bool{
		"http":  true,
		"https": true,
		"git":   true,
		"ssh":   true,
		"file":  true, // Allow for testing
	}

	if !validSchemes[u.Scheme] {
		return fmt.Errorf("unsupported URL scheme: %s", u.Scheme)
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
		return fmt.Errorf("remote 'origin' does not exist")
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
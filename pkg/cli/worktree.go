package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	
	"l8s/pkg/git"
)

// GetContainerNameFromWorktree generates a deterministic container name from the current worktree
func GetContainerNameFromWorktree(prefix string) (string, error) {
	// Get the worktree root
	worktreePath, err := git.GetWorktreeRoot()
	if err != nil {
		return "", fmt.Errorf("not in a git repository: %w", err)
	}
	
	// Get the repository name
	repoName, err := git.GetRepositoryName(worktreePath)
	if err != nil {
		return "", fmt.Errorf("failed to get repository name: %w", err)
	}
	
	// Generate container name
	return GenerateContainerName(prefix, repoName, worktreePath), nil
}

// GenerateContainerName creates a deterministic container name from repo name and worktree path
func GenerateContainerName(prefix, repoName, worktreePath string) string {
	// Get absolute path for consistent hashing
	absPath, err := filepath.Abs(worktreePath)
	if err != nil {
		absPath = worktreePath // Fall back to provided path
	}
	
	// Generate SHA256 hash of the absolute path
	hash := sha256.Sum256([]byte(absPath))
	shortHash := hex.EncodeToString(hash[:])[:6]
	
	// Format: dev-<repo_name>-<hash>
	return fmt.Sprintf("%s-%s-%s", prefix, repoName, shortHash)
}

// GetExpectedContainerName returns the expected container name for the current worktree
// Returns empty string if not in a git repository
func GetExpectedContainerName(prefix string) string {
	name, err := GetContainerNameFromWorktree(prefix)
	if err != nil {
		return ""
	}
	return name
}
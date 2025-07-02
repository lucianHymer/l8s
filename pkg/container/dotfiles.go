package container

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CopyDotfiles copies dotfiles from source directory to target directory
func CopyDotfiles(sourceDir, targetDir, containerUser string) error {
	// Walk through the source directory
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from source directory
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Skip the source directory itself
		if relPath == "." {
			return nil
		}

		// Check if we should copy this file
		if !shouldCopyFile(filepath.Base(path)) {
			// If it's a directory, skip the entire subtree
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Construct target path
		targetPath := filepath.Join(targetDir, relPath)

		// Handle directories
		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		// Copy the file
		return copyFile(path, targetPath, info.Mode())
	})
}

// shouldCopyFile determines if a file should be copied based on its name
func shouldCopyFile(filename string) bool {
	// Skip hidden directories that are version control related
	if filename == ".git" || filename == ".svn" || filename == ".hg" {
		return false
	}

	// Skip backup files
	if strings.HasSuffix(filename, "~") || strings.HasSuffix(filename, ".bak") {
		return false
	}

	// Skip swap files
	if strings.HasSuffix(filename, ".swp") || strings.HasSuffix(filename, ".swo") {
		return false
	}

	// Only copy files that start with a dot (dotfiles)
	// or are in a directory that starts with a dot
	return strings.HasPrefix(filename, ".")
}

// CopyDotfilesToContainer copies dotfiles to a container via Podman
func CopyDotfilesToContainer(ctx context.Context, client PodmanClient, containerName, dotfilesDir, containerUser string) error {
	// First, ensure the dotfiles directory exists
	if _, err := os.Stat(dotfilesDir); os.IsNotExist(err) {
		return fmt.Errorf("dotfiles directory does not exist: %s", dotfilesDir)
	}

	// Create temp directory for staging files
	tempDir, err := os.MkdirTemp("", "l8s-dotfiles-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Copy dotfiles to temp directory
	targetDir := filepath.Join(tempDir, "home", containerUser)
	if err := CopyDotfiles(dotfilesDir, targetDir, containerUser); err != nil {
		return fmt.Errorf("failed to copy dotfiles: %w", err)
	}

	// Walk through the temp directory and copy each file to the container
	return filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the base directory
		if path == targetDir {
			return nil
		}

		// Get relative path from target directory
		relPath, err := filepath.Rel(targetDir, path)
		if err != nil {
			return err
		}

		// Container destination path
		containerPath := filepath.Join("/home", containerUser, relPath)

		// If it's a directory, create it in the container
		if info.IsDir() {
			mkdirCmd := []string{"mkdir", "-p", containerPath}
			if err := client.ExecContainer(ctx, containerName, mkdirCmd); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", containerPath, err)
			}
			
			// Set permissions
			chmodCmd := []string{"chmod", fmt.Sprintf("%o", info.Mode().Perm()), containerPath}
			if err := client.ExecContainer(ctx, containerName, chmodCmd); err != nil {
				return fmt.Errorf("failed to set permissions on %s: %w", containerPath, err)
			}
			
			return nil
		}

		// Copy the file to the container
		if err := client.CopyToContainer(ctx, containerName, path, containerPath); err != nil {
			return fmt.Errorf("failed to copy %s to container: %w", relPath, err)
		}

		// Set ownership
		chownCmd := []string{"chown", fmt.Sprintf("%s:%s", containerUser, containerUser), containerPath}
		if err := client.ExecContainer(ctx, containerName, chownCmd); err != nil {
			return fmt.Errorf("failed to set ownership on %s: %w", containerPath, err)
		}

		// Set permissions
		chmodCmd := []string{"chmod", fmt.Sprintf("%o", info.Mode().Perm()), containerPath}
		if err := client.ExecContainer(ctx, containerName, chmodCmd); err != nil {
			return fmt.Errorf("failed to set permissions on %s: %w", containerPath, err)
		}

		return nil
	})
}

// copyFile copies a single file preserving permissions
func copyFile(src, dst string, mode os.FileMode) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return nil
}
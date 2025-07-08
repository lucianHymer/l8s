package commands

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/l8s/l8s/pkg/embed"
	"github.com/spf13/cobra"
)

// InitDotfilesCmd creates the init-dotfiles command
func InitDotfilesCmd() *cobra.Command {
	var template string
	var force bool
	
	cmd := &cobra.Command{
		Use:   "init-dotfiles",
		Short: "Initialize user dotfiles from templates",
		Long: `Initialize user dotfiles in ~/.config/l8s/dotfiles/ from embedded templates.

This command copies the default dotfiles that are embedded in the l8s binary
to your user configuration directory. You can then customize these files to
suit your preferences.

The dotfiles will be used when creating new containers, providing a consistent
development environment across all your l8s containers.`,
		Example: `  # Initialize with default dotfiles
  l8s init-dotfiles
  
  # Overwrite existing dotfiles
  l8s init-dotfiles --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return initializeDotfiles(template, force)
		},
	}
	
	cmd.Flags().StringVar(&template, "template", "minimal", "Template to use (minimal|full)")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing dotfiles")
	
	return cmd
}

// initializeDotfiles copies embedded dotfiles to user config directory
func initializeDotfiles(template string, force bool) error {
	// Validate template
	if template != "minimal" && template != "full" {
		return fmt.Errorf("unknown template: %s (use 'minimal' or 'full')", template)
	}
	
	// Get target directory
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	
	targetDir := filepath.Join(home, ".config", "l8s", "dotfiles")
	
	// Check if directory exists
	if _, err := os.Stat(targetDir); err == nil && !force {
		return fmt.Errorf("dotfiles directory already exists at %s\nUse --force to overwrite", targetDir)
	}
	
	// Create directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Copy embedded dotfiles
	if err := copyEmbeddedDotfilesToDir(targetDir); err != nil {
		return fmt.Errorf("failed to copy dotfiles: %w", err)
	}
	
	fmt.Printf("Successfully initialized dotfiles in %s\n", targetDir)
	fmt.Println("\nYou can now customize these files to suit your preferences.")
	fmt.Println("They will be automatically copied to new l8s containers.")
	
	return nil
}

// copyEmbeddedDotfilesToDir copies all embedded dotfiles to the specified directory
func copyEmbeddedDotfilesToDir(targetDir string) error {
	// Get embedded filesystem
	fsys, err := embed.GetDotfilesFS()
	if err != nil {
		return fmt.Errorf("failed to get embedded dotfiles: %w", err)
	}
	
	// Walk through all files
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories
		if d.IsDir() {
			return nil
		}
		
		// Read file content
		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}
		
		// Write to target directory
		targetPath := filepath.Join(targetDir, path)
		if err := os.WriteFile(targetPath, content, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", targetPath, err)
		}
		
		return nil
	})
}
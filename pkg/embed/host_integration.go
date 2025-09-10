package embed

import (
	_ "embed"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed all:host-integration
var hostIntegrationFS embed.FS

// GetHostIntegrationFS returns the embedded host integration filesystem
func GetHostIntegrationFS() (fs.FS, error) {
	return fs.Sub(hostIntegrationFS, "host-integration")
}

// ExtractZSHPlugin extracts the ZSH plugin files to the specified directory
func ExtractZSHPlugin(destDir string) error {
	hostFS, err := GetHostIntegrationFS()
	if err != nil {
		return fmt.Errorf("failed to get host integration filesystem: %w", err)
	}

	// Walk through the oh-my-zsh/l8s directory
	return fs.WalkDir(hostFS, "oh-my-zsh/l8s", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate the destination path
		relPath, err := filepath.Rel("oh-my-zsh/l8s", path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(destDir, relPath)

		if d.IsDir() {
			// Create directory
			return os.MkdirAll(destPath, 0755)
		}

		// Read and write file
		content, err := fs.ReadFile(hostFS, path)
		if err != nil {
			return err
		}

		// Check if the file should be executable (like test scripts)
		mode := fs.FileMode(0644)
		if filepath.Ext(path) == ".sh" || path == "oh-my-zsh/l8s/_l8s" {
			mode = 0755
		}

		return os.WriteFile(destPath, content, mode)
	})
}
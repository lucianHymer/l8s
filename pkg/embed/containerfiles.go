package embed

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed containers/Containerfile
var Containerfile string

//go:embed containers/Containerfile.test
var ContainerfileTest string

// ExtractContainerfile writes the embedded Containerfile to a temporary file
// and returns the path to that file. The caller is responsible for cleaning up.
func ExtractContainerfile() (string, error) {
	return extractToTemp(Containerfile, "Containerfile")
}

// ExtractContainerfileTest writes the embedded test Containerfile to a temporary file
// and returns the path to that file. The caller is responsible for cleaning up.
func ExtractContainerfileTest() (string, error) {
	return extractToTemp(ContainerfileTest, "Containerfile.test")
}

func extractToTemp(content, filename string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "l8s-build-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	tmpFile := filepath.Join(tmpDir, filename)
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to write Containerfile: %w", err)
	}

	return tmpFile, nil
}
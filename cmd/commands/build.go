package commands

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/l8s/l8s/pkg/container"
	"github.com/spf13/cobra"
)

// BuildCmd creates the build command
func BuildCmd() *cobra.Command {
	var testImage bool

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build or rebuild the base container image",
		Long:  `Build or rebuild the base container image from the Containerfile.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBuild(cmd, args, testImage)
		},
	}

	cmd.Flags().BoolVar(&testImage, "test", false, "Build the test image instead of the base image")

	return cmd
}

func runBuild(cmd *cobra.Command, args []string, testImage bool) error {
	// Determine which Containerfile to use
	containerfile := "containers/Containerfile"
	imageName := "localhost/l8s-fedora:latest"
	
	if testImage {
		containerfile = "containers/Containerfile.test"
		imageName = "localhost/l8s-test:latest"
	}

	// Get absolute path
	absPath, err := filepath.Abs(containerfile)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	fmt.Printf("Building image %s from %s...\n", imageName, containerfile)

	// Build the image
	ctx := context.Background()
	if err := container.BuildImage(ctx, absPath, imageName); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	fmt.Printf("✓ Image built successfully: %s\n", imageName)
	return nil
}
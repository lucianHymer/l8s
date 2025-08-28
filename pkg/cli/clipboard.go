package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// extractClipboardContent detects clipboard type and extracts content to a temporary file
// Returns: content type ("png" or "txt"), path to temp file, error
func extractClipboardContent() (string, string, error) {
	// First check if there's an image in clipboard
	if hasImageInClipboard() {
		path, err := extractImageFromClipboard()
		if err != nil {
			return "", "", err
		}
		return "png", path, nil
	}

	// Check for text content
	if hasTextInClipboard() {
		path, err := extractTextFromClipboard()
		if err != nil {
			return "", "", err
		}
		return "txt", path, nil
	}

	return "", "", fmt.Errorf("no image or text found in clipboard")
}

// hasImageInClipboard checks if clipboard contains an image
func hasImageInClipboard() bool {
	// Use osascript to check clipboard info
	cmd := exec.Command("osascript", "-e", "clipboard info")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Check for common image formats
	outputStr := string(output)
	return strings.Contains(outputStr, "PNGf") ||
		strings.Contains(outputStr, "JPEG") ||
		strings.Contains(outputStr, "TIFF")
}

// extractImageFromClipboard extracts an image from clipboard to a temporary file
func extractImageFromClipboard() (string, error) {
	// Create temp file for image
	tempFile := filepath.Join(os.TempDir(), "l8s-clipboard.png")

	// AppleScript to extract PNG from clipboard
	script := fmt.Sprintf(`
		set thePNG to the clipboard as «class PNGf»
		set theFile to open for access POSIX file "%s" with write permission
		write thePNG to theFile
		close access theFile
	`, tempFile)

	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to extract image from clipboard: %w", err)
	}

	return tempFile, nil
}

// hasTextInClipboard checks if clipboard contains text
func hasTextInClipboard() bool {
	// pbpaste will return non-empty output if there's text
	cmd := exec.Command("pbpaste")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(output) > 0
}

// extractTextFromClipboard extracts text from clipboard to a temporary file
func extractTextFromClipboard() (string, error) {
	// Create temp file for text
	tempFile := filepath.Join(os.TempDir(), "l8s-clipboard.txt")

	// Use pbpaste to get text content
	cmd := exec.Command("pbpaste")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to extract text from clipboard: %w", err)
	}

	// Write to temp file
	if err := os.WriteFile(tempFile, output, 0644); err != nil {
		return "", fmt.Errorf("failed to write clipboard text to file: %w", err)
	}

	return tempFile, nil
}
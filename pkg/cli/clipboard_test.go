package cli

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractClipboardContent(t *testing.T) {
	// Skip tests on non-macOS platforms
	if runtime.GOOS != "darwin" {
		t.Skip("Clipboard tests only run on macOS")
	}

	tests := []struct {
		name        string
		setup       func()
		cleanup     func()
		wantType    string
		wantErr     bool
		errContains string
	}{
		{
			name: "text in clipboard",
			setup: func() {
				// This test requires manual setup - putting text in clipboard
				t.Skip("Requires manual clipboard setup")
			},
			cleanup:  func() {},
			wantType: "txt",
			wantErr:  false,
		},
		{
			name: "image in clipboard",
			setup: func() {
				// This test requires manual setup - putting image in clipboard
				t.Skip("Requires manual clipboard setup")
			},
			cleanup:  func() {},
			wantType: "png",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			contentType, path, err := extractClipboardContent()
			
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantType, contentType)
			assert.NotEmpty(t, path)

			// Cleanup temp file
			if path != "" {
				os.Remove(path)
			}
		})
	}
}

func TestHasImageInClipboard(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Clipboard tests only run on macOS")
	}

	// This test would need manual clipboard setup or mocking
	t.Skip("Requires manual clipboard setup or command mocking")
}

func TestHasTextInClipboard(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Clipboard tests only run on macOS")
	}

	// This test would need manual clipboard setup or mocking
	t.Skip("Requires manual clipboard setup or command mocking")
}

func TestExtractImageFromClipboard(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Clipboard tests only run on macOS")
	}

	// This test would need an image in clipboard
	t.Skip("Requires image in clipboard")
}

func TestExtractTextFromClipboard(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Clipboard tests only run on macOS")
	}

	// This test would need text in clipboard
	t.Skip("Requires text in clipboard")
}
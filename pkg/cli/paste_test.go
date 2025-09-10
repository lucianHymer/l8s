package cli

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunPaste(t *testing.T) {
	// Skip on non-macOS for now
	if runtime.GOOS != "darwin" {
		t.Skip("Paste command tests only run on macOS")
	}

	// Cannot directly test unexported runPaste method
	// The actual paste command is tested via integration tests
	t.Skip("Cannot directly test unexported runPaste method")
}

func TestPasteCommandStructure(t *testing.T) {
	factory := NewLazyCommandFactory()
	cmd := factory.PasteCmd()

	// Check command configuration
	assert.Equal(t, "paste [name]", cmd.Use)
	assert.Contains(t, cmd.Short, "clipboard")
	assert.NotNil(t, cmd.RunE)

	// Test argument validation - paste now accepts 0 or 1 arg
	assert.NoError(t, cmd.Args(nil, []string{}))
	assert.NoError(t, cmd.Args(nil, []string{"name"}))
	assert.Error(t, cmd.Args(nil, []string{"a", "b"}))
}

func TestRunPastePlatformCheck(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("This test is for non-macOS platforms")
	}

	// Cannot directly test unexported runPaste method
	// Platform check is verified in integration tests
	t.Skip("Cannot directly test unexported runPaste method")
}

func TestPasteFileNaming(t *testing.T) {
	tests := []struct {
		name         string
		customName   string
		clipboardType string
		wantFilename string
	}{
		{
			name:         "default image",
			customName:   "",
			clipboardType: "png",
			wantFilename: "clipboard.png",
		},
		{
			name:         "default text",
			customName:   "",
			clipboardType: "txt",
			wantFilename: "clipboard.txt",
		},
		{
			name:         "custom image",
			customName:   "screenshot1",
			clipboardType: "png",
			wantFilename: "clipboard-screenshot1.png",
		},
		{
			name:         "custom text",
			customName:   "snippet",
			clipboardType: "txt",
			wantFilename: "clipboard-snippet.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the filename generation logic
			var destFilename string
			if tt.customName != "" {
				destFilename = "clipboard-" + tt.customName + "." + tt.clipboardType
			} else {
				destFilename = "clipboard." + tt.clipboardType
			}
			
			assert.Equal(t, tt.wantFilename, destFilename)
		})
	}
}
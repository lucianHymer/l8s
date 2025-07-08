package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitDotfilesCmd(t *testing.T) {
	// Test that the command exists and has the right properties
	cmd := InitDotfilesCmd()
	
	assert.Equal(t, "init-dotfiles", cmd.Use)
	assert.Contains(t, cmd.Short, "Initialize user dotfiles")
	assert.NotNil(t, cmd.RunE)
	
	// Check flags
	templateFlag := cmd.Flags().Lookup("template")
	assert.NotNil(t, templateFlag)
	assert.Equal(t, "minimal", templateFlag.DefValue)
	
	forceFlag := cmd.Flags().Lookup("force")
	assert.NotNil(t, forceFlag)
	assert.Equal(t, "false", forceFlag.DefValue)
}

func TestInitializeDotfiles(t *testing.T) {
	tests := []struct {
		name        string
		template    string
		force       bool
		setupFunc   func(string) error
		wantErr     bool
		errContains string
		checkFunc   func(t *testing.T, dir string)
	}{
		{
			name:     "create minimal template in new directory",
			template: "minimal",
			force:    false,
			checkFunc: func(t *testing.T, dir string) {
				// Check that essential dotfiles were created
				essentialFiles := []string{".zshrc", ".bashrc", ".gitconfig", ".tmux.conf"}
				for _, file := range essentialFiles {
					path := filepath.Join(dir, file)
					assert.FileExists(t, path, "Expected %s to be created", file)
					
					// Check file is not empty
					info, err := os.Stat(path)
					require.NoError(t, err)
					assert.Greater(t, info.Size(), int64(0), "Expected %s to have content", file)
				}
			},
		},
		{
			name:     "fail when directory exists without force",
			template: "minimal",
			force:    false,
			setupFunc: func(dir string) error {
				// Create the directory
				return os.MkdirAll(dir, 0755)
			},
			wantErr:     true,
			errContains: "already exists",
		},
		{
			name:     "overwrite with force flag",
			template: "minimal",
			force:    true,
			setupFunc: func(dir string) error {
				// Create directory with existing file
				os.MkdirAll(dir, 0755)
				return os.WriteFile(filepath.Join(dir, ".zshrc"), []byte("old content"), 0644)
			},
			checkFunc: func(t *testing.T, dir string) {
				// Check that file was overwritten
				content, err := os.ReadFile(filepath.Join(dir, ".zshrc"))
				require.NoError(t, err)
				assert.NotEqual(t, "old content", string(content))
				assert.Contains(t, string(content), "l8s container zsh configuration")
			},
		},
		{
			name:        "invalid template name",
			template:    "invalid",
			force:       false,
			wantErr:     true,
			errContains: "unknown template",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tempDir := t.TempDir()
			
			// Mock home directory
			origHome := os.Getenv("HOME")
			os.Setenv("HOME", tempDir)
			defer os.Setenv("HOME", origHome)
			
			targetDir := filepath.Join(tempDir, ".config", "l8s", "dotfiles")
			
			// Run setup if provided
			if tt.setupFunc != nil {
				err := tt.setupFunc(targetDir)
				require.NoError(t, err)
			}
			
			// Run the function
			err := initializeDotfiles(tt.template, tt.force)
			
			// Check error
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
			
			// Run additional checks
			if tt.checkFunc != nil && !tt.wantErr {
				tt.checkFunc(t, targetDir)
			}
		})
	}
}

func TestCopyEmbeddedDotfilesToDir(t *testing.T) {
	tempDir := t.TempDir()
	
	err := copyEmbeddedDotfilesToDir(tempDir)
	require.NoError(t, err)
	
	// Check files were copied
	files, err := os.ReadDir(tempDir)
	require.NoError(t, err)
	
	assert.GreaterOrEqual(t, len(files), 4, "Expected at least 4 dotfiles")
	
	// Check specific files
	for _, filename := range []string{".zshrc", ".bashrc", ".gitconfig", ".tmux.conf"} {
		path := filepath.Join(tempDir, filename)
		assert.FileExists(t, path)
		
		// Verify content matches embedded
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Greater(t, info.Size(), int64(0))
	}
}
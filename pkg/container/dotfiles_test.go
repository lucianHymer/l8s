package container

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCopyDotfiles(t *testing.T) {
	tests := []struct {
		name          string
		setupDotfiles func(string) error
		containerUser string
		wantErr       bool
		errContains   string
		expectedFiles []string
	}{
		{
			name: "copy basic dotfiles",
			setupDotfiles: func(dir string) error {
				files := map[string]string{
					".zshrc":     "# ZSH configuration\nexport PATH=$PATH:/usr/local/bin",
					".tmux.conf": "# TMUX configuration\nset -g mouse on",
					".gitconfig": "[user]\n  name = Test User\n  email = test@example.com",
				}
				for name, content := range files {
					if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			containerUser: "dev",
			wantErr:       false,
			expectedFiles: []string{".zshrc", ".tmux.conf", ".gitconfig"},
		},
		{
			name: "copy nested config files",
			setupDotfiles: func(dir string) error {
				// Create nested directories
				configDirs := []string{
					filepath.Join(dir, ".config", "nvim"),
					filepath.Join(dir, ".config", "nvim", "lua"),
					filepath.Join(dir, ".config", "claude"),
				}
				for _, d := range configDirs {
					if err := os.MkdirAll(d, 0755); err != nil {
						return err
					}
				}
				
				// Create config files
				files := map[string]string{
					".config/nvim/init.lua":         "-- Neovim configuration",
					".config/nvim/lua/plugins.lua":  "-- Plugin configuration",
					".config/nvim/lua/settings.lua": "-- Settings",
					".config/claude/config.yaml":    "model: claude-3-opus",
				}
				for name, content := range files {
					if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			containerUser: "dev",
			wantErr:       false,
			expectedFiles: []string{
				".config/nvim/init.lua",
				".config/nvim/lua/plugins.lua",
				".config/nvim/lua/settings.lua",
				".config/claude/config.yaml",
			},
		},
		{
			name: "dotfiles directory not found",
			setupDotfiles: func(dir string) error {
				// Don't create directory
				return nil
			},
			containerUser: "dev",
			wantErr:       true,
			errContains:   "dotfiles directory not found",
		},
		{
			name: "skip non-dotfiles",
			setupDotfiles: func(dir string) error {
				files := map[string]string{
					".zshrc":    "# ZSH config",
					"README.md": "# This should not be copied",
					"install.sh": "#!/bin/bash\necho 'Should not be copied'",
				}
				for name, content := range files {
					if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
						return err
					}
				}
				return nil
			},
			containerUser: "dev",
			wantErr:       false,
			expectedFiles: []string{".zshrc"},
		},
		{
			name: "preserve file permissions",
			setupDotfiles: func(dir string) error {
				// Create files with specific permissions
				files := map[string]os.FileMode{
					".zshrc":           0644,
					".ssh/config":      0600,
					".local/bin/script": 0755,
				}
				
				// Create directories
				if err := os.MkdirAll(filepath.Join(dir, ".ssh"), 0700); err != nil {
					return err
				}
				if err := os.MkdirAll(filepath.Join(dir, ".local/bin"), 0755); err != nil {
					return err
				}
				
				for name, mode := range files {
					path := filepath.Join(dir, name)
					if err := os.WriteFile(path, []byte("content"), mode); err != nil {
						return err
					}
				}
				return nil
			},
			containerUser: "dev",
			wantErr:       false,
			expectedFiles: []string{".zshrc", ".ssh/config", ".local/bin/script"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directories
			dotfilesDir := t.TempDir()
			targetDir := t.TempDir()
			
			// Setup dotfiles
			if tt.setupDotfiles != nil {
				err := tt.setupDotfiles(dotfilesDir)
				require.NoError(t, err)
			}
			
			// If testing "not found", use a non-existent directory
			if tt.errContains == "dotfiles directory not found" {
				dotfilesDir = "/non/existent/path"
			}
			
			// Copy dotfiles
			err := CopyDotfiles(dotfilesDir, targetDir, tt.containerUser)
			
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				
				// Verify expected files were copied
				for _, file := range tt.expectedFiles {
					targetPath := filepath.Join(targetDir, file)
					assert.FileExists(t, targetPath)
					
					// Verify content matches
					sourcePath := filepath.Join(dotfilesDir, file)
					sourceContent, err := os.ReadFile(sourcePath)
					require.NoError(t, err)
					targetContent, err := os.ReadFile(targetPath)
					require.NoError(t, err)
					assert.Equal(t, sourceContent, targetContent)
					
					// Verify permissions match
					sourceInfo, err := os.Stat(sourcePath)
					require.NoError(t, err)
					targetInfo, err := os.Stat(targetPath)
					require.NoError(t, err)
					assert.Equal(t, sourceInfo.Mode(), targetInfo.Mode())
				}
			}
		})
	}
}

func TestShouldCopyFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"dotfile", ".zshrc", true},
		{"hidden directory", ".config", true},
		{"regular file", "README.md", false},
		{"script file", "install.sh", false},
		{"git directory", ".git", false}, // Should skip .git
		{"DS_Store", ".DS_Store", false}, // Should skip macOS files
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldCopyFile(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDotfilesPath(t *testing.T) {
	tests := []struct {
		name         string
		appPath      string
		relativePath string
		expected     string
	}{
		{
			name:         "standard dotfiles path",
			appPath:      "/usr/local/bin/l8s",
			relativePath: "../share/l8s/dotfiles",
			expected:     "/usr/local/share/l8s/dotfiles",
		},
		{
			name:         "development path",
			appPath:      "/home/user/go/src/l8s/l8s",
			relativePath: "./dotfiles",
			expected:     "/home/user/go/src/l8s/dotfiles",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This would be the actual implementation
			result := filepath.Clean(filepath.Join(filepath.Dir(tt.appPath), tt.relativePath))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCopyDotfilesToContainer(t *testing.T) {
	tests := []struct {
		name          string
		containerName string
		containerUser string
		setupDotfiles func(string) error
		mockPodman    func(*MockPodmanClient)
		wantErr       bool
		errContains   string
	}{
		{
			name:          "successful copy via podman exec",
			containerName: "dev-myproject",
			containerUser: "dev",
			setupDotfiles: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, ".zshrc"), []byte("# ZSH config"), 0644)
			},
			mockPodman: func(m *MockPodmanClient) {
				// Mock container exec for creating directories
				m.On("ExecContainer", mock.Anything, "dev-myproject", 
					[]string{"mkdir", "-p", "/home/dev"}).Return(nil)
				
				// Mock container exec for copying files
				m.On("CopyToContainer", mock.Anything, "dev-myproject", 
					mock.AnythingOfType("string"), "/home/dev/.zshrc").Return(nil)
				
				// Mock container exec for chown
				m.On("ExecContainer", mock.Anything, "dev-myproject", 
					[]string{"chown", "-R", "dev:dev", "/home/dev"}).Return(nil)
			},
			wantErr: false,
		},
		{
			name:          "container not found",
			containerName: "dev-nonexistent",
			containerUser: "dev",
			setupDotfiles: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, ".zshrc"), []byte("# ZSH config"), 0644)
			},
			mockPodman: func(m *MockPodmanClient) {
				m.On("ExecContainer", mock.Anything, "dev-nonexistent", 
					mock.Anything).Return(assert.AnError)
			},
			wantErr:     true,
			errContains: "failed to copy dotfiles",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp dotfiles directory
			dotfilesDir := t.TempDir()
			if tt.setupDotfiles != nil {
				err := tt.setupDotfiles(dotfilesDir)
				require.NoError(t, err)
			}
			
			// Create mock podman client
			mockClient := new(MockPodmanClient)
			if tt.mockPodman != nil {
				tt.mockPodman(mockClient)
			}
			
			// Test copying to container
			err := CopyDotfilesToContainer(mockClient, tt.containerName, 
				dotfilesDir, tt.containerUser)
			
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
			
			mockClient.AssertExpectations(t)
		})
	}
}
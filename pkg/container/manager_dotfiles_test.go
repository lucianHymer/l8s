package container

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/l8s/l8s/pkg/logging"
	"github.com/stretchr/testify/mock"
)

func TestManagerGetDotfilesPath(t *testing.T) {
	// Create a temp directory for testing
	tempDir := t.TempDir()
	
	tests := []struct {
		name          string
		setupFunc     func() (*Manager, func())
		expectedPath  string
		expectEmbedded bool
	}{
		{
			name: "CLI flag takes highest priority",
			setupFunc: func() (*Manager, func()) {
				cfg := Config{
					DotfilesPath: "/config/path",
					ContainerUser: "testuser",
				}
				m := &Manager{
					config: cfg,
					logger: logging.Default(),
					cliDotfilesPath: "/cli/path",
				}
				return m, func() {}
			},
			expectedPath: "/cli/path",
			expectEmbedded: false,
		},
		{
			name: "Environment variable takes second priority",
			setupFunc: func() (*Manager, func()) {
				os.Setenv("L8S_DOTFILES", "/env/path")
				cfg := Config{
					DotfilesPath: "/config/path",
					ContainerUser: "testuser",
				}
				m := &Manager{
					config: cfg,
					logger: logging.Default(),
				}
				return m, func() { os.Unsetenv("L8S_DOTFILES") }
			},
			expectedPath: "/env/path",
			expectEmbedded: false,
		},
		{
			name: "Config file takes third priority",
			setupFunc: func() (*Manager, func()) {
				cfg := Config{
					DotfilesPath: "/config/path",
					ContainerUser: "testuser",
				}
				m := &Manager{
					config: cfg,
					logger: logging.Default(),
				}
				return m, func() {}
			},
			expectedPath: "/config/path",
			expectEmbedded: false,
		},
		{
			name: "User dotfiles directory takes fourth priority",
			setupFunc: func() (*Manager, func()) {
				// Create user dotfiles directory
				userDotfiles := filepath.Join(tempDir, ".config", "l8s", "dotfiles")
				os.MkdirAll(userDotfiles, 0755)
				
				// Mock home directory
				oldHome := os.Getenv("HOME")
				os.Setenv("HOME", tempDir)
				
				cfg := Config{
					ContainerUser: "testuser",
				}
				m := &Manager{
					config: cfg,
					logger: logging.Default(),
				}
				return m, func() { os.Setenv("HOME", oldHome) }
			},
			expectedPath: filepath.Join(tempDir, ".config", "l8s", "dotfiles"),
			expectEmbedded: false,
		},
		{
			name: "Falls back to embedded defaults",
			setupFunc: func() (*Manager, func()) {
				cfg := Config{
					ContainerUser: "testuser",
				}
				m := &Manager{
					config: cfg,
					logger: logging.Default(),
				}
				return m, func() {}
			},
			expectedPath: "",
			expectEmbedded: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, cleanup := tt.setupFunc()
			defer cleanup()
			
			path, useEmbedded := m.getDotfilesPath()
			
			if path != tt.expectedPath {
				t.Errorf("getDotfilesPath() path = %v, want %v", path, tt.expectedPath)
			}
			
			if useEmbedded != tt.expectEmbedded {
				t.Errorf("getDotfilesPath() useEmbedded = %v, want %v", useEmbedded, tt.expectEmbedded)
			}
		})
	}
}

func TestCopyDotfilesWithPriority(t *testing.T) {
	ctx := context.Background()
	
	tests := []struct {
		name      string
		setupFunc func() (*Manager, *MockPodmanClient, func())
		checkFunc func(t *testing.T, mockClient *MockPodmanClient)
	}{
		{
			name: "Uses embedded dotfiles when no user dotfiles exist",
			setupFunc: func() (*Manager, *MockPodmanClient, func()) {
				cfg := Config{
					ContainerUser: "testuser",
				}
				
				mockClient := &MockPodmanClient{}
				
				// Mock the CopyToContainer calls for embedded dotfiles
				mockClient.On("CopyToContainer", ctx, "test-container", 
					mock.AnythingOfType("string"), 
					mock.AnythingOfType("string")).Return(nil)
				
				// Mock ExecContainer for any command (stat, chown, chmod)
				mockClient.On("ExecContainer", ctx, "test-container",
					mock.AnythingOfType("[]string")).Return(nil)
				
				// Mock ExecContainerWithInput for git config
				mockClient.On("ExecContainerWithInput", ctx, "test-container", 
					mock.AnythingOfType("[]string"), 
					mock.AnythingOfType("string")).Return(nil).Maybe()
				
				m := &Manager{
					config: cfg,
					client: mockClient,
					logger: logging.Default(),
				}
				
				return m, mockClient, func() {}
			},
			checkFunc: func(t *testing.T, mockClient *MockPodmanClient) {
				// Verify that CopyToContainer was called for dotfiles
				mockClient.AssertCalled(t, "CopyToContainer", ctx, "test-container", 
					mock.MatchedBy(func(src string) bool { return true }), 
					mock.MatchedBy(func(dst string) bool { return true }))
			},
		},
		{
			name: "Uses user dotfiles when available",
			setupFunc: func() (*Manager, *MockPodmanClient, func()) {
				tempDir := t.TempDir()
				cfg := Config{
					ContainerUser: "testuser",
					DotfilesPath:  tempDir,
				}
				
				// Create a test dotfile
				testFile := filepath.Join(tempDir, ".testrc")
				os.WriteFile(testFile, []byte("test content"), 0644)
				
				mockClient := &MockPodmanClient{}
				
				// Mock the CopyToContainer calls
				mockClient.On("CopyToContainer", ctx, "test-container", 
					mock.AnythingOfType("string"), 
					mock.AnythingOfType("string")).Return(nil)
				
				// Mock ExecContainer for any command (stat, chown, chmod)
				mockClient.On("ExecContainer", ctx, "test-container",
					mock.AnythingOfType("[]string")).Return(nil)
				
				// Mock ExecContainerWithInput for git config
				mockClient.On("ExecContainerWithInput", ctx, "test-container", 
					mock.AnythingOfType("[]string"), 
					mock.AnythingOfType("string")).Return(nil).Maybe()
				
				m := &Manager{
					config: cfg,
					client: mockClient,
					logger: logging.Default(),
				}
				
				return m, mockClient, func() {}
			},
			checkFunc: func(t *testing.T, mockClient *MockPodmanClient) {
				// Verify that CopyToContainer was called
				mockClient.AssertCalled(t, "CopyToContainer", ctx, "test-container", 
					mock.MatchedBy(func(src string) bool { return true }), 
					mock.MatchedBy(func(dst string) bool { return true }))
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, mockClient, cleanup := tt.setupFunc()
			defer cleanup()
			
			err := m.copyDotfiles(ctx, "test-container")
			if err != nil {
				t.Errorf("copyDotfiles() error = %v", err)
			}
			
			// Run additional checks
			if tt.checkFunc != nil {
				tt.checkFunc(t, mockClient)
			}
		})
	}
}
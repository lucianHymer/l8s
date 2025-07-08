package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create a test git repository
func createTestRepo(t *testing.T) string {
	tmpDir := t.TempDir()
	
	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	err := cmd.Run()
	require.NoError(t, err, "Failed to init git repo")
	
	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err)
	
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err)
	
	// Create initial commit
	testFile := filepath.Join(tmpDir, "README.md")
	err = os.WriteFile(testFile, []byte("# Test Repo"), 0644)
	require.NoError(t, err)
	
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err)
	
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err)
	
	// Ensure we're on main branch (for consistency across git versions)
	cmd = exec.Command("git", "branch", "-M", "main")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err)
	
	return tmpDir
}

func TestCloneRepository(t *testing.T) {
	tests := []struct {
		name        string
		gitURL      string
		branch      string
		targetPath  string
		setupRepo   bool
		wantErr     bool
		errContains string
	}{
		{
			name:       "successful clone",
			gitURL:     "", // Will be set to test repo
			branch:     "main",
			targetPath: "",
			setupRepo:  true,
			wantErr:    false,
		},
		{
			name:        "invalid git URL",
			gitURL:      "not-a-git-url",
			branch:      "main",
			targetPath:  "",
			setupRepo:   false,
			wantErr:     true,
			errContains: "invalid repository",
		},
		{
			name:        "non-existent branch",
			gitURL:      "", // Will be set to test repo
			branch:      "non-existent-branch",
			targetPath:  "",
			setupRepo:   true,
			wantErr:     true,
			errContains: "branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupRepo {
				sourceRepo := createTestRepo(t)
				if tt.gitURL == "" {
					tt.gitURL = "file://" + sourceRepo
				}
			}
			
			if tt.targetPath == "" {
				tt.targetPath = filepath.Join(t.TempDir(), "cloned")
			}

			err := CloneRepository(tt.gitURL, tt.branch, tt.targetPath)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				// Verify clone was successful
				assert.DirExists(t, filepath.Join(tt.targetPath, ".git"))
				assert.FileExists(t, filepath.Join(tt.targetPath, "README.md"))
			}
		})
	}
}

func TestAddRemote(t *testing.T) {
	tests := []struct {
		name        string
		remoteName  string
		remoteURL   string
		wantErr     bool
		errContains string
	}{
		{
			name:       "add new remote",
			remoteName: "myproject",
			remoteURL:  "ssh://dev@localhost:2200/workspace/project",
			wantErr:    false,
		},
		{
			name:        "duplicate remote name",
			remoteName:  "origin",
			remoteURL:   "ssh://dev@localhost:2200/workspace/project",
			wantErr:     true,
			errContains: "already exists",
		},
		{
			name:        "empty remote name",
			remoteName:  "",
			remoteURL:   "ssh://dev@localhost:2200/workspace/project",
			wantErr:     true,
			errContains: "remote name is required",
		},
		{
			name:        "empty remote URL",
			remoteName:  "myproject",
			remoteURL:   "",
			wantErr:     true,
			errContains: "remote URL is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoPath := createTestRepo(t)
			
			// Add origin remote for duplicate test
			if tt.remoteName == "origin" {
				cmd := exec.Command("git", "remote", "add", "origin", "https://github.com/test/repo.git")
				cmd.Dir = repoPath
				err := cmd.Run()
				require.NoError(t, err)
			}

			err := AddRemote(repoPath, tt.remoteName, tt.remoteURL)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				
				// Verify remote was added
				cmd := exec.Command("git", "remote", "get-url", tt.remoteName)
				cmd.Dir = repoPath
				output, err := cmd.Output()
				require.NoError(t, err)
				assert.Contains(t, string(output), tt.remoteURL)
			}
		})
	}
}

func TestRemoveRemote(t *testing.T) {
	tests := []struct {
		name        string
		remoteName  string
		setupRemote bool
		wantErr     bool
		errContains string
	}{
		{
			name:        "remove existing remote",
			remoteName:  "myproject",
			setupRemote: true,
			wantErr:     false,
		},
		{
			name:        "remove non-existent remote",
			remoteName:  "nonexistent",
			setupRemote: false,
			wantErr:     true,
			errContains: "No such remote",
		},
		{
			name:        "empty remote name",
			remoteName:  "",
			setupRemote: false,
			wantErr:     true,
			errContains: "remote name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoPath := createTestRepo(t)
			
			if tt.setupRemote {
				cmd := exec.Command("git", "remote", "add", tt.remoteName, "ssh://dev@localhost:2200/workspace/project")
				cmd.Dir = repoPath
				err := cmd.Run()
				require.NoError(t, err)
			}

			err := RemoveRemote(repoPath, tt.remoteName)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				
				// Verify remote was removed
				cmd := exec.Command("git", "remote")
				cmd.Dir = repoPath
				output, err := cmd.Output()
				require.NoError(t, err)
				assert.NotContains(t, string(output), tt.remoteName)
			}
		})
	}
}

func TestSetUpstream(t *testing.T) {
	tests := []struct {
		name        string
		branch      string
		remoteName  string
		setupBranch bool
		wantErr     bool
		errContains string
	}{
		{
			name:        "set upstream for existing branch",
			branch:      "main",
			remoteName:  "myproject",
			setupBranch: true,
			wantErr:     false,
		},
		{
			name:        "set upstream for non-existent branch",
			branch:      "feature",
			remoteName:  "myproject",
			setupBranch: false,
			wantErr:     true,
			errContains: "branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoPath := createTestRepo(t)
			
			// Add remote
			cmd := exec.Command("git", "remote", "add", tt.remoteName, "ssh://dev@localhost:2200/workspace/project")
			cmd.Dir = repoPath
			err := cmd.Run()
			require.NoError(t, err)
			
			if tt.setupBranch && tt.branch != "main" {
				// Create and checkout branch
				cmd = exec.Command("git", "checkout", "-b", tt.branch)
				cmd.Dir = repoPath
				err = cmd.Run()
				require.NoError(t, err)
			}

			err = SetUpstream(repoPath, tt.branch, tt.remoteName)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				
				// Verify upstream was set
				cmd = exec.Command("git", "config", fmt.Sprintf("branch.%s.remote", tt.branch))
				cmd.Dir = repoPath
				output, err := cmd.Output()
				require.NoError(t, err)
				assert.Contains(t, string(output), tt.remoteName)
			}
		})
	}
}

func TestGetCurrentBranch(t *testing.T) {
	t.Run("get current branch", func(t *testing.T) {
		repoPath := createTestRepo(t)
		
		branch, err := GetCurrentBranch(repoPath)
		require.NoError(t, err)
		assert.Contains(t, []string{"main", "master"}, branch)
	})

	t.Run("not a git repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		
		_, err := GetCurrentBranch(tmpDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a git repository")
	})
}

func TestListRemotes(t *testing.T) {
	t.Run("list multiple remotes", func(t *testing.T) {
		repoPath := createTestRepo(t)
		
		// Add some remotes
		remotes := map[string]string{
			"origin":    "https://github.com/user/repo.git",
			"myproject": "ssh://dev@localhost:2200/workspace/project",
			"backup":    "https://gitlab.com/user/repo.git",
		}
		
		for name, url := range remotes {
			cmd := exec.Command("git", "remote", "add", name, url)
			cmd.Dir = repoPath
			err := cmd.Run()
			require.NoError(t, err)
		}
		
		result, err := ListRemotes(repoPath)
		require.NoError(t, err)
		assert.Len(t, result, 3)
		
		for name, url := range remotes {
			assert.Equal(t, url, result[name])
		}
	})

	t.Run("no remotes", func(t *testing.T) {
		repoPath := createTestRepo(t)
		
		result, err := ListRemotes(repoPath)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestValidateGitURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid https URL", "https://github.com/user/repo.git", false},
		{"valid https URL without .git", "https://github.com/user/repo", false},
		{"valid SSH URL", "git@github.com:user/repo.git", false},
		{"valid SSH URL with protocol", "ssh://git@github.com/user/repo.git", false},
		{"valid git protocol", "git://github.com/user/repo.git", false},
		{"empty URL", "", true},
		{"invalid URL", "not-a-url", true},
		{"file URL", "file:///path/to/repo", true},
		{"local path", "/path/to/repo", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGitURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}


func TestChangeUpstreamToOrigin(t *testing.T) {
	tests := []struct {
		name          string
		branch        string
		currentRemote string
		hasOrigin     bool
		wantErr       bool
		errContains   string
	}{
		{
			name:          "change upstream from container to origin",
			branch:        "main",
			currentRemote: "myproject",
			hasOrigin:     true,
			wantErr:       false,
		},
		{
			name:          "no origin remote exists",
			branch:        "main",
			currentRemote: "myproject",
			hasOrigin:     false,
			wantErr:       true,
			errContains:   "origin remote not found",
		},
		{
			name:          "branch not tracked",
			branch:        "untracked",
			currentRemote: "",
			hasOrigin:     true,
			wantErr:       false, // Should succeed but do nothing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoPath := createTestRepo(t)
			
			// Add origin remote if needed
			if tt.hasOrigin {
				cmd := exec.Command("git", "remote", "add", "origin", "https://github.com/user/repo.git")
				cmd.Dir = repoPath
				err := cmd.Run()
				require.NoError(t, err)
			}
			
			// Add current remote and set upstream
			if tt.currentRemote != "" {
				cmd := exec.Command("git", "remote", "add", tt.currentRemote, "ssh://dev@localhost:2200/workspace/project")
				cmd.Dir = repoPath
				err := cmd.Run()
				require.NoError(t, err)
				
				// Set upstream to current remote
				err = SetUpstream(repoPath, tt.branch, tt.currentRemote)
				require.NoError(t, err)
			}
			
			// Change upstream back to origin
			err := ChangeUpstreamToOrigin(repoPath, tt.branch)
			
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				
				// Verify upstream is now origin (if it was tracked)
				if tt.currentRemote != "" {
					cmd := exec.Command("git", "config", fmt.Sprintf("branch.%s.remote", tt.branch))
					cmd.Dir = repoPath
					output, err := cmd.Output()
					require.NoError(t, err)
					assert.Contains(t, string(output), "origin")
				}
			}
		})
	}
}

func TestGenerateSSHRemoteURL(t *testing.T) {
	tests := []struct {
		name           string
		containerName  string
		sshPort        int
		containerUser  string
		repoPath       string
		expected       string
	}{
		{
			name:          "standard SSH remote URL",
			containerName: "myproject",
			sshPort:       2200,
			containerUser: "dev",
			repoPath:      "/workspace/project",
			expected:      "dev-myproject:/workspace/project",
		},
		{
			name:          "custom user",
			containerName: "test",
			sshPort:       2201,
			containerUser: "lucian",
			repoPath:      "/workspace/myapp",
			expected:      "dev-test:/workspace/myapp",
		},
		{
			name:          "non-standard port",
			containerName: "app",
			sshPort:       3000,
			containerUser: "dev",
			repoPath:      "/workspace/app",
			expected:      "dev-app:/workspace/app",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := GenerateSSHRemoteURL(tt.containerName, tt.sshPort, tt.containerUser, tt.repoPath)
			assert.Equal(t, tt.expected, url)
		})
	}
}
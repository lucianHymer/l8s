package embed

import (
	"io/fs"
	"testing"
)

func TestGetDotfilesFS(t *testing.T) {
	// Test that we can get the embedded filesystem
	fsys, err := GetDotfilesFS()
	if err != nil {
		t.Fatalf("GetDotfilesFS() error = %v", err)
	}
	
	// Test that essential dotfiles exist
	essentialFiles := []string{
		".zshrc",
		".bashrc", 
		".gitconfig",
		".tmux.conf",
	}
	
	for _, file := range essentialFiles {
		t.Run("File_"+file, func(t *testing.T) {
			// Check file exists
			info, err := fs.Stat(fsys, file)
			if err != nil {
				t.Errorf("expected file %s to exist, got error: %v", file, err)
				return
			}
			
			// Check it's a regular file
			if info.IsDir() {
				t.Errorf("expected %s to be a file, but it's a directory", file)
			}
			
			// Check file is not empty
			if info.Size() == 0 {
				t.Errorf("expected %s to have content, but it's empty", file)
			}
		})
	}
}

func TestReadEmbeddedDotfile(t *testing.T) {
	fsys, err := GetDotfilesFS()
	if err != nil {
		t.Fatalf("GetDotfilesFS() error = %v", err)
	}
	
	// Test reading .gitconfig
	content, err := fs.ReadFile(fsys, ".gitconfig")
	if err != nil {
		t.Fatalf("failed to read .gitconfig: %v", err)
	}
	
	// Check content is not empty
	if len(content) == 0 {
		t.Error("expected .gitconfig to have content")
	}
	
	// Check for expected content
	contentStr := string(content)
	expectedPatterns := []string{
		"[core]",
		"editor",
	}
	
	for _, pattern := range expectedPatterns {
		if !contains(contentStr, pattern) {
			t.Errorf("expected .gitconfig to contain %q", pattern)
		}
	}
}

func TestListEmbeddedDotfiles(t *testing.T) {
	fsys, err := GetDotfilesFS()
	if err != nil {
		t.Fatalf("GetDotfilesFS() error = %v", err)
	}
	
	// List all files
	var files []string
	err = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && path != "." {
			files = append(files, path)
		}
		return nil
	})
	
	if err != nil {
		t.Fatalf("failed to walk embedded files: %v", err)
	}
	
	// Should have at least 4 dotfiles
	if len(files) < 4 {
		t.Errorf("expected at least 4 dotfiles, got %d: %v", len(files), files)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || len(s) > len(substr) && contains(s[1:], substr)
}
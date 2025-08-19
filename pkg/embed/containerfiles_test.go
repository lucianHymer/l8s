package embed

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContainerfileEmbedding(t *testing.T) {
	t.Run("Containerfile is embedded", func(t *testing.T) {
		if len(Containerfile) == 0 {
			t.Error("Containerfile is empty")
		}
		
		// Check for expected content
		if !strings.Contains(Containerfile, "FROM fedora:latest") {
			t.Error("Containerfile doesn't contain expected base image")
		}
		
		if !strings.Contains(Containerfile, "RUN useradd") {
			t.Error("Containerfile doesn't contain user creation")
		}
	})
	
	t.Run("Test Containerfile is embedded", func(t *testing.T) {
		if len(ContainerfileTest) == 0 {
			t.Error("ContainerfileTest is empty")
		}
		
		// Check for expected content
		if !strings.Contains(ContainerfileTest, "FROM fedora:latest") {
			t.Error("ContainerfileTest doesn't contain expected base image")
		}
	})
}

func TestExtractContainerfile(t *testing.T) {
	t.Run("ExtractContainerfile creates temp file", func(t *testing.T) {
		path, err := ExtractContainerfile()
		if err != nil {
			t.Fatalf("ExtractContainerfile failed: %v", err)
		}
		defer os.RemoveAll(filepath.Dir(path))
		
		// Check file exists
		if _, err := os.Stat(path); err != nil {
			t.Errorf("Extracted file doesn't exist: %v", err)
		}
		
		// Check file content
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read extracted file: %v", err)
		}
		
		if string(content) != Containerfile {
			t.Error("Extracted content doesn't match embedded Containerfile")
		}
		
		// Check filename
		if filepath.Base(path) != "Containerfile" {
			t.Errorf("Expected filename 'Containerfile', got %s", filepath.Base(path))
		}
	})
	
	t.Run("ExtractContainerfileTest creates temp file", func(t *testing.T) {
		path, err := ExtractContainerfileTest()
		if err != nil {
			t.Fatalf("ExtractContainerfileTest failed: %v", err)
		}
		defer os.RemoveAll(filepath.Dir(path))
		
		// Check file exists
		if _, err := os.Stat(path); err != nil {
			t.Errorf("Extracted file doesn't exist: %v", err)
		}
		
		// Check file content
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read extracted file: %v", err)
		}
		
		if string(content) != ContainerfileTest {
			t.Error("Extracted content doesn't match embedded ContainerfileTest")
		}
		
		// Check filename
		if filepath.Base(path) != "Containerfile.test" {
			t.Errorf("Expected filename 'Containerfile.test', got %s", filepath.Base(path))
		}
	})
	
	t.Run("Multiple extractions create different temp dirs", func(t *testing.T) {
		path1, err := ExtractContainerfile()
		if err != nil {
			t.Fatalf("First extraction failed: %v", err)
		}
		defer os.RemoveAll(filepath.Dir(path1))
		
		path2, err := ExtractContainerfile()
		if err != nil {
			t.Fatalf("Second extraction failed: %v", err)
		}
		defer os.RemoveAll(filepath.Dir(path2))
		
		if filepath.Dir(path1) == filepath.Dir(path2) {
			t.Error("Multiple extractions should create different temp directories")
		}
	})
}
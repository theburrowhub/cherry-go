package git

import (
	"cherry-go/internal/config"
	"cherry-go/internal/logger"
	"os"
	"path/filepath"
	"testing"
)

func TestShouldExclude(t *testing.T) {
	excludes := []string{"*.tmp", "test_*", "node_modules"}

	testCases := []struct {
		path     string
		expected bool
	}{
		{"file.txt", false},
		{"file.tmp", true},
		{"test_file.go", true},
		{"node_modules", true},
		{"src/main.go", false},
	}

	for _, tc := range testCases {
		result := shouldExclude(tc.path, excludes)
		if result != tc.expected {
			t.Errorf("shouldExclude(%s) = %t, expected %t", tc.path, result, tc.expected)
		}
	}
}

func TestGetAuth(t *testing.T) {
	logger.Init() // Initialize logger for tests
	
	// Test auto auth with HTTPS URL (should return nil if no env vars set)
	_, err := getAuth(config.AuthConfig{Type: "auto"}, "https://github.com/user/repo.git")
	if err != nil {
		t.Errorf("Expected no error for auto auth, got %v", err)
	}
	// Auth might be nil if no environment variables are set, which is fine
	
	// Test auto auth with SSH URL (should try SSH agent)
	_, err = getAuth(config.AuthConfig{Type: "auto"}, "git@github.com:user/repo.git")
	// This might fail if SSH agent is not available, which is expected in test environment
	if err != nil {
		t.Logf("SSH auth failed (expected in test environment): %v", err)
	}
	
	// Test basic auth without environment variables (should fail)
	basicConfig := config.AuthConfig{
		Type:     "basic",
		Username: "user",
	}
	_, err = getAuth(basicConfig, "https://github.com/user/repo.git")
	if err == nil {
		t.Error("Expected error for basic auth without password environment variable")
	}
}

func TestCopyFile(t *testing.T) {
	logger.Init() // Initialize logger for tests

	// Create temporary directories
	tmpDir, err := os.MkdirTemp("", "cherry-go-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source file
	srcPath := filepath.Join(tmpDir, "source.txt")
	srcContent := "test content"
	if err := os.WriteFile(srcPath, []byte(srcContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Copy file
	dstPath := filepath.Join(tmpDir, "subdir", "dest.txt")
	if err := copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("Failed to copy file: %v", err)
	}

	// Verify destination file
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(dstContent) != srcContent {
		t.Errorf("Expected content %s, got %s", srcContent, string(dstContent))
	}
}

func TestCopyDir(t *testing.T) {
	logger.Init() // Initialize logger for tests

	// Create temporary directories
	tmpDir, err := os.MkdirTemp("", "cherry-go-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source directory structure
	srcDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(filepath.Join(srcDir, "subdir"), 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create files
	files := map[string]string{
		"file1.txt":        "content1",
		"file2.tmp":        "temp content",
		"subdir/file3.txt": "content3",
	}

	for filePath, content := range files {
		fullPath := filepath.Join(srcDir, filePath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", filePath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", filePath, err)
		}
	}

	// Copy directory with excludes
	dstDir := filepath.Join(tmpDir, "dst")
	excludes := []string{"*.tmp"}

	if err := copyDir(srcDir, dstDir, excludes); err != nil {
		t.Fatalf("Failed to copy directory: %v", err)
	}

	// Verify copied files
	expectedFiles := []string{
		"file1.txt",
		"subdir/file3.txt",
	}

	for _, filePath := range expectedFiles {
		fullPath := filepath.Join(dstDir, filePath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist", filePath)
		}
	}

	// Verify excluded files don't exist
	excludedFile := filepath.Join(dstDir, "file2.tmp")
	if _, err := os.Stat(excludedFile); !os.IsNotExist(err) {
		t.Error("Expected file2.tmp to be excluded")
	}
}

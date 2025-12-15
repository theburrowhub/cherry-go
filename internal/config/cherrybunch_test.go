package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCherryBunch(t *testing.T) {
	// Create a temporary cherry bunch file
	tmpDir := t.TempDir()
	cbFile := filepath.Join(tmpDir, "test.cherrybunch")

	cbContent := `name: test-bunch
description: Test cherry bunch
version: "1.0"
repository: https://github.com/test/repo.git
files:
  - path: file1.txt
    local_path: local1.txt
    branch: main
  - path: file2.txt
    local_path: local2.txt
    branch: develop
directories:
  - path: src/
    local_path: local-src/
    branch: main
    exclude:
      - "*.tmp"
      - "cache/"
`

	err := os.WriteFile(cbFile, []byte(cbContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test loading the cherry bunch
	cb, err := LoadCherryBunch(cbFile)
	if err != nil {
		t.Fatalf("Failed to load cherry bunch: %v", err)
	}

	// Verify basic fields
	if cb.Name != "test-bunch" {
		t.Errorf("Expected name 'test-bunch', got '%s'", cb.Name)
	}
	if cb.Description != "Test cherry bunch" {
		t.Errorf("Expected description 'Test cherry bunch', got '%s'", cb.Description)
	}
	if cb.Version != "1.0" {
		t.Errorf("Expected version '1.0', got '%s'", cb.Version)
	}
	if cb.Repository != "https://github.com/test/repo.git" {
		t.Errorf("Expected repository 'https://github.com/test/repo.git', got '%s'", cb.Repository)
	}

	// Verify files
	if len(cb.Files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(cb.Files))
	}
	if cb.Files[0].Path != "file1.txt" {
		t.Errorf("Expected first file path 'file1.txt', got '%s'", cb.Files[0].Path)
	}
	if cb.Files[0].LocalPath != "local1.txt" {
		t.Errorf("Expected first file local path 'local1.txt', got '%s'", cb.Files[0].LocalPath)
	}
	if cb.Files[0].Branch != "main" {
		t.Errorf("Expected first file branch 'main', got '%s'", cb.Files[0].Branch)
	}

	// Verify directories
	if len(cb.Directories) != 1 {
		t.Errorf("Expected 1 directory, got %d", len(cb.Directories))
	}
	if cb.Directories[0].Path != "src/" {
		t.Errorf("Expected directory path 'src/', got '%s'", cb.Directories[0].Path)
	}
	if cb.Directories[0].LocalPath != "local-src/" {
		t.Errorf("Expected directory local path 'local-src/', got '%s'", cb.Directories[0].LocalPath)
	}
	if len(cb.Directories[0].Exclude) != 2 {
		t.Errorf("Expected 2 exclude patterns, got %d", len(cb.Directories[0].Exclude))
	}
}

func TestLoadCherryBunchFromData(t *testing.T) {
	cbContent := `name: data-test
description: Test from data
version: "2.0"
repository: https://github.com/data/repo.git
files:
  - path: test.go
    local_path: test.go
    branch: main
`

	cb, err := LoadCherryBunchFromData([]byte(cbContent))
	if err != nil {
		t.Fatalf("Failed to load cherry bunch from data: %v", err)
	}

	if cb.Name != "data-test" {
		t.Errorf("Expected name 'data-test', got '%s'", cb.Name)
	}
	if cb.Version != "2.0" {
		t.Errorf("Expected version '2.0', got '%s'", cb.Version)
	}
}

func TestLoadCherryBunchValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing name",
			content: `description: Test
repository: https://github.com/test/repo.git`,
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "missing repository",
			content: `name: test
description: Test`,
			wantErr: true,
			errMsg:  "repository is required",
		},
		{
			name: "invalid yaml",
			content: `name: test
repository: https://github.com/test/repo.git
files:
  - path: test
    invalid_indent`,
			wantErr: true,
		},
		{
			name: "default version",
			content: `name: test
repository: https://github.com/test/repo.git`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb, err := LoadCherryBunchFromData([]byte(tt.content))

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if tt.errMsg != "" && err != nil {
					if !containsString(err.Error(), tt.errMsg) {
						t.Errorf("Expected error message to contain '%s', got '%s'", tt.errMsg, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				// Check default version is set
				if cb != nil && cb.Version == "" {
					t.Errorf("Expected default version to be set")
				}
			}
		})
	}
}

func TestApplyCherryBunch(t *testing.T) {
	config := DefaultConfig()

	cb := &CherryBunch{
		Name:        "test-apply",
		Description: "Test application",
		Version:     "1.0",
		Repository:  "https://github.com/test/apply.git",
		Files: []CherryBunchFileSpec{
			{Path: "README.md", LocalPath: "README.md", Branch: "main"},
		},
		Directories: []CherryBunchDirSpec{
			{Path: "src/", LocalPath: "src/", Branch: "main", Exclude: []string{"*.tmp"}},
		},
	}

	err := config.ApplyCherryBunch(cb)
	if err != nil {
		t.Fatalf("Failed to apply cherry bunch: %v", err)
	}

	// Check that source was added
	if len(config.Sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(config.Sources))
	}

	source := config.Sources[0]
	if source.Name != "test-apply" {
		t.Errorf("Expected source name 'test-apply', got '%s'", source.Name)
	}
	if source.Repository != "https://github.com/test/apply.git" {
		t.Errorf("Expected source repository 'https://github.com/test/apply.git', got '%s'", source.Repository)
	}
	if len(source.Paths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(source.Paths))
	}

	// Check file path
	filePath := source.Paths[0]
	if filePath.Include != "README.md" {
		t.Errorf("Expected file include 'README.md', got '%s'", filePath.Include)
	}
	if filePath.LocalPath != "README.md" {
		t.Errorf("Expected file local path 'README.md', got '%s'", filePath.LocalPath)
	}

	// Check directory path
	dirPath := source.Paths[1]
	if dirPath.Include != "src/" {
		t.Errorf("Expected directory include 'src/', got '%s'", dirPath.Include)
	}
	if len(dirPath.Exclude) != 1 {
		t.Errorf("Expected 1 exclude pattern, got %d", len(dirPath.Exclude))
	}
}

func TestSaveCherryBunch(t *testing.T) {
	tmpDir := t.TempDir()
	cbFile := filepath.Join(tmpDir, "save-test.cherrybunch")

	cb := &CherryBunch{
		Name:        "save-test",
		Description: "Test saving",
		Version:     "1.0",
		Repository:  "https://github.com/test/save.git",
		Files: []CherryBunchFileSpec{
			{Path: "main.go", LocalPath: "main.go", Branch: "main"},
		},
	}

	err := cb.Save(cbFile)
	if err != nil {
		t.Fatalf("Failed to save cherry bunch: %v", err)
	}

	// Verify file was created
	if _, statErr := os.Stat(cbFile); os.IsNotExist(statErr) {
		t.Errorf("Cherry bunch file was not created")
	}

	// Load it back and verify
	loadedCb, err := LoadCherryBunch(cbFile)
	if err != nil {
		t.Fatalf("Failed to load saved cherry bunch: %v", err)
	}

	if loadedCb.Name != cb.Name {
		t.Errorf("Expected name '%s', got '%s'", cb.Name, loadedCb.Name)
	}
	if loadedCb.Repository != cb.Repository {
		t.Errorf("Expected repository '%s', got '%s'", cb.Repository, loadedCb.Repository)
	}
}

func TestIsURL(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"https://example.com", true},
		{"http://example.com", true},
		{"ftp://example.com", false},
		{"./local/file.txt", false},
		{"/absolute/path.txt", false},
		{"relative/path.txt", false},
		{"https://", true},
		{"http://", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isURL(tt.input)
			if result != tt.expected {
				t.Errorf("isURL(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

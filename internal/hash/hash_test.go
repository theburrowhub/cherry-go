package hash

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashFile(t *testing.T) {
	// Create temporary file
	tmpDir, err := os.MkdirTemp("", "hash-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hasher := NewFileHasher()
	hash1, err := hasher.HashFile(testFile)
	if err != nil {
		t.Fatalf("Failed to hash file: %v", err)
	}

	// Hash should be consistent
	hash2, err := hasher.HashFile(testFile)
	if err != nil {
		t.Fatalf("Failed to hash file again: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("Hash should be consistent: %s != %s", hash1, hash2)
	}

	// Expected SHA256 of "Hello, World!"
	expected := "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f"
	if hash1 != expected {
		t.Errorf("Expected hash %s, got %s", expected, hash1)
	}
}

func TestHashDirectory(t *testing.T) {
	// Create temporary directory structure
	tmpDir, err := os.MkdirTemp("", "hash-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test files
	files := map[string]string{
		"file1.txt":        "content1",
		"file2.txt":        "content2",
		"file3.tmp":        "temp content",
		"subdir/file4.txt": "content4",
	}

	for filePath, content := range files {
		fullPath := filepath.Join(tmpDir, filePath)
		if mkdirErr := os.MkdirAll(filepath.Dir(fullPath), 0755); mkdirErr != nil {
			t.Fatalf("Failed to create directory for %s: %v", filePath, mkdirErr)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", filePath, err)
		}
	}

	hasher := NewFileHasher()
	excludes := []string{"*.tmp"}

	hashes, err := hasher.HashDirectory(tmpDir, excludes)
	if err != nil {
		t.Fatalf("Failed to hash directory: %v", err)
	}

	// Should have 3 files (excluding *.tmp)
	expectedFiles := 3
	if len(hashes) != expectedFiles {
		t.Errorf("Expected %d files, got %d", expectedFiles, len(hashes))
	}

	// Check specific files
	if _, exists := hashes["file1.txt"]; !exists {
		t.Error("file1.txt should be in hashes")
	}
	if _, exists := hashes["file2.txt"]; !exists {
		t.Error("file2.txt should be in hashes")
	}
	if _, exists := hashes["file3.tmp"]; exists {
		t.Error("file3.tmp should be excluded")
	}
	if _, exists := hashes[filepath.Join("subdir", "file4.txt")]; !exists {
		t.Error("subdir/file4.txt should be in hashes")
	}
}

func TestCompareHashes(t *testing.T) {
	hasher := NewFileHasher()

	oldHashes := map[string]string{
		"file1.txt": "hash1",
		"file2.txt": "hash2",
		"file3.txt": "hash3",
	}

	newHashes := map[string]string{
		"file1.txt": "hash1",     // unchanged
		"file2.txt": "hash2_new", // modified
		"file4.txt": "hash4",     // added
		// file3.txt is removed
	}

	modified, added, removed := hasher.CompareHashes(oldHashes, newHashes)

	// Check modified
	if len(modified) != 1 || modified[0] != "file2.txt" {
		t.Errorf("Expected modified: [file2.txt], got: %v", modified)
	}

	// Check added
	if len(added) != 1 || added[0] != "file4.txt" {
		t.Errorf("Expected added: [file4.txt], got: %v", added)
	}

	// Check removed
	if len(removed) != 1 || removed[0] != "file3.txt" {
		t.Errorf("Expected removed: [file3.txt], got: %v", removed)
	}
}

func TestVerifyFileIntegrity(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "hash-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := "original content"
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hasher := NewFileHasher()
	originalHash, err := hasher.HashFile(testFile)
	if err != nil {
		t.Fatalf("Failed to hash file: %v", err)
	}

	expectedHashes := map[string]string{
		"test.txt": originalHash,
	}

	// Test 1: No conflicts (file unchanged)
	conflicts, err := hasher.VerifyFileIntegrity(tmpDir, expectedHashes)
	if err != nil {
		t.Fatalf("Failed to verify integrity: %v", err)
	}
	if len(conflicts) != 0 {
		t.Errorf("Expected no conflicts, got: %v", conflicts)
	}

	// Test 2: File modified
	modifiedContent := "modified content"
	if err := os.WriteFile(testFile, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	conflicts, err = hasher.VerifyFileIntegrity(tmpDir, expectedHashes)
	if err != nil {
		t.Fatalf("Failed to verify integrity: %v", err)
	}
	if len(conflicts) != 1 {
		t.Errorf("Expected 1 conflict, got: %d", len(conflicts))
	}
	if conflicts[0].Type != ConflictTypeModified {
		t.Errorf("Expected conflict type modified, got: %s", conflicts[0].Type)
	}

	// Test 3: File deleted
	if removeErr := os.Remove(testFile); removeErr != nil {
		t.Fatalf("Failed to remove test file: %v", removeErr)
	}

	conflicts, err = hasher.VerifyFileIntegrity(tmpDir, expectedHashes)
	if err != nil {
		t.Fatalf("Failed to verify integrity: %v", err)
	}
	if len(conflicts) != 1 {
		t.Errorf("Expected 1 conflict, got: %d", len(conflicts))
	}
	if conflicts[0].Type != ConflictTypeDeleted {
		t.Errorf("Expected conflict type deleted, got: %s", conflicts[0].Type)
	}
}

func TestFileConflictString(t *testing.T) {
	testCases := []struct {
		conflict FileConflict
		expected string
	}{
		{
			FileConflict{
				Path:         "test.txt",
				Type:         ConflictTypeModified,
				ExpectedHash: "abcdef1234567890",
				ActualHash:   "1234567890abcdef",
			},
			"Modified: test.txt (expected: abcdef12, actual: 12345678)",
		},
		{
			FileConflict{
				Path:         "deleted.txt",
				Type:         ConflictTypeDeleted,
				ExpectedHash: "abcdef1234567890",
				ActualHash:   "",
			},
			"Deleted: deleted.txt (expected: abcdef12)",
		},
		{
			FileConflict{
				Path:         "new.txt",
				Type:         ConflictTypeAdded,
				ExpectedHash: "",
				ActualHash:   "1234567890abcdef",
			},
			"Added: new.txt (actual: 12345678)",
		},
	}

	for i, tc := range testCases {
		result := tc.conflict.String()
		if result != tc.expected {
			t.Errorf("Test case %d: expected %q, got %q", i, tc.expected, result)
		}
	}
}

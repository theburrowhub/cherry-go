package merge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestThreeWayMerge_CleanMerge(t *testing.T) {
	// Base content
	base := []byte("line1\nline2\nline3\n")
	// Local adds line at end
	local := []byte("line1\nline2\nline3\nlocal addition\n")
	// Remote modifies line2
	remote := []byte("line1\nmodified line2\nline3\n")

	result, err := ThreeWayMerge(base, local, remote)
	if err != nil {
		t.Fatalf("ThreeWayMerge failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected merge to succeed")
	}

	if result.HasConflict {
		t.Error("Expected no conflicts")
	}

	// Check that merged content contains both changes
	content := string(result.Content)
	if !strings.Contains(content, "modified line2") {
		t.Error("Merged content should contain remote's modification")
	}
	if !strings.Contains(content, "local addition") {
		t.Error("Merged content should contain local's addition")
	}
}

func TestThreeWayMerge_ConflictingChanges(t *testing.T) {
	// Base content
	base := []byte("line1\nline2\nline3\n")
	// Local modifies line2
	local := []byte("line1\nlocal change to line2\nline3\n")
	// Remote also modifies line2 differently
	remote := []byte("line1\nremote change to line2\nline3\n")

	result, err := ThreeWayMerge(base, local, remote)
	if err != nil {
		t.Fatalf("ThreeWayMerge failed: %v", err)
	}

	if result.Success {
		t.Error("Expected merge to fail due to conflict")
	}

	if !result.HasConflict {
		t.Error("Expected conflict to be detected")
	}

	// Content should contain conflict markers
	if !ContainsConflictMarkers(result.Content) {
		t.Error("Merged content should contain conflict markers")
	}
}

func TestThreeWayMerge_IdenticalChanges(t *testing.T) {
	// Base content
	base := []byte("line1\nline2\nline3\n")
	// Both local and remote make the same change
	local := []byte("line1\nsame change\nline3\n")
	remote := []byte("line1\nsame change\nline3\n")

	result, err := ThreeWayMerge(base, local, remote)
	if err != nil {
		t.Fatalf("ThreeWayMerge failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected merge to succeed for identical changes")
	}

	if result.HasConflict {
		t.Error("Expected no conflicts for identical changes")
	}
}

func TestThreeWayMerge_OnlyLocalChanges(t *testing.T) {
	base := []byte("line1\nline2\nline3\n")
	local := []byte("line1\nlocal modified\nline3\n")
	remote := []byte("line1\nline2\nline3\n") // Unchanged from base

	result, err := ThreeWayMerge(base, local, remote)
	if err != nil {
		t.Fatalf("ThreeWayMerge failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected merge to succeed")
	}

	// Should keep local changes
	if !strings.Contains(string(result.Content), "local modified") {
		t.Error("Should keep local changes when remote is unchanged")
	}
}

func TestThreeWayMerge_OnlyRemoteChanges(t *testing.T) {
	base := []byte("line1\nline2\nline3\n")
	local := []byte("line1\nline2\nline3\n") // Unchanged from base
	remote := []byte("line1\nremote modified\nline3\n")

	result, err := ThreeWayMerge(base, local, remote)
	if err != nil {
		t.Fatalf("ThreeWayMerge failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected merge to succeed")
	}

	// Should have remote changes
	if !strings.Contains(string(result.Content), "remote modified") {
		t.Error("Should have remote changes when local is unchanged")
	}
}

func TestPatchBasedMerge_PreservesLocalAdditions(t *testing.T) {
	// This test verifies the key behavior: local additions are preserved
	// while remote modifications are applied
	base := []byte("# Test file\n\nOriginal line\n")
	// Local added a new line
	local := []byte("# Test file\n\nOriginal line\nLocal addition\n")
	// Remote modified the original line
	remote := []byte("# Test file\n\nModified line\n")

	result, err := ThreeWayMerge(base, local, remote)
	if err != nil {
		t.Fatalf("ThreeWayMerge failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected merge to succeed, got conflict. Content:\n%s", result.Content)
	}

	content := string(result.Content)

	// Should have the remote's modification
	if !strings.Contains(content, "Modified line") {
		t.Error("Should contain remote's modification")
	}

	// Should preserve local addition
	if !strings.Contains(content, "Local addition") {
		t.Error("Should preserve local addition")
	}
}

func TestPatchBasedMerge_RemoteFixAndLocalAddition(t *testing.T) {
	// Simulates the user's exact scenario:
	// - Base has "Change made from source"
	// - Local has "Change made from source" + "Change mada in project" (typo + addition)
	// - Remote has "Change made in source" (fixed "from" to "in")
	// Expected: "Change made in source" + "Change mada in project"
	base := []byte("# Test file\n\nChange made from source\n")
	local := []byte("# Test file\n\nChange made from source\nChange mada in project\n")
	remote := []byte("# Test file\n\nChange made in source\n")

	result, err := ThreeWayMerge(base, local, remote)
	if err != nil {
		t.Fatalf("ThreeWayMerge failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected merge to succeed, got conflict. Content:\n%s", result.Content)
	}

	content := string(result.Content)

	// Should have remote's fix ("in" instead of "from")
	if !strings.Contains(content, "Change made in source") {
		t.Error("Should contain remote's fix 'Change made in source'")
	}

	// Should preserve local addition
	if !strings.Contains(content, "Change mada in project") {
		t.Error("Should preserve local addition 'Change mada in project'")
	}

	// Should NOT have the old version
	if strings.Contains(content, "Change made from source") {
		t.Error("Should not contain old version 'Change made from source'")
	}
}

func TestMergeFile(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "merge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	basePath := filepath.Join(tempDir, "base.txt")
	localPath := filepath.Join(tempDir, "local.txt")
	remotePath := filepath.Join(tempDir, "remote.txt")

	os.WriteFile(basePath, []byte("original content\n"), 0644)
	os.WriteFile(localPath, []byte("local content\n"), 0644)
	os.WriteFile(remotePath, []byte("remote content\n"), 0644)

	result, err := MergeFile(basePath, localPath, remotePath)
	if err != nil {
		t.Fatalf("MergeFile failed: %v", err)
	}

	// Both local and remote changed the same content, should conflict
	if !result.HasConflict {
		t.Error("Expected conflict when both sides modify same content")
	}
}

func TestContainsConflictMarkers(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "with conflict markers",
			content:  []byte("line1\n<<<<<<< LOCAL\nlocal\n=======\nremote\n>>>>>>> REMOTE\n"),
			expected: true,
		},
		{
			name:     "without conflict markers",
			content:  []byte("line1\nline2\nline3\n"),
			expected: false,
		},
		{
			name:     "partial markers",
			content:  []byte("line1\n<<<<<<<\nline3\n"),
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ContainsConflictMarkers(tc.content)
			if result != tc.expected {
				t.Errorf("ContainsConflictMarkers(%q) = %v, want %v", tc.content, result, tc.expected)
			}
		})
	}
}

func TestIsBinaryFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "binary-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create text file
	textPath := filepath.Join(tempDir, "text.txt")
	os.WriteFile(textPath, []byte("hello world\n"), 0644)

	// Create binary file (with null bytes)
	binaryPath := filepath.Join(tempDir, "binary.bin")
	os.WriteFile(binaryPath, []byte("hello\x00world\n"), 0644)

	if isBinaryFile(textPath) {
		t.Error("Text file should not be detected as binary")
	}

	if !isBinaryFile(binaryPath) {
		t.Error("Binary file should be detected as binary")
	}
}

func TestHasConflicts(t *testing.T) {
	resultsWithConflict := []FileMergeResult{
		{Path: "file1.txt", Result: MergeResult{Success: true}},
		{Path: "file2.txt", Result: MergeResult{HasConflict: true}},
	}

	resultsWithoutConflict := []FileMergeResult{
		{Path: "file1.txt", Result: MergeResult{Success: true}},
		{Path: "file2.txt", Result: MergeResult{Success: true}},
	}

	if !HasConflicts(resultsWithConflict) {
		t.Error("Should detect conflicts")
	}

	if HasConflicts(resultsWithoutConflict) {
		t.Error("Should not detect conflicts")
	}
}

func TestGetConflictedFiles(t *testing.T) {
	results := []FileMergeResult{
		{Path: "file1.txt", Result: MergeResult{Success: true}},
		{Path: "file2.txt", Result: MergeResult{HasConflict: true}},
		{Path: "file3.txt", Error: os.ErrNotExist},
	}

	conflicted := GetConflictedFiles(results)

	if len(conflicted) != 2 {
		t.Errorf("Expected 2 conflicted files, got %d", len(conflicted))
	}

	expected := map[string]bool{"file2.txt": true, "file3.txt": true}
	for _, f := range conflicted {
		if !expected[f] {
			t.Errorf("Unexpected conflicted file: %s", f)
		}
	}
}

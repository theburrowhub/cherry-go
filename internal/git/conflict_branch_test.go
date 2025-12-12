package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetMergeInstructions(t *testing.T) {
	result := &ConflictBranchResult{
		BranchName:     "cherry-go/sync/mylib-20241212-120000",
		OriginalBranch: "main",
		FilesCommitted: []string{"src/utils.go", "src/config.go"},
	}

	instructions := GetMergeInstructions(result)

	// Verify key content is present
	if !strings.Contains(instructions, result.BranchName) {
		t.Error("Instructions should contain branch name")
	}

	if !strings.Contains(instructions, "git merge") {
		t.Error("Instructions should contain merge command")
	}

	for _, file := range result.FilesCommitted {
		if !strings.Contains(instructions, file) {
			t.Errorf("Instructions should list file: %s", file)
		}
	}

	if !strings.Contains(instructions, "git branch -d") {
		t.Error("Instructions should explain how to delete conflict branch")
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		input    string
		length   int
		expected string
	}{
		{"short", 10, "short     ║"},
		{"exactly10!", 10, "exactly10!"},
		{"longer than expected", 5, "longer than expected"},
	}

	for _, tc := range tests {
		result := padRight(tc.input, tc.length)
		if result != tc.expected {
			t.Errorf("padRight(%q, %d) = %q, want %q", tc.input, tc.length, result, tc.expected)
		}
	}
}

// Integration test - requires git to be installed
func TestCreateConflictBranch_Integration(t *testing.T) {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available, skipping integration test")
	}

	// Create temp directory for test repo
	tempDir, err := os.MkdirTemp("", "conflict-branch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tempDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tempDir
	cmd.Run()

	// Create initial file and commit
	initialFile := filepath.Join(tempDir, "file.txt")
	os.WriteFile(initialFile, []byte("initial content\n"), 0644)

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tempDir
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "initial commit")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	// Test creating conflict branch
	files := map[string][]byte{
		"file.txt":     []byte("remote content\n"),
		"new_file.txt": []byte("new file content\n"),
	}

	result, err := CreateConflictBranch(tempDir, "cherry-go/sync", "test-source", files)
	if err != nil {
		t.Fatalf("CreateConflictBranch failed: %v", err)
	}

	// Verify result
	if result.BranchName == "" {
		t.Error("Branch name should not be empty")
	}

	if !strings.HasPrefix(result.BranchName, "cherry-go/sync/test-source-") {
		t.Errorf("Branch name should have correct prefix, got: %s", result.BranchName)
	}

	if len(result.FilesCommitted) != 2 {
		t.Errorf("Expected 2 files committed, got %d", len(result.FilesCommitted))
	}

	// Verify we're back on original branch
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = tempDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	currentBranch := strings.TrimSpace(string(output))
	if currentBranch != result.OriginalBranch {
		t.Errorf("Should be back on original branch %s, got %s", result.OriginalBranch, currentBranch)
	}

	// Verify conflict branch exists
	cmd = exec.Command("git", "branch", "--list", result.BranchName)
	cmd.Dir = tempDir
	output, err = cmd.Output()
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}

	if !strings.Contains(string(output), "test-source") {
		t.Error("Conflict branch should exist")
	}

	// Test deleting conflict branch
	err = DeleteConflictBranch(tempDir, result.BranchName)
	if err != nil {
		t.Errorf("DeleteConflictBranch failed: %v", err)
	}
}

func TestCreateConflictBranch_NotGitRepo(t *testing.T) {
	// Create temp directory (not a git repo)
	tempDir, err := os.MkdirTemp("", "not-git-repo-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	files := map[string][]byte{
		"file.txt": []byte("content"),
	}

	_, err = CreateConflictBranch(tempDir, "prefix", "source", files)
	if err == nil {
		t.Error("CreateConflictBranch should fail in non-git directory")
	}
}

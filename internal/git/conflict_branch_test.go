package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
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

func TestListConflictBranches(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "list-branches-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repo
	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create initial commit
	worktree, _ := repo.Worktree()
	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)
	worktree.Add("test.txt")
	worktree.Commit("initial", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@test.com",
			When:  time.Now(),
		},
	})

	// Create some conflict branches
	files := map[string][]byte{
		"conflict1.txt": []byte("conflict 1"),
	}
	result1, err := CreateConflictBranch(tempDir, "cherry-go/sync", "source1", files)
	if err != nil {
		t.Fatalf("Failed to create conflict branch 1: %v", err)
	}

	result2, err := CreateConflictBranch(tempDir, "cherry-go/sync", "source2", files)
	if err != nil {
		t.Fatalf("Failed to create conflict branch 2: %v", err)
	}

	// Create a non-conflict branch
	worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("feature/test"),
		Create: true,
	})
	worktree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/heads/" + result1.OriginalBranch),
	})

	// List conflict branches
	branches, err := ListConflictBranches(tempDir, "cherry-go/sync")
	if err != nil {
		t.Fatalf("ListConflictBranches failed: %v", err)
	}

	if len(branches) != 2 {
		t.Errorf("Expected 2 conflict branches, got %d", len(branches))
	}

	// Verify branch names
	found1, found2 := false, false
	for _, branch := range branches {
		if branch == result1.BranchName {
			found1 = true
		}
		if branch == result2.BranchName {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Error("Not all expected conflict branches were found")
	}
}

func TestDeleteAllConflictBranches(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "delete-all-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repo
	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create initial commit
	worktree, _ := repo.Worktree()
	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)
	worktree.Add("test.txt")
	worktree.Commit("initial", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@test.com",
			When:  time.Now(),
		},
	})

	// Create some conflict branches
	files := map[string][]byte{
		"conflict.txt": []byte("conflict"),
	}
	CreateConflictBranch(tempDir, "cherry-go/sync", "source1", files)
	CreateConflictBranch(tempDir, "cherry-go/sync", "source2", files)

	// Delete all conflict branches
	deleted, err := DeleteAllConflictBranches(tempDir, "cherry-go/sync")
	if err != nil {
		t.Fatalf("DeleteAllConflictBranches failed: %v", err)
	}

	if len(deleted) != 2 {
		t.Errorf("Expected 2 deleted branches, got %d", len(deleted))
	}

	// Verify branches are deleted
	branches, err := ListConflictBranches(tempDir, "cherry-go/sync")
	if err != nil {
		t.Fatalf("ListConflictBranches failed: %v", err)
	}

	if len(branches) != 0 {
		t.Errorf("Expected 0 conflict branches after deletion, got %d", len(branches))
	}
}

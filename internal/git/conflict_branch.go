package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// ConflictBranchResult contains information about a created conflict branch
type ConflictBranchResult struct {
	BranchName     string
	OriginalBranch string
	FilesCommitted []string
}

// CreateConflictBranch creates a new branch with the remote content for manual merge
func CreateConflictBranch(workDir string, branchPrefix string, sourceName string, files map[string][]byte) (*ConflictBranchResult, error) {
	repo, err := git.PlainOpen(workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Get current branch name
	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}
	originalBranch := head.Name().Short()

	// Generate branch name with timestamp
	timestamp := time.Now().Format("20060102-150405")
	branchName := fmt.Sprintf("%s/%s-%s", branchPrefix, sourceName, timestamp)

	// Get worktree
	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	// Create and checkout new branch
	branchRef := plumbing.NewBranchReferenceName(branchName)
	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: branchRef,
		Create: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create branch %s: %w", branchName, err)
	}

	// Write remote files to the branch
	var committedFiles []string
	for relPath, content := range files {
		fullPath := filepath.Join(workDir, relPath)

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			// Try to checkout back to original branch on error
			_ = worktree.Checkout(&git.CheckoutOptions{Branch: head.Name()})
			return nil, fmt.Errorf("failed to create directory for %s: %w", relPath, err)
		}

		// Write file
		if err := os.WriteFile(fullPath, content, 0644); err != nil {
			_ = worktree.Checkout(&git.CheckoutOptions{Branch: head.Name()})
			return nil, fmt.Errorf("failed to write file %s: %w", relPath, err)
		}

		// Stage the file
		if _, err := worktree.Add(relPath); err != nil {
			_ = worktree.Checkout(&git.CheckoutOptions{Branch: head.Name()})
			return nil, fmt.Errorf("failed to stage file %s: %w", relPath, err)
		}

		committedFiles = append(committedFiles, relPath)
	}

	// Create commit with remote changes
	commitMessage := fmt.Sprintf("cherry-go: remote changes from %s\n\nThis branch contains the remote changes that conflicted with local modifications.\nUse 'git merge %s' from your original branch to resolve conflicts.", sourceName, branchName)

	_, err = worktree.Commit(commitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "cherry-go",
			Email: "cherry-go@local",
			When:  time.Now(),
		},
	})
	if err != nil {
		_ = worktree.Checkout(&git.CheckoutOptions{Branch: head.Name()})
		return nil, fmt.Errorf("failed to create commit: %w", err)
	}

	// Checkout back to original branch
	err = worktree.Checkout(&git.CheckoutOptions{
		Branch: head.Name(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to checkout back to %s: %w", originalBranch, err)
	}

	return &ConflictBranchResult{
		BranchName:     branchName,
		OriginalBranch: originalBranch,
		FilesCommitted: committedFiles,
	}, nil
}

// GetMergeInstructions generates instructions for manual merge resolution
func GetMergeInstructions(result *ConflictBranchResult) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("⚠️  Merge Conflicts - Remote changes saved to branch\n\n")
	sb.WriteString(fmt.Sprintf("Branch: %s\n", result.BranchName))

	if len(result.FilesCommitted) > 0 {
		sb.WriteString("\nFiles with conflicts:\n")
		for _, file := range result.FilesCommitted {
			sb.WriteString(fmt.Sprintf("  • %s\n", file))
		}
	}

	sb.WriteString("\nNext steps:\n")
	sb.WriteString("Review the changes in the branch and merge when ready.\n")
	sb.WriteString("The branch contains the remote version - adjust as needed before merging.\n\n")
	sb.WriteString(fmt.Sprintf("  git diff %s              # Review changes\n", result.BranchName))
	sb.WriteString(fmt.Sprintf("  git merge %s             # Merge when ready\n", result.BranchName))
	sb.WriteString(fmt.Sprintf("  git branch -d %s   # Delete branch after merge\n", result.BranchName))
	sb.WriteString("\n")

	return sb.String()
}

// DeleteConflictBranch deletes a conflict branch after successful resolution
// Note: Currently used primarily for testing; may be used for cleanup in future versions
func DeleteConflictBranch(workDir string, branchName string) error {
	repo, err := git.PlainOpen(workDir)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	branchRef := plumbing.NewBranchReferenceName(branchName)
	err = repo.Storer.RemoveReference(branchRef)
	if err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", branchName, err)
	}

	return nil
}

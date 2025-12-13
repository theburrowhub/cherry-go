package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitUtils provides simple Git utility functions
type GitUtils struct{}

// NewGitUtils creates a new GitUtils instance
func NewGitUtils() *GitUtils {
	return &GitUtils{}
}

// GetRepositoryRoot returns the root directory of the Git repository
func (g *GitUtils) GetRepositoryRoot(path string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = path

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository or git not available: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetRemoteURL returns the URL of the specified remote
func (g *GitUtils) GetRemoteURL(path, remote string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", remote)
	cmd.Dir = path

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetCurrentBranch returns the current branch name
func (g *GitUtils) GetCurrentBranch(path string) (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = path

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	branch := strings.TrimSpace(string(output))
	if branch == "" {
		// Fallback for detached HEAD
		cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
		cmd.Dir = path

		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get branch info: %w", err)
		}

		branch = strings.TrimSpace(string(output))
		if branch == "HEAD" {
			return "main", nil // Default fallback
		}
	}

	return branch, nil
}

// IsGitRepository checks if the path is within a Git repository
func (g *GitUtils) IsGitRepository(path string) bool {
	_, err := g.GetRepositoryRoot(path)
	return err == nil
}

// ListFiles returns all files in the repository relative to the repo root
func (g *GitUtils) ListFiles(path string) ([]string, error) {
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = path

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list git files: %w", err)
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(files) == 1 && files[0] == "" {
		return []string{}, nil
	}

	return files, nil
}

// ListDirectories returns all directories in the repository
func (g *GitUtils) ListDirectories(path string) ([]string, error) {
	repoRoot, err := g.GetRepositoryRoot(path)
	if err != nil {
		return nil, err
	}

	var dirs []string
	err = filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Skip .git directory
			if strings.Contains(path, ".git") {
				return filepath.SkipDir
			}

			// Get relative path from repo root
			relPath, err := filepath.Rel(repoRoot, path)
			if err != nil {
				return err
			}

			// Skip root directory
			if relPath != "." {
				dirs = append(dirs, relPath)
			}
		}

		return nil
	})

	return dirs, err
}

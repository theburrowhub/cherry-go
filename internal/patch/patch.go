package patch

import (
	"cherry-go/internal/config"
	"cherry-go/internal/logger"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// ConflictResolution represents the type of conflict resolution needed
type ConflictResolution int

const (
	ResolutionStandard ConflictResolution = iota // Standard overwrite
	ResolutionPatch                              // Apply patch
	ResolutionConflict                           // Manual resolution needed
)

// ConflictAnalysis represents the analysis of a file conflict
type ConflictAnalysis struct {
	FilePath           string
	Resolution         ConflictResolution
	LocalHash          string
	RemoteHash         string
	LastKnownCommit    string
	CurrentCommit      string
	PatchContent       string
	ConflictDetails    string
}

// PatchManager handles patch operations
type PatchManager struct {
	repoPath string
	repo     *git.Repository
}

// NewPatchManager creates a new patch manager
func NewPatchManager(repoPath string, repo *git.Repository) *PatchManager {
	return &PatchManager{
		repoPath: repoPath,
		repo:     repo,
	}
}

// AnalyzeConflict analyzes a file conflict and determines resolution strategy
func (pm *PatchManager) AnalyzeConflict(filePath string, tracking config.FileTraking, currentCommit string) (*ConflictAnalysis, error) {
	analysis := &ConflictAnalysis{
		FilePath:        filePath,
		LastKnownCommit: tracking.LastCommit,
		CurrentCommit:   currentCommit,
		LocalHash:       "", // Will be calculated
		RemoteHash:      tracking.Hash,
	}

	// Calculate current local file hash
	localFilePath := filePath // This should be the actual local file path
	if _, err := os.Stat(localFilePath); os.IsNotExist(err) {
		analysis.Resolution = ResolutionStandard
		analysis.ConflictDetails = "Local file does not exist, will be created"
		return analysis, nil
	}

	// Calculate local file hash
	localContent, err := os.ReadFile(localFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read local file: %w", err)
	}

	localHash := calculateHash(localContent)
	analysis.LocalHash = localHash

	// Check if local file matches the last known state
	if localHash == tracking.Hash && !tracking.Modified {
		// File hasn't been modified locally, safe to overwrite
		analysis.Resolution = ResolutionStandard
		analysis.ConflictDetails = "Local file unchanged, safe to overwrite"
		return analysis, nil
	}

	// Local file has been modified, need to check if we can apply patch
	if tracking.LastCommit == "" {
		// No commit tracking available, cannot create patch
		analysis.Resolution = ResolutionConflict
		analysis.ConflictDetails = "Local file modified but no commit history available for patching"
		return analysis, nil
	}

	// Try to generate patch from last known commit to current commit
	patch, err := pm.generatePatch(filePath, tracking.LastCommit, currentCommit)
	if err != nil {
		analysis.Resolution = ResolutionConflict
		analysis.ConflictDetails = fmt.Sprintf("Failed to generate patch: %v", err)
		return analysis, nil
	}

	analysis.PatchContent = patch
	analysis.Resolution = ResolutionPatch
	analysis.ConflictDetails = "Local file modified, patch can be applied"

	return analysis, nil
}

// generatePatch generates a patch between two commits for a specific file
func (pm *PatchManager) generatePatch(filePath, fromCommit, toCommit string) (string, error) {
	// Get the file content at both commits
	fromContent, err := pm.getFileAtCommit(filePath, fromCommit)
	if err != nil {
		return "", fmt.Errorf("failed to get file at commit %s: %w", fromCommit, err)
	}

	toContent, err := pm.getFileAtCommit(filePath, toCommit)
	if err != nil {
		return "", fmt.Errorf("failed to get file at commit %s: %w", toCommit, err)
	}

	// Create temporary files for diff
	tmpDir, err := os.MkdirTemp("", "cherry-go-patch")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	fromFile := filepath.Join(tmpDir, "from")
	toFile := filepath.Join(tmpDir, "to")

	if err := os.WriteFile(fromFile, fromContent, 0644); err != nil {
		return "", fmt.Errorf("failed to write from file: %w", err)
	}

	if err := os.WriteFile(toFile, toContent, 0644); err != nil {
		return "", fmt.Errorf("failed to write to file: %w", err)
	}

	// Generate patch using git diff
	cmd := exec.Command("git", "diff", "--no-index", fromFile, toFile)
	output, err := cmd.Output()
	if err != nil {
		// git diff returns exit code 1 when files differ, which is expected
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return string(output), nil
		}
		return "", fmt.Errorf("failed to generate patch: %w", err)
	}

	return string(output), nil
}

// getFileAtCommit retrieves file content at a specific commit
func (pm *PatchManager) getFileAtCommit(filePath, commitHash string) ([]byte, error) {
	commit, err := pm.repo.CommitObject(plumbing.NewHash(commitHash))
	if err != nil {
		return nil, fmt.Errorf("failed to get commit object: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	file, err := tree.File(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file from tree: %w", err)
	}

	content, err := file.Contents()
	if err != nil {
		return nil, fmt.Errorf("failed to get file contents: %w", err)
	}

	return []byte(content), nil
}

// ApplyPatch applies a patch to a local file
func (pm *PatchManager) ApplyPatch(localFilePath, patchContent string) error {
	if logger.IsDryRun() {
		logger.DryRunInfo("Would apply patch to: %s", localFilePath)
		logger.Debug("Patch content preview:")
		lines := strings.Split(patchContent, "\n")
		for i, line := range lines {
			if i < 10 { // Show first 10 lines
				logger.Debug("  %s", line)
			} else {
				logger.Debug("  ... (%d more lines)", len(lines)-10)
				break
			}
		}
		return nil
	}

	// Create temporary patch file
	tmpDir, err := os.MkdirTemp("", "cherry-go-patch")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	patchFile := filepath.Join(tmpDir, "changes.patch")
	if err := os.WriteFile(patchFile, []byte(patchContent), 0644); err != nil {
		return fmt.Errorf("failed to write patch file: %w", err)
	}

	// Apply patch using git apply
	cmd := exec.Command("git", "apply", "--verbose", patchFile)
	cmd.Dir = filepath.Dir(localFilePath)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to apply patch: %w\nOutput: %s", err, string(output))
	}

	logger.Info("✅ Patch applied successfully to %s", localFilePath)
	logger.Debug("Patch output: %s", string(output))
	
	return nil
}

// CanApplyPatch checks if a patch can be applied cleanly
func (pm *PatchManager) CanApplyPatch(localFilePath, patchContent string) (bool, error) {
	// Create temporary patch file
	tmpDir, err := os.MkdirTemp("", "cherry-go-patch-check")
	if err != nil {
		return false, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	patchFile := filepath.Join(tmpDir, "changes.patch")
	if err := os.WriteFile(patchFile, []byte(patchContent), 0644); err != nil {
		return false, fmt.Errorf("failed to write patch file: %w", err)
	}

	// Check if patch can be applied using git apply --check
	cmd := exec.Command("git", "apply", "--check", patchFile)
	cmd.Dir = filepath.Dir(localFilePath)
	
	err = cmd.Run()
	return err == nil, nil
}

// calculateHash calculates SHA256 hash of content
func calculateHash(content []byte) string {
	// This should use the same hash function as the hash package
	// For now, we'll implement a simple version
	return fmt.Sprintf("%x", content) // Simplified for now
}

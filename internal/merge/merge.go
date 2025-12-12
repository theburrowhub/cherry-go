package merge

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// MergeResult represents the result of a merge operation
type MergeResult struct {
	Success     bool   // Whether the merge was successful (no conflicts)
	Content     []byte // The merged content (may contain conflict markers if Success is false)
	HasConflict bool   // Whether there were conflicts that couldn't be auto-resolved
}

// ThreeWayMerge performs a three-way merge using git merge-file
// base: the common ancestor content (last synced version)
// local: the current local content
// remote: the new remote content
func ThreeWayMerge(base, local, remote []byte) (MergeResult, error) {
	// Create temporary files for the merge
	tempDir, err := os.MkdirTemp("", "cherry-go-merge-*")
	if err != nil {
		return MergeResult{}, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	baseFile := filepath.Join(tempDir, "base")
	localFile := filepath.Join(tempDir, "local")
	remoteFile := filepath.Join(tempDir, "remote")

	// Write content to temp files
	if err := os.WriteFile(baseFile, base, 0644); err != nil {
		return MergeResult{}, fmt.Errorf("failed to write base file: %w", err)
	}
	if err := os.WriteFile(localFile, local, 0644); err != nil {
		return MergeResult{}, fmt.Errorf("failed to write local file: %w", err)
	}
	if err := os.WriteFile(remoteFile, remote, 0644); err != nil {
		return MergeResult{}, fmt.Errorf("failed to write remote file: %w", err)
	}

	// Run git merge-file
	// -p: print result to stdout
	// --diff3: show base version in conflict markers
	// git merge-file returns:
	//   0: merge was successful
	//   >0: number of conflicts (merge attempted but has conflicts)
	//   <0: error occurred
	cmd := exec.Command("git", "merge-file", "-p", "--diff3",
		"-L", "LOCAL",
		"-L", "BASE",
		"-L", "REMOTE",
		localFile, baseFile, remoteFile)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		return MergeResult{}, fmt.Errorf("failed to run git merge-file: %w (stderr: %s)", err, stderr.String())
	}

	result := MergeResult{
		Content:     stdout.Bytes(),
		Success:     exitCode == 0,
		HasConflict: exitCode > 0,
	}

	return result, nil
}

// MergeFile performs a three-way merge on files specified by paths
func MergeFile(basePath, localPath, remotePath string) (MergeResult, error) {
	base, err := os.ReadFile(basePath)
	if err != nil {
		return MergeResult{}, fmt.Errorf("failed to read base file: %w", err)
	}

	local, err := os.ReadFile(localPath)
	if err != nil {
		return MergeResult{}, fmt.Errorf("failed to read local file: %w", err)
	}

	remote, err := os.ReadFile(remotePath)
	if err != nil {
		return MergeResult{}, fmt.Errorf("failed to read remote file: %w", err)
	}

	return ThreeWayMerge(base, local, remote)
}

// MergeFiles performs three-way merge on multiple files
// Returns a map of relative paths to merge results and any files that couldn't be merged
type FileMergeResult struct {
	Path      string
	Result    MergeResult
	Error     error
	IsBinary  bool
	IsNewFile bool // File only exists in remote
	IsDeleted bool // File was deleted locally
}

// MergeDirectory merges all files from a remote directory with local files using base snapshots
func MergeDirectory(baseContent, localDir, remoteDir string, files []string) []FileMergeResult {
	var results []FileMergeResult

	for _, relPath := range files {
		basePath := filepath.Join(baseContent, relPath)
		localPath := filepath.Join(localDir, relPath)
		remotePath := filepath.Join(remoteDir, relPath)

		result := FileMergeResult{Path: relPath}

		// Check if file exists in each location
		baseExists := fileExists(basePath)
		localExists := fileExists(localPath)
		remoteExists := fileExists(remotePath)

		// Handle various cases
		switch {
		case !remoteExists:
			// File removed from remote - skip, let user decide
			continue

		case !baseExists && !localExists:
			// New file from remote - just copy
			result.IsNewFile = true
			content, err := os.ReadFile(remotePath)
			if err != nil {
				result.Error = err
			} else {
				result.Result = MergeResult{Success: true, Content: content}
			}

		case !localExists:
			// File deleted locally but exists in remote - conflict
			result.IsDeleted = true
			result.Result = MergeResult{HasConflict: true}

		case !baseExists:
			// No base version - can't do three-way merge
			// Check if files are identical
			localContent, _ := os.ReadFile(localPath)
			remoteContent, _ := os.ReadFile(remotePath)
			if bytes.Equal(localContent, remoteContent) {
				result.Result = MergeResult{Success: true, Content: localContent}
			} else {
				// Files differ and no base - treat as conflict
				result.Result = MergeResult{HasConflict: true}
			}

		default:
			// Check if file is binary
			if isBinaryFile(localPath) || isBinaryFile(remotePath) {
				result.IsBinary = true
				// For binary files, check if they're identical
				localContent, _ := os.ReadFile(localPath)
				remoteContent, _ := os.ReadFile(remotePath)
				if bytes.Equal(localContent, remoteContent) {
					result.Result = MergeResult{Success: true, Content: localContent}
				} else {
					result.Result = MergeResult{HasConflict: true}
				}
			} else {
				// Perform three-way merge
				mergeResult, err := MergeFile(basePath, localPath, remotePath)
				if err != nil {
					result.Error = err
				} else {
					result.Result = mergeResult
				}
			}
		}

		results = append(results, result)
	}

	return results
}

// HasConflicts checks if any merge results have conflicts
func HasConflicts(results []FileMergeResult) bool {
	for _, r := range results {
		if r.Result.HasConflict || r.Error != nil {
			return true
		}
	}
	return false
}

// GetConflictedFiles returns a list of files that have conflicts
func GetConflictedFiles(results []FileMergeResult) []string {
	var conflicted []string
	for _, r := range results {
		if r.Result.HasConflict || r.Error != nil {
			conflicted = append(conflicted, r.Path)
		}
	}
	return conflicted
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// isBinaryFile checks if a file is binary by reading its first bytes
func isBinaryFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read first 8000 bytes (same as git)
	buf := make([]byte, 8000)
	n, err := file.Read(buf)
	if err != nil {
		return false
	}

	// Check for null bytes (binary indicator)
	return bytes.Contains(buf[:n], []byte{0})
}

// ContainsConflictMarkers checks if content has git conflict markers
func ContainsConflictMarkers(content []byte) bool {
	markers := []string{
		"<<<<<<<",
		"=======",
		">>>>>>>",
	}
	for _, marker := range markers {
		if bytes.Contains(content, []byte(marker)) {
			return true
		}
	}
	return false
}

// FormatConflictSummary formats a summary of conflicts for display
func FormatConflictSummary(results []FileMergeResult) string {
	var lines []string
	lines = append(lines, "Merge conflicts detected in the following files:")
	lines = append(lines, "")

	for _, r := range results {
		if !r.Result.HasConflict && r.Error == nil {
			continue
		}

		var reason string
		switch {
		case r.Error != nil:
			reason = fmt.Sprintf("error: %v", r.Error)
		case r.IsBinary:
			reason = "binary file changed"
		case r.IsDeleted:
			reason = "deleted locally, modified remotely"
		case r.Result.HasConflict:
			reason = "content conflict"
		default:
			reason = "unknown conflict"
		}

		lines = append(lines, fmt.Sprintf("  - %s (%s)", r.Path, reason))
	}

	return strings.Join(lines, "\n")
}

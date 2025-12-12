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

// ThreeWayMerge performs a patch-based merge (default behavior)
// This approach:
// 1. Creates a patch from base→remote (what changed in remote)
// 2. Applies that patch on top of local content
// This preserves local changes and only conflicts when the patch cannot be applied
//
// base: the common ancestor content (last synced version)
// local: the current local content
// remote: the new remote content
func ThreeWayMerge(base, local, remote []byte) (MergeResult, error) {
	// Use patch-based merge by default
	return PatchBasedMerge(base, local, remote)
}

// PatchBasedMerge applies remote changes as a patch on top of local content
// This preserves local additions while incorporating remote modifications
func PatchBasedMerge(base, local, remote []byte) (MergeResult, error) {
	// If base equals remote, no remote changes - keep local as is
	if bytes.Equal(base, remote) {
		return MergeResult{
			Success: true,
			Content: local,
		}, nil
	}

	// If base equals local, no local changes - take remote
	if bytes.Equal(base, local) {
		return MergeResult{
			Success: true,
			Content: remote,
		}, nil
	}

	// If local equals remote, both made same changes
	if bytes.Equal(local, remote) {
		return MergeResult{
			Success: true,
			Content: local,
		}, nil
	}

	// Try semantic line-based merge first
	// This preserves local additions while applying remote modifications
	result, success := semanticLineMerge(base, local, remote)
	if success {
		return MergeResult{
			Success: true,
			Content: result,
		}, nil
	}

	// Fallback to git-based merge for complex cases
	tempDir, err := os.MkdirTemp("", "cherry-go-patch-*")
	if err != nil {
		return MergeResult{}, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	return applyPatchWithGit(tempDir, base, local, remote)
}

// semanticLineMerge performs a line-based merge that preserves local additions
// Returns the merged content and true if successful, or nil and false if there's a real conflict
func semanticLineMerge(base, local, remote []byte) ([]byte, bool) {
	baseLines := strings.Split(string(base), "\n")
	localLines := strings.Split(string(local), "\n")
	remoteLines := strings.Split(string(remote), "\n")

	// Build sets for quick lookup
	baseSet := make(map[string]bool)
	for _, line := range baseLines {
		if line != "" {
			baseSet[line] = true
		}
	}

	localSet := make(map[string]bool)
	for _, line := range localLines {
		if line != "" {
			localSet[line] = true
		}
	}

	remoteSet := make(map[string]bool)
	for _, line := range remoteLines {
		if line != "" {
			remoteSet[line] = true
		}
	}

	// Find what each side did
	// Lines removed from base by local
	var localRemovals []string
	for _, line := range baseLines {
		if line != "" && !localSet[line] {
			localRemovals = append(localRemovals, line)
		}
	}

	// Lines removed from base by remote
	var remoteRemovals []string
	for _, line := range baseLines {
		if line != "" && !remoteSet[line] {
			remoteRemovals = append(remoteRemovals, line)
		}
	}

	// Lines added by local (not in base)
	var localAdditions []string
	for _, line := range localLines {
		if line != "" && !baseSet[line] {
			localAdditions = append(localAdditions, line)
		}
	}

	// Lines added by remote (not in base)
	var remoteAdditions []string
	for _, line := range remoteLines {
		if line != "" && !baseSet[line] {
			remoteAdditions = append(remoteAdditions, line)
		}
	}

	// CONFLICT DETECTION: If both sides removed the same line and added different content,
	// that's a real conflict (both modified the same line differently)
	if len(localRemovals) > 0 && len(remoteRemovals) > 0 {
		// Check if they removed the same lines
		localRemovalSet := make(map[string]bool)
		for _, line := range localRemovals {
			localRemovalSet[line] = true
		}

		for _, line := range remoteRemovals {
			if localRemovalSet[line] {
				// Both removed the same line
				// If both also added different content, it's a conflict
				if len(localAdditions) > 0 && len(remoteAdditions) > 0 {
					// Check if they added the same thing
					localAddSet := make(map[string]bool)
					for _, add := range localAdditions {
						localAddSet[add] = true
					}

					hasDifferentAdditions := false
					for _, add := range remoteAdditions {
						if !localAddSet[add] {
							hasDifferentAdditions = true
							break
						}
					}

					if hasDifferentAdditions {
						// Real conflict: both modified the same line differently
						return nil, false
					}
				}
			}
		}
	}

	// SPECIAL CASE: Local only added lines (didn't modify existing ones)
	// In this case, take remote changes and append local additions
	if len(localRemovals) == 0 && len(localAdditions) > 0 {
		// Local only added lines - apply remote changes and keep local additions
		var result []string
		result = append(result, remoteLines...)

		// Remove trailing empty line if present
		if len(result) > 0 && result[len(result)-1] == "" {
			result = result[:len(result)-1]
		}

		// Append local additions (that aren't already in remote)
		for _, add := range localAdditions {
			if !remoteSet[add] {
				result = append(result, add)
			}
		}

		resultStr := strings.Join(result, "\n")
		if !strings.HasSuffix(resultStr, "\n") && len(resultStr) > 0 {
			resultStr += "\n"
		}
		return []byte(resultStr), true
	}

	// For other cases, fall back to git merge
	return nil, false
}

// applyPatchWithGit creates a git repo and applies remote changes as a patch on local
// This preserves local additions while incorporating remote modifications
func applyPatchWithGit(tempDir string, base, local, remote []byte) (MergeResult, error) {
	gitDir := filepath.Join(tempDir, "repo")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		return fallbackToGitMergeFile(base, local, remote)
	}

	// Init git repo
	initCmd := exec.Command("git", "init")
	initCmd.Dir = gitDir
	if err := initCmd.Run(); err != nil {
		return fallbackToGitMergeFile(base, local, remote)
	}

	// Configure git
	exec.Command("git", "-C", gitDir, "config", "user.email", "cherry-go@local").Run()
	exec.Command("git", "-C", gitDir, "config", "user.name", "cherry-go").Run()

	targetFile := filepath.Join(gitDir, "file")

	// Step 1: Create base commit
	if err := os.WriteFile(targetFile, base, 0644); err != nil {
		return fallbackToGitMergeFile(base, local, remote)
	}
	exec.Command("git", "-C", gitDir, "add", ".").Run()
	exec.Command("git", "-C", gitDir, "commit", "-m", "base").Run()

	// Step 2: Apply local changes and commit
	if err := os.WriteFile(targetFile, local, 0644); err != nil {
		return fallbackToGitMergeFile(base, local, remote)
	}
	exec.Command("git", "-C", gitDir, "add", ".").Run()
	exec.Command("git", "-C", gitDir, "commit", "-m", "local").Run()

	// Step 3: Create patch from base to remote
	baseFile := filepath.Join(tempDir, "base_for_diff")
	remoteFile := filepath.Join(tempDir, "remote_for_diff")
	os.WriteFile(baseFile, base, 0644)
	os.WriteFile(remoteFile, remote, 0644)

	diffCmd := exec.Command("diff", "-u", baseFile, remoteFile)
	patchContent, _ := diffCmd.Output()

	if len(patchContent) == 0 {
		return MergeResult{Success: true, Content: local}, nil
	}

	// Fix patch header to reference the correct file in repo
	patchStr := string(patchContent)
	patchStr = strings.Replace(patchStr, baseFile, "a/file", 1)
	patchStr = strings.Replace(patchStr, remoteFile, "b/file", 1)

	patchFile := filepath.Join(tempDir, "remote.patch")
	os.WriteFile(patchFile, []byte(patchStr), 0644)

	// Step 4: Try git apply --3way (applies patch with 3-way merge on conflicts)
	applyCmd := exec.Command("git", "-C", gitDir, "apply", "--3way", patchFile)
	applyErr := applyCmd.Run()

	// Read result
	resultContent, err := os.ReadFile(targetFile)
	if err != nil {
		return fallbackToGitMergeFile(base, local, remote)
	}

	// Check result
	if applyErr != nil || ContainsConflictMarkers(resultContent) {
		// If git apply --3way left conflict markers, use them
		if ContainsConflictMarkers(resultContent) {
			return MergeResult{
				Content:     resultContent,
				Success:     false,
				HasConflict: true,
			}, nil
		}
		// Otherwise fallback to git merge-file for proper conflict markers
		return fallbackToGitMergeFile(base, local, remote)
	}

	return MergeResult{
		Success: true,
		Content: resultContent,
	}, nil
}

// fallbackToGitMergeFile uses the traditional three-way merge as last resort
func fallbackToGitMergeFile(base, local, remote []byte) (MergeResult, error) {
	tempDir, err := os.MkdirTemp("", "cherry-go-merge-fallback-*")
	if err != nil {
		return MergeResult{}, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	baseFile := filepath.Join(tempDir, "base")
	localFile := filepath.Join(tempDir, "local")
	remoteFile := filepath.Join(tempDir, "remote")

	os.WriteFile(baseFile, base, 0644)
	os.WriteFile(localFile, local, 0644)
	os.WriteFile(remoteFile, remote, 0644)

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

	return MergeResult{
		Content:     stdout.Bytes(),
		Success:     exitCode == 0,
		HasConflict: exitCode > 0,
	}, nil
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

// ShowDiff displays the difference between local and remote content (side by side)
func ShowDiff(localPath, remotePath, fileName string) {
	local, err1 := os.ReadFile(localPath)
	remote, err2 := os.ReadFile(remotePath)

	if err1 != nil || err2 != nil {
		fmt.Printf("\n[Cannot show diff for %s]\n", fileName)
		return
	}

	ShowDiffFromContent(local, remote, fileName)
}

// ShowDiffFromContent displays local and remote content side by side
func ShowDiffFromContent(local, remote []byte, fileName string) {
	showSideBySide(local, remote, fileName)
}

// showSideBySide displays local and remote content in parallel columns
func showSideBySide(local, remote []byte, fileName string) {
	localLines := strings.Split(string(local), "\n")
	remoteLines := strings.Split(string(remote), "\n")

	// Column width for each side
	const colWidth = 38
	const separator = " │ "

	// Header
	fmt.Println()
	fmt.Printf("┌─── %s ───\n", fileName)
	fmt.Printf("│\n")
	fmt.Printf("│  \033[36m%-*s\033[0m%s\033[33m%-*s\033[0m\n", colWidth, "LOCAL (your changes)", separator, colWidth, "REMOTE (source)")
	fmt.Printf("│  %s%s%s\n", strings.Repeat("─", colWidth), "─┼─", strings.Repeat("─", colWidth))

	// Get max lines
	maxLines := len(localLines)
	if len(remoteLines) > maxLines {
		maxLines = len(remoteLines)
	}

	// Limit output
	maxDisplay := 40
	if maxLines > maxDisplay {
		maxLines = maxDisplay
	}

	// Display side by side
	for i := 0; i < maxLines; i++ {
		localLine := ""
		remoteLine := ""

		if i < len(localLines) {
			localLine = localLines[i]
		}
		if i < len(remoteLines) {
			remoteLine = remoteLines[i]
		}

		// Truncate long lines
		if len(localLine) > colWidth {
			localLine = localLine[:colWidth-3] + "..."
		}
		if len(remoteLine) > colWidth {
			remoteLine = remoteLine[:colWidth-3] + "..."
		}

		// Determine if lines differ
		isDiff := localLine != remoteLine

		if isDiff {
			// Highlight differences
			fmt.Printf("│  \033[36m%-*s\033[0m%s\033[33m%-*s\033[0m  \033[31m◄\033[0m\n", colWidth, localLine, separator, colWidth, remoteLine)
		} else {
			fmt.Printf("│  %-*s%s%-*s\n", colWidth, localLine, separator, colWidth, remoteLine)
		}
	}

	if len(localLines) > maxDisplay || len(remoteLines) > maxDisplay {
		fmt.Printf("│  ... (%d more lines)\n", max(len(localLines), len(remoteLines))-maxDisplay)
	}

	fmt.Println("│")
	fmt.Printf("│  \033[31m◄\033[0m = lines that differ\n")
	fmt.Println("└" + strings.Repeat("─", 85))
}

// max returns the larger of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

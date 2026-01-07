package merge

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cherry-go/internal/logger"
)

// MergeResult represents the result of a merge operation
type MergeResult struct {
	Success     bool   // Whether the merge was successful (no conflicts)
	Content     []byte // The merged content (may contain conflict markers if Success is false)
	HasConflict bool   // Whether there were conflicts that couldn't be auto-resolved
}

// ThreeWayMerge performs a git merge-file based three-way merge with diff3 style
// This uses git's native merge algorithm directly
//
// base: the common ancestor content (from git history or empty)
// local: the current local content
// remote: the new remote content
func ThreeWayMerge(base, local, remote []byte) (MergeResult, error) {
	// Quick checks for trivial cases
	if bytes.Equal(base, remote) {
		// No remote changes - keep local as is
		return MergeResult{
			Success: true,
			Content: local,
		}, nil
	}

	if bytes.Equal(base, local) {
		// No local changes - take remote
		return MergeResult{
			Success: true,
			Content: remote,
		}, nil
	}

	if bytes.Equal(local, remote) {
		// Both made same changes
		return MergeResult{
			Success: true,
			Content: local,
		}, nil
	}

	// Use git merge-file for all other cases
	return gitMergeFileDiff3(base, local, remote)
}

// gitMergeFileDiff3 uses git merge-file with diff3 style for three-way merge
func gitMergeFileDiff3(base, local, remote []byte) (MergeResult, error) {
	tempDir, err := os.MkdirTemp("", "cherry-go-merge-fallback-*")
	if err != nil {
		return MergeResult{}, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	baseFile := filepath.Join(tempDir, "base")
	localFile := filepath.Join(tempDir, "local")
	remoteFile := filepath.Join(tempDir, "remote")

	if err := os.WriteFile(baseFile, base, 0644); err != nil {
		return MergeResult{}, fmt.Errorf("failed to write base file: %w", err)
	}
	if err := os.WriteFile(localFile, local, 0644); err != nil {
		return MergeResult{}, fmt.Errorf("failed to write local file: %w", err)
	}
	if err := os.WriteFile(remoteFile, remote, 0644); err != nil {
		return MergeResult{}, fmt.Errorf("failed to write remote file: %w", err)
	}

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

// isBinaryFile checks if a file is binary by reading its first bytes
// Note: Used primarily for testing
func isBinaryFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()

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

// ShowDiffFromContent displays a three-way diff (base, local, remote) with merge preview
// Only shows detailed diff if verbosity level >= 2, otherwise shows summary
func ShowDiffFromContent(base, local, remote []byte, fileName string) {
	if logger.ShouldShowDiffs() {
		// Verbosity >= 2: Show detailed diff
		showDiff3(base, local, remote, fileName)
	} else {
		// Verbosity < 2: Show only summary
		showConflictSummary(base, local, remote, fileName)
	}
}

// showConflictSummary shows a brief summary without detailed diff
func showConflictSummary(base, local, remote []byte, fileName string) {
	// If verbosity is 0, don't show anything (summary will be in final compact log)
	if logger.GetVerbosityLevel() == 0 {
		return
	}

	// Perform merge to determine the type of conflict
	mergeResult, _ := ThreeWayMerge(base, local, remote)

	baseLines := len(strings.Split(string(base), "\n"))
	localLines := len(strings.Split(string(local), "\n"))
	remoteLines := len(strings.Split(string(remote), "\n"))

	if mergeResult.Success {
		fmt.Printf("\n  • %s: Auto-merge successful (%d lines in base, %d local, %d remote)\n",
			fileName, baseLines, localLines, remoteLines)
	} else {
		fmt.Printf("\n  • %s: Merge conflict detected (%d lines in base, %d local, %d remote)\n",
			fileName, baseLines, localLines, remoteLines)
		fmt.Printf("    → Use -v or --verbose flag multiple times to see detailed diff\n")
	}
}

// showDiff3 displays a three-way diff showing BASE, LOCAL, REMOTE in 3 columns
// and the MERGE RESULT below spanning the full width
func showDiff3(base, local, remote []byte, fileName string) {
	// Perform merge to get the result
	mergeResult, _ := ThreeWayMerge(base, local, remote)

	baseLines := strings.Split(string(base), "\n")
	localLines := strings.Split(string(local), "\n")
	remoteLines := strings.Split(string(remote), "\n")
	resultLines := strings.Split(string(mergeResult.Content), "\n")

	const colWidth = 36
	const separator = " │ "
	const totalWidth = colWidth*3 + len(separator)*2

	// Header
	fmt.Println()
	fmt.Printf("┌─── %s ───\n", fileName)
	fmt.Printf("│\n")

	// Show merge status
	if mergeResult.Success {
		fmt.Printf("│  \033[32m✓ Auto-merge successful\033[0m\n")
	} else if mergeResult.HasConflict {
		fmt.Printf("│  \033[31m✗ Merge conflict detected\033[0m\n")
	}
	fmt.Printf("│\n")

	// Column headers for the 3 versions
	fmt.Printf("│  \033[90m%-*s\033[0m%s", colWidth, "BASE (last sync)", separator)
	fmt.Printf("\033[36m%-*s\033[0m%s", colWidth, "LOCAL (yours)", separator)
	fmt.Printf("\033[33m%-*s\033[0m\n", colWidth, "REMOTE (source)")

	fmt.Printf("│  %s%s%s%s%s\n",
		strings.Repeat("─", colWidth), "─┼─",
		strings.Repeat("─", colWidth), "─┼─",
		strings.Repeat("─", colWidth))

	// Get max lines for the three versions
	maxLines := max(len(baseLines), max(len(localLines), len(remoteLines)))

	// Limit output
	maxDisplay := 30
	displayLines := maxLines
	if maxLines > maxDisplay {
		displayLines = maxDisplay
	}

	// Display three columns (BASE, LOCAL, REMOTE)
	for i := 0; i < displayLines; i++ {
		baseLine := ""
		localLine := ""
		remoteLine := ""

		if i < len(baseLines) {
			baseLine = baseLines[i]
		}
		if i < len(localLines) {
			localLine = localLines[i]
		}
		if i < len(remoteLines) {
			remoteLine = remoteLines[i]
		}

		// Truncate long lines
		if len(baseLine) > colWidth {
			baseLine = baseLine[:colWidth-3] + "..."
		}
		if len(localLine) > colWidth {
			localLine = localLine[:colWidth-3] + "..."
		}
		if len(remoteLine) > colWidth {
			remoteLine = remoteLine[:colWidth-3] + "..."
		}

		// Determine if lines changed
		localChanged := localLine != baseLine
		remoteChanged := remoteLine != baseLine
		hasChange := localChanged || remoteChanged

		// Print with appropriate formatting
		fmt.Print("│  ")

		// BASE column (grey)
		fmt.Printf("\033[90m%-*s\033[0m%s", colWidth, baseLine, separator)

		// LOCAL column (cyan if changed)
		if localChanged {
			fmt.Printf("\033[36m%-*s\033[0m%s", colWidth, localLine, separator)
		} else {
			fmt.Printf("%-*s%s", colWidth, localLine, separator)
		}

		// REMOTE column (yellow if changed)
		if remoteChanged {
			fmt.Printf("\033[33m%-*s\033[0m", colWidth, remoteLine)
		} else {
			fmt.Printf("%-*s", colWidth, remoteLine)
		}

		// Mark changed lines
		if hasChange {
			fmt.Print("  \033[31m◄\033[0m")
		}

		fmt.Println()
	}

	if maxLines > maxDisplay {
		fmt.Printf("│  ... (%d more lines)\n", maxLines-maxDisplay)
	}

	// Separator before RESULT section
	fmt.Printf("│\n")
	fmt.Printf("│  %s\n", strings.Repeat("═", totalWidth))

	// RESULT header
	if mergeResult.Success {
		fmt.Printf("│  \033[32mRESULT (merged successfully)\033[0m\n")
	} else {
		fmt.Printf("│  \033[31mRESULT (with conflicts)\033[0m\n")
	}
	fmt.Printf("│  %s\n", strings.Repeat("─", totalWidth))

	// Display RESULT spanning full width
	maxResultDisplay := 30
	displayResultLines := len(resultLines)
	if len(resultLines) > maxResultDisplay {
		displayResultLines = maxResultDisplay
	}

	for i := 0; i < displayResultLines; i++ {
		resultLine := resultLines[i]

		// Truncate if needed
		if len(resultLine) > totalWidth {
			resultLine = resultLine[:totalWidth-3] + "..."
		}

		// Color based on content
		if strings.Contains(resultLine, "<<<<<<<") || strings.Contains(resultLine, ">>>>>>>") || strings.Contains(resultLine, "|||||||") || strings.Contains(resultLine, "=======") {
			fmt.Printf("│  \033[31m%-*s\033[0m\n", totalWidth, resultLine)
		} else if mergeResult.Success {
			fmt.Printf("│  \033[32m%-*s\033[0m\n", totalWidth, resultLine)
		} else {
			fmt.Printf("│  %-*s\n", totalWidth, resultLine)
		}
	}

	if len(resultLines) > maxResultDisplay {
		fmt.Printf("│  ... (%d more lines)\n", len(resultLines)-maxResultDisplay)
	}

	fmt.Println("│")
	fmt.Printf("│  \033[31m◄\033[0m = changed from base\n")
	fmt.Println("└" + strings.Repeat("─", totalWidth+2))
}

// max returns the larger of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

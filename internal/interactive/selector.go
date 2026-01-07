package interactive

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/koki-develop/go-fzf"
)

// FileItem represents a file in the selector
type FileItem struct {
	Path     string
	IsDir    bool
	Selected bool
}

// String returns the display string for the file item
func (f FileItem) String() string {
	if f.IsDir {
		return fmt.Sprintf("[dir] %s", f.Path)
	}
	return fmt.Sprintf("[file] %s", f.Path)
}

// Selector provides interactive file and directory selection
type Selector struct {
	fzf *fzf.FZF
}

// NewSelector creates a new interactive selector
func NewSelector() (*Selector, error) {
	f, err := fzf.New(fzf.WithNoLimit(true))
	if err != nil {
		return nil, fmt.Errorf("failed to create fzf instance: %w", err)
	}

	return &Selector{fzf: f}, nil
}

// SelectFiles presents an interactive file selector and returns selected file paths
func (s *Selector) SelectFiles(files []string, prompt string) ([]string, error) {
	if len(files) == 0 {
		return []string{}, nil
	}

	// Sort files for better presentation
	sort.Strings(files)

	// Create file items
	items := make([]FileItem, len(files))
	for i, file := range files {
		items[i] = FileItem{
			Path:  file,
			IsDir: false,
		}
	}

	// Note: go-fzf doesn't support dynamic prompts in this version
	// We'll display the prompt as a message instead
	if prompt != "" {
		fmt.Printf("\n%s\n", prompt)
		fmt.Println("Use arrow keys to navigate, Space to select, Enter to confirm, Ctrl+C to cancel")
	}

	// Run the selector
	indices, err := s.fzf.Find(items, func(i int) string {
		return items[i].String()
	})
	if err != nil {
		return nil, fmt.Errorf("selection cancelled or failed: %w", err)
	}

	// Extract selected file paths
	selected := make([]string, len(indices))
	for i, idx := range indices {
		selected[i] = items[idx].Path
	}

	return selected, nil
}

// SelectDirectories presents an interactive directory selector and returns selected directory paths
func (s *Selector) SelectDirectories(directories []string, prompt string) ([]string, error) {
	if len(directories) == 0 {
		return []string{}, nil
	}

	// Sort directories for better presentation
	sort.Strings(directories)

	// Create directory items
	items := make([]FileItem, len(directories))
	for i, dir := range directories {
		items[i] = FileItem{
			Path:  dir,
			IsDir: true,
		}
	}

	// Note: go-fzf doesn't support dynamic prompts in this version
	// We'll display the prompt as a message instead
	if prompt != "" {
		fmt.Printf("\n%s\n", prompt)
		fmt.Println("Use arrow keys to navigate, Space to select, Enter to confirm, Ctrl+C to cancel")
	}

	// Run the selector
	indices, err := s.fzf.Find(items, func(i int) string {
		return items[i].String()
	})
	if err != nil {
		return nil, fmt.Errorf("selection cancelled or failed: %w", err)
	}

	// Extract selected directory paths
	selected := make([]string, len(indices))
	for i, idx := range indices {
		selected[i] = items[idx].Path
	}

	return selected, nil
}

// SelectMixed presents an interactive selector for both files and directories
func (s *Selector) SelectMixed(files []string, directories []string, prompt string) (selectedFiles []string, selectedDirs []string, err error) {
	if len(files) == 0 && len(directories) == 0 {
		return []string{}, []string{}, nil
	}

	// Create combined items
	var items []FileItem

	// Add files
	for _, file := range files {
		items = append(items, FileItem{
			Path:  file,
			IsDir: false,
		})
	}

	// Add directories
	for _, dir := range directories {
		items = append(items, FileItem{
			Path:  dir,
			IsDir: true,
		})
	}

	// Sort items by type and then by path
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsDir != items[j].IsDir {
			return !items[i].IsDir // Files first, then directories
		}
		return items[i].Path < items[j].Path
	})

	// Note: go-fzf doesn't support dynamic prompts in this version
	// We'll display the prompt as a message instead
	if prompt != "" {
		fmt.Printf("\n%s\n", prompt)
		fmt.Println("Use arrow keys to navigate, Space to select, Enter to confirm, Ctrl+C to cancel")
	}

	// Run the selector
	indices, err := s.fzf.Find(items, func(i int) string {
		return items[i].String()
	})
	if err != nil {
		return nil, nil, fmt.Errorf("selection cancelled or failed: %w", err)
	}

	// Separate selected files and directories
	for _, idx := range indices {
		item := items[idx]
		if item.IsDir {
			selectedDirs = append(selectedDirs, item.Path)
		} else {
			selectedFiles = append(selectedFiles, item.Path)
		}
	}

	return selectedFiles, selectedDirs, nil
}

// PathConfig represents path configuration for a selected item
type PathConfig struct {
	SourcePath string
	LocalPath  string
	Branch     string
}

// ConfigurePaths asks the user to configure custom paths for selected items
func ConfigurePaths(items []string, itemType string, defaultBranch string) ([]PathConfig, error) {
	configs := make([]PathConfig, len(items))

	fmt.Printf("\n=== Path configuration for %s ===\n", itemType)
	fmt.Println("Press Enter to use the same source path as destination.")
	fmt.Println("Press Enter to use the default branch.")
	fmt.Println()

	for i, item := range items {
		fmt.Printf("Configuring: %s\n", item)

		// Local path configuration
		fmt.Printf("Local path [%s]: ", item)
		var localPath string
		fmt.Scanln(&localPath)
		if strings.TrimSpace(localPath) == "" {
			localPath = item
		}

		// Branch configuration
		fmt.Printf("Branch [%s]: ", defaultBranch)
		var branch string
		fmt.Scanln(&branch)
		if strings.TrimSpace(branch) == "" {
			branch = defaultBranch
		}

		configs[i] = PathConfig{
			SourcePath: item,
			LocalPath:  localPath,
			Branch:     branch,
		}

		fmt.Println()
	}

	return configs, nil
}

// AskYesNo asks a yes/no question and returns the result
func AskYesNo(question string, defaultYes bool) bool {
	var defaultStr string
	if defaultYes {
		defaultStr = "Y/n"
	} else {
		defaultStr = "y/N"
	}

	fmt.Printf("%s [%s]: ", question, defaultStr)
	var response string
	fmt.Scanln(&response)

	response = strings.TrimSpace(strings.ToLower(response))

	if response == "" {
		return defaultYes
	}

	return response == "y" || response == "yes"
}

// FilterGitFiles filters out common Git-related files and directories that shouldn't be included
func FilterGitFiles(files []string) []string {
	filtered := make([]string, 0, len(files))

	for _, file := range files {
		// Skip .git directory and its contents
		if strings.HasPrefix(file, ".git/") || file == ".git" {
			continue
		}

		// Skip common temporary and build files
		base := filepath.Base(file)
		if strings.HasPrefix(base, ".") && (base == ".DS_Store" || base == ".gitkeep") {
			continue
		}

		filtered = append(filtered, file)
	}

	return filtered
}

// FilterGitDirectories filters out common Git-related directories that shouldn't be included
func FilterGitDirectories(directories []string) []string {
	filtered := make([]string, 0, len(directories))

	for _, dir := range directories {
		// Skip .git directory
		if dir == ".git" || strings.HasPrefix(dir, ".git/") {
			continue
		}

		// Skip common build and cache directories
		base := filepath.Base(dir)
		if base == "node_modules" || base == ".vscode" || base == ".idea" ||
			base == "dist" || base == "build" || base == "target" {
			continue
		}

		filtered = append(filtered, dir)
	}

	return filtered
}

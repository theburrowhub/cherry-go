package cmd

import (
	"bufio"
	"cherry-go/internal/config"
	"cherry-go/internal/git"
	"cherry-go/internal/interactive"
	"cherry-go/internal/logger"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	cherryBunchOutputFile string
	cherryBunchBranch     string
)

// cherryBunchCmd represents the cherrybunch command
var cherryBunchCmd = &cobra.Command{
	Use:   "cherrybunch",
	Short: "Manage cherry bunch templates",
	Long: `Manage cherry bunch templates for quick repository setup.

Cherry bunches are YAML template files that describe sets of files and directories
to synchronize from repositories. This command helps create and manage these templates.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// cherryBunchCreateCmd represents the cherrybunch create command
var cherryBunchCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a cherry bunch template interactively",
	Long: `Create a cherry bunch template interactively by selecting files and directories
from the current Git repository.

This command will:
1. Detect the current Git repository
2. Allow you to select files and directories to include
3. Configure destination paths and branches
4. Generate a .cherrybunch file

Examples:
  # Create a cherry bunch in the current directory
  cherry-go cherrybunch create
  
  # Create with specific output file and branch
  cherry-go cherrybunch create --output python.cherrybunch --branch main`,
	Run: runCherryBunchCreate,
}

func runCherryBunchCreate(cmd *cobra.Command, args []string) {
	logger.Info("Creating cherry bunch template...")

	// Check if we're in a Git repository
	gitUtils := git.NewGitUtils()
	repoRoot, err := gitUtils.GetRepositoryRoot(".")
	if err != nil {
		logger.Fatal("Not in a Git repository: %v", err)
	}

	logger.Info("Git repository detected: %s", repoRoot)

	// Get repository URL
	repoURL, err := gitUtils.GetRemoteURL(".", "origin")
	if err != nil {
		logger.Warning("Could not detect repository URL: %v", err)
		repoURL = "https://github.com/user/repo.git" // Placeholder
	}

	// Get current branch if not specified
	if cherryBunchBranch == "" {
		cherryBunchBranch, err = gitUtils.GetCurrentBranch(".")
		if err != nil {
			logger.Warning("Could not detect current branch: %v", err)
			cherryBunchBranch = "main"
		}
	}

	// Interactive setup
	scanner := bufio.NewScanner(os.Stdin)
	
	// Get basic information
	fmt.Print("Cherry bunch name: ")
	scanner.Scan()
	name := strings.TrimSpace(scanner.Text())
	if name == "" {
		name = "my-cherrybunch"
	}

	fmt.Print("Description (optional): ")
	scanner.Scan()
	description := strings.TrimSpace(scanner.Text())

	fmt.Printf("Repository URL [%s]: ", repoURL)
	scanner.Scan()
	inputURL := strings.TrimSpace(scanner.Text())
	if inputURL != "" {
		repoURL = inputURL
	}

	fmt.Printf("Default branch [%s]: ", cherryBunchBranch)
	scanner.Scan()
	inputBranch := strings.TrimSpace(scanner.Text())
	if inputBranch != "" {
		cherryBunchBranch = inputBranch
	}

	// Create cherry bunch
	cherryBunch := &config.CherryBunch{
		Name:        name,
		Description: description,
		Version:     "1.0",
		Repository:  repoURL,
		Files:       []config.CherryBunchFileSpec{},
		Directories: []config.CherryBunchDirSpec{},
	}

	// Get all files and directories from the repository
	allFiles, err := gitUtils.ListFiles(".")
	if err != nil {
		logger.Fatal("Failed to list repository files: %v", err)
	}

	allDirs, err := gitUtils.ListDirectories(".")
	if err != nil {
		logger.Fatal("Failed to list repository directories: %v", err)
	}

	// Filter out Git-related files and directories
	allFiles = interactive.FilterGitFiles(allFiles)
	allDirs = interactive.FilterGitDirectories(allDirs)

	logger.Info("Found %d files and %d directories in repository", len(allFiles), len(allDirs))

	// Create interactive selector
	selector, err := interactive.NewSelector()
	if err != nil {
		logger.Fatal("Failed to create interactive selector: %v", err)
	}

	// Interactive selection of files and directories
	fmt.Println("\n=== Interactive file and directory selection ===")
	fmt.Println("Use arrow keys to navigate, Tab to select, Enter to confirm")
	fmt.Println("Ctrl+C to cancel")

	selectedFiles, selectedDirs, err := selector.SelectMixed(
		allFiles, 
		allDirs, 
		"Select files and directories for the cherry bunch",
	)
	if err != nil {
		logger.Fatal("Selection failed: %v", err)
	}

	logger.Info("Selected %d files and %d directories", len(selectedFiles), len(selectedDirs))

	if len(selectedFiles) == 0 && len(selectedDirs) == 0 {
		logger.Fatal("No files or directories selected")
	}

	// Ask if user wants to configure custom paths
	configureCustomPaths := interactive.AskYesNo(
		"Do you want to configure specific paths for the selected items?", 
		false,
	)

	// Configure file paths
	if len(selectedFiles) > 0 {
		var fileConfigs []interactive.PathConfig
		if configureCustomPaths {
			fileConfigs, err = interactive.ConfigurePaths(selectedFiles, "files", cherryBunchBranch)
			if err != nil {
				logger.Fatal("Failed to configure file paths: %v", err)
			}
		} else {
			// Use same paths for source and destination
			fileConfigs = make([]interactive.PathConfig, len(selectedFiles))
			for i, file := range selectedFiles {
				fileConfigs[i] = interactive.PathConfig{
					SourcePath: file,
					LocalPath:  file,
					Branch:     cherryBunchBranch,
				}
			}
		}

		// Convert to CherryBunch file specs
		for _, pathConfig := range fileConfigs {
			fileSpec := config.CherryBunchFileSpec{
				Path:      pathConfig.SourcePath,
				LocalPath: pathConfig.LocalPath,
				Branch:    pathConfig.Branch,
			}
			cherryBunch.Files = append(cherryBunch.Files, fileSpec)
		}
	}

	// Configure directory paths
	if len(selectedDirs) > 0 {
		var dirConfigs []interactive.PathConfig
		if configureCustomPaths {
			dirConfigs, err = interactive.ConfigurePaths(selectedDirs, "directories", cherryBunchBranch)
			if err != nil {
				logger.Fatal("Failed to configure directory paths: %v", err)
			}
		} else {
			// Use same paths for source and destination
			dirConfigs = make([]interactive.PathConfig, len(selectedDirs))
			for i, dir := range selectedDirs {
				dirConfigs[i] = interactive.PathConfig{
					SourcePath: dir,
					LocalPath:  dir,
					Branch:     cherryBunchBranch,
				}
			}
		}

		// Convert to CherryBunch directory specs
		for _, pathConfig := range dirConfigs {
			// Ask for exclude patterns if configuring custom paths
			var exclude []string
			if configureCustomPaths {
				fmt.Printf("Exclude patterns for %s (comma-separated, optional): ", pathConfig.SourcePath)
				scanner.Scan()
				excludeStr := strings.TrimSpace(scanner.Text())
				if excludeStr != "" {
					exclude = strings.Split(excludeStr, ",")
					for i, pattern := range exclude {
						exclude[i] = strings.TrimSpace(pattern)
					}
				}
			}

			dirSpec := config.CherryBunchDirSpec{
				Path:      pathConfig.SourcePath,
				LocalPath: pathConfig.LocalPath,
				Branch:    pathConfig.Branch,
				Exclude:   exclude,
			}
			cherryBunch.Directories = append(cherryBunch.Directories, dirSpec)
		}
	}

	if len(cherryBunch.Files) == 0 && len(cherryBunch.Directories) == 0 {
		logger.Fatal("No files or directories added to cherry bunch")
	}

	// Determine output file
	if cherryBunchOutputFile == "" {
		cherryBunchOutputFile = name + ".cherrybunch"
	}

	if dryRun {
		logger.Info("Dry run mode - would save cherry bunch to: %s", cherryBunchOutputFile)
		return
	}

	// Save cherry bunch
	if err := cherryBunch.Save(cherryBunchOutputFile); err != nil {
		logger.Fatal("Failed to save cherry bunch: %v", err)
	}

	logger.Info("Cherry bunch created successfully: %s", cherryBunchOutputFile)
	logger.Info("Files: %d", len(cherryBunch.Files))
	logger.Info("Directories: %d", len(cherryBunch.Directories))
	logger.Info("You can now share this file or use it with 'cherry-go add cherrybunch %s'", cherryBunchOutputFile)
}

func init() {
	rootCmd.AddCommand(cherryBunchCmd)
	cherryBunchCmd.AddCommand(cherryBunchCreateCmd)

	// Flags for create command
	cherryBunchCreateCmd.Flags().StringVar(&cherryBunchOutputFile, "output", "", "output file name (default: <name>.cherrybunch)")
	cherryBunchCreateCmd.Flags().StringVar(&cherryBunchBranch, "branch", "", "default branch to use (default: current branch)")
}

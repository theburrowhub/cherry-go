package cmd

import (
	"cherry-go/internal/config"
	"cherry-go/internal/logger"
	"cherry-go/internal/utils"

	"github.com/spf13/cobra"
)

var (
	fileRepoName  string
	fileLocalPath string
	fileBranch    string
)

// addFileCmd represents the add file command
var addFileCmd = &cobra.Command{
	Use:   "file [url-path]",
	Short: "Add a file to track from a repository",
	Args:  cobra.ExactArgs(1),
	Long: `Add a specific file to track. The file is automatically synced when added.

Format: cherry-go add file REPOSITORY_URL/path/to/file.ext

The repository is auto-detected from the URL. If multiple repositories are configured
and the URL doesn't specify a repository, you must specify --repo.

Examples:
  # Add a file with full URL (repository auto-detected)
  cherry-go add file https://github.com/user/library.git/src/main.go
  
  # Add a file from SSH repository
  cherry-go add file git@github.com:user/repo.git/README.md
  
  # Add with custom local path
  cherry-go add file https://github.com/user/lib.git/utils.go --local-path internal/utils.go
  
  # Add from specific branch
  cherry-go add file https://github.com/user/lib.git/config.json --branch v1.2.0
  
  # Add from configured repository (if only one exists)
  cherry-go add file src/main.go`,
	Run: func(cmd *cobra.Command, args []string) {
		urlPath := args[0]

		// Parse the URL path to extract repository URL and file path
		repoURL, filePath := utils.ParseURLPath(urlPath)

		var source *config.Source
		var exists bool

		if repoURL != "" {
			// Repository URL provided in the path, find or create repository
			repoName := utils.ExtractRepoName(repoURL)

			// Override with explicit repo name if provided
			if fileRepoName != "" {
				repoName = fileRepoName
			}

			source, exists = cfg.GetSource(repoName)
			if !exists {
				// Auto-add repository if it doesn't exist
				logger.Info("Repository '%s' not found, adding automatically...", repoName)

				authType := detectAuthType(repoURL)
				auth := config.AuthConfig{
					Type:     authType,
					Username: "", // Will be detected automatically
					SSHKey:   "",
				}

				source = &config.Source{
					Name:       repoName,
					Repository: repoURL,
					Auth:       auth,
					Paths:      []config.PathSpec{},
				}

				cfg.AddSource(*source)
				logger.Info("✅ Auto-added repository '%s'", repoName)
			}

			fileRepoName = repoName
		} else {
			// No repository URL in path, try to auto-detect from existing repositories
			if fileRepoName == "" {
				if len(cfg.Sources) == 0 {
					logger.Fatal("No repositories configured. Add one first with: cherry-go add repo <URL>")
				} else if len(cfg.Sources) == 1 {
					// Only one repository, use it
					source = &cfg.Sources[0]
					fileRepoName = source.Name
					logger.Debug("Auto-detected repository: %s", fileRepoName)
				} else {
					logger.Fatal("Multiple repositories configured. Specify which one to use with --repo or use full URL format: cherry-go add file REPO_URL/path/to/file")
				}
			} else {
				// Repository name provided explicitly
				source, exists = cfg.GetSource(fileRepoName)
				if !exists {
					logger.Fatal("Repository '%s' not found. Available repositories: %v", fileRepoName, getRepositoryNames())
				}
			}
		}

		// Set local path - default to same as source path
		localPath := fileLocalPath
		if localPath == "" {
			localPath = filePath
		}

		// Check if this file is already being tracked
		for _, pathSpec := range source.Paths {
			if pathSpec.Include == filePath {
				logger.Fatal("File '%s' is already being tracked in repository '%s'", filePath, fileRepoName)
			}
		}

		// Create new path spec for the file
		newPathSpec := config.PathSpec{
			Include:   filePath,
			LocalPath: localPath,
			Branch:    fileBranch,
			Files:     make(map[string]config.FileTraking), // Will be populated during sync
		}

		// Add the path spec to the source
		source.Paths = append(source.Paths, newPathSpec)

		// Update the source in configuration
		for i, cfgSource := range cfg.Sources {
			if cfgSource.Name == fileRepoName {
				cfg.Sources[i] = *source
				break
			}
		}

		// Try to sync the file first before adding to tracking
		var syncSuccess bool
		if !logger.IsDryRun() {
			logger.Info("🔄 Syncing file for the first time...")

			// Perform initial sync of the file
			if err := performInitialSync(fileRepoName); err != nil {
				logger.Error("Failed to sync file: %v", err)
				logger.Error("File will not be added to tracking due to sync failure")
				return
			} else {
				logger.Info("✅ File synced successfully!")
				syncSuccess = true
			}
		} else {
			logger.DryRunInfo("Would sync the file automatically")
			syncSuccess = true // Assume success in dry-run
		}

		// Only save configuration if sync was successful
		if syncSuccess {
			if !logger.IsDryRun() {
				if err := cfg.Save(configFile); err != nil {
					logger.Fatal("Failed to save configuration: %v", err)
				}
			}

			logger.Info("✅ Added file tracking:")
			logger.Info("  Repository: %s", fileRepoName)
			logger.Info("  Source path: %s", filePath)
			logger.Info("  Local path: %s", localPath)
			if fileBranch != "" {
				logger.Info("  Branch/Tag: %s", fileBranch)
			} else {
				logger.Info("  Branch/Tag: (default)")
			}

			if logger.IsDryRun() {
				logger.DryRunInfo("Configuration would be saved to: %s", configFile)
			} else {
				logger.Info("Configuration saved to: %s", configFile)
			}
		}
	},
}

// getRepositoryNames returns a list of configured repository names
func getRepositoryNames() []string {
	var names []string
	for _, source := range cfg.Sources {
		names = append(names, source.Name)
	}
	return names
}

func init() {
	addCmd.AddCommand(addFileCmd)

	addFileCmd.Flags().StringVar(&fileRepoName, "repo", "", "repository name (auto-detected if only one configured)")
	addFileCmd.Flags().StringVar(&fileLocalPath, "local-path", "", "local path for the file (defaults to same as source path)")
	addFileCmd.Flags().StringVar(&fileBranch, "branch", "", "branch or tag to track (defaults to main/master)")
}

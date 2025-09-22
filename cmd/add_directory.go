package cmd

import (
	"cherry-go/internal/config"
	"cherry-go/internal/logger"
	"cherry-go/internal/utils"
	"strings"

	"github.com/spf13/cobra"
)

var (
	dirRepoName  string
	dirLocalPath string
	dirBranch    string
	dirExcludes  []string
)

// addDirectoryCmd represents the add directory command
var addDirectoryCmd = &cobra.Command{
	Use:   "directory [url-path]",
	Short: "Add a directory to track from a repository",
	Args:  cobra.ExactArgs(1),
	Long: `Add a directory to track. All files in the directory are automatically synced when added.

Format: cherry-go add directory REPOSITORY_URL/path/to/dir/

The repository is auto-detected from the URL. If multiple repositories are configured
and the URL doesn't specify a repository, you must specify --repo.

When syncing a directory:
- New files will be added automatically
- Modified files will be updated
- Deleted files will be removed from local
- Excluded patterns will be ignored

Examples:
  # Add a directory with full URL (repository auto-detected)
  cherry-go add directory https://github.com/user/library.git/src/
  
  # Add from SSH repository
  cherry-go add directory git@github.com:user/repo.git/lib/
  
  # Add with custom local path
  cherry-go add directory https://github.com/user/lib.git/utils/ --local-path internal/utils/
  
  # Add from specific branch with exclusions
  cherry-go add directory https://github.com/user/lib.git/src/ --branch develop --exclude "*.test.go,tmp/"
  
  # Add from configured repository (if only one exists)
  cherry-go add directory src/`,
	Run: func(cmd *cobra.Command, args []string) {
		urlPath := args[0]

		// Parse the URL path to extract repository URL and directory path
		repoURL, dirPath := utils.ParseURLPath(urlPath)

		// Ensure directory path ends with /
		if dirPath != "" && !strings.HasSuffix(dirPath, "/") {
			dirPath += "/"
		}

		var source *config.Source
		var exists bool

		if repoURL != "" {
			// Repository URL provided in the path, find or create repository
			repoName := utils.ExtractRepoName(repoURL)

			// Override with explicit repo name if provided
			if dirRepoName != "" {
				repoName = dirRepoName
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

			dirRepoName = repoName
		} else {
			// No repository URL in path, try to auto-detect from existing repositories
			if dirRepoName == "" {
				if len(cfg.Sources) == 0 {
					logger.Fatal("No repositories configured. Add one first with: cherry-go add repo <URL>")
				} else if len(cfg.Sources) == 1 {
					// Only one repository, use it
					source = &cfg.Sources[0]
					dirRepoName = source.Name
					logger.Debug("Auto-detected repository: %s", dirRepoName)
				} else {
					logger.Fatal("Multiple repositories configured. Specify which one to use with --repo or use full URL format: cherry-go add directory REPO_URL/path/to/dir/")
				}
			} else {
				// Repository name provided explicitly
				source, exists = cfg.GetSource(dirRepoName)
				if !exists {
					logger.Fatal("Repository '%s' not found. Available repositories: %v", dirRepoName, getRepositoryNames())
				}
			}
		}

		// Set local path - default to same as source path
		localPath := dirLocalPath
		if localPath == "" {
			localPath = dirPath
		}

		// Ensure local path ends with / if it's a directory
		if localPath != "" && !strings.HasSuffix(localPath, "/") {
			localPath += "/"
		}

		// Check if this directory is already being tracked
		for _, pathSpec := range source.Paths {
			if pathSpec.Include == dirPath {
				logger.Fatal("Directory '%s' is already being tracked in repository '%s'", dirPath, dirRepoName)
			}
			// Check for overlapping paths
			if strings.HasPrefix(dirPath, pathSpec.Include) || strings.HasPrefix(pathSpec.Include, dirPath) {
				logger.Fatal("Directory '%s' overlaps with existing tracked path '%s' in repository '%s'", dirPath, pathSpec.Include, dirRepoName)
			}
		}

		// Create new path spec for the directory
		newPathSpec := config.PathSpec{
			Include:   dirPath,
			LocalPath: localPath,
			Branch:    dirBranch,
			Exclude:   dirExcludes,
			Files:     make(map[string]string), // Will be populated during sync
		}

		// Add the path spec to the source
		source.Paths = append(source.Paths, newPathSpec)

		// Update the source in configuration
		for i, cfgSource := range cfg.Sources {
			if cfgSource.Name == dirRepoName {
				cfg.Sources[i] = *source
				break
			}
		}

		// Try to sync the directory first before adding to tracking
		var syncSuccess bool
		if !logger.IsDryRun() {
			logger.Info("🔄 Syncing directory for the first time...")

			// Perform initial sync of the directory
			if err := performInitialSync(dirRepoName); err != nil {
				logger.Error("Failed to sync directory: %v", err)
				logger.Error("Directory will not be added to tracking due to sync failure")
				return
			} else {
				logger.Info("✅ Directory synced successfully!")
				syncSuccess = true
			}
		} else {
			logger.DryRunInfo("Would sync the directory automatically")
			syncSuccess = true // Assume success in dry-run
		}

		// Only save configuration if sync was successful
		if syncSuccess {
			if !logger.IsDryRun() {
				if err := cfg.Save(configFile); err != nil {
					logger.Fatal("Failed to save configuration: %v", err)
				}
			}

			logger.Info("✅ Added directory tracking:")
			logger.Info("  Repository: %s", dirRepoName)
			logger.Info("  Source path: %s", dirPath)
			logger.Info("  Local path: %s", localPath)
			if dirBranch != "" {
				logger.Info("  Branch/Tag: %s", dirBranch)
			} else {
				logger.Info("  Branch/Tag: (default)")
			}
			if len(dirExcludes) > 0 {
				logger.Info("  Excludes: %v", dirExcludes)
			}

			if logger.IsDryRun() {
				logger.DryRunInfo("Configuration would be saved to: %s", configFile)
			} else {
				logger.Info("Configuration saved to: %s", configFile)
			}
		}
	},
}

func init() {
	addCmd.AddCommand(addDirectoryCmd)

	addDirectoryCmd.Flags().StringVar(&dirRepoName, "repo", "", "repository name (auto-detected if only one configured)")
	addDirectoryCmd.Flags().StringVar(&dirLocalPath, "local-path", "", "local path for the directory (defaults to same as source path)")
	addDirectoryCmd.Flags().StringVar(&dirBranch, "branch", "", "branch or tag to track (defaults to main/master)")
	addDirectoryCmd.Flags().StringSliceVar(&dirExcludes, "exclude", []string{}, "patterns to exclude (e.g., *.tmp,test_*)")
}

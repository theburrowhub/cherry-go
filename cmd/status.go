package cmd

import (
	"cherry-go/internal/logger"

	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of tracked sources",
	Long: `Display the current status of all tracked source repositories,
including their configuration and last sync information.

Examples:
  cherry-go status
  cherry-go status --verbose`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(cfg.Sources) == 0 {
			logger.Info("No sources configured")
			return
		}

		logger.Info("Cherry-go Status Report")
		logger.Info("Configuration file: %s", configFile)
		logger.Info("")

		for i, source := range cfg.Sources {
			logger.Info("Source %d: %s", i+1, source.Name)
			logger.Info("  Repository: %s", source.Repository)
			logger.Info("  Authentication: %s", getAuthTypeDisplay(source.Auth.Type))
			logger.Info("  Paths (%d):", len(source.Paths))

			for j, path := range source.Paths {
				localPathDisplay := path.LocalPath
				if localPathDisplay == "" {
					localPathDisplay = path.Include // Default: same as source path
				}

				branchDisplay := path.Branch
				if branchDisplay == "" {
					branchDisplay = "(default)"
				}

				logger.Info("    %d. %s -> %s [%s]", j+1, path.Include, localPathDisplay, branchDisplay)

				if len(path.Exclude) > 0 {
					logger.Info("       Excludes: %v", path.Exclude)
				}

				if len(path.Files) > 0 {
					logger.Info("       Tracked files: %d", len(path.Files))
				}
			}
			logger.Info("")
		}

		logger.Info("Sync Options:")
		logger.Info("  Auto-commit: %t", cfg.Options.AutoCommit)
		logger.Info("  Commit prefix: %s", cfg.Options.CommitPrefix)
		logger.Info("  Create branch: %t", cfg.Options.CreateBranch)
		if cfg.Options.CreateBranch {
			logger.Info("  Branch prefix: %s", cfg.Options.BranchPrefix)
		}
	},
}

func getBranchOrDefault(branch string) string {
	if branch == "" {
		return "main"
	}
	return branch
}

func getAuthTypeDisplay(authType string) string {
	if authType == "" {
		return "none"
	}
	return authType
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

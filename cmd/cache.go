package cmd

import (
	"cherry-go/internal/cache"
	"cherry-go/internal/logger"
	"fmt"

	"github.com/spf13/cobra"
)

// cacheCmd represents the cache command
var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage the global repository cache",
	Long: `Manage the global repository cache stored in ~/.cache/cherry-go/repos.

This cache is shared across all cherry-go projects to avoid duplicating
repository downloads.

Available subcommands:
  list  - List cached repositories
  clean - Clean old cached repositories
	info  - Show cache information`,
	Run: func(cmd *cobra.Command, args []string) {
		// Show help when cache is called without subcommands
		_ = cmd.Help()
	},
}

// cacheListCmd represents the cache list command
var cacheListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cached repositories",
	Long:  `List all repositories currently stored in the global cache.`,
	Run: func(cmd *cobra.Command, args []string) {
		cacheManager, err := cache.NewManager()
		if err != nil {
			logger.Fatal("Failed to initialize cache manager: %v", err)
		}

		repos, err := cacheManager.ListCachedRepositories()
		if err != nil {
			logger.Fatal("Failed to list cached repositories: %v", err)
		}

		if len(repos) == 0 {
			logger.Info("No repositories in cache")
			return
		}

		logger.Info("Cached Repositories (%d):", len(repos))
		for i, repo := range repos {
			logger.Info("  %d. %s", i+1, repo.String())
		}
	},
}

// cacheInfoCmd represents the cache info command
var cacheInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show cache information",
	Long:  `Display information about the global repository cache.`,
	Run: func(cmd *cobra.Command, args []string) {
		cacheManager, err := cache.NewManager()
		if err != nil {
			logger.Fatal("Failed to initialize cache manager: %v", err)
		}

		logger.Info("Cache Information:")
		logger.Info("  Cache directory: %s", cacheManager.GetCacheDir())

		repos, err := cacheManager.ListCachedRepositories()
		if err != nil {
			logger.Error("Failed to list cached repositories: %v", err)
		} else {
			logger.Info("  Cached repositories: %d", len(repos))
		}

		size, err := cacheManager.GetCacheSize()
		if err != nil {
			logger.Error("Failed to calculate cache size: %v", err)
		} else {
			logger.Info("  Cache size: %s", formatBytes(size))
		}
	},
}

// cacheCleanCmd represents the cache clean command
var cacheCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean old cached repositories",
	Long: `Remove old cached repositories to free up disk space.

By default, repositories older than 30 days are removed.`,
	Run: func(cmd *cobra.Command, args []string) {
		cacheManager, err := cache.NewManager()
		if err != nil {
			logger.Fatal("Failed to initialize cache manager: %v", err)
		}

		maxAge := int64(30) // 30 days default

		if logger.IsDryRun() {
			logger.DryRunInfo("Would clean repositories older than %d days", maxAge)
			return
		}

		logger.Info("Cleaning cache (removing repositories older than %d days)...", maxAge)

		if err := cacheManager.CleanCache(maxAge); err != nil {
			logger.Fatal("Failed to clean cache: %v", err)
		}

		logger.Info("✅ Cache cleaned successfully")
	},
}

// formatBytes formats bytes into human readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func init() {
	rootCmd.AddCommand(cacheCmd)

	// Add subcommands
	cacheCmd.AddCommand(cacheListCmd)
	cacheCmd.AddCommand(cacheInfoCmd)
	cacheCmd.AddCommand(cacheCleanCmd)
}

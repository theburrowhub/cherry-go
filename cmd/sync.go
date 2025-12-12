package cmd

import (
	"cherry-go/internal/config"
	"cherry-go/internal/git"
	"cherry-go/internal/logger"
	"fmt"
	"os"
	"sync"

	"github.com/spf13/cobra"
)

var (
	syncAll          bool
	forceSync        bool
	branchOnConflict bool
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync [source-name]",
	Short: "Synchronize files from tracked repositories",
	Long: `Synchronize files from one or all tracked source repositories.
This will pull the latest changes and update local files accordingly.

By default, cherry-go will attempt a three-way merge when local changes
are detected. If the merge succeeds, changes are applied automatically.
If conflicts cannot be resolved, the sync is aborted.

Examples:
  # Sync all sources (with automatic merge)
  cherry-go sync --all
  
  # Sync specific source
  cherry-go sync mylib
  
  # Dry run sync
  cherry-go sync --all --dry-run
  
  # Force sync (override local changes)
  cherry-go sync --all --force
  
  # Create branch on conflict for manual resolution
  cherry-go sync --all --branch-on-conflict`,
	Run: func(cmd *cobra.Command, args []string) {
		var sourceName string
		if len(args) > 0 {
			sourceName = args[0]
		}

		if !syncAll && sourceName == "" {
			logger.Fatal("Either specify a source name or use --all flag")
		}

		if syncAll && sourceName != "" {
			logger.Fatal("Cannot specify both --all and a source name")
		}

		if forceSync && branchOnConflict {
			logger.Fatal("Cannot specify both --force and --branch-on-conflict")
		}

		workDir, err := os.Getwd()
		if err != nil {
			logger.Fatal("Failed to get current directory: %v", err)
		}

		// Determine sync mode
		mode := getSyncMode()

		if syncAll {
			syncAllSources(workDir, mode)
		} else {
			syncSingleSource(sourceName, workDir, mode)
		}
	},
}

// getSyncMode determines the sync mode based on flags
func getSyncMode() git.SyncMode {
	if forceSync {
		return git.SyncModeForce
	}
	if branchOnConflict {
		return git.SyncModeBranch
	}
	return git.SyncModeMerge // Default: attempt three-way merge
}

func syncAllSources(workDir string, mode git.SyncMode) {
	if len(cfg.Sources) == 0 {
		logger.Info("No sources configured to sync")
		return
	}

	logger.Info("Syncing %d source(s)...", len(cfg.Sources))

	// Use goroutines for concurrent syncing
	var wg sync.WaitGroup
	results := make(chan git.SyncResult, len(cfg.Sources))

	for _, source := range cfg.Sources {
		wg.Add(1)
		go func(src config.Source) {
			defer wg.Done()
			result := syncSource(&src, workDir, mode)
			results <- result
		}(source)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var totalUpdated int
	var hasErrors bool
	var hasBranchCreated bool

	for result := range results {
		if result.Error != nil {
			logger.Error("Failed to sync %s: %v", result.SourceName, result.Error)
			hasErrors = true
		} else if result.BranchCreated != "" {
			logger.Info("Conflict branch created for %s: %s", result.SourceName, result.BranchCreated)
			hasBranchCreated = true
		} else if result.HasChanges {
			logger.Info("Successfully synced %s (%d paths updated)", result.SourceName, len(result.UpdatedPaths))
			totalUpdated += len(result.UpdatedPaths)
		} else {
			logger.Info("Source %s is up to date", result.SourceName)
		}
	}

	if hasErrors {
		logger.Error("Some sources failed to sync")
	} else if hasBranchCreated {
		logger.Info("Sync completed with conflicts. Review created branches for manual resolution.")
	} else {
		logger.Info("Sync completed successfully. Total paths updated: %d", totalUpdated)
	}
}

func syncSingleSource(name string, workDir string, mode git.SyncMode) {
	source, exists := cfg.GetSource(name)
	if !exists {
		logger.Fatal("Source '%s' not found", name)
	}

	logger.Info("Syncing source '%s'...", name)
	result := syncSource(source, workDir, mode)

	if result.Error != nil {
		logger.Fatal("Failed to sync %s: %v", result.SourceName, result.Error)
	}

	if result.BranchCreated != "" {
		// Branch was created for conflict resolution
		logger.Info("Conflict branch created: %s", result.BranchCreated)
		if result.MergeInstructions != "" {
			fmt.Println(result.MergeInstructions)
		}
	} else if result.HasChanges {
		logger.Info("Successfully synced %s (%d paths updated)", result.SourceName, len(result.UpdatedPaths))
	} else {
		logger.Info("Source %s is up to date", result.SourceName)
	}
}

func syncSource(source *config.Source, workDir string, mode git.SyncMode) git.SyncResult {
	result := git.SyncResult{
		SourceName: source.Name,
	}

	// Create repository wrapper
	repo, err := git.NewRepository(source)
	if err != nil {
		result.Error = fmt.Errorf("failed to initialize repository: %w", err)
		return result
	}

	// Pull latest changes
	if err := repo.Pull(); err != nil {
		result.Error = fmt.Errorf("failed to pull changes: %w", err)
		return result
	}

	// Get latest commit hash
	commitHash, err := repo.GetLatestCommit()
	if err != nil {
		result.Error = fmt.Errorf("failed to get commit hash: %w", err)
		return result
	}
	result.CommitHash = commitHash

	// Copy paths to local directory with the specified mode
	copyResult, err := repo.CopyPaths(mode, workDir)
	if err != nil {
		result.Error = fmt.Errorf("failed to copy paths: %w", err)
		return result
	}

	result.UpdatedPaths = copyResult.UpdatedPaths
	result.Conflicts = copyResult.Conflicts
	result.HasChanges = len(copyResult.UpdatedPaths) > 0
	result.BranchCreated = copyResult.BranchCreated
	result.MergeInstructions = copyResult.MergeInstructions

	// Handle conflicts in merge mode (abort)
	if len(copyResult.Conflicts) > 0 && mode == git.SyncModeMerge {
		logger.Error("Sync aborted due to merge conflicts. Use --force to override or --branch-on-conflict for manual resolution.")
		if !logger.IsDryRun() {
			result.Error = fmt.Errorf("merge conflicts detected, sync aborted")
			return result
		}
	}

	// Save updated configuration with new hashes
	if result.HasChanges && !logger.IsDryRun() {
		// Update the source in the configuration
		for i, cfgSource := range cfg.Sources {
			if cfgSource.Name == source.Name {
				cfg.Sources[i] = *source
				break
			}
		}

		// Save configuration
		if err := cfg.Save(configFile); err != nil {
			logger.Error("Failed to save updated configuration: %v", err)
		} else {
			logger.Debug("Updated configuration saved with new file hashes")
		}
	}

	// Create commit if auto-commit is enabled and there are changes
	if cfg.Options.AutoCommit && result.HasChanges && !logger.IsDryRun() {
		commitMessage := fmt.Sprintf("%s %s from %s (%s)",
			cfg.Options.CommitPrefix,
			source.Name,
			source.Repository,
			commitHash[:8])

		if err := git.CreateCommit(workDir, commitMessage, copyResult.UpdatedPaths); err != nil {
			logger.Error("Failed to create commit: %v", err)
		}
	}

	return result
}

func init() {
	rootCmd.AddCommand(syncCmd)

	syncCmd.Flags().BoolVar(&syncAll, "all", false, "sync all configured sources")
	syncCmd.Flags().BoolVar(&forceSync, "force", false, "force sync and override local changes")
	syncCmd.Flags().BoolVar(&branchOnConflict, "branch-on-conflict", false,
		"create a branch with remote changes when merge conflicts are detected")
}

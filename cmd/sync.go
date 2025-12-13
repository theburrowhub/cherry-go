package cmd

import (
	"cherry-go/internal/config"
	"cherry-go/internal/git"
	"cherry-go/internal/logger"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

var (
	syncAll          bool
	forceSync        bool
	mergeSync        bool
	branchOnConflict bool
	markConflicts    bool
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync [source-name]",
	Short: "Synchronize files from tracked repositories",
	Long: `Synchronize files from one or all tracked source repositories.
This will pull the latest changes and update local files accordingly.

By default, cherry-go will detect and report conflicts WITHOUT making changes.
This allows you to review what would change before deciding how to proceed.

Use --merge to attempt automatic merging, or --force to overwrite local changes.

Examples:
  # Check for updates and conflicts (default - no changes made)
  cherry-go sync --all
  
  # Sync with automatic merge
  cherry-go sync --all --merge
  
  # Force sync (override local changes)
  cherry-go sync --all --force
  
  # Merge with branch creation on conflict
  cherry-go sync --all --merge --branch-on-conflict
  
  # Merge with conflict markers for manual resolution
  cherry-go sync --all --merge --mark-conflicts
  
  # Dry run to preview changes
  cherry-go sync --all --dry-run`,
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

		if forceSync && mergeSync {
			logger.Fatal("Cannot specify both --force and --merge")
		}

		if forceSync && branchOnConflict {
			logger.Fatal("Cannot specify both --force and --branch-on-conflict")
		}

		if branchOnConflict && !mergeSync {
			logger.Fatal("--branch-on-conflict requires --merge flag")
		}

		if markConflicts && !mergeSync {
			logger.Fatal("--mark-conflicts requires --merge flag")
		}

		if markConflicts && branchOnConflict {
			logger.Fatal("Cannot specify both --mark-conflicts and --branch-on-conflict")
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
	if mergeSync {
		if branchOnConflict {
			return git.SyncModeBranch
		}
		if markConflicts {
			return git.SyncModeMarkConflicts
		}
		return git.SyncModeMerge
	}
	return git.SyncModeDetect // Default: only detect conflicts, don't make changes
}

func syncAllSources(workDir string, mode git.SyncMode) {
	if len(cfg.Sources) == 0 {
		logger.Info("No sources configured to sync")
		return
	}

	if mode == git.SyncModeDetect {
		logger.Info("Checking %d source(s) for updates...", len(cfg.Sources))
	} else {
		logger.Info("Syncing %d source(s)...", len(cfg.Sources))
	}

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
	var hasConflicts bool
	var branchesCreated []git.SyncResult
	var conflictResults []git.SyncResult

	for result := range results {
		if result.Error != nil {
			logger.Error("Failed to sync %s: %v", result.SourceName, result.Error)
			hasErrors = true
		} else if result.BranchCreated != "" {
			branchesCreated = append(branchesCreated, result)
		} else if len(result.Conflicts) > 0 && mode == git.SyncModeDetect {
			hasConflicts = true
			conflictResults = append(conflictResults, result)
		} else if result.HasChanges {
			logger.Info("Successfully synced %s (%d paths updated)", result.SourceName, len(result.UpdatedPaths))
			totalUpdated += len(result.UpdatedPaths)
		} else {
			logger.Info("Source %s is up to date", result.SourceName)
		}
	}

	if hasErrors {
		logger.Error("Some sources failed to sync")
	} else if len(branchesCreated) > 0 {
		// Show detailed instructions for conflict resolution
		printConflictResolutionInstructions(branchesCreated)
	} else if hasConflicts {
		// Show instructions for detected conflicts
		printDetectedConflictsInstructions(conflictResults)
	} else {
		if mode == git.SyncModeDetect {
			logger.Info("Check completed. %d paths updated (no conflicts detected)", totalUpdated)
		} else {
			logger.Info("Sync completed successfully. Total paths updated: %d", totalUpdated)
		}
	}
}

func syncSingleSource(name string, workDir string, mode git.SyncMode) {
	source, exists := cfg.GetSource(name)
	if !exists {
		logger.Fatal("Source '%s' not found", name)
	}

	if mode == git.SyncModeDetect {
		logger.Info("Checking source '%s' for updates...", name)
	} else {
		logger.Info("Syncing source '%s'...", name)
	}
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
	} else if len(result.Conflicts) > 0 && mode == git.SyncModeDetect {
		// Conflicts detected in detect mode
		printDetectedConflictsInstructions([]git.SyncResult{result})
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
	if pullErr := repo.Pull(); pullErr != nil {
		result.Error = fmt.Errorf("failed to pull changes: %w", pullErr)
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
		logger.Error("Sync aborted due to merge conflicts. Use --force to override, --branch-on-conflict, or --mark-conflicts for manual resolution.")
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
	// BUT skip commit if using --mark-conflicts mode with conflicts (user needs to resolve manually)
	shouldCommit := cfg.Options.AutoCommit && result.HasChanges && !logger.IsDryRun()

	// Don't commit if mark-conflicts mode and there are conflicts
	if mode == git.SyncModeMarkConflicts && len(copyResult.Conflicts) > 0 {
		shouldCommit = false
		logger.Info("Changes staged but not committed - resolve conflict markers and commit manually")
	}

	if shouldCommit {
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

// printDetectedConflictsInstructions prints instructions when conflicts are detected in detect mode
func printDetectedConflictsInstructions(results []git.SyncResult) {
	// If verbosity is 0, print compact single-line format
	if logger.GetVerbosityLevel() == 0 {
		var allConflicts []string
		var sourceName string
		for _, result := range results {
			sourceName = result.SourceName
			for _, conflict := range result.Conflicts {
				allConflicts = append(allConflicts, conflict.Path)
			}
		}

		conflictsList := strings.Join(allConflicts, ", ")
		logger.Warning("⚠️  Differences detected in %s: %s. Use --merge (auto-merge), --merge --branch-on-conflict (branch), --merge --mark-conflicts (markers), or --force (overwrite)", sourceName, conflictsList)
		return
	}

	// Verbose output
	fmt.Println()
	fmt.Println("\033[33m⚠ DIFFERENCES DETECTED\033[0m")
	fmt.Println()

	for _, result := range results {
		fmt.Printf("  Source: \033[36m%s\033[0m\n", result.SourceName)
		for _, conflict := range result.Conflicts {
			fmt.Printf("    • %s\n", conflict.Path)
		}
	}

	fmt.Println()
	fmt.Println("\033[1mHow to proceed:\033[0m")
	fmt.Println()
	fmt.Println("  \033[32m--merge\033[0m                        Auto-merge (preserves local changes)")
	fmt.Println("  \033[32m--merge --branch-on-conflict\033[0m   Merge with manual control via git branch")
	fmt.Println("  \033[32m--merge --mark-conflicts\033[0m       Write conflict markers to files for manual resolution")
	fmt.Println("  \033[31m--force\033[0m                        Overwrite with remote version")
	fmt.Println()
}

// printConflictResolutionInstructions prints instructions for resolving merge conflicts via branch
func printConflictResolutionInstructions(results []git.SyncResult) {
	fmt.Println()
	fmt.Println("\033[33m⚠️  Merge Conflicts - Remote changes saved to branch\033[0m")
	fmt.Println()

	for _, result := range results {
		fmt.Printf("Source: \033[36m%s\033[0m\n", result.SourceName)
		fmt.Printf("Branch: \033[32m%s\033[0m\n", result.BranchCreated)

		if len(result.Conflicts) > 0 {
			fmt.Println("\nFiles with conflicts:")
			for _, conflict := range result.Conflicts {
				fmt.Printf("  • %s\n", conflict.Path)
			}
		}

		fmt.Println("\n\033[1mNext steps:\033[0m")
		fmt.Println("Review the changes in the branch and merge when ready.")
		fmt.Println("The branch contains the remote version - adjust as needed before merging.")
		fmt.Println()
		fmt.Printf("  git diff %s              # Review changes\n", result.BranchCreated)
		fmt.Printf("  git merge %s             # Merge when ready\n", result.BranchCreated)
		fmt.Printf("  git branch -d %s   # Delete branch after merge\n", result.BranchCreated)
		fmt.Println()
	}
}

func init() {
	rootCmd.AddCommand(syncCmd)

	syncCmd.Flags().BoolVar(&syncAll, "all", false, "sync all configured sources")
	syncCmd.Flags().BoolVar(&mergeSync, "merge", false, "attempt to merge remote changes with local modifications")
	syncCmd.Flags().BoolVar(&forceSync, "force", false, "force sync and override local changes")
	syncCmd.Flags().BoolVar(&branchOnConflict, "branch-on-conflict", false,
		"with --merge, create a branch with remote changes when merge conflicts are detected")
	syncCmd.Flags().BoolVar(&markConflicts, "mark-conflicts", false,
		"with --merge, write conflict markers to files for manual resolution (no commit)")
}

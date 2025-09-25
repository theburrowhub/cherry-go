package cmd

import (
	"cherry-go/internal/config"
	"cherry-go/internal/git"
	"cherry-go/internal/interactive"
	"cherry-go/internal/logger"
	"fmt"
	"os"
	"sync"

	"github.com/spf13/cobra"
)

var (
	forceSync       bool
	overrideAutocommit *bool // Pointer to distinguish between not set and false
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync [source-name]",
	Short: "Synchronize files from tracked repositories",
	Long: `Synchronize files from tracked source repositories.
This will pull the latest changes and update local files accordingly.

If no source name is provided, all configured sources will be synchronized
(with confirmation prompt in interactive mode).

Examples:
  # Sync all sources (with confirmation prompt)
  cherry-go sync
  
  # Sync specific source
  cherry-go sync mylib
  
  # Dry run sync
  cherry-go sync --dry-run
  
  # Force sync (override local changes)
  cherry-go sync --force
  
  # Override autocommit setting for this execution
  cherry-go sync --autocommit=true   # Force commit even if auto_commit is false
  cherry-go sync --autocommit=false  # Skip commit even if auto_commit is true`,
	Run: func(cmd *cobra.Command, args []string) {
		var sourceName string
		if len(args) > 0 {
			sourceName = args[0]
		}

		workDir, err := os.Getwd()
		if err != nil {
			logger.Fatal("Failed to get current directory: %v", err)
		}
		
		if sourceName == "" {
			// No source specified, sync all with confirmation
			syncAllSourcesWithConfirmation(workDir)
		} else {
			// Specific source specified
			syncSingleSource(sourceName, workDir)
		}
	},
}

func syncAllSourcesWithConfirmation(workDir string) {
	if len(cfg.Sources) == 0 {
		logger.Info("No sources configured to sync")
		return
	}
	
	// Show what will be synced
	logger.Info("The following %d source(s) will be synchronized:", len(cfg.Sources))
	for i, source := range cfg.Sources {
		pathCount := len(source.Paths)
		logger.Info("  %d. %s (%s) - %d path(s)", i+1, source.Name, source.Repository, pathCount)
	}
	logger.Info("")
	
	// Ask for confirmation in interactive mode
	if interactive.ShouldPrompt() && !logger.IsDryRun() {
		if !interactive.Confirm("Do you want to proceed with synchronization?") {
			logger.Info("Synchronization cancelled by user")
			return
		}
		logger.Info("")
	} else if !interactive.ShouldPrompt() {
		logger.Info("Non-interactive mode detected, proceeding automatically...")
	}
	
	// Proceed with sync
	syncAllSources(workDir)
}

func syncAllSources(workDir string) {
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
			result := syncSource(&src, workDir)
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

	for result := range results {
		if result.Error != nil {
			logger.Error("Failed to sync %s: %v", result.SourceName, result.Error)
			hasErrors = true
		} else if result.HasChanges {
			logger.Info("Successfully synced %s (%d paths updated)", result.SourceName, len(result.UpdatedPaths))
			totalUpdated += len(result.UpdatedPaths)
		} else {
			logger.Info("Source %s is up to date", result.SourceName)
		}
	}

	if hasErrors {
		logger.Error("Some sources failed to sync")
	} else {
		logger.Info("Sync completed successfully. Total paths updated: %d", totalUpdated)
	}
}

func syncSingleSource(name string, workDir string) {
	source, exists := cfg.GetSource(name)
	if !exists {
		logger.Fatal("Source '%s' not found", name)
	}

	logger.Info("Syncing source '%s'...", name)
	result := syncSource(source, workDir)

	if result.Error != nil {
		logger.Fatal("Failed to sync %s: %v", result.SourceName, result.Error)
	}

	if result.HasChanges {
		logger.Info("Successfully synced %s (%d paths updated)", result.SourceName, len(result.UpdatedPaths))
	} else {
		logger.Info("Source %s is up to date", result.SourceName)
	}
}

func syncSource(source *config.Source, workDir string) git.SyncResult {
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

	// Copy paths to local directory
	updatedPaths, conflicts, err := repo.CopyPaths(forceSync)
	if err != nil {
		result.Error = fmt.Errorf("failed to copy paths: %w", err)
		return result
	}

	result.UpdatedPaths = updatedPaths
	result.Conflicts = conflicts
	result.HasChanges = len(updatedPaths) > 0

	// Handle conflicts
	if len(conflicts) > 0 && !forceSync {
		logger.Error("Sync aborted due to conflicts. Use --force to override or resolve manually.")
		if !logger.IsDryRun() {
			result.Error = fmt.Errorf("conflicts detected, sync aborted")
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

	// Determine if we should commit based on config and override
	shouldCommit := cfg.Options.AutoCommit
	if overrideAutocommit != nil {
		shouldCommit = *overrideAutocommit
		if *overrideAutocommit {
			logger.Info("🔧 Overriding autocommit: forcing commit for this execution")
		} else {
			logger.Info("🔧 Overriding autocommit: skipping commit for this execution")
		}
	}

	// Create commit if enabled and there are changes
	if shouldCommit && result.HasChanges && !logger.IsDryRun() {
		commitMessage := fmt.Sprintf("%s %s from %s (%s)", 
			cfg.Options.CommitPrefix, 
			source.Name, 
			source.Repository, 
			commitHash[:8])
		
		if err := git.CreateCommit(workDir, commitMessage, updatedPaths); err != nil {
			logger.Error("Failed to create commit: %v", err)
		} else {
			logger.Info("✅ Created commit: %s", commitMessage)
		}
	} else if !shouldCommit && result.HasChanges {
		if overrideAutocommit != nil {
			logger.Info("📝 Changes synced but not committed (autocommit overridden to false)")
		} else {
			logger.Info("📝 Changes synced but not committed (autocommit disabled in config)")
		}
	}

	return result
}

func init() {
	rootCmd.AddCommand(syncCmd)
	
	syncCmd.Flags().BoolVar(&forceSync, "force", false, "force sync and override local changes")
	
	// Use a custom flag function to handle the pointer
	syncCmd.Flags().BoolVar(new(bool), "autocommit", false, "override autocommit setting (true=force commit, false=skip commit)")
	
	// Custom handling for the autocommit flag
	syncCmd.PreRun = func(cmd *cobra.Command, args []string) {
		if cmd.Flags().Changed("autocommit") {
			autocommitValue, _ := cmd.Flags().GetBool("autocommit")
			overrideAutocommit = &autocommitValue
		}
	}
}

package cmd

import (
	"cherry-go/internal/git"
	"cherry-go/internal/logger"
	"os"

	"github.com/spf13/cobra"
)

var (
	cleanupAll bool
)

// cleanupCmd represents the cleanup command
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up conflict branches created by cherry-go",
	Long: `Clean up conflict branches that were created during sync operations with conflicts.

When cherry-go encounters conflicts during sync with --merge --branch-on-conflict,
it creates branches with the prefix configured in your .cherry-go.yaml (default: cherry-go/sync/).

This command helps you clean up these branches after you've resolved the conflicts.

Examples:
  # List all conflict branches
  cherry-go cleanup
  
  # Delete all conflict branches
  cherry-go cleanup --all`,
	Run: func(cmd *cobra.Command, args []string) {
		workDir, err := os.Getwd()
		if err != nil {
			logger.Fatal("Failed to get current directory: %v", err)
		}

		// Get branch prefix from config
		branchPrefix := cfg.Options.BranchPrefix
		if branchPrefix == "" {
			branchPrefix = "cherry-go/sync"
		}

		if cleanupAll {
			deleteAllConflictBranches(workDir, branchPrefix)
		} else {
			listConflictBranches(workDir, branchPrefix)
		}
	},
}

func listConflictBranches(workDir string, branchPrefix string) {
	branches, err := git.ListConflictBranches(workDir, branchPrefix)
	if err != nil {
		logger.Fatal("Failed to list conflict branches: %v", err)
	}

	if len(branches) == 0 {
		logger.Info("No conflict branches found with prefix '%s'", branchPrefix)
		return
	}

	logger.Info("Found %d conflict branch(es):", len(branches))
	for i, branch := range branches {
		logger.Info("  %d. %s", i+1, branch)
	}
	logger.Info("")
	logger.Info("To delete all conflict branches, run:")
	logger.Info("  cherry-go cleanup --all")
}

func deleteAllConflictBranches(workDir string, branchPrefix string) {
	branches, err := git.ListConflictBranches(workDir, branchPrefix)
	if err != nil {
		logger.Fatal("Failed to list conflict branches: %v", err)
	}

	if len(branches) == 0 {
		logger.Info("No conflict branches found with prefix '%s'", branchPrefix)
		return
	}

	logger.Info("Found %d conflict branch(es) to delete:", len(branches))
	for i, branch := range branches {
		logger.Info("  %d. %s", i+1, branch)
	}
	logger.Info("")

	if logger.IsDryRun() {
		logger.DryRunInfo("Would delete %d conflict branch(es)", len(branches))
		return
	}

	deleted, err := git.DeleteAllConflictBranches(workDir, branchPrefix)
	if err != nil {
		logger.Error("Failed to delete all branches: %v", err)
		if len(deleted) > 0 {
			logger.Info("Successfully deleted %d branch(es):", len(deleted))
			for _, branch := range deleted {
				logger.Info("  ✓ %s", branch)
			}
		}
		os.Exit(1)
	}

	logger.Info("✅ Successfully deleted %d conflict branch(es)", len(deleted))
	for _, branch := range deleted {
		logger.Info("  ✓ %s", branch)
	}
}

func init() {
	rootCmd.AddCommand(cleanupCmd)

	cleanupCmd.Flags().BoolVar(&cleanupAll, "all", false, "delete all conflict branches")
}

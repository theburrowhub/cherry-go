package cmd

import (
	"cherry-go/internal/logger"

	"github.com/spf13/cobra"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove [source-name]",
	Short: "Remove a source repository from tracking",
	Long: `Remove a source repository and stop tracking its files.

Examples:
  cherry-go remove mylib
  cherry-go remove private`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sourceName := args[0]

		// Check if source exists
		if _, exists := cfg.GetSource(sourceName); !exists {
			logger.Fatal("Source '%s' not found", sourceName)
		}

		// Remove from configuration
		if !cfg.RemoveSource(sourceName) {
			logger.Fatal("Failed to remove source '%s'", sourceName)
		}

		// Save configuration
		if !logger.IsDryRun() {
			if err := cfg.Save(configFile); err != nil {
				logger.Fatal("Failed to save configuration: %v", err)
			}
		}

		logger.Info("Removed source '%s'", sourceName)
		if logger.IsDryRun() {
			logger.DryRunInfo("Configuration would be saved to: %s", configFile)
		} else {
			logger.Info("Configuration saved to: %s", configFile)
		}
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}

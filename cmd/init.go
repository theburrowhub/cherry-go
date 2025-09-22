package cmd

import (
	"cherry-go/internal/config"
	"cherry-go/internal/logger"
	"os"

	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new cherry-go configuration file",
	Long: `Initialize a new cherry-go configuration file in the current directory.
This will create a .cherry-go.yaml file with default settings.

If a configuration file already exists, this command will exit with an error
to prevent accidentally overwriting existing configuration.

Examples:
  cherry-go init
  cherry-go init --config custom-config.yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		// Check if configuration file already exists
		if _, err := os.Stat(configFile); err == nil {
			logger.Fatal("Configuration file already exists: %s\nUse 'cherry-go status' to view current configuration or remove the file to reinitialize.", configFile)
		}

		// Create default configuration
		defaultCfg := config.DefaultConfig()

		if logger.IsDryRun() {
			logger.DryRunInfo("Would create configuration file: %s", configFile)
			logger.Info("Default configuration:")
			logger.Info("  Version: %s", defaultCfg.Version)
			logger.Info("  Auto-commit: %t", defaultCfg.Options.AutoCommit)
			logger.Info("  Commit prefix: %s", defaultCfg.Options.CommitPrefix)
			return
		}

		// Save default configuration
		if err := defaultCfg.Save(configFile); err != nil {
			logger.Fatal("Failed to create configuration file: %v", err)
		}

		logger.Info("✅ Initialized cherry-go configuration: %s", configFile)
		logger.Info("")
		logger.Info("Next steps:")
		logger.Info("1. Add files directly (auto-detects and adds repository):")
		logger.Info("   cherry-go add file https://github.com/user/repo.git/src/main.go")
		logger.Info("   cherry-go add directory https://github.com/user/repo.git/src/")
		logger.Info("")
		logger.Info("Or step-by-step:")
		logger.Info("1. Add repository: cherry-go add repo https://github.com/user/repo.git")
		logger.Info("2. Add files: cherry-go add file src/main.go")
		logger.Info("3. Check status: cherry-go status")
		logger.Info("")
		logger.Info("Edit %s to customize configuration", configFile)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

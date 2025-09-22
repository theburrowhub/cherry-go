package cmd

import (
	"cherry-go/internal/config"
	"cherry-go/internal/logger"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	configFile string
	dryRun     bool
	verbose    bool
	cfg        *config.Config
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cherry-go",
	Short: "A tool for partial versioning of files from other Git repositories",
	Long: `Cherry-go allows you to selectively version files and directories from other Git repositories
into your local repository, keeping them synchronized when changes occur in the source.

Features:
- Select specific files/directories from remote repositories
- Automatic synchronization with source repositories
- Support for private repositories with authentication
- Concurrent operations for multiple sources
- Dry-run mode for testing changes
- Configurable via YAML file`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if verbose {
			logger.Debug("Verbose mode enabled")
		}

		if dryRun {
			logger.SetDryRun(true)
			logger.Info("Running in dry-run mode - no changes will be made")
		}

		// Load configuration
		var err error
		cfg, err = config.Load(configFile)
		if err != nil {
			logger.Fatal("Failed to load configuration: %v", err)
		}

		logger.Debug("Configuration loaded from: %s", configFile)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is .cherry-go.yaml)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "simulate actions without making changes")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if configFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(configFile)
	} else {
		// Get current working directory
		cwd, err := os.Getwd()
		cobra.CheckErr(err)

		// Search config in current directory only (project-specific)
		viper.AddConfigPath(cwd)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".cherry-go")
		
		// Set default config file path to current directory
		configFile = filepath.Join(cwd, ".cherry-go.yaml")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		logger.Debug("Using config file: %s", viper.ConfigFileUsed())
	}
}

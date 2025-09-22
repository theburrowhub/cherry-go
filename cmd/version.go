package cmd

import (
	"cherry-go/internal/logger"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version will be set during build
	Version = "0.1.0"
	// CommitHash will be set during build
	CommitHash = "unknown"
	// BuildTime will be set during build
	BuildTime = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display version, commit hash, and build time information for cherry-go.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("Cherry-go version: %s", Version)
		logger.Info("Commit hash: %s", CommitHash)
		logger.Info("Build time: %s", BuildTime)

		if verbose {
			logger.Info("Go version: %s", fmt.Sprintf("%s", "go1.21+"))
			logger.Info("Platform: %s", "cross-platform")
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

package cmd

import (
	"cherry-go/internal/git"
	"cherry-go/internal/logger"
	"fmt"
	"os"
)

// performInitialSync performs the initial sync for a newly added file/directory
func performInitialSync(repoName string) error {
	// Get current working directory
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Get the source from configuration
	source, exists := cfg.GetSource(repoName)
	if !exists {
		return fmt.Errorf("repository '%s' not found", repoName)
	}

	// Perform sync for this specific source using default merge mode
	result := syncSource(source, workDir, git.SyncModeMerge)

	if result.Error != nil {
		return result.Error
	}

	if len(result.Conflicts) > 0 {
		logger.Warning("Conflicts detected during initial sync:")
		for _, conflict := range result.Conflicts {
			logger.Warning("  - %s", conflict.String())
		}
	}

	return nil
}

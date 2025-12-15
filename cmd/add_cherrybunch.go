package cmd

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"cherry-go/internal/config"
	"cherry-go/internal/logger"
)

var (
	cherryBunchName string
)

// addCherryBunchCmd represents the add cherrybunch command
var addCherryBunchCmd = &cobra.Command{
	Use:     "cherrybunch [URL or file]",
	Aliases: []string{"cb"},
	Short:   "Add a cherry bunch template to initialize file sets",
	Long: `Add a cherry bunch template to initialize file sets from a repository.

Cherry bunches are YAML template files that describe a set of files and directories
to synchronize from a repository, making it easy to quickly set up common configurations.

Examples:
  # Add a cherry bunch from a URL
  cherry-go add cherrybunch https://raw.githubusercontent.com/user/bunches/main/python.cherrybunch
  
  # Add a cherry bunch from a local file
  cherry-go add cherrybunch ./templates/python.cherrybunch
  
  # Add with custom name
  cherry-go add cb --name my-python-setup https://example.com/python.cherrybunch

The cherry bunch file should have a .cherrybunch extension and contain:
- name: Template name
- description: Optional description
- repository: Source repository URL
- files: List of files to sync
- directories: List of directories to sync`,
	Args: cobra.ExactArgs(1),
	Run:  runAddCherryBunch,
}

func runAddCherryBunch(cmd *cobra.Command, args []string) {
	source := args[0]

	logger.Info("Adding cherry bunch from: %s", source)

	// Load the cherry bunch
	var cherryBunch *config.CherryBunch
	var err error

	if isURL(source) {
		cherryBunch, err = loadCherryBunchFromURL(source)
	} else {
		cherryBunch, err = config.LoadCherryBunch(source)
	}

	if err != nil {
		logger.Fatal("Failed to load cherry bunch: %v", err)
	}

	// Override name if provided
	if cherryBunchName != "" {
		cherryBunch.Name = cherryBunchName
	}

	logger.Info("Loaded cherry bunch: %s", cherryBunch.Name)
	logger.Info("Description: %s", cherryBunch.Description)
	logger.Info("Repository: %s", cherryBunch.Repository)
	logger.Info("Files: %d", len(cherryBunch.Files))
	logger.Info("Directories: %d", len(cherryBunch.Directories))

	if dryRun {
		logger.Info("Dry run mode - would apply cherry bunch to configuration")
		return
	}

	// Apply cherry bunch to configuration
	if err := cfg.ApplyCherryBunch(cherryBunch); err != nil {
		logger.Fatal("Failed to apply cherry bunch: %v", err)
	}

	// Save configuration
	if err := cfg.Save(configFile); err != nil {
		logger.Fatal("Failed to save configuration: %v", err)
	}

	logger.Info("Cherry bunch '%s' added successfully!", cherryBunch.Name)
	logger.Info("Run 'cherry-go sync %s' to synchronize the files", cherryBunch.Name)
}

func loadCherryBunchFromURL(url string) (*config.CherryBunch, error) {
	logger.Debug("Downloading cherry bunch from URL: %s", url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download cherry bunch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download cherry bunch: HTTP %d", resp.StatusCode)
	}

	// Read response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Load from data
	return config.LoadCherryBunchFromData(data)
}

func isURL(str string) bool {
	return strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://")
}

func init() {
	addCmd.AddCommand(addCherryBunchCmd)

	// Flags
	addCherryBunchCmd.Flags().StringVar(&cherryBunchName, "name", "", "custom name for the cherry bunch (overrides the name in the file)")
}

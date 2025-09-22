package cmd

import (
	"cherry-go/internal/config"
	"cherry-go/internal/logger"
	"cherry-go/internal/utils"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

var (
	repoName     string
	repoAuthType string
	repoAuthUser string
	repoSSHKey   string
)

// addRepoCmd represents the add repo command
var addRepoCmd = &cobra.Command{
	Use:   "repo [repository-url]",
	Short: "Add a new repository to track",
	Args:  cobra.ExactArgs(1),
	Long: `Add a new repository configuration. This creates a repository entry
that can be used later to track files and directories from any branch or tag.

The repository name is automatically extracted from the URL unless specified with --name.
Authentication type is automatically detected based on the repository URL.

Examples:
  # Add a public repository (name auto-detected)
  cherry-go add repo https://github.com/user/library.git
  
  # Add with custom name
  cherry-go add repo https://github.com/user/library.git --name mylib
  
  # Add a private repository with SSH
  cherry-go add repo git@github.com:company/private.git
  
  # Add with custom SSH key
  cherry-go add repo git@git.company.com:team/repo.git --auth-ssh-key ~/.ssh/company_key`,
	Run: func(cmd *cobra.Command, args []string) {
		repoURL := args[0]

		// Auto-generate repository name if not provided
		if repoName == "" {
			repoName = utils.ExtractRepoName(repoURL)
			logger.Debug("Auto-detected repository name: %s", repoName)
		}

		// Auto-detect authentication type if not specified
		if repoAuthType == "" || repoAuthType == "auto" {
			repoAuthType = detectAuthType(repoURL)
		}

		// Create auth config
		auth := config.AuthConfig{
			Type:     repoAuthType,
			Username: repoAuthUser,
			SSHKey:   repoSSHKey,
		}

		// Check if repository already exists
		if _, exists := cfg.GetSource(repoName); exists {
			logger.Fatal("Repository '%s' already exists. Use a different name or remove the existing one first.", repoName)
		}

		// Create source without paths (they will be added later)
		source := config.Source{
			Name:       repoName,
			Repository: repoURL,
			Auth:       auth,
			Paths:      []config.PathSpec{}, // Empty initially
		}

		// Add to configuration
		cfg.AddSource(source)

		// Save configuration
		if !logger.IsDryRun() {
			if err := cfg.Save(configFile); err != nil {
				logger.Fatal("Failed to save configuration: %v", err)
			}
		}

		logger.Info("✅ Added repository '%s'", repoName)
		logger.Info("  URL: %s", repoURL)
		logger.Info("  Authentication: %s", repoAuthType)
		logger.Info("")
		logger.Info("Next steps:")
		logger.Info("  Add files: cherry-go add file %s/path/to/file.ext", repoURL)
		logger.Info("  Add directories: cherry-go add directory %s/path/to/dir/", repoURL)
		
		if logger.IsDryRun() {
			logger.DryRunInfo("Configuration would be saved to: %s", configFile)
		} else {
			logger.Info("Configuration saved to: %s", configFile)
		}
	},
}

// detectAuthType automatically detects the authentication type based on URL
func detectAuthType(repoURL string) string {
	// Parse URL to determine protocol
	if strings.HasPrefix(repoURL, "git@") {
		return "ssh"
	}
	
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		logger.Debug("Failed to parse URL, defaulting to auto: %v", err)
		return "auto"
	}
	
	switch {
	case parsedURL.Scheme == "ssh":
		return "ssh"
	case parsedURL.Scheme == "https":
		// Check for common Git hosting services
		switch parsedURL.Host {
		case "github.com", "gitlab.com":
			return "auto" // Will try environment variables
		default:
			return "auto"
		}
	default:
		return "auto"
	}
}

func init() {
	addCmd.AddCommand(addRepoCmd)

	addRepoCmd.Flags().StringVar(&repoName, "name", "", "repository name (auto-detected from URL if not provided)")
	addRepoCmd.Flags().StringVar(&repoAuthType, "auth-type", "auto", "authentication type (auto, ssh, basic)")
	addRepoCmd.Flags().StringVar(&repoAuthUser, "auth-user", "", "username for basic auth")
	addRepoCmd.Flags().StringVar(&repoSSHKey, "auth-ssh-key", "", "path to SSH private key")
}

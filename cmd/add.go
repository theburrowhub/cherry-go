package cmd

import (
	"github.com/spf13/cobra"
)

// addCmd represents the add command (parent command)
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add repositories, files, or directories to track",
	Long: `Add repositories, files, or directories to track from remote Git repositories.

This command has three subcommands:

1. add repo     - Add a repository configuration
2. add file     - Add a specific file to track from a repository  
3. add directory - Add a directory to track from a repository

Workflow:
  1. First, add a repository: cherry-go add repo --name mylib --url https://github.com/user/lib.git
  2. Then, add files or directories: cherry-go add file --repo mylib --path src/main.go
  3. Finally, sync: cherry-go sync mylib

Examples:
  # Add a repository
  cherry-go add repo --name mylib --url https://github.com/user/library.git
  
  # Add a file from that repository
  cherry-go add file --repo mylib --path src/main.go --local-path internal/main.go
  
  # Add a directory from that repository  
	cherry-go add directory --repo mylib --path src/ --local-path internal/mylib/`,
	Run: func(cmd *cobra.Command, args []string) {
		// Show help when add is called without subcommands
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}

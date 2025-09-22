package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	Version string      `yaml:"version"`
	Sources []Source    `yaml:"sources"`
	Options SyncOptions `yaml:"options,omitempty"`
}

// Source represents a remote repository source
type Source struct {
	Name       string     `yaml:"name"`
	Repository string     `yaml:"repository"`
	Auth       AuthConfig `yaml:"auth,omitempty"`
	Paths      []PathSpec `yaml:"paths"`
}

// PathSpec represents a path specification with includes and excludes
type PathSpec struct {
	Include   string            `yaml:"include"`
	Exclude   []string          `yaml:"exclude,omitempty"`
	LocalPath string            `yaml:"local_path,omitempty"` // Exact local path where file/dir should be placed
	Branch    string            `yaml:"branch,omitempty"`     // Branch or tag to track for this specific path
	Files     map[string]string `yaml:"files,omitempty"`      // filename -> hash mapping
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Type     string `yaml:"type,omitempty"`     // "ssh", "basic", "auto"
	Username string `yaml:"username,omitempty"` // For basic auth only
	SSHKey   string `yaml:"ssh_key,omitempty"`  // Optional: specific SSH key path
	// Note: Tokens and passwords are NOT stored in config for security
	// Use environment variables or SSH agent instead
}

// SyncOptions represents synchronization options
type SyncOptions struct {
	AutoCommit   bool   `yaml:"auto_commit"`
	CommitPrefix string `yaml:"commit_prefix,omitempty"`
	CreateBranch bool   `yaml:"create_branch"`
	BranchPrefix string `yaml:"branch_prefix,omitempty"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Version: "1.0",
		Sources: []Source{},
		Options: SyncOptions{
			AutoCommit:   true,
			CommitPrefix: "cherry-go: sync",
			CreateBranch: false,
			BranchPrefix: "cherry-go/sync",
		},
	}
}

// Load loads configuration from a file
func Load(configPath string) (*Config, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults for missing fields
	if config.Version == "" {
		config.Version = "1.0"
	}
	if config.Options.CommitPrefix == "" {
		config.Options.CommitPrefix = "cherry-go: sync"
	}
	if config.Options.BranchPrefix == "" {
		config.Options.BranchPrefix = "cherry-go/sync"
	}

	return &config, nil
}

// Save saves configuration to a file
func (c *Config) Save(configPath string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// AddSource adds a new source to the configuration
func (c *Config) AddSource(source Source) {
	// Check if source already exists
	for i, existing := range c.Sources {
		if existing.Name == source.Name {
			c.Sources[i] = source
			return
		}
	}
	c.Sources = append(c.Sources, source)
}

// RemoveSource removes a source from the configuration
func (c *Config) RemoveSource(name string) bool {
	for i, source := range c.Sources {
		if source.Name == name {
			c.Sources = append(c.Sources[:i], c.Sources[i+1:]...)
			return true
		}
	}
	return false
}

// GetSource returns a source by name
func (c *Config) GetSource(name string) (*Source, bool) {
	for _, source := range c.Sources {
		if source.Name == name {
			return &source, true
		}
	}
	return nil, false
}

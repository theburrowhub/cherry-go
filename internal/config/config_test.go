package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", cfg.Version)
	}

	// LocalPrefix has been removed, so we don't test it anymore

	if !cfg.Options.AutoCommit {
		t.Error("Expected auto-commit to be true by default")
	}

	if cfg.Options.CommitPrefix != "cherry-go: sync" {
		t.Errorf("Expected commit prefix 'cherry-go: sync', got %s", cfg.Options.CommitPrefix)
	}
}

func TestAddSource(t *testing.T) {
	cfg := DefaultConfig()

	source := Source{
		Name:       "test-source",
		Repository: "https://github.com/test/repo.git",
		Paths: []PathSpec{
			{Include: "src/"},
		},
	}

	cfg.AddSource(source)

	if len(cfg.Sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(cfg.Sources))
	}

	if cfg.Sources[0].Name != "test-source" {
		t.Errorf("Expected source name 'test-source', got %s", cfg.Sources[0].Name)
	}
}

func TestRemoveSource(t *testing.T) {
	cfg := DefaultConfig()

	source := Source{
		Name:       "test-source",
		Repository: "https://github.com/test/repo.git",
	}

	cfg.AddSource(source)

	if !cfg.RemoveSource("test-source") {
		t.Error("Expected RemoveSource to return true")
	}

	if len(cfg.Sources) != 0 {
		t.Errorf("Expected 0 sources after removal, got %d", len(cfg.Sources))
	}

	if cfg.RemoveSource("non-existent") {
		t.Error("Expected RemoveSource to return false for non-existent source")
	}
}

func TestGetSource(t *testing.T) {
	cfg := DefaultConfig()

	source := Source{
		Name:       "test-source",
		Repository: "https://github.com/test/repo.git",
	}

	cfg.AddSource(source)

	found, exists := cfg.GetSource("test-source")
	if !exists {
		t.Error("Expected source to exist")
	}

	if found.Name != "test-source" {
		t.Errorf("Expected source name 'test-source', got %s", found.Name)
	}

	_, exists = cfg.GetSource("non-existent")
	if exists {
		t.Error("Expected source to not exist")
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "cherry-go-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "test-config.yaml")

	// Create test configuration
	cfg := DefaultConfig()

	source := Source{
		Name:       "test-source",
		Repository: "https://github.com/test/repo.git",
		Paths: []PathSpec{
			{
				Include:   "src/",
				Exclude:   []string{"*.tmp"},
				LocalPath: "local/src",
				Branch:    "develop",
			},
		},
		Auth: AuthConfig{
			Type: "ssh",
		},
	}

	cfg.AddSource(source)

	// Save configuration
	if saveErr := cfg.Save(configPath); saveErr != nil {
		t.Fatalf("Failed to save config: %v", saveErr)
	}

	// Load configuration
	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded configuration (LocalPrefix removed)

	if len(loadedCfg.Sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(loadedCfg.Sources))
	}

	loadedSource := loadedCfg.Sources[0]
	if loadedSource.Name != "test-source" {
		t.Errorf("Expected source name 'test-source', got %s", loadedSource.Name)
	}

	if len(loadedSource.Paths) != 1 {
		t.Errorf("Expected 1 path, got %d", len(loadedSource.Paths))
	}

	if loadedSource.Paths[0].Branch != "develop" {
		t.Errorf("Expected path branch 'develop', got %s", loadedSource.Paths[0].Branch)
	}

	if loadedSource.Auth.Type != "ssh" {
		t.Errorf("Expected auth type 'ssh', got %s", loadedSource.Auth.Type)
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	cfg, err := Load("non-existent-file.yaml")
	if err != nil {
		t.Errorf("Expected no error for non-existent file, got %v", err)
	}

	// Should return default configuration
	if cfg.Version != "1.0" {
		t.Errorf("Expected default version 1.0, got %s", cfg.Version)
	}
}

package cache

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

// BaseContentManager handles snapshots of synced content for three-way merge
type BaseContentManager struct {
	baseDir string
}

// NewBaseContentManager creates a new base content manager
func NewBaseContentManager() (*BaseContentManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	baseDir := filepath.Join(homeDir, ".cache", "cherry-go", "base-content")

	// Ensure base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base content directory: %w", err)
	}

	return &BaseContentManager{
		baseDir: baseDir,
	}, nil
}

// GetBaseDir returns the base content directory path
func (m *BaseContentManager) GetBaseDir() string {
	return m.baseDir
}

// getSnapshotPath returns the path for a specific source/path snapshot
func (m *BaseContentManager) getSnapshotPath(sourceName, pathSpec string) string {
	// Hash the pathSpec to create a safe directory name
	pathHash := fmt.Sprintf("%x", sha256.Sum256([]byte(pathSpec)))[:16]
	return filepath.Join(m.baseDir, sourceName, pathHash)
}

// SaveSnapshot saves the content of files after a successful sync
func (m *BaseContentManager) SaveSnapshot(sourceName, pathSpec string, files map[string][]byte) error {
	snapshotPath := m.getSnapshotPath(sourceName, pathSpec)

	// Remove existing snapshot if any
	if err := os.RemoveAll(snapshotPath); err != nil {
		return fmt.Errorf("failed to remove existing snapshot: %w", err)
	}

	// Create snapshot directory
	if err := os.MkdirAll(snapshotPath, 0755); err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	// Save each file
	for relPath, content := range files {
		filePath := filepath.Join(snapshotPath, relPath)

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", relPath, err)
		}

		if err := os.WriteFile(filePath, content, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", relPath, err)
		}
	}

	return nil
}

// GetSnapshot retrieves the base content for three-way merge
func (m *BaseContentManager) GetSnapshot(sourceName, pathSpec string) (map[string][]byte, error) {
	snapshotPath := m.getSnapshotPath(sourceName, pathSpec)

	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		return nil, nil // No snapshot exists
	}

	files := make(map[string][]byte)

	err := filepath.Walk(snapshotPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(snapshotPath, path)
		if err != nil {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		files[relPath] = content
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot: %w", err)
	}

	return files, nil
}

// GetFileContent retrieves a single file from the snapshot
func (m *BaseContentManager) GetFileContent(sourceName, pathSpec, relPath string) ([]byte, error) {
	snapshotPath := m.getSnapshotPath(sourceName, pathSpec)
	filePath := filepath.Join(snapshotPath, relPath)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil // File doesn't exist in snapshot
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return content, nil
}

// HasSnapshot checks if a snapshot exists for the given source/path
func (m *BaseContentManager) HasSnapshot(sourceName, pathSpec string) bool {
	snapshotPath := m.getSnapshotPath(sourceName, pathSpec)
	_, err := os.Stat(snapshotPath)
	return err == nil
}

// DeleteSnapshot removes a snapshot for a source/path
func (m *BaseContentManager) DeleteSnapshot(sourceName, pathSpec string) error {
	snapshotPath := m.getSnapshotPath(sourceName, pathSpec)
	return os.RemoveAll(snapshotPath)
}

// DeleteSourceSnapshots removes all snapshots for a source
func (m *BaseContentManager) DeleteSourceSnapshots(sourceName string) error {
	sourcePath := filepath.Join(m.baseDir, sourceName)
	return os.RemoveAll(sourcePath)
}

// CleanOrphanedSnapshots removes snapshots for sources that no longer exist
func (m *BaseContentManager) CleanOrphanedSnapshots(validSources []string) error {
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read base content directory: %w", err)
	}

	validSet := make(map[string]bool)
	for _, source := range validSources {
		validSet[source] = true
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if !validSet[entry.Name()] {
			sourcePath := filepath.Join(m.baseDir, entry.Name())
			if err := os.RemoveAll(sourcePath); err != nil {
				return fmt.Errorf("failed to remove orphaned snapshot %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

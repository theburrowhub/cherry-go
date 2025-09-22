package hash

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileHasher handles file hashing operations
type FileHasher struct{}

// NewFileHasher creates a new file hasher
func NewFileHasher() *FileHasher {
	return &FileHasher{}
}

// HashFile calculates SHA256 hash of a file
func (fh *FileHasher) HashFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to hash file %s: %w", filePath, err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// HashDirectory calculates hashes for all files in a directory
func (fh *FileHasher) HashDirectory(dirPath string, excludes []string) (map[string]string, error) {
	hashes := make(map[string]string)

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path from the base directory
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}

		// Check if file should be excluded
		if fh.shouldExclude(relPath, excludes) {
			return nil
		}

		// Calculate hash
		hash, err := fh.HashFile(path)
		if err != nil {
			return err
		}

		hashes[relPath] = hash
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to hash directory %s: %w", dirPath, err)
	}

	return hashes, nil
}

// shouldExclude checks if a file should be excluded based on patterns
func (fh *FileHasher) shouldExclude(path string, excludes []string) bool {
	for _, exclude := range excludes {
		if matched, _ := filepath.Match(exclude, filepath.Base(path)); matched {
			return true
		}
		if strings.Contains(path, exclude) {
			return true
		}
	}
	return false
}

// CompareHashes compares two hash maps and returns differences
func (fh *FileHasher) CompareHashes(oldHashes, newHashes map[string]string) (modified, added, removed []string) {
	// Check for modified and removed files
	for file, oldHash := range oldHashes {
		if newHash, exists := newHashes[file]; exists {
			if oldHash != newHash {
				modified = append(modified, file)
			}
		} else {
			removed = append(removed, file)
		}
	}

	// Check for added files
	for file := range newHashes {
		if _, exists := oldHashes[file]; !exists {
			added = append(added, file)
		}
	}

	return modified, added, removed
}

// VerifyFileIntegrity checks if local files match expected hashes
func (fh *FileHasher) VerifyFileIntegrity(baseDir string, expectedHashes map[string]string) (conflicts []FileConflict, err error) {
	for relPath, expectedHash := range expectedHashes {
		fullPath := filepath.Join(baseDir, relPath)
		
		// Check if file exists
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			conflicts = append(conflicts, FileConflict{
				Path:         relPath,
				Type:         ConflictTypeDeleted,
				ExpectedHash: expectedHash,
				ActualHash:   "",
			})
			continue
		}

		// Calculate current hash
		actualHash, err := fh.HashFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to hash file %s: %w", fullPath, err)
		}

		// Compare hashes
		if actualHash != expectedHash {
			conflicts = append(conflicts, FileConflict{
				Path:         relPath,
				Type:         ConflictTypeModified,
				ExpectedHash: expectedHash,
				ActualHash:   actualHash,
			})
		}
	}

	return conflicts, nil
}

// ConflictType represents the type of file conflict
type ConflictType string

const (
	ConflictTypeModified ConflictType = "modified"
	ConflictTypeDeleted  ConflictType = "deleted"
	ConflictTypeAdded    ConflictType = "added"
)

// FileConflict represents a conflict between expected and actual file state
type FileConflict struct {
	Path         string
	Type         ConflictType
	ExpectedHash string
	ActualHash   string
}

// String returns a human-readable description of the conflict
func (fc FileConflict) String() string {
	switch fc.Type {
	case ConflictTypeModified:
		return fmt.Sprintf("Modified: %s (expected: %s, actual: %s)", fc.Path, fc.ExpectedHash[:8], fc.ActualHash[:8])
	case ConflictTypeDeleted:
		return fmt.Sprintf("Deleted: %s (expected: %s)", fc.Path, fc.ExpectedHash[:8])
	case ConflictTypeAdded:
		return fmt.Sprintf("Added: %s (actual: %s)", fc.Path, fc.ActualHash[:8])
	default:
		return fmt.Sprintf("Unknown conflict: %s", fc.Path)
	}
}

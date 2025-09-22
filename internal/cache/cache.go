package cache

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Manager handles the global cache directory for repositories
type Manager struct {
	cacheDir string
}

// NewManager creates a new cache manager
func NewManager() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}
	
	cacheDir := filepath.Join(homeDir, ".cache", "cherry-go", "repos")
	
	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}
	
	return &Manager{
		cacheDir: cacheDir,
	}, nil
}

// GetCacheDir returns the cache directory path
func (m *Manager) GetCacheDir() string {
	return m.cacheDir
}

// GetRepositoryPath returns the path where a repository should be cached
func (m *Manager) GetRepositoryPath(repoURL string) string {
	// Create a safe directory name from the repository URL
	repoHash := m.hashRepositoryURL(repoURL)
	repoName := m.extractRepositoryName(repoURL)
	
	// Combine name and hash for uniqueness
	dirName := fmt.Sprintf("%s-%s", repoName, repoHash[:8])
	
	return filepath.Join(m.cacheDir, dirName)
}

// hashRepositoryURL creates a hash from the repository URL for uniqueness
func (m *Manager) hashRepositoryURL(repoURL string) string {
	hasher := sha256.New()
	hasher.Write([]byte(repoURL))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// extractRepositoryName extracts a clean repository name from URL
func (m *Manager) extractRepositoryName(repoURL string) string {
	// Remove protocol
	name := repoURL
	if strings.HasPrefix(name, "https://") {
		name = strings.TrimPrefix(name, "https://")
	}
	if strings.HasPrefix(name, "http://") {
		name = strings.TrimPrefix(name, "http://")
	}
	if strings.HasPrefix(name, "git@") {
		name = strings.TrimPrefix(name, "git@")
		name = strings.Replace(name, ":", "/", 1)
	}
	
	// Remove .git suffix
	name = strings.TrimSuffix(name, ".git")
	
	// Replace special characters with dashes
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, ".", "-")
	
	// Limit length
	if len(name) > 50 {
		name = name[:50]
	}
	
	return name
}

// RepositoryExists checks if a repository is already cached
func (m *Manager) RepositoryExists(repoURL string) bool {
	repoPath := m.GetRepositoryPath(repoURL)
	gitDir := filepath.Join(repoPath, ".git")
	
	_, err := os.Stat(gitDir)
	return err == nil
}

// ListCachedRepositories returns a list of cached repositories
func (m *Manager) ListCachedRepositories() ([]CachedRepository, error) {
	entries, err := os.ReadDir(m.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []CachedRepository{}, nil
		}
		return nil, fmt.Errorf("failed to read cache directory: %w", err)
	}
	
	var repos []CachedRepository
	
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		repoPath := filepath.Join(m.cacheDir, entry.Name())
		gitDir := filepath.Join(repoPath, ".git")
		
		// Check if it's a valid git repository
		if _, err := os.Stat(gitDir); err == nil {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			
			repos = append(repos, CachedRepository{
				Name:         entry.Name(),
				Path:         repoPath,
				LastModified: info.ModTime(),
			})
		}
	}
	
	return repos, nil
}

// CleanCache removes old or unused cached repositories
func (m *Manager) CleanCache(maxAge int64) error {
	repos, err := m.ListCachedRepositories()
	if err != nil {
		return err
	}
	
	currentTime := time.Now().Unix()
	
	for _, repo := range repos {
		// Check if repository is older than maxAge days
		daysSinceModified := repo.LastModified.Unix()
		
		if (currentTime - daysSinceModified) > (maxAge * 24 * 60 * 60) {
			if err := os.RemoveAll(repo.Path); err != nil {
				return fmt.Errorf("failed to remove cached repository %s: %w", repo.Name, err)
			}
		}
	}
	
	return nil
}

// GetCacheSize returns the total size of the cache directory
func (m *Manager) GetCacheSize() (int64, error) {
	var size int64
	
	err := filepath.Walk(m.cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	
	return size, err
}

// CachedRepository represents a cached repository
type CachedRepository struct {
	Name         string
	Path         string
	LastModified time.Time
}

// String returns a string representation of the cached repository
func (cr CachedRepository) String() string {
	return fmt.Sprintf("%s (%s)", cr.Name, cr.LastModified.Format("2006-01-02 15:04:05"))
}

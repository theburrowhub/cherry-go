package git

import (
	"cherry-go/internal/cache"
	"cherry-go/internal/config"
	"cherry-go/internal/hash"
	"cherry-go/internal/logger"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// Repository represents a Git repository wrapper
type Repository struct {
	repo   *git.Repository
	path   string
	source *config.Source
}

// SyncResult represents the result of a sync operation
type SyncResult struct {
	SourceName   string
	UpdatedPaths []string
	CommitHash   string
	HasChanges   bool
	Conflicts    []hash.FileConflict
	Error        error
}

// NewRepository creates a new repository wrapper using global cache
func NewRepository(source *config.Source) (*Repository, error) {
	// Initialize cache manager
	cacheManager, err := cache.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache manager: %w", err)
	}
	
	// Get repository path in cache
	repoPath := cacheManager.GetRepositoryPath(source.Repository)
	
	var repo *git.Repository
	
	// Check if repository already exists in cache
	if cacheManager.RepositoryExists(source.Repository) {
		logger.Debug("Using cached repository: %s", repoPath)
		repo, err = git.PlainOpen(repoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open cached repository: %w", err)
		}
	} else {
		// Clone repository to cache
		logger.Info("Cloning repository %s to cache: %s", source.Repository, repoPath)
		repo, err = cloneRepository(source, repoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to clone repository: %w", err)
		}
	}
	
	return &Repository{
		repo:   repo,
		path:   repoPath,
		source: source,
	}, nil
}

// cloneRepository clones a repository with authentication (full clone for branch flexibility)
func cloneRepository(source *config.Source, repoPath string) (*git.Repository, error) {
	auth, err := getAuth(source.Auth, source.Repository)
	if err != nil {
		return nil, fmt.Errorf("failed to get authentication: %w", err)
	}
	
	cloneOptions := &git.CloneOptions{
		URL:  source.Repository,
		Auth: auth,
		// Don't specify SingleBranch or ReferenceName to get all branches
		// This allows us to checkout any branch/tag later
	}
	
	if logger.IsDryRun() {
		logger.DryRunInfo("Would clone repository %s to %s", source.Repository, repoPath)
		return nil, nil
	}
	
	return git.PlainClone(repoPath, false, cloneOptions)
}

// getAuth creates authentication based on config and repository URL
func getAuth(authConfig config.AuthConfig, repoURL string) (transport.AuthMethod, error) {
	// Handle SSH URLs specially (they don't parse well with url.Parse)
	if strings.HasPrefix(repoURL, "git@") {
		// SSH URL detected
		if authConfig.Type == "" || authConfig.Type == "auto" || authConfig.Type == "ssh" {
			return getSSHAuth(authConfig.SSHKey)
		}
	}
	
	// Parse repository URL to determine protocol for HTTPS URLs
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		// If parsing fails and it looks like SSH, try SSH auth
		if strings.Contains(repoURL, "@") && strings.Contains(repoURL, ":") {
			logger.Debug("URL parsing failed, assuming SSH format")
			return getSSHAuth(authConfig.SSHKey)
		}
		return nil, fmt.Errorf("failed to parse repository URL: %w", err)
	}
	
	// Auto-detect authentication method if not specified
	if authConfig.Type == "" || authConfig.Type == "auto" {
		return getAutoAuth(parsedURL)
	}
	
	switch authConfig.Type {
	case "ssh":
		return getSSHAuth(authConfig.SSHKey)
		
	case "basic":
		return getBasicAuth(authConfig.Username)
		
	default:
		return nil, nil // No authentication
	}
}

// getAutoAuth automatically detects and configures authentication
func getAutoAuth(parsedURL *url.URL) (transport.AuthMethod, error) {
	switch {
	case parsedURL.Scheme == "ssh" || strings.HasPrefix(parsedURL.String(), "git@"):
		// For SSH URLs, use SSH authentication
		logger.Debug("Auto-detecting SSH authentication for %s", parsedURL.Host)
		return getSSHAuth("")
		
	case parsedURL.Scheme == "https":
		// For HTTPS URLs, try token from environment first
		logger.Debug("Auto-detecting HTTPS authentication for %s", parsedURL.Host)
		auth, err := getHTTPSAuth()
		if err == nil && auth != nil {
			return auth, nil
		}
		
		// If no environment auth found, try without authentication for public repos
		logger.Debug("No HTTPS authentication found, trying without auth for public repository")
		return nil, nil
		
	default:
		logger.Debug("No authentication method auto-detected for %s", parsedURL.String())
		return nil, nil
	}
}

// getSSHAuth configures SSH authentication using SSH agent or specific key
func getSSHAuth(keyPath string) (transport.AuthMethod, error) {
	if keyPath != "" {
		// Use specific SSH key
		logger.Debug("Using SSH key: %s", keyPath)
		publicKeys, err := ssh.NewPublicKeysFromFile("git", keyPath, "")
		if err != nil {
			return nil, fmt.Errorf("failed to load SSH key %s: %w", keyPath, err)
		}
		return publicKeys, nil
	}
	
	// Try to use SSH agent
	logger.Debug("Using SSH agent authentication")
	sshAuth, err := ssh.NewSSHAgentAuth("git")
	if err != nil {
		logger.Debug("SSH agent not available: %v", err)
		// Fallback to default SSH key
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		
		defaultKeyPath := filepath.Join(homeDir, ".ssh", "id_rsa")
		if _, err := os.Stat(defaultKeyPath); err == nil {
			logger.Debug("Falling back to default SSH key: %s", defaultKeyPath)
			publicKeys, err := ssh.NewPublicKeysFromFile("git", defaultKeyPath, "")
			if err != nil {
				return nil, fmt.Errorf("failed to load default SSH key: %w", err)
			}
			return publicKeys, nil
		}
		
		return nil, fmt.Errorf("no SSH authentication method available")
	}
	
	return sshAuth, nil
}

// getHTTPSAuth configures HTTPS authentication using environment variables
func getHTTPSAuth() (transport.AuthMethod, error) {
	// Try GitHub token from environment
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		logger.Debug("Using GitHub token from environment")
		return &http.BasicAuth{
			Username: "token",
			Password: token,
		}, nil
	}
	
	// Try GitLab token from environment
	if token := os.Getenv("GITLAB_TOKEN"); token != "" {
		logger.Debug("Using GitLab token from environment")
		return &http.BasicAuth{
			Username: "oauth2",
			Password: token,
		}, nil
	}
	
	// Try generic Git token from environment
	if token := os.Getenv("GIT_TOKEN"); token != "" {
		logger.Debug("Using generic Git token from environment")
		return &http.BasicAuth{
			Username: "token",
			Password: token,
		}, nil
	}
	
	// Try Git credentials from environment
	if username := os.Getenv("GIT_USERNAME"); username != "" {
		if password := os.Getenv("GIT_PASSWORD"); password != "" {
			logger.Debug("Using Git credentials from environment")
			return &http.BasicAuth{
				Username: username,
				Password: password,
			}, nil
		}
	}
	
	logger.Debug("No HTTPS authentication found in environment variables")
	return nil, nil
}

// getBasicAuth configures basic authentication using environment variables
func getBasicAuth(username string) (transport.AuthMethod, error) {
	if username == "" {
		username = os.Getenv("GIT_USERNAME")
		if username == "" {
			return nil, fmt.Errorf("username is required for basic authentication")
		}
	}
	
	password := os.Getenv("GIT_PASSWORD")
	if password == "" {
		return nil, fmt.Errorf("password is required for basic authentication (set GIT_PASSWORD environment variable)")
	}
	
	logger.Debug("Using basic authentication for user: %s", username)
	return &http.BasicAuth{
		Username: username,
		Password: password,
	}, nil
}

// Pull fetches the latest changes from remote
func (r *Repository) Pull() error {
	if logger.IsDryRun() {
		logger.DryRunInfo("Would pull latest changes for %s", r.source.Name)
		return nil
	}

	workTree, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	auth, err := getAuth(r.source.Auth, r.source.Repository)
	if err != nil {
		return fmt.Errorf("failed to get authentication: %w", err)
	}

	pullOptions := &git.PullOptions{
		Auth: auth,
	}

	err = workTree.Pull(pullOptions)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to pull: %w", err)
	}

	return nil
}

// GetLatestCommit returns the latest commit hash
func (r *Repository) GetLatestCommit() (string, error) {
	ref, err := r.repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	return ref.Hash().String(), nil
}

// CopyPaths copies specified paths from the repository to local directory
func (r *Repository) CopyPaths(force bool) ([]string, []hash.FileConflict, error) {
	var updatedPaths []string
	var allConflicts []hash.FileConflict
	hasher := hash.NewFileHasher()
	
	for i, pathSpec := range r.source.Paths {
		// Checkout the specific branch/tag for this path
		if err := r.checkoutBranch(pathSpec.Branch); err != nil {
			logger.Error("Failed to checkout branch '%s' for %s: %v", pathSpec.Branch, pathSpec.Include, err)
			continue
		}
		// Determine local path - use specified path or default to same as source
		localPath := pathSpec.LocalPath
		if localPath == "" {
			localPath = pathSpec.Include
		}
		
		sourcePath := filepath.Join(r.path, pathSpec.Include)
		
		// Check if source path exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			logger.Error("Source path does not exist: %s", sourcePath)
			continue
		}
		
		// For directories, we need to check conflicts in the target directory
		// For files, we check the specific file location
		var conflictCheckPath string
		srcInfo, err := os.Stat(sourcePath)
		if err != nil {
			logger.Error("Failed to stat source path %s: %v", sourcePath, err)
			continue
		}
		
		if srcInfo.IsDir() {
			conflictCheckPath = localPath
		} else {
			conflictCheckPath = filepath.Dir(localPath)
		}
		
		// Check for conflicts with existing files
		if pathSpec.Files != nil && len(pathSpec.Files) > 0 {
			conflicts, err := hasher.VerifyFileIntegrity(conflictCheckPath, pathSpec.Files)
			if err != nil {
				logger.Error("Failed to verify file integrity for %s: %v", pathSpec.Include, err)
			} else if len(conflicts) > 0 {
				logger.Error("⚠️  Conflicts detected in %s:", pathSpec.Include)
				for _, conflict := range conflicts {
					logger.Error("  - %s", conflict.String())
					allConflicts = append(allConflicts, conflict)
				}
				
				if !force && !logger.IsDryRun() {
					logger.Error("Skipping %s due to conflicts. Use --force to override or resolve conflicts manually.", pathSpec.Include)
					continue
				} else if force {
					logger.Info("🔧 Force mode: Overriding local changes in %s", pathSpec.Include)
				}
			}
		}
		
		// Copy files/directories
		if err := copyPath(sourcePath, localPath, pathSpec.Exclude); err != nil {
			logger.Error("Failed to copy %s: %v", pathSpec.Include, err)
			continue
		}
		
		// Calculate new hashes for tracking
		var newHashes map[string]string
		
		if srcInfo.IsDir() {
			newHashes, err = hasher.HashDirectory(sourcePath, pathSpec.Exclude)
		} else {
			hash, hashErr := hasher.HashFile(sourcePath)
			if hashErr == nil {
				newHashes = map[string]string{
					filepath.Base(sourcePath): hash,
				}
			} else {
				err = hashErr
			}
		}
		
		if err != nil {
			logger.Error("Failed to calculate hashes for %s: %v", pathSpec.Include, err)
		} else {
			// Update path spec with new hashes
			r.source.Paths[i].Files = newHashes
			logger.Debug("Updated hashes for %s: %d files tracked", pathSpec.Include, len(newHashes))
		}
		
		updatedPaths = append(updatedPaths, pathSpec.Include)
		logger.Info("Copied %s to %s", pathSpec.Include, localPath)
	}
	
	return updatedPaths, allConflicts, nil
}

// checkoutBranch checks out a specific branch or tag
func (r *Repository) checkoutBranch(branch string) error {
	if branch == "" {
		// Try to detect default branch
		branch = r.detectDefaultBranch()
	}
	
	if logger.IsDryRun() {
		logger.DryRunInfo("Would checkout branch/tag: %s", branch)
		return nil
	}
	
	workTree, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}
	
	// Try to checkout as branch first
	branchRef := plumbing.ReferenceName("refs/heads/" + branch)
	err = workTree.Checkout(&git.CheckoutOptions{
		Branch: branchRef,
	})
	
	if err != nil {
		// If branch checkout fails, try as tag
		tagRef := plumbing.ReferenceName("refs/tags/" + branch)
		err = workTree.Checkout(&git.CheckoutOptions{
			Branch: tagRef,
		})
		
		if err != nil {
			// If both fail, try to resolve as a commit hash
			hash := plumbing.NewHash(branch)
			if hash.IsZero() {
				return fmt.Errorf("failed to checkout '%s': not a valid branch, tag, or commit", branch)
			}
			
			err = workTree.Checkout(&git.CheckoutOptions{
				Hash: hash,
			})
			
			if err != nil {
				return fmt.Errorf("failed to checkout '%s': %w", branch, err)
			}
		}
	}
	
	logger.Debug("Checked out branch/tag: %s", branch)
	return nil
}

// detectDefaultBranch tries to detect the default branch of the repository
func (r *Repository) detectDefaultBranch() string {
	// Try common default branch names
	defaultBranches := []string{"main", "master", "develop", "dev"}
	
	for _, branch := range defaultBranches {
		branchRef := plumbing.ReferenceName("refs/heads/" + branch)
		_, err := r.repo.Reference(branchRef, true)
		if err == nil {
			logger.Debug("Detected default branch: %s", branch)
			return branch
		}
	}
	
	// If no common branch found, try to get HEAD reference
	head, err := r.repo.Head()
	if err == nil {
		branchName := head.Name().Short()
		logger.Debug("Using HEAD branch: %s", branchName)
		return branchName
	}
	
	// Final fallback
	logger.Debug("Could not detect default branch, using 'main'")
	return "main"
}

// copyPath copies a file or directory from source to destination
func copyPath(src, dst string, excludes []string) error {
	if logger.IsDryRun() {
		logger.DryRunInfo("Would copy %s to %s", src, dst)
		return nil
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if srcInfo.IsDir() {
		return copyDir(src, dst, excludes)
	}
	return copyFile(src, dst)
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	srcData, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, srcData, 0644)
}

// copyDir recursively copies a directory
func copyDir(src, dst string, excludes []string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		// Check if path should be excluded
		if shouldExclude(entry.Name(), excludes) {
			logger.Debug("Excluding %s", entry.Name())
			continue
		}

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath, excludes); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// shouldExclude checks if a path should be excluded based on patterns
func shouldExclude(path string, excludes []string) bool {
	for _, exclude := range excludes {
		if matched, _ := filepath.Match(exclude, path); matched {
			return true
		}
		if strings.Contains(path, exclude) {
			return true
		}
	}
	return false
}

// CreateCommit creates a commit with the updated files
func CreateCommit(workDir string, message string, updatedPaths []string) error {
	if logger.IsDryRun() {
		logger.DryRunInfo("Would create commit with message: %s", message)
		logger.DryRunInfo("Updated paths: %v", updatedPaths)
		return nil
	}

	repo, err := git.PlainOpen(workDir)
	if err != nil {
		return fmt.Errorf("failed to open local repository: %w", err)
	}

	workTree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Add all updated paths
	for _, path := range updatedPaths {
		if _, err := workTree.Add(path); err != nil {
			logger.Error("Failed to add %s: %v", path, err)
		}
	}

	// Create commit
	commit, err := workTree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "cherry-go",
			Email: "cherry-go@local",
			When:  time.Now(),
		},
	})

	if err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	logger.Info("Created commit: %s", commit.String())
	return nil
}

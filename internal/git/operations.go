package git

import (
	"cherry-go/internal/cache"
	"cherry-go/internal/config"
	"cherry-go/internal/hash"
	"cherry-go/internal/logger"
	"cherry-go/internal/merge"
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

// SyncMode defines the synchronization mode
type SyncMode int

const (
	SyncModeDetect SyncMode = iota // Default: only detect and report conflicts, no changes
	SyncModeMerge                  // Attempt three-way merge
	SyncModeForce                  // Force overwrite local changes
	SyncModeBranch                 // Create branch on conflict for manual resolution (used with merge)
)

// Repository represents a Git repository wrapper
type Repository struct {
	repo   *git.Repository
	path   string
	source *config.Source
}

// SyncResult represents the result of a sync operation
type SyncResult struct {
	SourceName        string
	UpdatedPaths      []string
	CommitHash        string
	HasChanges        bool
	Conflicts         []hash.FileConflict
	BranchCreated     string // Name of conflict branch if created
	MergeInstructions string // Instructions for manual merge
	Error             error
}

// CopyResult represents the result of copying paths
type CopyResult struct {
	UpdatedPaths      []string
	Conflicts         []hash.FileConflict
	BranchCreated     string
	MergeInstructions string
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
// mode: SyncModeMerge (default), SyncModeForce, or SyncModeBranch
// workDir: the local working directory (for branch creation)
func (r *Repository) CopyPaths(mode SyncMode, workDir string) (*CopyResult, error) {
	result := &CopyResult{}
	hasher := hash.NewFileHasher()

	// Initialize base content manager for three-way merge
	baseManager, err := cache.NewBaseContentManager()
	if err != nil {
		logger.Debug("Failed to initialize base content manager: %v", err)
		// Continue without base content - will fall back to hash-based conflict detection
	}

	// Collect files for potential branch creation
	var conflictFiles map[string][]byte

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

		srcInfo, err := os.Stat(sourcePath)
		if err != nil {
			logger.Error("Failed to stat source path %s: %v", sourcePath, err)
			continue
		}

		// Process based on mode
		pathResult, pathConflicts := r.processPath(processPathInput{
			pathSpec:    pathSpec,
			sourcePath:  sourcePath,
			localPath:   localPath,
			srcInfo:     srcInfo,
			mode:        mode,
			hasher:      hasher,
			baseManager: baseManager,
			workDir:     workDir,
		})

		if len(pathConflicts) > 0 {
			result.Conflicts = append(result.Conflicts, pathConflicts...)

			// Collect conflict files for branch creation
			if mode == SyncModeBranch {
				if conflictFiles == nil {
					conflictFiles = make(map[string][]byte)
				}
				// Read remote files for branch
				remoteFiles := r.readRemoteFiles(sourcePath, localPath, srcInfo.IsDir(), pathSpec.Exclude)
				for k, v := range remoteFiles {
					conflictFiles[k] = v
				}
			}
		}

		if pathResult.updated {
			result.UpdatedPaths = append(result.UpdatedPaths, pathSpec.Include)

			// Update hashes in path spec
			r.source.Paths[i].Files = pathResult.newHashes

			// Save base content for future merges
			if baseManager != nil && !logger.IsDryRun() {
				baseContent := r.readRemoteFiles(sourcePath, localPath, srcInfo.IsDir(), pathSpec.Exclude)
				if err := baseManager.SaveSnapshot(r.source.Name, pathSpec.Include, baseContent); err != nil {
					logger.Debug("Failed to save base content snapshot: %v", err)
				}
			}

			logger.Info("Synced %s to %s", pathSpec.Include, localPath)
		}
	}

	// Create conflict branch if needed
	if mode == SyncModeBranch && len(result.Conflicts) > 0 && conflictFiles != nil && len(conflictFiles) > 0 {
		branchPrefix := "cherry-go/sync"
		if r.source.Name != "" {
			branchPrefix = "cherry-go/sync"
		}

		branchResult, err := CreateConflictBranch(workDir, branchPrefix, r.source.Name, conflictFiles)
		if err != nil {
			logger.Error("Failed to create conflict branch: %v", err)
		} else {
			result.BranchCreated = branchResult.BranchName
			result.MergeInstructions = GetMergeInstructions(branchResult)
		}
	}

	return result, nil
}

// processPathInput contains input parameters for processPath
type processPathInput struct {
	pathSpec    config.PathSpec
	sourcePath  string
	localPath   string
	srcInfo     os.FileInfo
	mode        SyncMode
	hasher      *hash.FileHasher
	baseManager *cache.BaseContentManager
	workDir     string
}

// processPathResult contains the result of processing a path
type processPathResult struct {
	updated   bool
	newHashes map[string]string
}

// processPath processes a single path spec according to the sync mode
func (r *Repository) processPath(input processPathInput) (processPathResult, []hash.FileConflict) {
	result := processPathResult{}
	var conflicts []hash.FileConflict

	// Check if local content differs from remote
	localDiffersFromRemote := r.contentDiffersFromRemote(input)

	// If local and remote are identical, nothing to do
	if !localDiffersFromRemote {
		result.newHashes = r.calculateHashes(input.sourcePath, input.srcInfo.IsDir(), input.hasher, input.pathSpec.Exclude)
		result.updated = false
		return result, conflicts
	}

	// Local differs from remote - handle based on mode
	switch input.mode {
	case SyncModeDetect:
		// Detect mode - NEVER copy, only report differences
		logger.Warning("⚠️  Differences detected in %s", input.pathSpec.Include)
		r.showConflictDiff(input)
		conflicts = r.getFileConflicts(input)

	case SyncModeForce:
		// Force mode - overwrite
		logger.Info("🔧 Force mode: Overriding local changes in %s", input.pathSpec.Include)
		if err := copyPath(input.sourcePath, input.localPath, input.pathSpec.Exclude); err != nil {
			logger.Error("Failed to copy %s: %v", input.pathSpec.Include, err)
			return result, conflicts
		}
		result.newHashes = r.calculateHashes(input.sourcePath, input.srcInfo.IsDir(), input.hasher, input.pathSpec.Exclude)
		result.updated = true

	case SyncModeMerge, SyncModeBranch:
		// Try three-way merge (preserves local changes when possible)
		mergeResult, mergeConflicts := r.attemptMerge(input)

		if len(mergeConflicts) > 0 {
			conflicts = mergeConflicts
			if input.mode == SyncModeMerge {
				// Show the conflict diff
				r.showConflictDiff(input)
				logger.Error("⚠️  Merge conflicts in %s - cannot auto-merge", input.pathSpec.Include)
				logger.Info("💡 Use --branch-on-conflict to create a branch for manual resolution:")
				logger.Info("   cherry-go sync --merge --branch-on-conflict")
			}
		} else if mergeResult.updated {
			result = mergeResult
			logger.Info("✓ Merged %s (local changes preserved)", input.pathSpec.Include)
		}
	}

	return result, conflicts
}

// contentDiffersFromRemote checks if local content differs from remote content
func (r *Repository) contentDiffersFromRemote(input processPathInput) bool {
	if input.srcInfo.IsDir() {
		// For directories, check each file
		differs := false
		filepath.Walk(input.sourcePath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			relPath, _ := filepath.Rel(input.sourcePath, path)
			localPath := filepath.Join(input.localPath, relPath)

			localContent, err := os.ReadFile(localPath)
			if err != nil {
				// Local doesn't exist - differs
				differs = true
				return filepath.SkipAll
			}

			remoteContent, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			if string(localContent) != string(remoteContent) {
				differs = true
				return filepath.SkipAll
			}
			return nil
		})
		return differs
	}

	// For single file
	localContent, err := os.ReadFile(input.localPath)
	if err != nil {
		// Local doesn't exist - differs
		return true
	}

	remoteContent, err := os.ReadFile(input.sourcePath)
	if err != nil {
		return false
	}

	return string(localContent) != string(remoteContent)
}

// showConflictDiff shows the diff between local and remote for conflict detection
func (r *Repository) showConflictDiff(input processPathInput) {
	if input.srcInfo.IsDir() {
		// For directories, show diff for each modified file
		filepath.Walk(input.sourcePath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			relPath, _ := filepath.Rel(input.sourcePath, path)
			localPath := filepath.Join(input.localPath, relPath)

			if _, err := os.Stat(localPath); err == nil {
				// Local file exists, check if different
				localContent, _ := os.ReadFile(localPath)
				remoteContent, _ := os.ReadFile(path)
				if string(localContent) != string(remoteContent) {
					merge.ShowDiffFromContent(localContent, remoteContent, relPath)
				}
			}
			return nil
		})
	} else {
		// For single file
		localContent, err := os.ReadFile(input.localPath)
		if err != nil {
			return
		}
		remoteContent, err := os.ReadFile(input.sourcePath)
		if err != nil {
			return
		}
		if string(localContent) != string(remoteContent) {
			merge.ShowDiffFromContent(localContent, remoteContent, filepath.Base(input.localPath))
		}
	}
}

// getFileConflicts returns file conflicts for the given path
func (r *Repository) getFileConflicts(input processPathInput) []hash.FileConflict {
	var conflicts []hash.FileConflict

	if input.srcInfo.IsDir() {
		filepath.Walk(input.sourcePath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			relPath, _ := filepath.Rel(input.sourcePath, path)
			localPath := filepath.Join(input.localPath, relPath)

			if _, err := os.Stat(localPath); err == nil {
				localContent, _ := os.ReadFile(localPath)
				remoteContent, _ := os.ReadFile(path)
				if string(localContent) != string(remoteContent) {
					conflicts = append(conflicts, hash.FileConflict{
						Path: relPath,
						Type: hash.ConflictTypeModified,
					})
				}
			}
			return nil
		})
	} else {
		conflicts = append(conflicts, hash.FileConflict{
			Path: filepath.Base(input.localPath),
			Type: hash.ConflictTypeModified,
		})
	}

	return conflicts
}

// hasLocalChanges checks if there are local changes in the given path
func (r *Repository) hasLocalChanges(pathSpec config.PathSpec, localPath string, hasher *hash.FileHasher, isDir bool) bool {
	if pathSpec.Files == nil || len(pathSpec.Files) == 0 {
		// No previous hashes - first sync
		return false
	}

	var conflictCheckPath string
	if isDir {
		conflictCheckPath = localPath
	} else {
		conflictCheckPath = filepath.Dir(localPath)
	}

	conflicts, err := hasher.VerifyFileIntegrity(conflictCheckPath, pathSpec.Files)
	if err != nil {
		logger.Debug("Failed to verify file integrity: %v", err)
		return false
	}

	return len(conflicts) > 0
}

// attemptMerge attempts a three-way merge for the given path
func (r *Repository) attemptMerge(input processPathInput) (processPathResult, []hash.FileConflict) {
	result := processPathResult{}
	var conflicts []hash.FileConflict

	// Check if we have base content for three-way merge
	if input.baseManager == nil || !input.baseManager.HasSnapshot(r.source.Name, input.pathSpec.Include) {
		// No base content - fall back to hash-based conflict detection
		logger.Debug("No base content available for %s, falling back to conflict detection", input.pathSpec.Include)

		// Report as conflict since we can't do proper merge
		var conflictCheckPath string
		if input.srcInfo.IsDir() {
			conflictCheckPath = input.localPath
		} else {
			conflictCheckPath = filepath.Dir(input.localPath)
		}

		hashConflicts, _ := input.hasher.VerifyFileIntegrity(conflictCheckPath, input.pathSpec.Files)
		for _, c := range hashConflicts {
			conflicts = append(conflicts, c)
			logger.Error("  - %s (no base content for merge)", c.Path)
		}
		return result, conflicts
	}

	// Get base content
	baseContent, err := input.baseManager.GetSnapshot(r.source.Name, input.pathSpec.Include)
	if err != nil {
		logger.Error("Failed to get base content: %v", err)
		return result, conflicts
	}

	// Perform merge
	if input.srcInfo.IsDir() {
		result, conflicts = r.mergeDirectory(input, baseContent)
	} else {
		result, conflicts = r.mergeFile(input, baseContent)
	}

	return result, conflicts
}

// mergeDirectory attempts to merge a directory
func (r *Repository) mergeDirectory(input processPathInput, baseContent map[string][]byte) (processPathResult, []hash.FileConflict) {
	result := processPathResult{newHashes: make(map[string]string)}
	var conflicts []hash.FileConflict

	// Get list of files to process
	var files []string
	err := filepath.Walk(input.sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		relPath, _ := filepath.Rel(input.sourcePath, path)
		if !shouldExclude(relPath, input.pathSpec.Exclude) {
			files = append(files, relPath)
		}
		return nil
	})
	if err != nil {
		logger.Error("Failed to walk source directory: %v", err)
		return result, conflicts
	}

	allMerged := true
	for _, relPath := range files {
		remotePath := filepath.Join(input.sourcePath, relPath)
		localPath := filepath.Join(input.localPath, relPath)

		// Read remote content
		remoteContent, err := os.ReadFile(remotePath)
		if err != nil {
			logger.Error("Failed to read remote file %s: %v", relPath, err)
			continue
		}

		// Check if local file exists
		localContent, localErr := os.ReadFile(localPath)
		if localErr != nil {
			// Local file doesn't exist - just copy
			if err := os.MkdirAll(filepath.Dir(localPath), 0755); err == nil {
				if err := os.WriteFile(localPath, remoteContent, 0644); err != nil {
					logger.Error("Failed to write file %s: %v", relPath, err)
				}
			}
			result.newHashes[relPath] = input.hasher.HashBytes(remoteContent)
			continue
		}

		// Get base content for this file
		base, hasBase := baseContent[relPath]

		if !hasBase {
			// No base - check if files are identical
			if string(localContent) == string(remoteContent) {
				result.newHashes[relPath] = input.hasher.HashBytes(remoteContent)
				continue
			}
			// Files differ and no base - conflict
			logger.Error("  - %s (no base content for merge)", relPath)
			merge.ShowDiffFromContent(localContent, remoteContent, relPath)
			conflicts = append(conflicts, hash.FileConflict{
				Path: relPath,
				Type: hash.ConflictTypeModified,
			})
			allMerged = false
			continue
		}

		// Check if local is unchanged from base
		if string(localContent) == string(base) {
			// Local unchanged - just take remote
			if err := os.WriteFile(localPath, remoteContent, 0644); err != nil {
				logger.Error("Failed to write file %s: %v", relPath, err)
			}
			result.newHashes[relPath] = input.hasher.HashBytes(remoteContent)
			continue
		}

		// Check if remote is unchanged from base
		if string(remoteContent) == string(base) {
			// Remote unchanged - keep local
			result.newHashes[relPath] = input.hasher.HashBytes(localContent)
			continue
		}

		// Both changed - attempt three-way merge
		mergeResult, err := merge.ThreeWayMerge(base, localContent, remoteContent)
		if err != nil {
			logger.Error("Failed to merge %s: %v", relPath, err)
			conflicts = append(conflicts, hash.FileConflict{
				Path: relPath,
				Type: hash.ConflictTypeModified,
			})
			allMerged = false
			continue
		}

		if mergeResult.HasConflict {
			logger.Error("  - %s (merge conflict - both local and remote modified)", relPath)
			merge.ShowDiffFromContent(localContent, remoteContent, relPath)
			conflicts = append(conflicts, hash.FileConflict{
				Path: relPath,
				Type: hash.ConflictTypeModified,
			})
			allMerged = false
			continue
		}

		// Merge successful - write result
		if !logger.IsDryRun() {
			if err := os.WriteFile(localPath, mergeResult.Content, 0644); err != nil {
				logger.Error("Failed to write merged file %s: %v", relPath, err)
				continue
			}
		}
		logger.Info("  ✓ Merged %s successfully", relPath)
		result.newHashes[relPath] = input.hasher.HashBytes(mergeResult.Content)
	}

	result.updated = allMerged && len(conflicts) == 0
	return result, conflicts
}

// mergeFile attempts to merge a single file
func (r *Repository) mergeFile(input processPathInput, baseContent map[string][]byte) (processPathResult, []hash.FileConflict) {
	result := processPathResult{newHashes: make(map[string]string)}
	var conflicts []hash.FileConflict

	fileName := filepath.Base(input.sourcePath)

	// Read remote content
	remoteContent, err := os.ReadFile(input.sourcePath)
	if err != nil {
		logger.Error("Failed to read remote file: %v", err)
		return result, conflicts
	}

	// Read local content
	localContent, err := os.ReadFile(input.localPath)
	if err != nil {
		// Local doesn't exist - just copy
		if err := copyPath(input.sourcePath, input.localPath, nil); err != nil {
			logger.Error("Failed to copy file: %v", err)
		}
		result.newHashes[fileName] = input.hasher.HashBytes(remoteContent)
		result.updated = true
		return result, conflicts
	}

	// Get base content
	base, hasBase := baseContent[fileName]
	if !hasBase {
		// Check if files are identical
		if string(localContent) == string(remoteContent) {
			result.newHashes[fileName] = input.hasher.HashBytes(remoteContent)
			result.updated = true
			return result, conflicts
		}
		// Conflict - no base for merge
		logger.Error("  - %s (no base content for merge)", fileName)
		merge.ShowDiffFromContent(localContent, remoteContent, fileName)
		conflicts = append(conflicts, hash.FileConflict{
			Path: fileName,
			Type: hash.ConflictTypeModified,
		})
		return result, conflicts
	}

	// Check if local unchanged
	if string(localContent) == string(base) {
		if err := os.WriteFile(input.localPath, remoteContent, 0644); err != nil {
			logger.Error("Failed to write file: %v", err)
		}
		result.newHashes[fileName] = input.hasher.HashBytes(remoteContent)
		result.updated = true
		return result, conflicts
	}

	// Check if remote unchanged
	if string(remoteContent) == string(base) {
		result.newHashes[fileName] = input.hasher.HashBytes(localContent)
		result.updated = true
		return result, conflicts
	}

	// Both changed - attempt merge
	mergeResult, err := merge.ThreeWayMerge(base, localContent, remoteContent)
	if err != nil {
		logger.Error("Failed to merge: %v", err)
		conflicts = append(conflicts, hash.FileConflict{
			Path: fileName,
			Type: hash.ConflictTypeModified,
		})
		return result, conflicts
	}

	if mergeResult.HasConflict {
		logger.Error("  - %s (merge conflict - both local and remote modified)", fileName)
		merge.ShowDiffFromContent(localContent, remoteContent, fileName)
		conflicts = append(conflicts, hash.FileConflict{
			Path: fileName,
			Type: hash.ConflictTypeModified,
		})
		return result, conflicts
	}

	// Merge successful
	if !logger.IsDryRun() {
		if err := os.WriteFile(input.localPath, mergeResult.Content, 0644); err != nil {
			logger.Error("Failed to write merged file: %v", err)
			return result, conflicts
		}
	}
	logger.Info("  ✓ Merged %s successfully", fileName)
	result.newHashes[fileName] = input.hasher.HashBytes(mergeResult.Content)
	result.updated = true
	return result, conflicts
}

// calculateHashes calculates hashes for files in the given path
func (r *Repository) calculateHashes(sourcePath string, isDir bool, hasher *hash.FileHasher, excludes []string) map[string]string {
	var newHashes map[string]string
	var err error

	if isDir {
		newHashes, err = hasher.HashDirectory(sourcePath, excludes)
	} else {
		h, hashErr := hasher.HashFile(sourcePath)
		if hashErr == nil {
			newHashes = map[string]string{
				filepath.Base(sourcePath): h,
			}
		} else {
			err = hashErr
		}
	}

	if err != nil {
		logger.Error("Failed to calculate hashes: %v", err)
		return nil
	}
	return newHashes
}

// readRemoteFiles reads all files from the remote path into a map
func (r *Repository) readRemoteFiles(sourcePath, localPath string, isDir bool, excludes []string) map[string][]byte {
	files := make(map[string][]byte)

	if !isDir {
		content, err := os.ReadFile(sourcePath)
		if err == nil {
			// Use localPath for the key to match where it will be written
			files[localPath] = content
		}
		return files
	}

	filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		relPath, _ := filepath.Rel(sourcePath, path)
		if shouldExclude(relPath, excludes) {
			return nil
		}
		content, err := os.ReadFile(path)
		if err == nil {
			// Use the full local path for branch creation
			fullLocalPath := filepath.Join(localPath, relPath)
			files[fullLocalPath] = content
		}
		return nil
	})

	return files
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

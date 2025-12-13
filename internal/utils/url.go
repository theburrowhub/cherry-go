package utils

import (
	"strings"
)

// ExtractRepoName extracts a repository name from a Git URL
func ExtractRepoName(repoURL string) string {
	// Remove protocol prefixes
	url := repoURL
	if strings.HasPrefix(url, "https://") {
		url = strings.TrimPrefix(url, "https://")
	}
	if strings.HasPrefix(url, "http://") {
		url = strings.TrimPrefix(url, "http://")
	}
	if strings.HasPrefix(url, "git@") {
		url = strings.TrimPrefix(url, "git@")
		// Convert git@host:owner/repo.git to host/owner/repo.git
		url = strings.Replace(url, ":", "/", 1)
	}

	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Split by / and get the last part (repository name)
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return "repo"
}

// ParseURLPath parses a URL path in the format: repo-url/path or just path
func ParseURLPath(urlPath string) (repoURL string, filePath string) {
	// Check if it contains a full URL
	if strings.Contains(urlPath, "://") || strings.HasPrefix(urlPath, "git@") {
		// Find where the repository URL ends and the path begins
		// Look for .git/ separator first
		if strings.Contains(urlPath, ".git/") {
			parts := strings.SplitN(urlPath, ".git/", 2)
			if len(parts) == 2 {
				return parts[0] + ".git", parts[1]
			}
		}

		// Handle URLs ending with .git but no path after
		if strings.HasSuffix(urlPath, ".git") {
			return urlPath, ""
		}

		// For URLs without .git in the middle, try to find the path separator
		if strings.HasPrefix(urlPath, "git@") {
			// Format: git@host:owner/repo/path/to/file
			colonIndex := strings.Index(urlPath, ":")
			if colonIndex != -1 {
				afterColon := urlPath[colonIndex+1:]
				slashCount := 0
				var repoEnd int

				for i, char := range afterColon {
					if char == '/' {
						slashCount++
						if slashCount == 2 { // After owner/repo
							repoEnd = colonIndex + 1 + i
							break
						}
					}
				}

				if repoEnd > 0 && repoEnd < len(urlPath)-1 {
					// Add .git if not present
					repoURL = urlPath[:repoEnd]
					if !strings.HasSuffix(repoURL, ".git") {
						repoURL += ".git"
					}
					return repoURL, urlPath[repoEnd+1:]
				}
			}
		} else {
			// HTTPS format: https://host/owner/repo/path/to/file
			parts := strings.Split(urlPath, "/")
			if len(parts) >= 5 { // https, "", host, owner, repo, ...
				repoURL = strings.Join(parts[:5], "/")
				if !strings.HasSuffix(repoURL, ".git") {
					repoURL += ".git"
				}
				if len(parts) > 5 {
					filePath = strings.Join(parts[5:], "/")
				}
				return repoURL, filePath
			}
		}

		// If we can't parse the path, assume the whole thing is a repository URL
		return urlPath, ""
	}

	// If no URL detected, assume it's just a file path
	return "", urlPath
}

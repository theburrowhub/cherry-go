## [0.1.3] - 2025-09-22

- fix: update GitHub Actions to latest versions

## [0.1.2] - 2025-09-22

- fix: build binaries
- Merge branch 'main' of github.com:theburrowhub/cherry-go
- docs: update README for release workflow, renaming section and clarifying actions
- ci: enhance release workflow with build and artifact management steps
- ci: remove obsolete build workflow from GitHub Actions

## [0.1.1] - 2025-09-22

- fix: add workflow_dispatch trigger to build workflow and improve version detection

## [0.1.0] - 2025-09-22

- feat: first commit
- ci: add GitHub Actions workflows for CI/CD including build, release, and PR checks
- chore: update GolangCI-Lint configuration to simplify settings and enable additional linters
- style: remove unnecessary whitespace in multiple files for cleaner code
- fix: correct 'grun' to 'run' in GolangCI-Lint configuration
- fix: enhance changelog generation logic for first and subsequent releases
- ci: update GitHub Actions permissions for pull request workflows
- ci: update GitHub Actions release workflow with additional permissions and credential persistence
- ci: add write permissions for contents and read permissions for actions in GitHub Actions workflow

# Changelog

All notable changes to cherry-go will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of cherry-go
- **Restructured CLI interface** with three-step workflow:
  - `add repo` - Add repository configurations (no branch specification)
  - `add file` - Add specific files to track (auto-synced on add, branch per file)
  - `add directory` - Add directories to track (auto-synced on add, branch per directory)  
- `remove`, `sync`, `status`, `init`, `version`, and `cache` commands
- Project-specific YAML configuration file support (`.cherry-go.yaml` in project directory)
- **`init` command** for generating initial configuration files
- **File integrity tracking** with SHA256 hashes for conflict detection
- **Conflict detection system** warns when local files have been modified
- **Force sync mode** to override local changes when needed
- **Flexible path management** - each file can be placed exactly where desired
- **Granular tracking control** - separate file and directory tracking
- **Repository-based organization** - configure repos once, track multiple paths
- **Branch/tag per path** - each file/directory can track different branches or tags
- **Global cache system** - repositories cached in ~/.cache/cherry-go/ for efficiency
- **Auto-sync on add** - files and directories sync automatically when added
- **Installation scripts** - automated local installation with backup support
- **Cache management** - commands to manage global repository cache
- Multiple authentication methods (SSH agent, environment variables)
- Concurrent synchronization using goroutines
- Dry-run mode for testing operations
- Path exclusion patterns for directories
- Auto-commit functionality
- Comprehensive logging system
- Cross-platform build support
- CI/CD integration examples
- Extensive documentation and usage examples

### Changed
- Configuration file location changed from global (`~/.cherry-go.yaml`) to project-specific (`./.cherry-go.yaml`)
- **Removed global `local_prefix`** - each path can now specify its own destination
- **Path specification changed** from `local_dir` to `local_path` for more precise control
- **Branch/tag moved from repository to path level** - more granular version control
- **Repository cache moved to global location** (`~/.cache/cherry-go/`) for efficiency
- **Security improvement**: Tokens and passwords are NO LONGER stored in configuration files
- **Authentication system redesigned** for better security and usability
- Each project now maintains its own independent configuration
- Default behavior: files are placed in same path as source unless `local_path` is specified

### Security Enhancements
- **SSH Agent Integration**: Automatic use of SSH agent for authentication
- **Environment Variable Support**: Tokens read from environment variables (GITHUB_TOKEN, GITLAB_TOKEN, etc.)
- **Auto-detection**: Automatically detects appropriate authentication method based on repository URL
- **No Secrets in Config**: Configuration files no longer contain sensitive authentication data

### Features
- **Selective File Tracking**: Choose specific files and directories from remote repositories
- **Authentication Support**: Token-based, SSH, and basic authentication for private repositories
- **Concurrent Operations**: Efficient syncing of multiple sources using goroutines and channels
- **Dry-run Mode**: Test operations without making actual changes
- **Flexible Configuration**: YAML-based configuration with CLI parameter overrides
- **Auto-commit**: Automatically commit synchronized changes with customizable messages
- **Path Management**: Custom local directories and exclusion patterns
- **Version Control Integration**: Maintains reference to source commits
- **Cross-platform**: Works on Linux, macOS, and Windows

### Documentation
- Comprehensive README with installation and usage instructions
- Detailed usage guide with examples
- CI/CD integration examples for GitHub Actions, GitLab CI, and Jenkins
- Use case documentation for vendor dependencies
- API documentation and code examples

### Testing
- Unit tests for configuration management
- Integration tests for Git operations
- File system operation tests
- Authentication mechanism tests

## [1.0.0] - TBD

### Added
- First stable release
- All core functionality implemented and tested
- Production-ready CLI tool
- Complete documentation suite

---

## Release Process

1. Update version in `cmd/version.go`
2. Update CHANGELOG.md
3. Create git tag: `git tag -a v1.0.0 -m "Release v1.0.0"`
4. Push tag: `git push origin v1.0.0`
5. Build release binaries: `./scripts/build.sh all`
6. Create GitHub release with binaries

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on how to contribute to this project.

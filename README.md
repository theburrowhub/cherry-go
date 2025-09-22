# Cherry-go

A command-line tool for partial versioning of files from other Git repositories. Cherry-go allows you to selectively sync specific files or directories from remote repositories into your local repository, keeping them synchronized when changes occur in the source.

## Features

- **Selective File Tracking**: Choose specific files and directories from remote repositories
- **Automatic Synchronization**: Keep tracked files up-to-date with source repositories
- **Conflict Detection**: Warns when local files have been modified before overwriting
- **File Integrity**: Tracks file hashes to detect local modifications
- **Private Repository Support**: Secure authentication for private repositories (token, SSH, basic auth)
- **Concurrent Operations**: Efficient syncing of multiple sources using goroutines
- **Dry-run Mode**: Test operations without making actual changes
- **Force Mode**: Override local changes when needed
- **Flexible Configuration**: YAML-based configuration with CLI overrides
- **Auto-commit**: Automatically commit synchronized changes
- **Path Exclusion**: Exclude specific files/patterns within tracked directories

## Installation

Cherry-go provides an intelligent installation script that automatically detects whether you're installing from a local repository or downloading from GitHub releases.

### Remote Installation (Recommended)

Install the latest release directly from GitHub:

```bash
curl -sSL https://raw.githubusercontent.com/theburrowhub/cherry-go/main/install.sh | bash
```

Or using wget:

```bash
wget -qO- https://raw.githubusercontent.com/theburrowhub/cherry-go/main/install.sh | bash
```

### Local Installation (Development)

If you have the source code locally:

```bash
git clone https://github.com/theburrowhub/cherry-go.git
cd cherry-go
./install.sh
```

### Installation Options

```bash
# Install to default location (~/.local/bin)
./install.sh

# Install to custom directory
./install.sh -d /usr/local/bin

# Force reinstallation without backup
./install.sh --force

# Force local build mode (requires Go)
./install.sh --local

# Force remote download mode
./install.sh --remote
```

### Platform Support

The installation script automatically detects your platform and downloads the appropriate binary:

- **Linux**: amd64, arm64
- **macOS**: amd64 (Intel), arm64 (Apple Silicon)

### Requirements

- **Remote installation**: No dependencies (downloads pre-built binaries)
- **Local installation**: Go 1.21 or later

### Using Makefile

```bash
# Quick local install
make install-quick

# Full local install with backup
make install-local

# Install to GOPATH/bin
make install
```

### From Source (Manual)

```bash
git clone https://github.com/theburrowhub/cherry-go.git
cd cherry-go
go build -o cherry-go
# Copy to your preferred location
cp cherry-go ~/.local/bin/
```

### Using Go Install

```bash
go install cherry-go@latest
```

### Uninstallation

```bash
# Remove local installation
./scripts/uninstall.sh

# Remove with backups
./scripts/uninstall.sh --remove-backups

# Using Makefile
make uninstall
```

## Quick Start

### Super Simple (One-liner approach)

```bash
# Initialize and add files in one go
cherry-go init
cherry-go add file https://github.com/user/library.git/src/main.go
cherry-go add directory https://github.com/user/library.git/docs/ --local-path docs/external/
```

### Step-by-step approach

1. **Initialize configuration**:
```bash
cherry-go init
```

2. **Add repository** (optional - auto-added when adding files):
```bash
cherry-go add repo https://github.com/user/library.git
```

3. **Add files or directories** (auto-synced):
```bash
# Add files with full URL (auto-detects repository)
cherry-go add file https://github.com/user/library.git/src/main.go

# Add from configured repository (if only one exists)
cherry-go add file src/utils.go --local-path internal/utils.go

# Add directories with custom options
cherry-go add directory https://github.com/user/library.git/docs/ --branch v1.2.0
```

4. **Check status and sync updates**:
```bash
# Check current status
cherry-go status

# Sync for updates (when needed)
cherry-go sync library
```

## Commands

### `init` - Initialize configuration

Initialize a new cherry-go configuration file in the current directory:

```bash
cherry-go init
```

This creates a `.cherry-go.yaml` file with default settings. If the file already exists, the command will fail to prevent overwriting existing configuration.

### `add` - Add repositories, files, or directories

The `add` command has three subcommands for a flexible workflow:

#### `add repo` - Add a repository configuration

Add a repository that can be used to track files and directories:

```bash
cherry-go add repo --name REPO_NAME --url REPOSITORY_URL
```

**Options**:
- `--name`: Repository name (required)
- `--url`: Repository URL (required)
- `--auth-type`: Authentication type (auto, ssh, basic) - defaults to "auto"
- `--auth-user`: Username for basic auth (password via GIT_PASSWORD env var)
- `--auth-ssh-key`: Path to SSH private key (optional - uses SSH agent by default)

**Note**: Branches and tags are specified when adding files/directories, not at the repository level.

**Examples**:

```bash
# Add public repository (name auto-detected as "library")
cherry-go add repo https://github.com/user/library.git

# Add with custom name
cherry-go add repo https://github.com/user/library.git --name mylib

# Add private repository with SSH (name auto-detected as "private")
cherry-go add repo git@github.com:company/private.git

# Add with custom SSH key
cherry-go add repo git@git.company.com:team/repo.git --auth-ssh-key ~/.ssh/company_key
```

#### `add file` - Add a specific file to track

Add a specific file from a previously configured repository. **The file is automatically synced when added.**

```bash
cherry-go add file REPOSITORY_URL/path/to/file.ext [OPTIONS]
# or
cherry-go add file path/to/file.ext --repo REPO_NAME [OPTIONS]
```

**Examples**:

```bash
# Add file with full URL (repository auto-detected, auto-synced)
cherry-go add file https://github.com/user/library.git/src/main.go

# Add from SSH repository (auto-synced)
cherry-go add file git@github.com:company/private.git/config.json

# Add with custom local path (auto-synced)
cherry-go add file https://github.com/user/lib.git/utils.go --local-path internal/utils.go

# Add from specific branch/tag (auto-synced)
cherry-go add file https://github.com/user/lib.git/README.md --branch v1.2.0

# Add from configured repository (if only one exists)
cherry-go add file src/main.go
```

#### `add directory` - Add a directory to track

Add a directory from a previously configured repository. **All files in the directory are automatically synced when added.**

```bash
cherry-go add directory REPOSITORY_URL/path/to/dir/ [OPTIONS]
# or
cherry-go add directory path/to/dir/ --repo REPO_NAME [OPTIONS]
```

**Examples**:

```bash
# Add directory with full URL (repository auto-detected, auto-synced)
cherry-go add directory https://github.com/user/library.git/src/

# Add from SSH repository (auto-synced)
cherry-go add directory git@github.com:company/private.git/lib/

# Add with custom local path (auto-synced)
cherry-go add directory https://github.com/user/lib.git/utils/ --local-path internal/utils/

# Add from specific branch with exclusions (auto-synced)
cherry-go add directory https://github.com/user/lib.git/src/ --branch develop --exclude "*.test.go,tmp/"

# Add from configured repository (if only one exists)
cherry-go add directory src/
```

**Directory Sync Behavior**:
- ✅ **New files**: Automatically added
- ✅ **Modified files**: Updated with conflict detection
- ✅ **Deleted files**: Removed from local copy
- ✅ **Excluded patterns**: Ignored during sync

### `remove` - Remove a source repository

Remove a source from tracking:

```bash
cherry-go remove SOURCE_NAME
```

### `sync` - Synchronize files

Sync files from tracked repositories:

```bash
# Sync all sources
cherry-go sync --all

# Sync specific source
cherry-go sync SOURCE_NAME

# Dry run (no changes made)
cherry-go sync --all --dry-run

# Force sync (override local changes)
cherry-go sync --all --force
```

### `cache` - Manage repository cache

Manage the global repository cache:

```bash
# View cache information
cherry-go cache info

# List cached repositories  
cherry-go cache list

# Clean old cached repositories
cherry-go cache clean
```

**Cache System**:
- **Location**: `~/.cache/cherry-go/repos/`
- **Shared**: All projects reuse the same cached repositories
- **Efficient**: No duplicate downloads across projects
- **Automatic**: Managed transparently by cherry-go

### `status` - Show status

Display current configuration and tracking status:

```bash
cherry-go status
```

## Configuration

Cherry-go uses a project-specific YAML configuration file (`.cherry-go.yaml`) stored in your project directory to manage settings:

```yaml
version: "1.0"
sources:
  - name: "mylib"
    repository: "https://github.com/user/library.git"
    auth:
      type: "auto"
    paths:
      - include: "src/utils/"
        exclude: ["*.tmp", "test_*"]
        local_path: "internal/utils"
        branch: "develop"
      - include: "README.md"
        local_path: "docs/external/README.md"
        branch: "v1.2.0"
      - include: "LICENSE"
        # No local_path specified - will use same path as source
        # No branch specified - will use default branch
options:
  auto_commit: true
  commit_prefix: "cherry-go: sync"
  create_branch: false
  branch_prefix: "cherry-go/sync"
```

### Configuration Fields

- **`sources`**: List of tracked repositories
  - **`paths[].include`**: Source path to track
  - **`paths[].local_path`**: Local destination path (optional - defaults to same as source)
  - **`paths[].branch`**: Branch or tag to track (optional - defaults to main/master)
  - **`paths[].exclude`**: Patterns to exclude from tracking
  - **`paths[].files`**: Hash tracking for conflict detection (automatically managed)
- **`options.auto_commit`**: Automatically commit changes (default: true)
- **`options.commit_prefix`**: Prefix for commit messages
- **`options.create_branch`**: Create branch for changes instead of direct commits
- **`options.branch_prefix`**: Prefix for created branches

### Path Management

Cherry-go gives you complete flexibility over where files are placed:

```bash
# Place files in exact same location as source
cherry-go add --name mylib --repo https://github.com/user/lib.git --paths "src/main.go"
# Result: src/main.go -> src/main.go

# Place files in custom location
cherry-go add --name mylib --repo https://github.com/user/lib.git --paths "src/" --local-path "vendor/mylib"
# Result: src/ -> vendor/mylib/

# Mix of default and custom paths
cherry-go add --name mylib --repo https://github.com/user/lib.git --paths "LICENSE,src/utils.go" --local-path "vendor/mylib/"
# Result: LICENSE -> vendor/mylib/LICENSE, src/utils.go -> vendor/mylib/src/utils.go
```

### Authentication

Cherry-go provides **secure, automatic authentication** without storing sensitive data in configuration files:

#### Automatic Authentication (Recommended)
```bash
# For SSH URLs - uses SSH agent automatically
cherry-go add --name repo --repo git@github.com:user/repo.git --paths "src/"

# For HTTPS URLs - uses environment variables
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx
cherry-go add --name repo --repo https://github.com/user/repo.git --paths "src/"
```

#### SSH Authentication
```bash
# Uses SSH agent by default
cherry-go add --name repo --repo git@github.com:user/repo.git --paths "src/"

# Or specify a specific SSH key
cherry-go add --name repo --repo git@github.com:user/repo.git \
  --auth-type ssh --auth-ssh-key ~/.ssh/id_rsa --paths "src/"
```

#### Basic Authentication
```bash
# Username from flag, password from environment
export GIT_PASSWORD=your_password
cherry-go add --name repo --repo https://git.company.com/repo.git \
  --auth-type basic --auth-user username --paths "src/"
```

#### Environment Variables

Cherry-go supports these environment variables for authentication:
- `GITHUB_TOKEN` - GitHub personal access token
- `GITLAB_TOKEN` - GitLab personal access token  
- `GIT_TOKEN` - Generic Git token
- `GIT_USERNAME` / `GIT_PASSWORD` - Basic auth credentials

**Security Benefits:**
- ✅ No tokens stored in configuration files
- ✅ Uses SSH agent when available
- ✅ Supports environment variables
- ✅ Automatic detection based on repository URL

## Global Options

- `--config`: Specify config file path (default: `.cherry-go.yaml` in current directory)
- `--dry-run`: Simulate actions without making changes
- `--verbose, -v`: Enable verbose output

**Note**: Configuration files are project-specific and should be stored in your project root directory.

### Project-Specific Configuration

Each project using cherry-go should have its own `.cherry-go.yaml` file:

```bash
# Project A
cd /path/to/project-a
cherry-go add --name utils --repo https://github.com/company/utils.git --paths "src/"
# Creates project-a/.cherry-go.yaml

# Project B  
cd /path/to/project-b
cherry-go add --name components --repo https://github.com/company/ui.git --paths "components/"
# Creates project-b/.cherry-go.yaml
```

This approach ensures:
- **Isolation**: Each project manages its own dependencies
- **Version Control**: Configuration is tracked with your project code
- **Team Collaboration**: Shared configuration across team members
- **Reproducibility**: Consistent dependency management across environments

## Conflict Detection and Resolution

Cherry-go automatically tracks file hashes to detect when local files have been modified. When syncing, it will warn you about conflicts:

```bash
# If local files have been modified
cherry-go sync --all
# ⚠️  Conflicts detected in mylib:
#   - Modified: src/utils.go (expected: abcd1234, actual: 5678efgh)
# Sync aborted due to conflicts. Use --force to override or resolve manually.

# Force sync to override local changes
cherry-go sync --all --force
# 🔧 Force mode: Overriding local changes in mylib

# Dry run to see what would happen
cherry-go sync --all --dry-run
# Shows conflicts without making changes
```

### Conflict Types

- **Modified**: Local file content differs from expected
- **Deleted**: Expected file is missing locally
- **Added**: Unexpected file exists locally

### Resolution Options

1. **Manual Resolution**: Edit local files to resolve conflicts
2. **Force Override**: Use `--force` flag to override local changes
3. **Selective Sync**: Sync only specific sources without conflicts

## Use Cases

### Vendor Dependencies
Track specific files from external libraries:
```bash
cherry-go add --name vendor-lib --repo https://github.com/lib/project.git \
  --paths "src/core/" --local-dir "vendor/lib-core"
```

### Shared Configuration
Sync configuration files across projects:
```bash
cherry-go add --name shared-config --repo https://github.com/company/configs.git \
  --paths "docker/,scripts/deploy.sh" --local-dir "."
```

### Documentation Sync
Keep documentation synchronized:
```bash
cherry-go add --name docs --repo https://github.com/company/docs.git \
  --paths "api-spec.md,deployment-guide.md" --local-dir "docs/"
```

## CI/CD Integration

Cherry-go can be integrated into CI/CD pipelines:

```yaml
# GitHub Actions example
- name: Sync external dependencies
  run: |
    cherry-go sync --all
    if git diff --quiet; then
      echo "No changes"
    else
      git add .
      git commit -m "Auto-sync external dependencies"
      git push
    fi
```

## Development

### Building

```bash
go build -o cherry-go
```

### Running Tests

```bash
go test ./...
```

### Running with Dry-run

Test operations without making changes:

```bash
cherry-go sync --all --dry-run --verbose
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

MIT License - see LICENSE file for details.


# Cherry-go Usage Guide

This guide provides detailed examples and use cases for cherry-go.

**Important**: Cherry-go uses project-specific configuration files. Each project should have its own `.cherry-go.yaml` file in the project root directory.

## Table of Contents

1. [Basic Usage](#basic-usage)
2. [Authentication](#authentication)
3. [Configuration File](#configuration-file)
4. [Advanced Use Cases](#advanced-use-cases)
5. [CI/CD Integration](#cicd-integration)
6. [Troubleshooting](#troubleshooting)

## Basic Usage

### Adding Your First Source

```bash
# Add a public repository
cherry-go add \
  --name "awesome-go-utils" \
  --repo "https://github.com/user/go-utils.git" \
  --paths "pkg/logger/,README.md"
```

### Checking Status

```bash
cherry-go status
```

### Syncing Changes

```bash
# Sync all sources
cherry-go sync --all

# Sync specific source
cherry-go sync awesome-go-utils

# Dry run to see what would happen
cherry-go sync --all --dry-run
```

## Authentication

### GitHub Token Authentication

```bash
# For private repositories
cherry-go add \
  --name "private-repo" \
  --repo "https://github.com/company/private.git" \
  --auth-type token \
  --auth-token "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" \
  --paths "src/core/"
```

### SSH Authentication

```bash
cherry-go add \
  --name "ssh-repo" \
  --repo "git@github.com:company/repo.git" \
  --auth-type ssh \
  --auth-ssh-key "~/.ssh/id_rsa" \
  --paths "lib/"
```

### Basic Authentication

```bash
cherry-go add \
  --name "basic-auth-repo" \
  --repo "https://git.company.com/repo.git" \
  --auth-type basic \
  --auth-user "username" \
  --auth-pass "password" \
  --paths "components/"
```

## Configuration File

Cherry-go uses a project-specific YAML configuration file (`.cherry-go.yaml`) in your project directory that you can edit directly:

```yaml
version: "1.0"
local_prefix: "vendor/external"

sources:
  - name: "shared-components"
    repository: "https://github.com/company/shared.git"
    branch: "stable"
    auth:
      type: "token"
      token: "${GITHUB_TOKEN}" # Use environment variables
    paths:
      - include: "components/ui/"
        exclude: ["*.test.js", "__tests__/"]
        local_dir: "src/components/shared"
      - include: "utils/validation.js"
        local_dir: "src/utils/"

options:
  auto_commit: true
  commit_prefix: "deps: sync"
  create_branch: false
```

### Environment Variables in Config

You can use environment variables in your configuration:

```yaml
sources:
  - name: "private-repo"
    auth:
      type: "token"
      token: "${GITHUB_TOKEN}"
```

## Advanced Use Cases

### 1. Vendor Dependencies

Track specific versions of external libraries:

```bash
cherry-go add \
  --name "vendor-lib" \
  --repo "https://github.com/lib/project.git" \
  --branch "v2.1.0" \
  --paths "src/core/,LICENSE" \
  --local-dir "vendor/lib-core"
```

### 2. Shared Configuration Files

Sync configuration across multiple projects:

```bash
cherry-go add \
  --name "shared-configs" \
  --repo "https://github.com/company/configs.git" \
  --paths "docker/Dockerfile.base,scripts/deploy.sh,.eslintrc.js" \
  --local-dir "."
```

### 3. Documentation Sync

Keep documentation up-to-date:

```bash
cherry-go add \
  --name "api-docs" \
  --repo "https://github.com/company/api-documentation.git" \
  --paths "openapi.yaml,docs/integration-guide.md" \
  --local-dir "docs/api/"
```

### 4. Multi-Repository Component Library

```bash
# UI Components
cherry-go add \
  --name "ui-components" \
  --repo "https://github.com/company/ui-library.git" \
  --paths "src/components/" \
  --local-dir "src/shared/ui"

# Utility Functions
cherry-go add \
  --name "utils" \
  --repo "https://github.com/company/utils.git" \
  --paths "src/validation/,src/formatting/" \
  --local-dir "src/shared/utils"
```

### 5. Template and Boilerplate Sync

```bash
cherry-go add \
  --name "project-templates" \
  --repo "https://github.com/company/templates.git" \
  --paths "github-workflows/,.gitignore.template" \
  --local-dir "."
```

## CI/CD Integration

### GitHub Actions

Create `.github/workflows/sync-deps.yml`:

```yaml
name: Sync Dependencies
on:
  schedule:
    - cron: '0 2 * * *' # Daily at 2 AM
  workflow_dispatch:

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Install cherry-go
      run: go install github.com/your-username/cherry-go@latest
    
    - name: Sync dependencies
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      run: cherry-go sync --all
    
    - name: Create PR if changes
      uses: peter-evans/create-pull-request@v5
      with:
        title: 'Auto-sync external dependencies'
        branch: cherry-go/sync
```

### GitLab CI

Create `.gitlab-ci.yml`:

```yaml
sync-dependencies:
  stage: sync
  image: golang:1.21
  script:
    - go install github.com/your-username/cherry-go@latest
    - cherry-go sync --all
    - |
      if git diff --quiet; then
        echo "No changes"
      else
        git config user.name "Cherry-go Bot"
        git config user.email "cherry-go@company.com"
        git add .
        git commit -m "cherry-go: sync dependencies"
        git push origin HEAD:cherry-go/sync
      fi
  only:
    - schedules
```

### Jenkins Pipeline

```groovy
pipeline {
    agent any
    
    triggers {
        cron('H 2 * * *') // Daily at 2 AM
    }
    
    stages {
        stage('Sync Dependencies') {
            steps {
                sh 'go install github.com/your-username/cherry-go@latest'
                sh 'cherry-go sync --all'
                
                script {
                    def changes = sh(
                        script: 'git diff --quiet; echo $?',
                        returnStdout: true
                    ).trim()
                    
                    if (changes != '0') {
                        sh '''
                            git config user.name "Cherry-go Bot"
                            git config user.email "cherry-go@company.com"
                            git add .
                            git commit -m "cherry-go: sync dependencies"
                            git push origin HEAD:cherry-go/sync
                        '''
                    }
                }
            }
        }
    }
}
```

## Troubleshooting

### Common Issues

#### 1. Authentication Errors

```bash
# Check if token has correct permissions
curl -H "Authorization: token YOUR_TOKEN" https://api.github.com/user

# For SSH, check key is loaded
ssh-add -l
```

#### 2. Path Not Found

```bash
# Use dry-run to debug
cherry-go sync --all --dry-run --verbose

# Check if path exists in repository
git ls-tree -r HEAD --name-only | grep "your-path"
```

#### 3. Permission Issues

```bash
# Check directory permissions
ls -la vendor/external/

# Ensure git repository is initialized
git status
```

### Debug Mode

Enable verbose logging:

```bash
cherry-go sync --all --verbose --dry-run
```

### Configuration Validation

Check your configuration:

```bash
cherry-go status
```

### Resetting State

If you need to start fresh:

```bash
# Remove cherry-go cache
rm -rf .cherry-go/

# Reset configuration
rm .cherry-go.yaml

# Start over
cherry-go add --name "new-source" ...
```

## Best Practices

1. **Use Dry Run First**: Always test with `--dry-run` before actual sync
2. **Project-Specific Configuration**: Keep `.cherry-go.yaml` in your project root and commit it to version control
3. **Environment Variables**: Store sensitive tokens in environment variables, not in the config file
4. **Specific Paths**: Be specific about which files/directories to track
5. **Regular Syncing**: Set up automated syncing in CI/CD
6. **Version Pinning**: Use specific branches or tags for stability
7. **Exclude Patterns**: Use exclude patterns to avoid unnecessary files
8. **Local Directories**: Use meaningful local directory names
9. **Commit Messages**: Configure meaningful commit prefixes
10. **Team Collaboration**: Share configuration through version control for consistent team setup

### Configuration Management

```bash
# ✅ Good: Each project has its own config
project-a/
├── .cherry-go.yaml    # Committed to git
├── src/
└── vendor/external/

project-b/  
├── .cherry-go.yaml    # Different config, also committed
├── components/
└── vendor/shared/

# ❌ Avoid: Global configuration that affects all projects
~/.cherry-go.yaml      # Don't use global config
```

## Examples Repository

Check out the `examples/` directory for more configuration examples and use cases.


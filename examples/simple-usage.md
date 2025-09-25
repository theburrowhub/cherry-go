# Simple Usage Examples

This document shows the new simplified syntax for cherry-go commands.

## Super Simple Workflow

### One-liner File Tracking

```bash
# Initialize project
cherry-go init

# Add files directly (auto-detects and adds repository)
cherry-go add file https://github.com/user/library.git/src/main.go
cherry-go add file git@github.com:company/private.git/config.json
cherry-go add directory https://github.com/user/utils.git/lib/ --local-path vendor/utils/

# Check status
cherry-go status

# Sync all sources (with confirmation)
cherry-go sync
```

## Step-by-step Examples

### Public Repository

```bash
# Initialize
cherry-go init

# Add repository (name auto-detected as "go-patterns")
cherry-go add repo https://github.com/tmrts/go-patterns.git

# Add files from the repository (no need to specify repo name)
cherry-go add file README.md
cherry-go add directory creational/ --local-path patterns/creational/

# Status shows everything
cherry-go status
```

### Private Repository with SSH

```bash
# Initialize
cherry-go init

# Add private repository directly while adding file
cherry-go add file git@github.com:company/private.git/src/auth.go --local-path internal/auth.go

# Add more files from same repository (auto-detected)
cherry-go add file config/database.yaml
cherry-go add directory scripts/ --branch production

# Sync all sources (with confirmation)
cherry-go sync
```

### Multiple Repositories

```bash
# Initialize
cherry-go init

# Add multiple repositories
cherry-go add repo https://github.com/user/frontend.git --name ui
cherry-go add repo https://github.com/user/backend.git --name api

# Add files specifying repository (since multiple exist)
cherry-go add file src/main.go --repo ui
cherry-go add file handlers/auth.go --repo api

# Or use full URL format
cherry-go add file https://github.com/user/frontend.git/components/Button.tsx
cherry-go add directory https://github.com/user/backend.git/middleware/ --repo api

# Sync specific repositories
cherry-go sync ui
cherry-go sync api
```

### Branch and Tag Tracking

```bash
# Initialize
cherry-go init

# Track files from different branches/tags
cherry-go add file https://github.com/user/repo.git/src/main.go --branch main
cherry-go add file https://github.com/user/repo.git/config.json --branch v1.2.0
cherry-go add directory https://github.com/user/repo.git/docs/ --branch stable

# Each file/directory can track different versions
cherry-go status
```

## Real-world Use Cases

### Vendor Dependencies

```bash
# Track specific utility files
cherry-go add file https://github.com/company/utils.git/logger.go --local-path internal/logger.go
cherry-go add file https://github.com/company/utils.git/config.go --local-path internal/config.go

# Track from specific stable version
cherry-go add directory https://github.com/company/ui.git/components/ --branch v2.1.0 --local-path src/components/
```

### Configuration Management

```bash
# Track configuration files from different sources
cherry-go add file https://github.com/company/configs.git/docker/Dockerfile
cherry-go add file https://github.com/company/configs.git/k8s/deployment.yaml --local-path k8s/
cherry-go add directory https://github.com/company/scripts.git/ci/ --local-path .github/workflows/
```

### Documentation Sync

```bash
# Keep documentation in sync
cherry-go add file https://github.com/company/docs.git/api-spec.md --local-path docs/
cherry-go add directory https://github.com/company/docs.git/guides/ --local-path docs/guides/
```

## Comparison: Before vs After

### Before (Verbose)

```bash
cherry-go add repo --name mylib --url https://github.com/user/library.git --branch main
cherry-go add file --repo mylib --path src/main.go --local-path internal/main.go
cherry-go add directory --repo mylib --path utils/ --local-path internal/utils/ --exclude "*.test.go"
```

### After (Simple)

```bash
cherry-go add file https://github.com/user/library.git/src/main.go --local-path internal/main.go
cherry-go add directory https://github.com/user/library.git/utils/ --local-path internal/utils/ --exclude "*.test.go"
```

**Benefits:**
- ✅ **60% fewer parameters** required
- ✅ **Auto-detection** of repository name and path
- ✅ **One-liner capability** for most use cases
- ✅ **Intelligent defaults** for common scenarios
- ✅ **Backward compatibility** with explicit parameters when needed

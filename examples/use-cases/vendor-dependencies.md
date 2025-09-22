# Vendor Dependencies Use Case

This example shows how to use cherry-go to manage vendor dependencies by selectively pulling specific components from external repositories.

## Scenario

Your Go project needs specific utilities from external repositories, but you don't want to include the entire repository as a dependency. You want to:

1. Track only specific packages/files
2. Keep them updated automatically
3. Maintain control over which versions to use

## Setup

### 1. Add Utility Libraries

```bash
# Add logging utilities
cherry-go add \
  --name "go-logging" \
  --repo "https://github.com/op/go-logging.git" \
  --paths "logging.go,format.go" \
  --local-dir "vendor/logging"

# Add HTTP utilities  
cherry-go add \
  --name "http-utils" \
  --repo "https://github.com/gorilla/mux.git" \
  --paths "mux.go,route.go,context.go" \
  --local-dir "vendor/http"

# Add configuration utilities
cherry-go add \
  --name "config-utils" \
  --repo "https://github.com/spf13/viper.git" \
  --branch "v1.15.0" \
  --paths "viper.go,util.go" \
  --local-dir "vendor/config"
```

### 2. Configuration File

Create `.cherry-go.yaml`:

```yaml
version: "1.0"
local_prefix: "vendor"

sources:
  - name: "go-logging"
    repository: "https://github.com/op/go-logging.git"
    branch: "master"
    paths:
      - include: "logging.go"
        local_dir: "vendor/logging"
      - include: "format.go"
        local_dir: "vendor/logging"
      - include: "LICENSE"
        local_dir: "vendor/logging"

  - name: "http-utils"
    repository: "https://github.com/gorilla/mux.git"
    branch: "main"
    paths:
      - include: "mux.go"
        local_dir: "vendor/http"
      - include: "route.go"
        local_dir: "vendor/http"
      - include: "context.go"
        local_dir: "vendor/http"
      - include: "LICENSE"
        local_dir: "vendor/http"

  - name: "config-utils"
    repository: "https://github.com/spf13/viper.git"
    branch: "v1.15.0"
    paths:
      - include: "viper.go"
        exclude: ["*_test.go"]
        local_dir: "vendor/config"
      - include: "util.go"
        exclude: ["*_test.go"]  
        local_dir: "vendor/config"

options:
  auto_commit: true
  commit_prefix: "vendor: update"
  create_branch: false
```

### 3. Directory Structure

After syncing, your project will have:

```
project/
├── vendor/
│   ├── logging/
│   │   ├── logging.go
│   │   ├── format.go
│   │   └── LICENSE
│   ├── http/
│   │   ├── mux.go
│   │   ├── route.go
│   │   ├── context.go
│   │   └── LICENSE
│   └── config/
│       ├── viper.go
│       └── util.go
├── src/
│   └── main.go
└── .cherry-go.yaml
```

## Usage in Code

### main.go

```go
package main

import (
    "project/vendor/logging"
    "project/vendor/http"
    "project/vendor/config"
)

func main() {
    // Use vendored logging
    log := logging.MustGetLogger("example")
    log.Info("Application starting")
    
    // Use vendored HTTP utilities
    router := http.NewRouter()
    router.HandleFunc("/", homeHandler)
    
    // Use vendored config utilities
    config.SetConfigName("config")
    config.ReadInConfig()
    
    log.Info("Server starting on port 8080")
    // ... rest of application
}
```

## Automation

### GitHub Actions Workflow

Create `.github/workflows/update-vendor.yml`:

```yaml
name: Update Vendor Dependencies

on:
  schedule:
    - cron: '0 6 * * MON' # Weekly on Monday at 6 AM
  workflow_dispatch:

jobs:
  update-vendor:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Setup Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Install cherry-go
      run: go install github.com/your-username/cherry-go@latest
    
    - name: Update vendor dependencies
      run: cherry-go sync --all
    
    - name: Run tests
      run: go test ./...
    
    - name: Create PR if changes
      uses: peter-evans/create-pull-request@v5
      if: success()
      with:
        title: 'Update vendor dependencies'
        body: |
          Automated update of vendor dependencies using cherry-go.
          
          Please review the changes and ensure all tests pass.
        branch: vendor/auto-update
        commit-message: 'vendor: automated dependency update'
```

## Benefits

1. **Selective Dependencies**: Only include what you need
2. **Version Control**: Pin to specific versions or branches
3. **Automatic Updates**: Keep dependencies current with automation
4. **License Compliance**: Automatically include license files
5. **No Module Conflicts**: Avoid Go module dependency conflicts
6. **Custom Organization**: Organize vendor code as needed

## Best Practices

1. **Pin Versions**: Use specific tags or commits for stability
2. **Include Licenses**: Always track license files
3. **Test Integration**: Run tests after updates
4. **Review Changes**: Use PRs for vendor updates
5. **Document Sources**: Keep clear documentation of what comes from where

## Makefile Integration

Add to your `Makefile`:

```makefile
.PHONY: vendor-update vendor-status

vendor-update:
	@echo "Updating vendor dependencies..."
	@cherry-go sync --all
	@echo "Running tests..."
	@go test ./...

vendor-status:
	@cherry-go status

vendor-clean:
	@echo "Cleaning vendor directory..."
	@rm -rf vendor/
	@cherry-go sync --all
```

This approach gives you fine-grained control over external dependencies while maintaining the benefits of version control and automation.

# Cherry Bunch Usage Guide

Cherry Bunches are YAML template files that describe sets of files and directories to synchronize from repositories, making it easy to quickly set up common configurations.

## What is a Cherry Bunch?

A Cherry Bunch is a `.cherrybunch` file that contains:
- **name**: Unique identifier for the template
- **description**: Optional description of what the template provides
- **version**: Template version (defaults to "1.0")
- **repository**: Source Git repository URL
- **auth**: Optional authentication configuration
- **files**: List of individual files to sync
- **directories**: List of directories to sync

## Cherry Bunch File Format

```yaml
name: python-setup
description: Basic Python project setup with common configuration files
version: "1.0"
repository: https://github.com/example/python-templates.git
auth:
  type: auto
files:
  - path: .gitignore
    local_path: .gitignore
    branch: main
  - path: requirements.txt
    local_path: requirements.txt
    branch: main
directories:
  - path: src/
    local_path: src/
    branch: main
    exclude:
      - "*.pyc"
      - "__pycache__"
```

## Using Cherry Bunches

### Adding a Cherry Bunch

You can add a Cherry Bunch from a local file or URL:

```bash
# From a local file
cherry-go add cherrybunch ./templates/python.cherrybunch

# From a URL
cherry-go add cherrybunch https://raw.githubusercontent.com/user/bunches/main/python.cherrybunch

# Using the alias 'cb'
cherry-go add cb ./templates/python.cherrybunch

# With a custom name
cherry-go add cb --name my-python-setup ./templates/python.cherrybunch
```

### Creating a Cherry Bunch

Use the interactive assistant to create a Cherry Bunch from your current Git repository:

```bash
# Create interactively
cherry-go cherrybunch create

# Create with specific output file
cherry-go cherrybunch create --output my-template.cherrybunch

# Create with specific branch
cherry-go cherrybunch create --branch develop --output dev-template.cherrybunch
```

The interactive assistant will:
1. Detect your current Git repository and gather basic information
2. Ask for cherry bunch details (name, description, repository URL)
3. **Present an interactive fzf-style selector** for files and directories
4. Allow multiselection using Space key, navigation with arrow keys
5. Ask if you want to configure custom destination paths
6. If custom paths are chosen, prompt for each selected item individually
7. Generate the `.cherrybunch` file

#### Interactive Selection Features

- **Fuzzy search**: Type to filter files and directories
- **Multiselection**: Use Space to select multiple items
- **Visual indicators**: [dir] for directories, [file] for files
- **Smart filtering**: Automatically excludes `.git`, `node_modules`, build artifacts
- **Keyboard shortcuts**:
  - Arrow keys: Navigate
  - Space: Toggle selection
  - Enter: Confirm selection
  - Ctrl+C: Cancel

#### Path Configuration Flow

After selection, you'll be asked:
```
Do you want to configure specific paths for the selected items? [y/N]:
```

- **No (default)**: Uses the same paths for source and destination
- **Yes**: Prompts individually for each selected item:
  ```
  Configuring: src/main.go
  Local path [src/main.go]: cmd/main.go
  Branch [main]: develop
  ```

### Synchronizing Cherry Bunch Files

After adding a Cherry Bunch, synchronize the files:

```bash
# Sync the cherry bunch (using the name from the .cherrybunch file)
cherry-go sync python-setup

# Or sync all sources
cherry-go sync
```

## File and Directory Specifications

### Files
```yaml
files:
  - path: src/main.go          # Path in source repository
    local_path: cmd/main.go    # Local destination path (optional)
    branch: main               # Branch to track (optional)
```

### Directories
```yaml
directories:
  - path: src/                 # Directory in source repository
    local_path: internal/      # Local destination path (optional)
    branch: main               # Branch to track (optional)
    exclude:                   # Patterns to exclude (optional)
      - "*.tmp"
      - "cache/"
      - "__pycache__"
```

## Authentication

Cherry Bunches support the same authentication methods as regular repositories:

```yaml
auth:
  type: auto        # Auto-detect (default)
  # type: ssh       # Use SSH key
  # type: basic     # Use username/password (not recommended)
```

## Best Practices

1. **Use descriptive names**: Choose clear, descriptive names for your Cherry Bunches
2. **Version your templates**: Use semantic versioning for your Cherry Bunch files
3. **Document your templates**: Include helpful descriptions
4. **Organize by use case**: Create specific Cherry Bunches for different project types
5. **Test your templates**: Always test Cherry Bunches before sharing them
6. **Use exclude patterns**: Exclude temporary files, build artifacts, and cache directories

## Example Use Cases

### Python Project Template
```yaml
name: python-project
description: Complete Python project setup with testing and CI
repository: https://github.com/templates/python-base.git
files:
  - path: .gitignore
  - path: pyproject.toml
  - path: requirements.txt
  - path: .github/workflows/ci.yml
    local_path: .github/workflows/ci.yml
directories:
  - path: src/
    exclude: ["*.pyc", "__pycache__"]
  - path: tests/
    exclude: [".pytest_cache"]
```

### Go Project Template
```yaml
name: go-project
description: Go project with standard layout
repository: https://github.com/templates/go-base.git
files:
  - path: .gitignore
  - path: go.mod.template
    local_path: go.mod
  - path: Makefile
directories:
  - path: cmd/
  - path: internal/
  - path: pkg/
```

### Frontend Template
```yaml
name: react-app
description: React application with TypeScript and testing setup
repository: https://github.com/templates/react-ts.git
files:
  - path: package.json.template
    local_path: package.json
  - path: tsconfig.json
  - path: .eslintrc.js
directories:
  - path: src/
    exclude: ["node_modules", "dist", "build"]
  - path: public/
```

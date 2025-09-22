#!/bin/bash

# Cherry-go Demo Script
# This script demonstrates basic usage of cherry-go

set -e

echo "🍒 Cherry-go Demo"
echo "=================="
echo

# Build the project first
echo "📦 Building cherry-go..."
go build -o cherry-go
echo "✅ Build completed"
echo

# Show current status (should be empty)
echo "📊 Current status:"
./cherry-go status
echo

# Add a public repository example (using a well-known Go repo)
echo "➕ Adding a public repository..."
./cherry-go add \
  --name "go-patterns" \
  --repo "https://github.com/tmrts/go-patterns.git" \
  --paths "README.md,creational/" \
  --local-dir "vendor/patterns" \
  --dry-run
echo

# Show status after adding source
echo "📊 Status after adding source:"
./cherry-go status
echo

# Demonstrate sync with dry-run
echo "🔄 Syncing (dry-run mode)..."
./cherry-go sync --all --dry-run --verbose
echo

# Show help for all commands
echo "📚 Available commands:"
./cherry-go --help
echo

echo "✨ Demo completed!"
echo ""
echo "📝 Notes:"
echo "- Configuration is stored in .cherry-go.yaml in the current project directory"
echo "- Each project should have its own cherry-go configuration"
echo "- To run actual sync (not dry-run), remove the --dry-run flag"
echo "- Example: ./cherry-go sync --all"


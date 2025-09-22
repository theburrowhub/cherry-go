#!/bin/bash

# Cherry-go Quick Installation Script
# Simple one-liner installation for cherry-go

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🍒 Installing cherry-go...${NC}"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed. Please install Go 1.21 or later.${NC}"
    exit 1
fi

# Create bin directory if it doesn't exist
mkdir -p "$HOME/.local/bin"

# Build and install
echo "Building cherry-go..."
go build -o "$HOME/.local/bin/cherry-go"

# Make executable
chmod +x "$HOME/.local/bin/cherry-go"

echo -e "${GREEN}✅ cherry-go installed to ~/.local/bin/cherry-go${NC}"

# Check if ~/.local/bin is in PATH
if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
    echo -e "${BLUE}ℹ️  Add ~/.local/bin to your PATH:${NC}"
    echo "    export PATH=\"\$PATH:\$HOME/.local/bin\""
    echo "    # Add this line to your ~/.bashrc or ~/.zshrc"
fi

echo -e "${GREEN}🎉 Installation complete!${NC}"
echo "Run 'cherry-go --help' to get started"

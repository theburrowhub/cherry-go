#!/bin/bash

# Cherry-go Installation Script
# This script installs or updates cherry-go locally

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="cherry-go"
INSTALL_DIR="${HOME}/.local/bin"
BACKUP_DIR="${HOME}/.local/backup"

# Functions
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
    exit 1
}

check_dependencies() {
    print_info "Checking dependencies..."
    
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go 1.21 or later."
    fi
    
    # Check Go version
    GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | grep -oE '[0-9]+\.[0-9]+')
    REQUIRED_VERSION="1.21"
    
    if ! printf '%s\n%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V -C; then
        print_error "Go version $GO_VERSION is too old. Please install Go $REQUIRED_VERSION or later."
    fi
    
    print_success "Dependencies check passed (Go $GO_VERSION)"
}

create_directories() {
    print_info "Creating installation directories..."
    
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$BACKUP_DIR"
    
    print_success "Directories created"
}

backup_existing() {
    local existing_binary="$INSTALL_DIR/$BINARY_NAME"
    
    if [ -f "$existing_binary" ]; then
        print_info "Backing up existing installation..."
        
        # Get current version if possible
        local current_version="unknown"
        if [ -x "$existing_binary" ]; then
            current_version=$("$existing_binary" version 2>/dev/null | head -n1 | grep -oE 'version: [^,]+' | cut -d' ' -f2 || echo "unknown")
        fi
        
        local backup_file="$BACKUP_DIR/${BINARY_NAME}-${current_version}-$(date +%Y%m%d-%H%M%S)"
        cp "$existing_binary" "$backup_file"
        
        print_success "Backed up to: $backup_file"
        return 0
    fi
    
    print_info "No existing installation found"
    return 1
}

build_binary() {
    print_info "Building cherry-go..."
    
    # Get version information
    local version="dev"
    local commit_hash="unknown"
    local build_time=$(date -u '+%Y-%m-%d_%H:%M:%S_UTC')
    
    # Try to get git information if available
    if command -v git &> /dev/null && git rev-parse --git-dir > /dev/null 2>&1; then
        version=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
        commit_hash=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    fi
    
    # Build flags
    local ldflags="-X cherry-go/cmd.Version=${version} -X cherry-go/cmd.CommitHash=${commit_hash} -X cherry-go/cmd.BuildTime=${build_time}"
    
    print_info "Version: $version"
    print_info "Commit: $commit_hash"
    print_info "Build time: $build_time"
    
    # Build the binary
    go build -ldflags "$ldflags" -o "$INSTALL_DIR/$BINARY_NAME"
    
    # Make it executable
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
    
    print_success "Binary built successfully"
}

verify_installation() {
    local installed_binary="$INSTALL_DIR/$BINARY_NAME"
    
    print_info "Verifying installation..."
    
    if [ ! -f "$installed_binary" ]; then
        print_error "Installation failed: binary not found"
    fi
    
    if [ ! -x "$installed_binary" ]; then
        print_error "Installation failed: binary is not executable"
    fi
    
    # Test the binary
    local version_output
    if version_output=$("$installed_binary" version 2>&1); then
        print_success "Installation verified"
        echo "$version_output"
    else
        print_error "Installation failed: binary is not working correctly"
    fi
}

update_path() {
    print_info "Checking PATH configuration..."
    
    # Check if install directory is in PATH
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        print_warning "Installation directory is not in PATH"
        print_info "Add the following line to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        echo ""
        echo "    export PATH=\"\$PATH:$INSTALL_DIR\""
        echo ""
        print_info "Then restart your shell or run: source ~/.bashrc (or ~/.zshrc)"
        echo ""
        print_info "Alternatively, you can run cherry-go directly: $INSTALL_DIR/$BINARY_NAME"
    else
        print_success "Installation directory is already in PATH"
    fi
}

show_usage() {
    cat << EOF
Cherry-go Installation Script

Usage: $0 [OPTIONS]

Options:
    -h, --help          Show this help message
    -d, --dir DIR       Set installation directory (default: ~/.local/bin)
    -f, --force         Force reinstallation without backup
    --no-backup         Skip backup of existing installation
    --dry-run           Show what would be done without actually doing it

Examples:
    $0                  # Install to default location
    $0 -d /usr/local/bin # Install to custom directory
    $0 --force          # Force reinstall without backup

EOF
}

# Parse command line arguments
FORCE_INSTALL=false
NO_BACKUP=false
DRY_RUN=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_usage
            exit 0
            ;;
        -d|--dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        -f|--force)
            FORCE_INSTALL=true
            shift
            ;;
        --no-backup)
            NO_BACKUP=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        *)
            print_error "Unknown option: $1. Use --help for usage information."
            ;;
    esac
done

# Main installation process
main() {
    echo "🍒 Cherry-go Installation Script"
    echo "================================"
    echo ""
    
    if [ "$DRY_RUN" = true ]; then
        print_warning "DRY RUN MODE - No changes will be made"
        echo ""
    fi
    
    print_info "Installation directory: $INSTALL_DIR"
    print_info "Backup directory: $BACKUP_DIR"
    echo ""
    
    if [ "$DRY_RUN" = false ]; then
        check_dependencies
        create_directories
        
        # Handle existing installation
        if [ "$FORCE_INSTALL" = false ] && [ "$NO_BACKUP" = false ]; then
            backup_existing
        fi
        
        build_binary
        verify_installation
        update_path
    else
        print_info "Would check dependencies"
        print_info "Would create directories: $INSTALL_DIR, $BACKUP_DIR"
        print_info "Would backup existing binary if found"
        print_info "Would build and install binary to: $INSTALL_DIR/$BINARY_NAME"
        print_info "Would verify installation"
        print_info "Would check PATH configuration"
    fi
    
    echo ""
    print_success "Installation completed successfully!"
    echo ""
    print_info "Next steps:"
    echo "  1. Run: cherry-go init"
    echo "  2. Add a repository: cherry-go add repo --name mylib --url https://github.com/user/repo.git"
    echo "  3. Track files: cherry-go add file --repo mylib --path src/main.go"
    echo "  4. Sync: cherry-go sync mylib"
    echo ""
    print_info "For more information, run: cherry-go --help"
}

# Run main function
main

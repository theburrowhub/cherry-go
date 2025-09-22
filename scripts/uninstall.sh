#!/bin/bash

# Cherry-go Uninstallation Script
# Removes cherry-go from the local system

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
BINARY_NAME="cherry-go"
INSTALL_DIR="${HOME}/.local/bin"
BACKUP_DIR="${HOME}/.local/backup"

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
}

show_usage() {
    cat << EOF
Cherry-go Uninstallation Script

Usage: $0 [OPTIONS]

Options:
    -h, --help          Show this help message
    -d, --dir DIR       Installation directory (default: ~/.local/bin)
    --remove-backups    Also remove backup files
    --dry-run           Show what would be done without actually doing it

Examples:
    $0                      # Remove cherry-go from default location
    $0 --remove-backups     # Remove cherry-go and all backups
    $0 --dry-run            # Show what would be removed

EOF
}

# Parse command line arguments
REMOVE_BACKUPS=false
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
        --remove-backups)
            REMOVE_BACKUPS=true
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

main() {
    echo "🗑️  Cherry-go Uninstallation Script"
    echo "==================================="
    echo ""
    
    if [ "$DRY_RUN" = true ]; then
        print_warning "DRY RUN MODE - No changes will be made"
        echo ""
    fi
    
    local binary_path="$INSTALL_DIR/$BINARY_NAME"
    local found_items=false
    
    # Check for main binary
    if [ -f "$binary_path" ]; then
        found_items=true
        if [ "$DRY_RUN" = true ]; then
            print_info "Would remove: $binary_path"
        else
            print_info "Removing cherry-go binary..."
            rm "$binary_path"
            print_success "Removed: $binary_path"
        fi
    else
        print_info "Cherry-go binary not found at: $binary_path"
    fi
    
    # Check for backups
    if [ -d "$BACKUP_DIR" ]; then
        local backup_files=$(find "$BACKUP_DIR" -name "${BINARY_NAME}-*" 2>/dev/null || true)
        if [ -n "$backup_files" ]; then
            found_items=true
            local backup_count=$(echo "$backup_files" | wc -l)
            print_info "Found $backup_count backup file(s) in $BACKUP_DIR"
            
            if [ "$REMOVE_BACKUPS" = true ]; then
                if [ "$DRY_RUN" = true ]; then
                    print_info "Would remove backup files:"
                    echo "$backup_files" | sed 's/^/  - /'
                else
                    print_info "Removing backup files..."
                    echo "$backup_files" | while read -r backup_file; do
                        if [ -f "$backup_file" ]; then
                            rm "$backup_file"
                            print_success "Removed backup: $(basename "$backup_file")"
                        fi
                    done
                fi
            else
                print_info "Backup files preserved. Use --remove-backups to remove them."
                echo "$backup_files" | sed 's/^/  - /'
            fi
        fi
    fi
    
    # Check for configuration files (inform user)
    local config_files_found=false
    print_info "Checking for configuration files..."
    
    # Common locations where users might have cherry-go configs
    local common_locations=(
        "$HOME/.cherry-go.yaml"
        "$(pwd)/.cherry-go.yaml"
    )
    
    for location in "${common_locations[@]}"; do
        if [ -f "$location" ]; then
            config_files_found=true
            print_warning "Configuration file found: $location"
        fi
    done
    
    if [ "$config_files_found" = true ]; then
        print_info "Configuration files are preserved. Remove them manually if needed."
    else
        print_info "No global configuration files found."
    fi
    
    echo ""
    
    if [ "$found_items" = true ]; then
        if [ "$DRY_RUN" = false ]; then
            print_success "Cherry-go uninstallation completed!"
        else
            print_info "Uninstallation plan shown above."
        fi
    else
        print_info "Cherry-go was not found on this system."
    fi
    
    echo ""
    print_info "Thank you for using cherry-go! 🍒"
}

# Run main function
main

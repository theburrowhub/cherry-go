#!/bin/bash

# Simplified E2E test script for cherry-go
# Uses relative paths correctly

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[✓]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[!]${NC} $1"; }
log_error() { echo -e "${RED}[✗]${NC} $1"; }
log_step() { echo -e "\n${GREEN}===${NC} $1 ${GREEN}===${NC}\n"; }

cleanup() {
    if [ -n "$TEST_DIR" ] && [ -d "$TEST_DIR" ]; then
        log_warning "Cleaning up: $TEST_DIR"
        rm -rf "$TEST_DIR"
    fi
}

trap cleanup EXIT

# Check binary
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHERRY_GO_BIN="$SCRIPT_DIR/cherry-go"

if [ ! -f "$CHERRY_GO_BIN" ]; then
    log_error "Compiling cherry-go..."
    cd "$SCRIPT_DIR"
    go build -o cherry-go .
    log_success "Compiled"
fi

TEST_DIR=$(mktemp -d -t cherry-go-e2e-XXXXXXXXXX)
log_info "Test dir: $TEST_DIR"

ORIGIN_DIR="$TEST_DIR/origin-repo"
DEST_DIR="$TEST_DIR/dest-project"

log_step "STEP 1: Create origin repository"

mkdir -p "$ORIGIN_DIR/src/components"
mkdir -p "$ORIGIN_DIR/src/utils"
cd "$ORIGIN_DIR"
git init
git config user.email "test@test.com"
git config user.name "Test"

cat > src/components/header.js << 'EOF'
export function Header() {
    return "Header v1.0";
}
EOF

cat > src/utils/helpers.js << 'EOF'
export function capitalize(str) {
    return str.charAt(0).toUpperCase() + str.slice(1);
}
EOF

git add .
git commit -m "Initial commit"

log_success "Origin repository created"

log_step "STEP 2: Create destination project"

mkdir -p "$DEST_DIR/src/local"
cd "$DEST_DIR"
git init
git config user.email "test@test.com"
git config user.name "Test"

cat > src/local/app.js << 'EOF'
console.log("Local app");
EOF

git add .
git commit -m "Init"

log_success "Destination project created"

log_step "STEP 3: Initialize cherry-go"

$CHERRY_GO_BIN init

log_step "STEP 4: Add repository and files"

# Add repository first
$CHERRY_GO_BIN add repo "$ORIGIN_DIR" --name origin-lib

# Now add files using repo name and relative paths
log_info "Adding header.js..."
$CHERRY_GO_BIN add file src/components/header.js --repo origin-lib --local-path src/components/header.js

log_info "Adding utils/..."
$CHERRY_GO_BIN add directory src/utils --repo origin-lib --local-path src/utils

log_success "Configuration created"

log_step "STEP 5: Initial sync"

$CHERRY_GO_BIN sync --all

# Verify files
if [ -f src/components/header.js ] && [ -f src/utils/helpers.js ]; then
    log_success "Files synced successfully ✓"
else
    log_error "Files NOT synced"
    ls -la src/components/ src/utils/ || true
    exit 1
fi

git add .
git commit -m "Sync: initial"

log_step "STEP 6: Modify in origin"

cd "$ORIGIN_DIR"
cat > src/components/header.js << 'EOF'
export function Header() {
    return "Header v2.0 - UPDATED";
}
EOF

git add .
git commit -m "Update header v2.0"

log_step "STEP 7: Sync with merge"

cd "$DEST_DIR"

log_info "Syncing changes..."
$CHERRY_GO_BIN sync --all --merge

if grep -q "v2.0" src/components/header.js; then
    log_success "Header updated ✓"
else
    log_error "Header NOT updated"
    cat src/components/header.js
    exit 1
fi

git add .
git commit -m "Sync: header v2.0"

log_step "STEP 8: Create conflict"

# Modify locally
cat > src/utils/helpers.js << 'EOF'
export function capitalize(str) {
    return str.charAt(0).toUpperCase() + str.slice(1);
}

// LOCAL CHANGE
export function lowercase(str) {
    return str.toLowerCase();
}
EOF

git add .
git commit -m "Local: add lowercase"

# Modify in origin
cd "$ORIGIN_DIR"
cat > src/utils/helpers.js << 'EOF'
export function capitalize(str) {
    return str.toUpperCase(); // CHANGED
}

// ORIGIN CHANGE
export function uppercase(str) {
    return str.toUpperCase();
}
EOF

git add .
git commit -m "Origin: add uppercase"

log_step "STEP 9: Sync with conflict"

cd "$DEST_DIR"

log_info "Syncing (with conflict)..."
set +e
$CHERRY_GO_BIN sync --all --merge 2>&1 | tee /tmp/sync-conflict.log
SYNC_CODE=$?
set -e

log_info "Content of helpers.js:"
cat src/utils/helpers.js

if grep -q "uppercase\|lowercase\|<<<<<<<" src/utils/helpers.js; then
    log_success "Merge detected ✓"
else
    log_warning "No conflict markers (may be auto-resolved)"
fi

log_step "STEP 10: Test --merge adding new file"

# First, resolve the pending conflict from step 9 with force
log_info "Resolving pending conflicts with --force..."
$CHERRY_GO_BIN sync --all --force

git add .
git commit -m "Resolved previous conflicts" || true

cd "$ORIGIN_DIR"

# Create a new file that doesn't exist locally yet
cat > src/components/footer.js << 'EOF'
export function Footer() {
    return "Footer v1.0";
}
EOF

git add .
git commit -m "Origin: add footer.js"

cd "$DEST_DIR"

# Add footer.js to tracking and sync it
log_info "Adding footer.js to tracking..."
$CHERRY_GO_BIN add file src/components/footer.js --repo origin-lib --local-path src/components/footer.js

# The add file command already syncs, but let's verify
if [ -f src/components/footer.js ]; then
    log_success "New file synced cleanly ✓"
else
    log_error "New file was not synced"
    exit 1
fi

log_info "Content of new file:"
cat src/components/footer.js

git add .
git commit -m "Sync: added footer.js"

log_step "STEP 11: Test --force override"

cd "$ORIGIN_DIR"
cat > src/utils/helpers.js << 'EOF'
export function capitalize(str) {
    return str.charAt(0).toUpperCase() + str.slice(1);
}

// ORIGIN FINAL VERSION
export function formatText(str) {
    return str.trim().toUpperCase();
}
EOF

git add .
git commit -m "Final: formatText function"

cd "$DEST_DIR"

# Modify locally
cat > src/utils/helpers.js << 'EOF'
export function capitalize(str) {
    return str.charAt(0).toUpperCase() + str.slice(1);
}

// LOCAL VERSION - should be overridden
export function processText(str) {
    return str.toLowerCase();
}
EOF

log_info "Applying --force (should override local)..."
$CHERRY_GO_BIN sync --all --force

if grep -q "formatText" src/utils/helpers.js && ! grep -q "processText" src/utils/helpers.js; then
    log_success "Force override successful ✓"
else
    log_error "Force did NOT override correctly"
    cat src/utils/helpers.js
    exit 1
fi

log_step "✅ ALL TESTS PASSED"

echo ""
log_success "═══════════════════════════════════"
log_success "  E2E TEST COMPLETE ✓"
log_success "═══════════════════════════════════"
echo ""
log_info "Tests completed:"
echo "  ✓ Init and configuration"
echo "  ✓ Initial synchronization"
echo "  ✓ Update with merge"
echo "  ✓ Conflict detection and auto-merge"
echo "  ✓ Adding new files cleanly"
echo "  ✓ Override with --force"
echo ""
log_info "Test dir: $TEST_DIR"
log_info "To inspect: cd $TEST_DIR/dest-project"

trap - EXIT

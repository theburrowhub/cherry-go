#!/bin/bash

# Build script for cherry-go with version information

set -e

# Default values
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
COMMIT_HASH=${COMMIT_HASH:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}
BUILD_TIME=${BUILD_TIME:-$(date -u '+%Y-%m-%d_%H:%M:%S_UTC')}

# Build flags
LDFLAGS="-X cherry-go/cmd.Version=${VERSION} -X cherry-go/cmd.CommitHash=${COMMIT_HASH} -X cherry-go/cmd.BuildTime=${BUILD_TIME}"

echo "Building cherry-go..."
echo "Version: ${VERSION}"
echo "Commit: ${COMMIT_HASH}"
echo "Build Time: ${BUILD_TIME}"
echo

# Build for current platform
echo "Building for current platform..."
go build -ldflags "${LDFLAGS}" -o cherry-go

echo "✅ Build completed: cherry-go"

# Optional: Build for multiple platforms
if [ "$1" = "all" ]; then
    echo
    echo "Building for multiple platforms..."
    
    # Create dist directory
    mkdir -p dist
    
    # Build for different platforms
    platforms=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64")
    
    for platform in "${platforms[@]}"; do
        IFS='/' read -r GOOS GOARCH <<< "$platform"
        output_name="cherry-go-${VERSION}-${GOOS}-${GOARCH}"
        
        if [ "$GOOS" = "windows" ]; then
            output_name="${output_name}.exe"
        fi
        
        echo "Building for ${GOOS}/${GOARCH}..."
        env GOOS="$GOOS" GOARCH="$GOARCH" go build -ldflags "${LDFLAGS}" -o "dist/${output_name}"
        
        # Create compressed archive
        if [ "$GOOS" = "windows" ]; then
            (cd dist && zip "${output_name%.exe}.zip" "$output_name")
        else
            (cd dist && tar -czf "${output_name}.tar.gz" "$output_name")
        fi
    done
    
    echo "✅ Multi-platform builds completed in dist/"
    ls -la dist/
fi

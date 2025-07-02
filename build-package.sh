#!/bin/bash
# Build and package haxorport-go-client for distribution
# Author: alwanandri2712
# Usage: bash build-package.sh
set -e

# Constants
APP_NAME="haxorport-go-client"
REPO_BRANCH="main"
VERSION=$(git describe --tags 2>/dev/null || echo "v1.0.0")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build settings
BUILD_OS="linux"
BUILD_ARCH="amd64"
DIST_DIR="dist"
OUTPUT_TAR="${APP_NAME}-${BUILD_OS}-${BUILD_ARCH}.tar.gz"

# Clean previous build
echo "🚀 [${APP_NAME}] Preparing build environment..."
rm -rf "$DIST_DIR" "$OUTPUT_TAR" "${APP_NAME}"*".tar.gz"
mkdir -p "$DIST_DIR"

# Build binary
echo "🔨 Building ${APP_NAME} ${VERSION} for ${BUILD_OS}/${BUILD_ARCH}..."
GOOS=${BUILD_OS} GOARCH=${BUILD_ARCH} go build \
    -ldflags "-X main.version=${VERSION} -X main.commit=${GIT_COMMIT} -X main.date=${BUILD_DATE}" \
    -o "$DIST_DIR/$APP_NAME" \
    main.go

# Make binary executable
chmod +x "$DIST_DIR/$APP_NAME"

# Create necessary directories in dist
echo "📂 Setting up distribution directory structure..."
mkdir -p "$DIST_DIR/logs"

# Copy configuration files
echo "📋 Copying configuration files..."
cp config.yaml "$DIST_DIR/config.yaml"
cp config_tcp.yaml "$DIST_DIR/config_tcp.yaml"

# Ensure connection_mode is set correctly in config files
echo "🔧 Verifying configuration settings..."

# Check if running on macOS
if [[ "$OSTYPE" == "darwin"* ]]; then
    # For macOS, sed requires an empty string after -i
    sed -i '' "s|connection_mode:.*|connection_mode: \"websocket\"  # Options: websocket, direct_tcp|g" "$DIST_DIR/config.yaml"
    sed -i '' "s|connection_mode:.*|connection_mode: \"direct_tcp\"  # Options: websocket, direct_tcp|g" "$DIST_DIR/config_tcp.yaml"
    sed -i '' "s|server_address:.*|server_address: control.haxorport.online|g" "$DIST_DIR/config.yaml"
    sed -i '' "s|server_address:.*|server_address: tcp.haxorport.online|g" "$DIST_DIR/config_tcp.yaml"
else
    # For Linux and other systems
    sed -i "s|connection_mode:.*|connection_mode: \"websocket\"  # Options: websocket, direct_tcp|g" "$DIST_DIR/config.yaml"
    sed -i "s|connection_mode:.*|connection_mode: \"direct_tcp\"  # Options: websocket, direct_tcp|g" "$DIST_DIR/config_tcp.yaml"
    sed -i "s|server_address:.*|server_address: control.haxorport.online|g" "$DIST_DIR/config.yaml"
    sed -i "s|server_address:.*|server_address: tcp.haxorport.online|g" "$DIST_DIR/config_tcp.yaml"
fi

# Copy installation and utility scripts
echo "📜 Copying installation and utility scripts..."
cp install.sh "$DIST_DIR/"
cp uninstall.sh "$DIST_DIR/"
cp setup.sh "$DIST_DIR/" 2>/dev/null || echo "ℹ️  setup.sh not found, skipping..."

# Generate version file
echo "🏷️  Generating version information..."
cat > "$DIST_DIR/version" <<- EOM
Version:    ${VERSION}
Git commit: ${GIT_COMMIT}
Build date: ${BUILD_DATE}
Platform:   ${BUILD_OS}/${BUILD_ARCH}
EOM

# Create package
echo "📦 Creating package ${OUTPUT_TAR}..."
# Use --no-xattrs to avoid macOS extended attributes that cause warnings on Linux
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS: Use gnutar if available, otherwise use BSD tar with --no-xattrs
    if command -v gtar &> /dev/null; then
        gtar --no-xattrs -czf "$OUTPUT_TAR" -C "$DIST_DIR" .
    else
        # BSD tar on macOS doesn't support --no-xattrs, use COPYFILE_DISABLE
        COPYFILE_DISABLE=1 tar -czf "$OUTPUT_TAR" -C "$DIST_DIR" .
    fi
else
    # Linux and other systems
    tar -czf "$OUTPUT_TAR" -C "$DIST_DIR" .
fi

# Create checksum
echo "🔒 Generating checksum..."
sha256sum "$OUTPUT_TAR" > "${OUTPUT_TAR}.sha256"

# Success message
echo "\n✅ [SUCCESS] Package created: ${OUTPUT_TAR}"
echo "   - Size: $(du -h "$OUTPUT_TAR" | cut -f1)"
echo "   - SHA256: $(cat "${OUTPUT_TAR}.sha256" | cut -d' ' -f1)"
echo "\n🚀 To install, run: tar -xzf ${OUTPUT_TAR} && cd ${DIST_DIR} && ./install.sh"

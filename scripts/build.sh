#!/bin/bash

# Script to build Haxorport Client
# Supports Linux, macOS, and Windows (via WSL/Git Bash)

# Function to display error message and exit
error_exit() {
    echo "ERROR: $1" >&2
    exit 1
}

# Detect OS and architecture
detect_platform() {
    # Detect OS
    case "$(uname -s)" in
        Linux*)     OS="linux" ;;
        Darwin*)    OS="darwin" ;;
        MINGW*|MSYS*) OS="windows" ;;
        *)          OS="unknown" ;;
    esac
    
    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)  ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *)             ARCH="unknown" ;;
    esac
    
    echo "Detected platform: $OS/$ARCH"
}

# Check if Go is installed
check_go() {
    if ! command -v go &> /dev/null; then
        error_exit "Go is not installed. Please install Go first: https://golang.org/doc/install"
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    echo "Go version: $GO_VERSION"
}

# Determine project root directory
set_root_dir() {
    # Try to use BASH_SOURCE if available
    if [ -n "${BASH_SOURCE[0]}" ]; then
        ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.."; pwd)"
    else
        # Fallback for other shells
        SCRIPT_DIR="$(cd "$(dirname "$0")"; pwd)"
        ROOT_DIR="$(cd "$SCRIPT_DIR/.."; pwd)"
    fi
    
    cd "$ROOT_DIR" || error_exit "Failed to change to project root directory"
    echo "Project directory: $ROOT_DIR"
}

# Get application version
get_version() {
    VERSION_FILE="$ROOT_DIR/cmd/version.go"
    if [ -f "$VERSION_FILE" ]; then
        VERSION=$(grep 'const Version = "' "$VERSION_FILE" | cut -d'"' -f2)
        echo "Building Haxorport Client v$VERSION"
    else
        VERSION="dev"
        echo "version.go file not found, using version: dev"
    fi
}

# Clean build directory
clean_build_dir() {
    echo "Cleaning build directory..."
    rm -rf "$ROOT_DIR/bin"
    mkdir -p "$ROOT_DIR/bin"
}

# Build for current platform
build_current_platform() {
    echo "Building for $OS/$ARCH..."
    
    if [ "$OS" = "windows" ]; then
        OUTPUT="$ROOT_DIR/bin/haxorport.exe"
    else
        OUTPUT="$ROOT_DIR/bin/haxorport"
    fi
    
    echo "Downloading dependencies..."
    go mod download || error_exit "Failed to download dependencies"
    
    echo "Building application..."
    GOOS=$OS GOARCH=$ARCH go build -o "$OUTPUT" "$ROOT_DIR/main.go" || error_exit "Build failed"
    
    # Create symlink for compatibility with older scripts that might expect 'haxor' instead of 'haxorport'
    if [ "$OS" != "windows" ]; then
        ln -sf "$ROOT_DIR/bin/haxorport" "$ROOT_DIR/bin/haxor"
        echo "Created symlink: $ROOT_DIR/bin/haxor -> $ROOT_DIR/bin/haxorport"
    fi
    
    echo "✅ Build successful!"
    echo "Binary location: $OUTPUT"
}

# Build for all platforms
build_all_platforms() {
    echo "Building for all platforms..."
    
    # Linux (amd64)
    echo "Building for linux/amd64..."
    GOOS=linux GOARCH=amd64 go build -o "$ROOT_DIR/bin/haxorport-linux-amd64" "$ROOT_DIR/main.go" || echo "⚠️ Build for linux/amd64 failed"
    
    # Linux (arm64)
    echo "Building for linux/arm64..."
    GOOS=linux GOARCH=arm64 go build -o "$ROOT_DIR/bin/haxorport-linux-arm64" "$ROOT_DIR/main.go" || echo "⚠️ Build for linux/arm64 failed"
    
    # macOS (amd64)
    echo "Building for darwin/amd64..."
    GOOS=darwin GOARCH=amd64 go build -o "$ROOT_DIR/bin/haxorport-darwin-amd64" "$ROOT_DIR/main.go" || echo "⚠️ Build for darwin/amd64 failed"
    
    # macOS (arm64)
    echo "Building for darwin/arm64..."
    GOOS=darwin GOARCH=arm64 go build -o "$ROOT_DIR/bin/haxorport-darwin-arm64" "$ROOT_DIR/main.go" || echo "⚠️ Build for darwin/arm64 failed"
    
    # Windows (amd64)
    echo "Building for windows/amd64..."
    GOOS=windows GOARCH=amd64 go build -o "$ROOT_DIR/bin/haxorport-windows-amd64.exe" "$ROOT_DIR/main.go" || echo "⚠️ Build for windows/amd64 failed"
    
    echo "✅ Multi-platform build completed!"
}

# Main script
echo "=== Haxorport Client Build Script ==="

# Initialization
detect_platform
check_go
set_root_dir
get_version
clean_build_dir

# Build application
build_current_platform

# Build for all platforms if --all flag is provided
if [ "$1" == "--all" ]; then
    build_all_platforms
fi

echo "\n=================================================="
echo "✅ Haxorport Client successfully built!"
echo "=================================================="
echo "Binary location: $ROOT_DIR/bin/"
echo ""
echo "To run:"
echo "  $ROOT_DIR/bin/haxorport --help"
echo "=================================================="

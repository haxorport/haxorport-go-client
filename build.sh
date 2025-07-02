#!/bin/bash

# All-in-one script for Haxorport Client
# Supports Linux, macOS, and Windows (via WSL/Git Bash)
# Author: Haxorport Team

# Colors for output formatting
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions for displaying colored messages
print_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

print_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

# Detect OS and architecture
detect_platform() {
    # Detect OS
    case "$(uname -s)" in
        Linux*)     
            OS="linux" 
            CONFIG_DIR="$HOME/.haxorport"
            if [ "$(id -u)" -eq 0 ]; then
                # If root user
                CONFIG_DIR="/etc/haxorport"
            fi
            ;;
        Darwin*)    
            OS="darwin" 
            CONFIG_DIR="$HOME/Library/Preferences/haxorport"
            ;;
        MINGW*|MSYS*) 
            OS="windows"
            CONFIG_DIR="$HOME/.haxorport/config"
            ;;
        *)          
            OS="unknown"
            CONFIG_DIR="$HOME/.haxorport"
            ;;
    esac
    
    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)  ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *)             ARCH="unknown" ;;
    esac
    
    CONFIG_FILE="$CONFIG_DIR/config.yaml"
    print_info "Detected platform: $OS/$ARCH"
}

# Check if Go is installed
check_go() {
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go first: https://golang.org/doc/install"
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    print_info "Go version: $GO_VERSION"
}

# Clean bin directory
clean_bin_dir() {
    print_info "Cleaning bin directory..."
    mkdir -p bin
}

# Build application
build_app() {
    print_info "Downloading dependencies..."
    go mod download || print_error "Failed to download dependencies"
    
    print_info "Building application..."
    if [ "$OS" = "windows" ]; then
        go build -o bin/haxorport.exe main.go || print_error "Build failed"
    else
        go build -o bin/haxorport main.go || print_error "Build failed"
    fi
    
    # Give execution permission to binary (for Linux and macOS)
    if [ "$OS" != "windows" ]; then
        chmod +x bin/haxorport
    fi
    
    print_success "Build successful!"
}

# Build for all platforms
build_all_platforms() {
    print_info "Building for all platforms..."
    
    # Linux (amd64)
    print_info "Building for linux/amd64..."
    GOOS=linux GOARCH=amd64 go build -o "bin/haxorport-linux-amd64" main.go || print_info "⚠️ Build for linux/amd64 failed"
    
    # Linux (arm64)
    print_info "Building for linux/arm64..."
    GOOS=linux GOARCH=arm64 go build -o "bin/haxorport-linux-arm64" main.go || print_info "⚠️ Build for linux/arm64 failed"
    
    # macOS (amd64)
    print_info "Building for darwin/amd64..."
    GOOS=darwin GOARCH=amd64 go build -o "bin/haxorport-darwin-amd64" main.go || print_info "⚠️ Build for darwin/amd64 failed"
    
    # macOS (arm64)
    print_info "Building for darwin/arm64..."
    GOOS=darwin GOARCH=arm64 go build -o "bin/haxorport-darwin-arm64" main.go || print_info "⚠️ Build for darwin/arm64 failed"
    
    # Windows (amd64)
    print_info "Building for windows/amd64..."
    GOOS=windows GOARCH=amd64 go build -o "bin/haxorport-windows-amd64.exe" main.go || print_info "⚠️ Build for windows/amd64 failed"
    
    print_success "Multi-platform build completed!"
}

# Run application after build (optional)
run_app() {
    if [ "$1" = "--run" ]; then
        print_info "Running application..."
        if [ "$OS" = "windows" ]; then
            ./bin/haxorport.exe "${@:2}"
        else
            ./bin/haxorport "${@:2}"
        fi
    fi
}

# Check internet connection
check_internet() {
    print_info "Checking internet connection..."
    if ping -c 1 control.haxorport.online &> /dev/null; then
        print_success "Internet connection OK"
        return 0
    else
        print_info "Cannot reach control.haxorport.online, trying with IP..."
        if ping -c 1 8.8.8.8 &> /dev/null; then
            print_info "Internet connection OK, but DNS may be problematic"
            return 0
        else
            print_error "No internet connection. Please check your network connection."
            return 1
        fi
    fi
}

# Check firewall
check_firewall() {
    print_info "Checking access to port 443..."
    if nc -z -w 5 control.haxorport.online 443 &> /dev/null; then
        print_success "Port 443 is accessible"
        return 0
    else
        print_info "Port 443 is not accessible. Firewall may be blocking the connection."
        return 1
    fi
}

# Update configuration
update_config() {
    print_info "Updating configuration..."
    
    # Check if configuration directory exists
    if [ ! -d "$CONFIG_DIR" ]; then
        print_info "Creating configuration directory: $CONFIG_DIR"
        mkdir -p "$CONFIG_DIR"
    fi
    
    # Backup old configuration if exists
    if [ -f "$CONFIG_FILE" ]; then
        print_info "Old configuration backup saved to $CONFIG_FILE.bak"
        cp "$CONFIG_FILE" "$CONFIG_FILE.bak"
    fi
    
    # Ask for token from user
    print_info "Please enter your authentication token from the Haxorport dashboard:"
    read -p "Token: " USER_TOKEN
    
    if [ -z "$USER_TOKEN" ]; then
        print_error "Token cannot be empty. Please run this script again and enter a valid token."
    fi
    
    # Create logs directory if it doesn't exist
    mkdir -p "$CONFIG_DIR/logs"
    
    # Update configuration
    cat > "$CONFIG_FILE" << EOF
# Haxorport Client Configuration
server_address: control.haxorport.online
control_port: 443
data_port: 0

# Authentication Configuration
auth_enabled: true
auth_token: $USER_TOKEN
auth_validation_url: https://haxorport.online/AuthToken/validate

# TLS Configuration
tls_enabled: true
tls_cert: ""
tls_key: ""

# Base domain for tunnel subdomains
base_domain: "haxorport.online"

# Logging Configuration
log_level: warn
log_file: "$CONFIG_DIR/logs/haxorport-client.log"
tunnels: []
EOF

    print_success "Configuration successfully updated!"
}

# Test connection
test_connection() {
    print_info "Testing connection to server..."
    
    # Check if haxorport is available
    if command -v haxorport &> /dev/null; then
        HAXOR_CMD="haxorport"
    elif [ -f "./bin/haxorport" ]; then
        HAXOR_CMD="./bin/haxorport"
    else
        print_error "Haxorport client not found. Make sure you are in the project directory or haxorport is installed."
    fi
    
    # Run version command to test connection
    print_info "Running version command to test connection..."
    VERSION_OUTPUT=$($HAXOR_CMD version 2>&1)
    VERSION_EXIT_CODE=$?
    
    if [ $VERSION_EXIT_CODE -eq 0 ]; then
        print_success "Connection successful! Version command ran correctly."
        print_debug "Output: $VERSION_OUTPUT"
    else
        print_info "Version command failed. Attempting further diagnosis..."
        print_debug "Error: $VERSION_OUTPUT"
        
        # Try with verbose logging
        print_info "Trying with debug log level..."
        DEBUG_CONFIG="$CONFIG_DIR/debug.yaml"
        
        # Create temporary debug configuration
        cp "$CONFIG_FILE" "$DEBUG_CONFIG"
        sed -i.bak 's/log_level: warn/log_level: debug/g' "$DEBUG_CONFIG"
        
        # Run with debug configuration
        DEBUG_OUTPUT=$($HAXOR_CMD -c "$DEBUG_CONFIG" version 2>&1)
        print_debug "Debug output: $DEBUG_OUTPUT"
        
        # Remove temporary debug configuration file
        rm -f "$DEBUG_CONFIG" "$DEBUG_CONFIG.bak"
        
        # Check for common issues
        if echo "$DEBUG_OUTPUT" | grep -q "bad handshake"; then
            print_info "Detected 'bad handshake' issue. Possible causes:"
            print_info "1. Server may be down or unavailable"
            print_info "2. Authentication token may be invalid"
            print_info "3. Firewall may be blocking WebSocket connections"
            print_info "4. Proxy may be interfering with WebSocket connections"
            
            # Try to check server status
            print_info "Checking server status..."
            if curl -s -o /dev/null -w "%{http_code}" https://control.haxorport.online 2>/dev/null | grep -q "200"; then
                print_success "Server responds well via HTTPS"
                print_info "The issue is likely with the token or WebSocket configuration"
            else
                print_info "Server does not respond well via HTTPS. Server may be down."
            fi
        fi
    fi
    
    print_info "You can try running the HTTP tunnel command:"
    print_info "$HAXOR_CMD http --port 80"
}

# Function to display help
show_help() {
    echo -e "${GREEN}Usage:${NC} $0 [OPTIONS] [COMMAND]"
    echo -e ""
    echo -e "${YELLOW}Options:${NC}"
    echo -e "  --help          Display this help"
    echo -e "  --all           Build for all platforms"
    echo -e "  --run COMMAND   Run the application after build"
    echo -e ""
    echo -e "${YELLOW}Commands:${NC}"
    echo -e "  build           Build the application (default)"
    echo -e "  config          Update configuration"
    echo -e "  test            Test connection to server"
    echo -e "  fix             Fix connection issues"
    echo -e ""
    echo -e "${YELLOW}Examples:${NC}"
    echo -e "  $0              Build the application"
    echo -e "  $0 --all        Build for all platforms"
    echo -e "  $0 config       Update configuration"
    echo -e "  $0 fix          Fix connection issues"
    echo -e "  $0 --run http   Build and run HTTP tunnel"
    echo -e ""
}

# Main script
echo -e "${GREEN}=== Haxorport Client Tool ===${NC}"

# Initialization
detect_platform

# Parse arguments
COMMAND="build"
BUILD_ALL=false
RUN_ARGS=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --help)
            show_help
            exit 0
            ;;
        --all)
            BUILD_ALL=true
            shift
            ;;
        --run)
            if [[ $# -gt 1 ]]; then
                RUN_ARGS="${@:2}"
                break
            else
                print_error "--run requires additional arguments"
            fi
            ;;
        build|config|test|fix)
            COMMAND=$1
            shift
            ;;
        *)
            print_error "Unknown option: $1. Use --help for assistance."
            ;;
    esac
done

# Execute command
case $COMMAND in
    build)
        check_go
        clean_bin_dir
        if [ "$BUILD_ALL" = true ]; then
            build_all_platforms
        else
            build_app
        fi
        
        # Display information
        echo -e "\n${GREEN}==================================================${NC}"
        echo -e "${GREEN}✅ Haxorport Client successfully built!${NC}"
        echo -e "${GREEN}==================================================${NC}"
        echo -e "Binary location: $(pwd)/bin/"
        echo -e ""
        
        # Check if haxorport is already installed globally
        if command -v haxorport &> /dev/null; then
            echo -e "Haxorport is already installed globally. You can use:"
            if [ "$OS" = "windows" ]; then
                echo -e "  haxorport.exe --help"
            else
                echo -e "  haxorport --help"
            fi
            echo -e ""
            echo -e "Or use the newly built binary:"
        fi
        
        echo -e "To run the newly built binary:"
        if [ "$OS" = "windows" ]; then
            echo -e "  ./bin/haxorport.exe --help"
        else
            echo -e "  ./bin/haxorport --help"
        fi
        echo -e ""
        echo -e "Usage examples:"
        echo -e "  haxorport http --port 80"
        echo -e "  haxorport tcp --local-port 22 --remote-port 2222"
        echo -e "${GREEN}==================================================${NC}"
        
        # Run application if RUN_ARGS is not empty
        if [ -n "$RUN_ARGS" ]; then
            print_info "Running application with arguments: $RUN_ARGS"
            if [ "$OS" = "windows" ]; then
                ./bin/haxorport.exe $RUN_ARGS
            else
                ./bin/haxorport $RUN_ARGS
            fi
        fi
        ;;
    config)
        update_config
        test_connection
        ;;
    test)
        check_internet
        check_firewall
        test_connection
        ;;
    fix)
        check_internet
        check_firewall
        update_config
        test_connection
        ;;
esac

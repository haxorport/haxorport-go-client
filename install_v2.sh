#!/bin/bash
# HaxorPort Go Client Installer v2
# Downloads and installs the latest haxorport-go-client binary
# Author: alwanandri2712
# Usage: bash install_v2.sh [download_url]

set -e

# Constants
APP_NAME="haxorport-go-client"
BINARY_NAME="haxorport-client"
ARCH="linux-amd64"
TAR_FILE="${APP_NAME}-${ARCH}.tar.gz"
CHECKSUM_FILE="${TAR_FILE}.sha256"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/haxorport"
TEMP_DIR="/tmp/haxorport-install"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

check_dependencies() {
    local deps=("curl" "tar" "sha256sum")
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            log_error "Required dependency '$dep' is not installed"
            exit 1
        fi
    done
}

detect_architecture() {
    local arch=$(uname -m)
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    
    case $arch in
        x86_64|amd64)
            ARCH="${os}-amd64"
            ;;
        aarch64|arm64)
            ARCH="${os}-arm64"
            ;;
        *)
            log_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
    
    TAR_FILE="${APP_NAME}-${ARCH}.tar.gz"
    CHECKSUM_FILE="${TAR_FILE}.sha256"
    log_info "Detected architecture: $ARCH"
}

download_binary() {
    local download_url="$1"
    
    # Use default HaxorPort URL if not provided
    if [[ -z "$download_url" ]]; then
        download_url="https://haxorport.online/"
        log_info "Using default download URL: $download_url"
    fi
    
    # Ensure URL ends with /
    if [[ "$download_url" != */ ]]; then
        download_url="${download_url}/"
    fi
    
    local tar_url="${download_url}${TAR_FILE}"
    local checksum_url="${download_url}${CHECKSUM_FILE}"
    
    log_info "Creating temporary directory: $TEMP_DIR"
    rm -rf "$TEMP_DIR"
    mkdir -p "$TEMP_DIR"
    cd "$TEMP_DIR"
    
    log_info "Downloading $TAR_FILE..."
    if ! curl -L -o "$TAR_FILE" "$tar_url"; then
        log_error "Failed to download $TAR_FILE from $tar_url"
        exit 1
    fi
    
    log_info "Downloading checksum file..."
    if ! curl -L -o "$CHECKSUM_FILE" "$checksum_url"; then
        log_warning "Failed to download checksum file, skipping verification"
    else
        log_info "Verifying checksum..."
        if sha256sum -c "$CHECKSUM_FILE"; then
            log_success "Checksum verification passed"
        else
            log_error "Checksum verification failed"
            exit 1
        fi
    fi
}

extract_binary() {
    log_info "Extracting $TAR_FILE..."
    if ! tar -xzf "$TAR_FILE"; then
        log_error "Failed to extract $TAR_FILE"
        exit 1
    fi
    
    if [[ ! -f "$APP_NAME" ]]; then
        log_error "Binary $APP_NAME not found in extracted files"
        exit 1
    fi
    
    chmod +x "$APP_NAME"
    log_success "Binary extracted successfully"
}

install_binary() {
    log_info "Installing binary to $INSTALL_DIR..."
    
    # Backup existing binary
    if [[ -f "$INSTALL_DIR/$BINARY_NAME" ]]; then
        log_info "Backing up existing binary..."
        cp "$INSTALL_DIR/$BINARY_NAME" "$INSTALL_DIR/${BINARY_NAME}.backup.$(date +%Y%m%d_%H%M%S)"
    fi
    
    # Install new binary
    cp "$APP_NAME" "$INSTALL_DIR/$BINARY_NAME"
    chown root:root "$INSTALL_DIR/$BINARY_NAME"
    chmod 755 "$INSTALL_DIR/$BINARY_NAME"
    
    log_success "Binary installed to $INSTALL_DIR/$BINARY_NAME"
}

setup_directories() {
    log_info "Setting up directories..."
    
    # Create config directory
    mkdir -p "$CONFIG_DIR"
    chown root:root "$CONFIG_DIR"
    chmod 755 "$CONFIG_DIR"
    
    log_success "Directories setup completed"
}

install_configs() {
    log_info "Installing configuration files..."
    
    # Install config files if they exist in the extracted package
    for config_file in "config.yaml" "config_tcp.yaml"; do
        if [[ -f "$config_file" ]]; then
            if [[ ! -f "$CONFIG_DIR/$config_file" ]]; then
                cp "$config_file" "$CONFIG_DIR/"
                chown root:root "$CONFIG_DIR/$config_file"
                chmod 644 "$CONFIG_DIR/$config_file"
                log_success "Installed $config_file to $CONFIG_DIR"
            else
                log_info "Configuration file $CONFIG_DIR/$config_file already exists, skipping"
            fi
        fi
    done
}

# Systemd service creation removed - users will run haxorport command directly

cleanup() {
    log_info "Cleaning up temporary files..."
    rm -rf "$TEMP_DIR"
}

show_usage() {
    echo "HaxorPort Go Client Installer v2"
    echo ""
    echo "Usage: $0 [OPTIONS] [download_url]"
    echo ""
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  --minimal      Minimal installation (same as default now)"
    echo ""
    echo "Arguments:"
    echo "  download_url   Optional. Download URL for the binary (default: https://haxorport.online/)"
    echo ""
    echo "Examples:"
    echo "  $0                                                    # Use default URL"
    echo "  $0 https://github.com/user/repo/releases/latest/download/"
    echo "  $0 --minimal https://releases.example.com/"
    echo "  $0 --minimal                                         # Use default URL, minimal install"
    echo ""
}

# Main installation function
main() {
    local download_url=""
    local minimal_install=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            --minimal)
                minimal_install=true
                shift
                ;;
            -*)
                log_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
            *)
                download_url="$1"
                shift
                ;;
        esac
    done
    
    # download_url is optional now, will use default if not provided
    
    log_info "Starting HaxorPort Go Client installation..."
    
    check_root
    check_dependencies
    detect_architecture
    download_binary "$download_url"
    extract_binary
    setup_directories
    install_binary
    install_configs
    
    # Service creation removed - users will run haxorport command directly
    
    cleanup
    
    log_success "Installation completed successfully!"
    log_info "Binary location: $INSTALL_DIR/$BINARY_NAME"
    log_info "Config directory: $CONFIG_DIR"
    
    log_info "Usage:"
    log_info "  Run HTTP tunnel: $BINARY_NAME http -p 80"
    log_info "  Run TCP tunnel:  $BINARY_NAME tcp -p 22"
    log_info "  Show help:       $BINARY_NAME --help"
}

# Run main function with all arguments
main "$@"
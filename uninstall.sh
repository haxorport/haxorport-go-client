#!/bin/bash

# Uninstaller for haxorport - Supports various operating systems
# This script will remove haxorport and all its files from the system

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Global variables - same as in install.sh for consistency
INSTALL_DIR="/opt/haxorport"
CONFIG_DIR="/etc/haxorport"
BIN_DIR="/usr/local/bin"
LOG_DIR="/var/log/haxorport"

# For macOS
if [[ "$OSTYPE" == "darwin"* ]]; then
    INSTALL_DIR="$HOME/Library/Application Support/haxorport"
    CONFIG_DIR="$HOME/Library/Preferences/haxorport"
    LOG_DIR="$HOME/Library/Logs/haxorport"
    
    # Check if using Apple Silicon
    if [[ $(uname -m) == "arm64" ]]; then
        # Use /opt/homebrew/bin for Apple Silicon if available
        if [ -d "/opt/homebrew/bin" ]; then
            BIN_DIR="/opt/homebrew/bin"
        else
            BIN_DIR="/usr/local/bin"
        fi
    else
        BIN_DIR="/usr/local/bin"
    fi
fi

# For Windows (WSL)
if grep -q Microsoft /proc/version 2>/dev/null; then
    INSTALL_DIR="$HOME/.haxorport"
    CONFIG_DIR="$HOME/.haxorport/config"
    LOG_DIR="$HOME/.haxorport/logs"
    BIN_DIR="$HOME/.local/bin"
fi

# Functions to display messages with colors
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to remove files and directories
remove_files() {
    local keep_configs=false
    
    # Check if we should keep configs
    if [ "$1" == "--keep-configs" ]; then
        keep_configs=true
    fi
    
    print_info "Removing application files..."
    
    # Remove application files
    if [[ "$OSTYPE" == "darwin"* ]] || grep -q Microsoft /proc/version 2>/dev/null; then
        # For macOS and WSL
        rm -rf "$INSTALL_DIR"
    else
        # For Linux, use sudo
        sudo rm -rf "$INSTALL_DIR"
    fi
    
    # Handle configuration files
    if [ -d "$CONFIG_DIR" ]; then
        if [ "$keep_configs" = true ]; then
            print_info "Keeping configuration files in $CONFIG_DIR"
        else
            print_info "Backing up configuration files..."
            local backup_dir="${CONFIG_DIR}_backup_$(date +%Y%m%d_%H%M%S)"
            
            # Create backup directory
            if [[ "$OSTYPE" == "darwin"* ]] || grep -q Microsoft /proc/version 2>/dev/null; then
                mkdir -p "$backup_dir"
                cp -r "$CONFIG_DIR/"* "$backup_dir/" 2>/dev/null || true
            else
                sudo mkdir -p "$backup_dir"
                sudo cp -r "$CONFIG_DIR/"* "$backup_dir/" 2>/dev/null || true
                sudo chown -R $(id -u):$(id -g) "$backup_dir"
            fi
            
            print_success "Configuration files backed up to: $backup_dir"
            
            # Remove config directory
            if [[ "$OSTYPE" == "darwin"* ]] || grep -q Microsoft /proc/version 2>/dev/null; then
                rm -rf "$CONFIG_DIR"
            else
                sudo rm -rf "$CONFIG_DIR"
            fi
        fi
    fi
    
    # Remove log files
    if [ -d "$LOG_DIR" ]; then
        print_info "Removing log files..."
        if [[ "$OSTYPE" == "darwin"* ]] || grep -q Microsoft /proc/version 2>/dev/null; then
            rm -rf "$LOG_DIR"
        else
            sudo rm -rf "$LOG_DIR"
        fi
    fi
    
    # Remove symlinks and binaries
    local removed_binaries=false
    
    if [ -L "$BIN_DIR/haxorport" ] || [ -f "$BIN_DIR/haxorport.bin" ]; then
        print_info "Removing binary symlinks..."
        if [[ "$OSTYPE" == "darwin"* ]] || grep -q Microsoft /proc/version 2>/dev/null; then
            rm -f "$BIN_DIR/haxorport"
            rm -f "$BIN_DIR/haxorport.bin"
            removed_binaries=true
        else
            sudo rm -f "$BIN_DIR/haxorport"
            sudo rm -f "$BIN_DIR/haxorport.bin"
            removed_binaries=true
        fi
    fi
    
    # Remove any remaining files in bin directory
    if [ -d "$BIN_DIR" ] && [ -z "$(ls -A "$BIN_DIR" 2>/dev/null)" ]; then
        if [[ "$OSTYPE" == "darwin"* ]] || grep -q Microsoft /proc/version 2>/dev/null; then
            rmdir "$BIN_DIR" 2>/dev/null || true
        else
            sudo rmdir "$BIN_DIR" 2>/dev/null || true
        fi
    fi
    
    if [ "$removed_binaries" = true ]; then
        print_success "All application files have been removed"
        if [ "$keep_configs" = true ]; then
            print_success "Configuration files have been preserved in $CONFIG_DIR"
        fi
    else
        print_warning "No application files were found to remove"
    fi
}

# Function to check if haxorport is running
check_running() {
    print_info "Checking if Haxorport is running..."
    
    # Check for running processes
    if pgrep -f "haxorport" > /dev/null; then
        print_warning "Haxorport is currently running. It will be terminated."
        
        if [[ "$OSTYPE" == "darwin"* ]] || grep -q Microsoft /proc/version 2>/dev/null; then
            # For macOS and WSL
            pkill -f "haxorport"
        else
            # For Linux
            sudo pkill -f "haxorport"
        fi
        
        # Wait a moment to ensure process is terminated
        sleep 1
        
        # Check again
        if pgrep -f "haxorport" > /dev/null; then
            print_error "Failed to terminate Haxorport processes. Please terminate them manually."
            print_info "You can use: pkill -f haxorport"
            return 1
        else
            print_success "Haxorport processes terminated."
        fi
    else
        print_info "No running Haxorport processes found."
    fi
    
    return 0
}

# Function to backup configuration
backup_config() {
    if [ -d "$CONFIG_DIR" ]; then
        print_info "Backing up configuration..."
        
        BACKUP_DIR="$HOME/haxorport-backup-$(date +%Y%m%d%H%M%S)"
        mkdir -p "$BACKUP_DIR"
        
        if [ -f "$CONFIG_DIR/config.yaml" ]; then
            cp -f "$CONFIG_DIR/config.yaml" "$BACKUP_DIR/"
            print_success "Configuration backed up to: $BACKUP_DIR/config.yaml"
        fi
    fi
}

# Main script
echo -e "${GREEN}=== Haxorport Uninstaller ===${NC}"
echo -e "This script will remove Haxorport from your system.\n"
echo -e "Detected operating system: ${YELLOW}$OSTYPE${NC}"

# Check if auto-confirm is enabled
AUTO_CONFIRM=false

# Parse command line arguments
KEEP_CONFIGS=false
AUTO_CONFIRM=false

for arg in "$@"; do
    case $arg in
        -y|--yes)
            AUTO_CONFIRM=true
            ;;
        --keep-configs)
            KEEP_CONFIGS=true
            ;;
    esac
done

# Check if script is being run through a pipe (non-interactive)
if [ ! -t 0 ]; then
    # Being run through a pipe (like curl | bash)
    AUTO_CONFIRM=true
    print_info "Detected non-interactive mode, proceeding with uninstallation automatically..."
fi

# Ask for confirmation if not auto-confirmed
if [ "$AUTO_CONFIRM" != "true" ]; then
    echo -e "${YELLOW}Warning: This will remove Haxorport and all its files from your system.${NC}"
    read -p "Do you want to backup your configuration before uninstalling? (y/n): " backup
    if [[ $backup == [yY] ]]; then
        backup_config
    fi
    
    read -p "Continue uninstallation? (y/n): " confirm
    if [[ $confirm != [yY] ]]; then
        print_warning "Uninstallation canceled."
        exit 0
    fi
else
    # Auto backup in non-interactive mode
    backup_config
fi

# Run uninstallation functions
check_running

if [ "$KEEP_CONFIGS" = true ]; then
    print_info "Keeping configuration files as requested..."
    remove_files --keep-configs
else
    remove_files
fi

echo -e "\n${GREEN}Haxorport has been successfully uninstalled from your system.${NC}"
echo -e "Thank you for using Haxorport!\n"

exit 0

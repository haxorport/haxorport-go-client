#!/bin/bash

# Installer for haxorport - Supports various operating systems
# This script will install haxorport and all its dependencies

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Global variables
REPO_URL="https://github.com/haxorport/haxorport-go-client.git"
REPO_BRANCH="main"
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
        print_info "Detected Apple Silicon (M1/M2)"
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
    mkdir -p "$BIN_DIR"
    export PATH="$PATH:$BIN_DIR"
    
    # Add BIN_DIR to PATH permanently if not already there
    if ! grep -q "$BIN_DIR" "$HOME/.bashrc" 2>/dev/null; then
        print_info "Adding $BIN_DIR to PATH in .bashrc"
        echo "export PATH=\"$PATH:$BIN_DIR\"" >> "$HOME/.bashrc"
    fi
    
    # If using zsh, also add to .zshrc
    if [ -f "$HOME/.zshrc" ] && ! grep -q "$BIN_DIR" "$HOME/.zshrc" 2>/dev/null; then
        print_info "Adding $BIN_DIR to PATH in .zshrc"
        echo "export PATH=\"$PATH:$BIN_DIR\"" >> "$HOME/.zshrc"
    fi
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

# Function to check if a command is available
check_command() {
    if ! command -v $1 &> /dev/null; then
        print_error "$1 not found."
        return 1
    else
        print_info "$1 is already installed."
        return 0
    fi
}

# Function to install dependencies based on OS
install_dependencies() {
    print_info "Checking and installing dependencies..."
    
    # Check Go
    if ! check_command go; then
        print_info "Go not found. Installing Go..."
        
        # Detect OS and install Go
        if [[ "$OSTYPE" == "linux-gnu"* ]]; then
            # Detect package manager
            if command -v apt-get &> /dev/null; then
                # Debian/Ubuntu
                sudo apt-get update
                sudo apt-get install -y golang-go
            elif command -v yum &> /dev/null; then
                # CentOS/RHEL
                sudo yum install -y golang
            elif command -v pacman &> /dev/null; then
                # Arch Linux
                sudo pacman -Sy go
            elif command -v zypper &> /dev/null; then
                # openSUSE
                sudo zypper install -y go
            else
                print_error "Package manager not recognized. Please install Go manually."
                exit 1
            fi
        elif [[ "$OSTYPE" == "darwin"* ]]; then
            # macOS
            if command -v brew &> /dev/null; then
                brew install go
            else
                print_error "Homebrew not found. Please install Homebrew first."
                print_info "Run: /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
                
                # Add special instructions for Apple Silicon
                if [[ $(uname -m) == "arm64" ]]; then
                    print_info "For Apple Silicon (M1/M2), after installing Homebrew, run:"
                    print_info "echo 'eval \"\$(/opt/homebrew/bin/brew shellenv)\"' >> ~/.zprofile"
                    print_info "eval \"\$(/opt/homebrew/bin/brew shellenv)\""
                fi
                
                exit 1
            fi
        else
            print_error "Operating system not supported for automatic installation. Please install Go manually."
            exit 1
        fi
    fi
    
    # Check Git
    if ! check_command git; then
        print_info "Git not found. Installing Git..."
        
        # Detect OS and install Git
        if [[ "$OSTYPE" == "linux-gnu"* ]]; then
            # Detect package manager
            if command -v apt-get &> /dev/null; then
                # Debian/Ubuntu
                sudo apt-get update
                sudo apt-get install -y git
            elif command -v yum &> /dev/null; then
                # CentOS/RHEL
                sudo yum install -y git
            elif command -v pacman &> /dev/null; then
                # Arch Linux
                sudo pacman -Sy git
            elif command -v zypper &> /dev/null; then
                # openSUSE
                sudo zypper install -y git
            else
                print_error "Package manager not recognized. Please install Git manually."
                exit 1
            fi
        elif [[ "$OSTYPE" == "darwin"* ]]; then
            # macOS
            if command -v brew &> /dev/null; then
                brew install git
            else
                print_error "Homebrew not found. Please install Homebrew first."
                exit 1
            fi
        else
            print_error "Operating system not supported for automatic installation. Please install Git manually."
            exit 1
        fi
    fi
    
    print_success "All dependencies successfully installed."
}

# Function to set up repository
setup_repository() {
    print_info "Setting up installation..."
    
    # Check if current directory is a valid Haxorport project
    if [ -f "go.mod" ] && [ -f "main.go" ]; then
        # Use current directory as installation source
        print_info "Using current directory as installation source: $PWD"
        REPO_DIR="$PWD"
    else
        # Clone repository from GitHub
        print_info "Current directory is not a valid Haxorport project. Downloading from GitHub..."
        
        # Create temporary directory
        TEMP_DIR="$(mktemp -d)"
        print_info "Created temporary directory: $TEMP_DIR"
        
        # Clone repository
        print_info "Cloning repository from $REPO_URL..."
        git clone --depth 1 -b $REPO_BRANCH $REPO_URL "$TEMP_DIR" || {
            print_error "Failed to clone repository."
            exit 1
        }
        
        # Set REPO_DIR to cloned repository
        REPO_DIR="$TEMP_DIR"
        print_info "Repository cloned to: $REPO_DIR"
    fi
    
    # Change to repository directory
    cd "$REPO_DIR"
    
    # Check if directory is a git repository
    if [ -d ".git" ]; then
        print_info "Using existing Git repository..."
        
        # Update repository
        git fetch --all
        
        # Checkout to specified branch
        CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
        if [ "$CURRENT_BRANCH" != "$REPO_BRANCH" ]; then
            print_info "Checking out to branch $REPO_BRANCH..."
            git checkout $REPO_BRANCH
            git pull origin $REPO_BRANCH
        else
            print_info "Already on branch $REPO_BRANCH. Updating..."
            git pull origin $REPO_BRANCH
        fi
    fi
}

# Function to compile the application
build_application() {
    print_info "Building application..."
    
    cd "$REPO_DIR"
    
    # Create bin directory if it doesn't exist
    mkdir -p bin
    
    # Clean any previous builds
    print_info "Cleaning previous builds..."
    rm -f bin/haxorport bin/haxor
    
    # Build the application
    print_info "Compiling Go code from branch $REPO_BRANCH..."
    go build -o bin/haxorport main.go
    
    if [ $? -ne 0 ]; then
        print_error "Failed to compile application."
        exit 1
    fi
    
    # Create symlink for backward compatibility
    ln -sf bin/haxorport bin/haxor
    
    print_success "Application successfully compiled from branch $REPO_BRANCH."
    cd ..
}

# Function to validate configuration
validate_config() {
    local config_file="$1"
    
    if [ ! -f "$config_file" ]; then
        print_error "Configuration file not found: $config_file"
        return 1
    fi
    
    # Check required fields
    local required_fields=("auth_token" "server_address" "connection_mode" "control_port")
    for field in "${required_fields[@]}"; do
        if ! grep -q "^$field:" "$config_file"; then
            print_error "Missing required field in config: $field"
            return 1
        fi
    done
    
    # Validate connection mode
    local connection_mode=$(grep "^connection_mode:" "$config_file" | awk '{print $2}' | tr -d '"' | tr -d ' ')
    if [[ "$connection_mode" != "websocket" && "$connection_mode" != "direct_tcp" ]]; then
        print_error "Invalid connection_mode: $connection_mode. Must be 'websocket' or 'direct_tcp'"
        return 1
    fi
    
    return 0
}

# Function to install the application
install_application() {
    print_info "Installing application..."
    
    cd "$REPO_DIR"
    
    # Create necessary directories with proper permissions
    if [[ "$OSTYPE" == "darwin"* ]] || grep -q Microsoft /proc/version 2>/dev/null; then
        # For macOS and WSL, no sudo needed
        mkdir -p "$INSTALL_DIR/bin" "$CONFIG_DIR" "$LOG_DIR"
        chmod 755 "$CONFIG_DIR" "$LOG_DIR"
    else
        # For Linux, use sudo to create directories
        sudo mkdir -p "$INSTALL_DIR/bin" "$CONFIG_DIR" "$LOG_DIR"
        sudo chmod 755 "$CONFIG_DIR" "$LOG_DIR"
        # Set ownership to current user if not root
        if [ "$(id -u)" != "0" ]; then
            sudo chown -R $(id -u):$(id -g) "$INSTALL_DIR" "$CONFIG_DIR" "$LOG_DIR"
        fi
    fi
    
    # Copy default configs if they don't exist or update if validation fails
    for config_file in config.yaml config_tcp.yaml; do
        if [ -f "$REPO_DIR/$config_file" ]; then
            config_needs_update=false
            
            # Check if file exists
            if [ ! -f "$CONFIG_DIR/$config_file" ]; then
                config_needs_update=true
            else
                # Validate existing configuration
                print_info "Validating configuration: $CONFIG_DIR/$config_file"
                if ! validate_config "$CONFIG_DIR/$config_file"; then
                    print_warning "Configuration validation failed. Will update with new template."
                    config_needs_update=true
                    
                    # Create backup of existing config
                    backup_file="$CONFIG_DIR/${config_file}.$(date +%Y%m%d%H%M%S).bak"
                    cp "$CONFIG_DIR/$config_file" "$backup_file"
                    print_info "Created backup of existing config: $backup_file"
                else
                    print_success "Configuration is valid"
                fi
            fi
            
            # Update config if needed
            if [ "$config_needs_update" = true ]; then
                cp "$REPO_DIR/$config_file" "$CONFIG_DIR/"
                print_success "Updated configuration file: $CONFIG_DIR/$config_file"
                
                # Set proper permissions
                if [[ "$OSTYPE" != "darwin"* ]] && ! grep -q Microsoft /proc/version 2>/dev/null; then
                    sudo chmod 644 "$CONFIG_DIR/$config_file"
                fi
            else
                print_info "Configuration file already exists and is valid: $CONFIG_DIR/$config_file"
            fi
        else
            print_warning "Default configuration file not found in repository: $config_file"
        fi
    done
    
    # Set permissions for installation directory
    if [[ "$OSTYPE" != "darwin"* ]] && ! grep -q Microsoft /proc/version 2>/dev/null; then
        sudo chmod 755 "$INSTALL_DIR"
        sudo chmod 755 "$INSTALL_DIR/bin"
    fi
    
    # Check if binary exists from build script
    if [ -f "$REPO_DIR/bin/haxorport" ]; then
        print_info "Using pre-built binary from build script..."
        if [[ "$OSTYPE" == "darwin"* ]] || grep -q Microsoft /proc/version 2>/dev/null; then
            # For macOS and WSL, no sudo needed
            cp -f "$REPO_DIR/bin/haxorport" "$INSTALL_DIR/bin/haxorport"
            cp -f "$REPO_DIR/bin/haxorport" "$BIN_DIR/haxorport.bin"
        else
            # For Linux, use sudo
            sudo cp -f "$REPO_DIR/bin/haxorport" "$INSTALL_DIR/bin/haxorport"
            sudo cp -f "$REPO_DIR/bin/haxorport" "$BIN_DIR/haxorport.bin"
        fi
    elif [ -f "$REPO_DIR/bin/haxor" ]; then
        print_info "Using pre-built binary from build script (haxor)..."
        if [[ "$OSTYPE" == "darwin"* ]] || grep -q Microsoft /proc/version 2>/dev/null; then
            # For macOS and WSL, no sudo needed
            cp -f "$REPO_DIR/bin/haxor" "$INSTALL_DIR/bin/haxorport"
            cp -f "$REPO_DIR/bin/haxor" "$BIN_DIR/haxorport.bin"
        else
            # For Linux, use sudo
            sudo cp -f "$REPO_DIR/bin/haxor" "$INSTALL_DIR/bin/haxorport"
            sudo cp -f "$REPO_DIR/bin/haxor" "$BIN_DIR/haxorport.bin"
        fi
    else
        # Build application if no binary exists
        print_info "No pre-built binary found, building from source..."
        if [[ "$OSTYPE" == "darwin"* ]] || grep -q Microsoft /proc/version 2>/dev/null; then
            # For macOS and WSL, build directly to install dir
            if ! go build -o "$INSTALL_DIR/bin/haxorport" .; then
                print_error "Failed to compile application"
                exit 1
            fi
            cp -f "$INSTALL_DIR/bin/haxorport" "$BIN_DIR/haxorport.bin"
        else
            # For Linux, build to temp location then copy with sudo
            if ! go build -o "haxorport.tmp" .; then
                print_error "Failed to compile application"
                exit 1
            fi
            sudo cp -f "haxorport.tmp" "$INSTALL_DIR/bin/haxorport"
            sudo cp -f "haxorport.tmp" "$BIN_DIR/haxorport.bin"
            rm -f "haxorport.tmp"
        fi
    fi
    
    # Set permission for binaries
    if [[ "$OSTYPE" == "darwin"* ]] || grep -q Microsoft /proc/version 2>/dev/null; then
        # For macOS and WSL, no sudo needed
        chmod +x "$INSTALL_DIR/bin/haxorport"
        chmod +x "$BIN_DIR/haxorport.bin"
    else
        # For Linux, use sudo
        sudo chmod +x "$INSTALL_DIR/bin/haxorport"
        sudo chmod +x "$BIN_DIR/haxorport.bin"
    fi
    
    # Adjust log configuration for OS
    for config_file in "$CONFIG_DIR/config.yaml" "$CONFIG_DIR/config_tcp.yaml"; do
        if [ -f "$config_file" ]; then
            if [[ "$OSTYPE" == "darwin"* ]] || grep -q Microsoft /proc/version 2>/dev/null; then
                # For macOS and WSL, use path relative to home
                sed -i '' "s|logs/|$LOG_DIR/|g" "$config_file"
            else
                # For Linux, use default path
                sed -i "s|logs/|$LOG_DIR/|g" "$config_file"
            fi
            print_info "Updated log path in $(basename "$config_file") to $LOG_DIR"
        fi
    done
    
    print_info "Log level set to 'warn' to reduce debug output"
    
    # Create wrapper script for haxorport with debugging
    cat > haxorport << 'EOF'
#!/bin/bash

# Debug information
debug() {
    if [ "$HAXORPORT_DEBUG" = "true" ]; then
        echo "DEBUG: $1" >&2
    fi
}

# Debug mode is disabled by default for better user experience
export HAXORPORT_DEBUG=false

# Hardcoded paths based on installation
INSTALL_DIR="/opt/haxorport"
CONFIG_DIR="/etc/haxorport"
BIN_DIR="/usr/local/bin"

# For macOS
if [[ "$OSTYPE" == "darwin"* ]]; then
    INSTALL_DIR="$HOME/Library/Application Support/haxorport"
    CONFIG_DIR="$HOME/Library/Preferences/haxorport"
    
    # Check if using Apple Silicon
    if [[ $(uname -m) == "arm64" ]]; then
        BIN_DIR="/opt/homebrew/bin"
    else
        BIN_DIR="/usr/local/bin"
    fi
fi

# For Windows (WSL)
if grep -q Microsoft /proc/version 2>/dev/null; then
    INSTALL_DIR="$HOME/.haxorport"
    CONFIG_DIR="$HOME/.haxorport/config"
    BIN_DIR="$HOME/.local/bin"
fi

# Show paths being checked
debug "Checking for binary at $BIN_DIR/haxorport.bin"
debug "Checking for binary at $INSTALL_DIR/bin/haxorport"

# Print command arguments for debugging
debug "Command arguments: $@"

# Determine which config file to use based on command
CONFIG_FILE="$CONFIG_DIR/config.yaml"
if [ "$1" = "tcp" ]; then
    CONFIG_FILE="$CONFIG_DIR/config_tcp.yaml"
    debug "TCP command detected, using config_tcp.yaml"
fi

debug "Using config file: $CONFIG_FILE"

# Use the direct binary path if it exists
if [ -f "$BIN_DIR/haxorport.bin" ]; then
    debug "Using binary at $BIN_DIR/haxorport.bin"
    debug "Running: $BIN_DIR/haxorport.bin --config $CONFIG_FILE $@"
    "$BIN_DIR/haxorport.bin" --config "$CONFIG_FILE" "$@"
    exit $?
fi

# Check installation directory
if [ -f "$INSTALL_DIR/bin/haxorport" ]; then
    debug "Using binary at $INSTALL_DIR/bin/haxorport"
    debug "Running: $INSTALL_DIR/bin/haxorport --config $CONFIG_FILE $@"
    "$INSTALL_DIR/bin/haxorport" --config "$CONFIG_FILE" "$@"
    exit $?
fi

# If we get here, no binary was found
echo "Error: haxorport binary not found at $INSTALL_DIR/bin/haxorport or $BIN_DIR/haxorport.bin"
echo "Please reinstall the application using: curl -sSL https://raw.githubusercontent.com/haxorport/haxorport-go-client/main/install.sh | bash"
exit 1
EOF
    
    # Give execution permission to wrapper script
    chmod +x haxorport
    
    # Copy wrapper script to bin directory
    if [[ "$OSTYPE" == "darwin"* ]] || grep -q Microsoft /proc/version 2>/dev/null; then
        # For macOS and WSL, no sudo needed
        cp haxorport "$BIN_DIR/haxorport"
    else
        # For Linux
        sudo cp haxorport "$BIN_DIR/haxorport"
    fi
    
    # Check if installation was successful
    if [ ! -f "$BIN_DIR/haxorport" ]; then
        print_error "Failed to install application."
        exit 1
    else
        print_success "Application successfully installed."
    fi
    
    cd ..
}

# Function to update the application
update_application() {
    print_info "Updating application..."
    
    # Update repository
    setup_repository
    
    # Rebuild application
    build_application
    
    # Reinstall application
    install_application
    
    print_success "Application successfully updated."
}

# Function to display usage information
show_usage() {
    echo -e "\n${GREEN}Haxorport successfully installed!${NC}"
    echo -e "\nUsage:"
    echo -e "  ${BLUE}haxorport auth-token YOUR_AUTH_TOKEN${NC} - Set authentication token"
    echo -e "  ${BLUE}haxorport http -p 2712${NC} - Create HTTP tunnel to localhost:2712"
    echo -e "  ${BLUE}haxorport tcp -p 22${NC} - Create TCP tunnel to port 22"
    echo -e "  ${BLUE}haxorport --help${NC} - Display help"
    
    echo -e "\nConfiguration:"
    echo -e "  Configuration file: ${YELLOW}$CONFIG_DIR/config.yaml${NC}"
    echo -e "  Log file: ${YELLOW}$LOG_DIR/haxorport-client.log${NC}"
    
    echo -e "\nTo update the application, run: ${YELLOW}$0 --update${NC}"
    
    if [[ "$OSTYPE" == "darwin"* ]] || grep -q Microsoft /proc/version 2>/dev/null; then
        echo -e "To uninstall, run: ${YELLOW}rm -rf $INSTALL_DIR $CONFIG_DIR $BIN_DIR/haxorport${NC}"
    else
        echo -e "To uninstall, run: ${YELLOW}sudo rm -rf $INSTALL_DIR $CONFIG_DIR $BIN_DIR/haxorport${NC}"
    fi
}

# Main script
echo -e "${GREEN}=== Haxorport Multi-Platform Installer ===${NC}"
echo -e "This installer will install Haxorport and all its dependencies.\n"
echo -e "Detected operating system: ${YELLOW}$OSTYPE${NC}"

# Check if this is an update request or auto-confirm request
if [ "$1" == "--update" ]; then
    update_application
    show_usage
    exit 0
fi

# Check if auto-confirm is enabled or if script is running through pipe
AUTO_CONFIRM=false

# Check for -y or --yes flags
for arg in "$@"; do
    if [ "$arg" == "-y" ] || [ "$arg" == "--yes" ]; then
        AUTO_CONFIRM=true
    fi
done

# Check if script is being run through a pipe (non-interactive)
if [ ! -t 0 ]; then
    # Being run through a pipe (like curl | bash)
    AUTO_CONFIRM=true
    print_info "Detected non-interactive mode, proceeding with installation automatically..."
fi

# Ask for confirmation if not auto-confirmed
if [ "$AUTO_CONFIRM" != "true" ]; then
    read -p "Continue installation? (y/n): " confirm
    if [[ $confirm != [yY] ]]; then
        print_warning "Installation canceled."
        exit 0
    fi
fi

# Run installation functions
install_dependencies
setup_repository
build_application
install_application
show_usage

exit 0

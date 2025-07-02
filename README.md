# 🚀 Haxorport Client

Haxorport Client is a client application for the Haxorport service, allowing you to create HTTP and TCP tunnels to expose local services to the internet.

## ✨ Features

- 🌐 **HTTP/HTTPS Tunnels**: Expose local web services with custom subdomains, supporting both HTTP and HTTPS protocols
- 🔌 **TCP Tunnels**: Expose local TCP services with remote ports
- 🔒 **Authentication**: Protect tunnels with basic or header authentication
- ⚙️ **Configuration**: Easily manage configuration through CLI
- 🔄 **Automatic Reconnection**: Connections will automatically reconnect if disconnected
- 🔀 **Multiple Connection Modes**: Support for WebSocket and Direct TCP connection modes
- 🔐 **TLS Support**: Secure connections with TLS for HTTP tunnels

## 🏗️ Architecture

Haxorport Client is built with a hexagonal architecture (ports and adapters) that separates business domain from technical infrastructure. This architecture enables:

1. **Separation of Concerns**: Business domain is separated from technical details
2. **Testability**: Components can be tested separately
3. **Flexibility**: Infrastructure implementations can be replaced without changing the business domain

Project structure:

```
haxorport-client/
├── cmd/                    # Command-line interface
├── internal/               # Internal code
│   ├── domain/             # Domain layer
│   │   ├── model/          # Domain models
│   │   └── port/           # Ports (interfaces)
│   ├── application/        # Application layer
│   │   └── service/        # Services
│   ├── infrastructure/     # Infrastructure layer
│   │   ├── config/         # Configuration implementation
│   │   ├── transport/      # Communication implementation
│   │   └── logger/         # Logger implementation
│   └── di/                 # Dependency injection
├── scripts/                # Build and run scripts
└── main.go                 # Entry point
```

## 💻 Installation

### 🚀 Easy Installation (All OS)

#### Option 1: Using Install Script (Recommended)

```bash
# Download and run the installer
curl -sSL https://raw.githubusercontent.com/haxorport/haxorport-go-client/main/install.sh | bash

# Or with sudo (if needed)
# curl -sSL https://raw.githubusercontent.com/haxorport/haxorport-go-client/main/install.sh | sudo bash
```

#### Option 2: Manual Installation

1. **Download the latest release**
   ```bash
   # Linux (64-bit)
   wget https://github.com/haxorport/haxorport-go-client/releases/latest/download/haxorport-linux-amd64.tar.gz
   
   # macOS (Intel)
   wget https://github.com/haxorport/haxorport-go-client/releases/latest/download/haxorport-darwin-amd64.tar.gz
   
   # macOS (Apple Silicon)
   wget https://github.com/haxorport/haxorport-go-client/releases/latest/download/haxorport-darwin-arm64.tar.gz
   ```

2. **Extract the package**
   ```bash
   tar -xzf haxorport-*.tar.gz
   cd haxorport-*
   ```

3. **Run the installer**
   ```bash
   ./install.sh
   ```

4. **Verify installation**
   ```bash
   haxorport --version
   ```

## 🔧 Configuration

Haxorport supports two types of tunnels, each with its own configuration file:

### 1. HTTP/HTTPS Tunnel (WebSocket Mode)

Default config file: `/etc/haxorport/config.yaml`

```yaml
# Authentication
auth_enabled: true
auth_token: "your-auth-token-here"
auth_validation_url: "https://haxorport.online/AuthToken/validate"

# Server Configuration
server_address: "control.haxorport.online"
control_port: 443
connection_mode: "websocket"  # Must be 'websocket' for HTTP tunnels
tls_enabled: true

# Tunnel Configuration
tunnels:
  - name: "my-web-app"
    type: "http"
    local_address: "localhost:8080"
    subdomain: "myapp"  # Optional, auto-generated if empty

# Logging
log_level: "info"  # debug, info, warn, error
log_file: "/var/log/haxorport/haxorport-client.log"
```

### 2. TCP Tunnel (Direct TCP Mode)

Config file: `/etc/haxorport/config_tcp.yaml`

```yaml
# Authentication
auth_enabled: true
auth_token: "your-auth-token-here"  # Required for all TCP tunnels
auth_validation_url: "https://haxorport.online/AuthToken/validate"

# Server Configuration
server_address: "control.haxorport.online"
control_port: 7000
connection_mode: "direct_tcp"  # Must be 'direct_tcp' for TCP tunnels
tls_enabled: false

# Tunnel Configuration
tunnels:
  - name: "my-tcp-service"
    type: "tcp"
    local_address: "localhost:22"
    remote_port: 2222  # Port on the remote server

# Logging
log_level: "info"
log_file: "/var/log/haxorport/haxorport-tcp-client.log"
```

**Important Note**: For TCP tunnels, authentication token is **always required** regardless of the `auth_enabled` setting. The token will be validated against the authentication server on each connection.

### Configuration Locations

- **Linux/Unix**: `/etc/haxorport/`
- **macOS**: `~/Library/Preferences/haxorport/`
- **Windows (WSL)**: `~/.haxorport/config/`

## 🚀 Usage

### Starting Tunnels

```bash
# Start HTTP tunnel (WebSocket mode)
haxorport http

# Start TCP tunnel (Direct TCP mode)
haxorport tcp

# Start with custom config file
haxorport http --config /path/to/custom-config.yaml

# Enable debug logging
LOG_LEVEL=debug haxorport http
```

### Managing Tunnels

```bash
# List active tunnels
haxorport list

# Stop all tunnels
haxorport stop
```

### Uninstallation

```bash
# Standard uninstallation (removes all files including configs)
sudo uninstall.sh

# Keep configuration files
sudo uninstall.sh --keep-configs

# Non-interactive uninstallation
sudo uninstall.sh --yes
```

## 🔄 Connection Modes

### WebSocket Mode (for HTTP/HTTPS Tunnels)
- Uses WebSocket protocol over port 443 (HTTPS)
- Recommended for web applications
- Supports TLS encryption
- Better compatibility with firewalls and proxies
- Validates auth token during WebSocket connection establishment

### Direct TCP Mode (for TCP Tunnels)
- Uses raw TCP connections
- Lower latency for non-HTTP traffic
- No TLS overhead (for better performance)
- Requires direct TCP access to the server
- **Always requires valid authentication token**
- Validates token via HTTP API before establishing tunnel

## 🔒 Security Considerations

1. **Authentication Token Required**: All TCP tunnels require a valid authentication token regardless of the `auth_enabled` setting
2. **Token Validation**: Every token is validated against the authentication server on each connection
3. **Subscription Limits**: The system enforces tunnel limits based on your subscription plan
4. **Never expose sensitive services** without proper authentication
5. **Regularly update** to the latest version
6. **Monitor logs** for suspicious activity
7. **Use TLS** for all HTTP traffic

## 🐛 Troubleshooting

### Common Issues

1. **Connection refused**
   - Check if the server is running
   - Verify firewall settings
   - Ensure the correct port is being used

2. **Authentication failed**
   - Verify your auth token in the appropriate config file (`config.yaml` for HTTP or `config_tcp.yaml` for TCP)
   - Check if the token has necessary permissions and hasn't expired
   - Ensure the validation URL is correct
   - For TCP tunnels, remember that auth token is always required regardless of `auth_enabled` setting
   - Check if you've reached your subscription's tunnel limit

3. **Port already in use**
   - Check for other instances of haxorport
   - Use a different remote port

### Viewing Logs

```bash
# Default log locations
# Linux/Unix: /var/log/haxorport/
# macOS: ~/Library/Logs/haxorport/
# Windows (WSL): ~/.haxorport/logs/

tail -f /var/log/haxorport/haxorport-client.log
```

### Enabling Debug Mode

```bash
# For more verbose output
LOG_LEVEL=debug haxorport http
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Thanks to all contributors who have helped with this project
- Special thanks to the open-source community for their invaluable resources

The script will automatically detect if it's being run through a pipe and will skip confirmation if so.

The installer script will:
- 🔎 Automatically detect your OS
- 📚 Install required dependencies
- 💿 Compile and install haxorport
- 📝 Create default configuration

After installation, you can immediately use the `haxorport` command.

### 🔑 Authentication Setup

Before using Haxorport, you need to set your authentication token:

```bash
# Set your authentication token
haxorport auth-token YOUR_AUTH_TOKEN
```

This token is required for connecting to the Haxorport service and creating tunnels.

## 🔧 Tunnel Configuration

Haxorport supports two types of tunnels with specific configurations:

### 🔄 Connection Modes

Haxorport supports two connection modes, each optimized for different use cases:

1. **WebSocket Mode** (`websocket`)
   - Used for HTTP/HTTPS tunnels
   - Provides better compatibility with proxies and firewalls
   - Supports automatic reconnection
   - Uses TLS encryption (HTTPS/WSS)
   - Connects to port 443 by default

2. **Direct TCP Mode** (`direct_tcp`)
   - Used for TCP tunnels (SSH, RDP, etc.)
   - Lower latency for raw TCP traffic
   - Better for protocols that don't work well over WebSocket
   - Connects to port 7000 by default
   - No TLS encryption (for maximum performance)

### HTTP Tunnel Configuration

HTTP tunnels use the following configuration (in `config.yaml` or `~/.haxorport/config.yaml`):

```yaml
# Required Settings
auth_enabled: true
auth_token: "YOUR_AUTH_TOKEN"
auth_validation_url: https://haxorport.online/AuthToken/validate
connection_mode: "websocket"  # Must be websocket for HTTP tunnels
server_address: control.haxorport.online
control_port: 443
tls_enabled: true  # Always enabled for WebSocket connections

# Optional Settings
base_domain: ""  # Leave empty to use default domain
log_level: "info"  # debug, info, warn, error
log_file: "logs/haxorport-client.log"

# Advanced Settings
data_port: 0  # 0 = auto-select
```

### TCP Tunnel Configuration

TCP tunnels use the following configuration (in `config_tcp.yaml` or `~/.haxorport/config_tcp.yaml`):

```yaml
# Required Settings
auth_enabled: true
auth_token: "YOUR_AUTH_TOKEN"
auth_validation_url: https://haxorport.online/AuthToken/validate
connection_mode: "direct_tcp"  # Must be direct_tcp for TCP tunnels
server_address: control.haxorport.online
control_port: 7000
tls_enabled: false  # Disabled for direct TCP connections

# Optional Settings
base_domain: "haxorport.online"
log_level: "info"  # debug, info, warn, error
log_file: "logs/haxorport-client-tcp.log"

# Advanced Settings
data_port: 0  # 0 = auto-select
```

### Important Notes

- **HTTP/HTTPS Tunnels**:
  - Must use WebSocket connection mode
  - Always use TLS encryption (HTTPS/WSS)
  - Connect to port 443 by default

- **TCP Tunnels**:
  - Must use Direct TCP connection mode
  - No TLS encryption (for better performance with raw TCP)
  - Connect to port 7000 by default
  - Ideal for SSH, RDP, and other TCP-based protocols

- **Common Settings**:
  - Set `LOG_LEVEL=debug` environment variable to enable detailed logging
  - Use `base_domain` to specify a custom domain for your tunnels
  - `data_port` is typically left as 0 for auto-selection
  - Always ensure your auth token is kept secure and never committed to version control

### 🔧 Manual Installation

#### 📂 From Source

1. Clone the repository:
   ```bash
   git clone https://github.com/haxorport/haxorport-go-client.git
   cd haxorport-go-client
   ```

2. Build the application:
   ```bash
   # Make sure Go is installed
   go build -o bin/haxorport main.go
   ```

3. (Optional) Move the binary to a directory in your PATH:
   ```bash
   # Linux/macOS
   sudo cp bin/haxorport /usr/local/bin/
   
   # Windows (PowerShell Admin)
   Copy-Item .\bin\haxorport.exe -Destination "$env:ProgramFiles\haxorport\"
   ```

### 🔧 Manual Installation

---

### 🛠️ Installer & Build Scripts Documentation

This project provides several installer and build scripts to support different installation and deployment scenarios:

#### 1. `install.sh`
- **Purpose:** Universal installer for Linux, macOS, and Windows (via WSL).
- **Usage:**
  ```bash
  curl -sSL https://raw.githubusercontent.com/haxorport/haxorport-go-client/main/install.sh | bash
  # or run directly if already available locally
  bash install.sh
  ```
- **Description:**
  - Automatically detects your OS and installs required dependencies.
  - Builds and installs the `haxorport` binary.
  - Creates a default configuration file.
  - Recommended for most users who want a quick and easy setup.

#### 2. `uninstall.sh`
- **Purpose:** Clean uninstaller for removing Haxorport from your system.
- **Usage:**
  ```bash
  # Interactive mode (with prompts)
  curl -sSL https://raw.githubusercontent.com/haxorport/haxorport-go-client/main/uninstall.sh | bash
  # or run directly if already available locally
  bash uninstall.sh
  
  # Non-interactive mode (no prompts)
  bash uninstall.sh -y
  ```
- **Description:**
  - Automatically detects your OS and removes all Haxorport files.
  - Terminates any running Haxorport processes.
  - Offers to backup your configuration before removal.
  - Supports non-interactive mode with `-y` or `--yes` flags.

#### 3. `build.sh`
- **Purpose:** Build and interactive configuration setup for developers or advanced users.
- **Usage:**
  ```bash
  bash build.sh
  ```
- **Description:**
  - Builds the project from source.
  - Interactively asks for your authentication token and writes it to the configuration file.
  - Backs up any existing configuration.
  - Recommended for development or manual setup scenarios.

#### 4. `build-package.sh`
- **Purpose:** Package the application and its configuration into a tar.gz archive for easy distribution (e.g., for VPS/server deployment).
- **Usage:**
  ```bash
  bash build-package.sh
  ```
- **Description:**
  - Builds the binary for Linux amd64
  - Copies configuration and installation scripts
  - Creates a tar.gz package ready for distribution

#### 5. Manual Installation from tar.gz Package

If you have a `haxorport-go-client-linux-amd64.tar.gz` file (created with build-package.sh), you can install it manually:

1. **Extract the file:**
   ```bash
   tar -xzvf haxorport-go-client-linux-amd64.tar.gz
   ```

2. **Move the binary to system directory:**
   ```bash
   sudo mv haxorport /usr/local/bin/
   ```

3. **Create configuration directory:**
   ```bash
   sudo mkdir -p /etc/haxorport
   ```

4. **Copy configuration file:**
   ```bash
   sudo cp config.yaml /etc/haxorport/
   ```

5. **Set execution permissions:**
   ```bash
   sudo chmod +x /usr/local/bin/haxorport
   ```

6. **Verify installation:**
   ```bash
   haxorport --version
   ```
  - Builds the Linux binary, copies config and installer script into a `dist/` directory.
  - Packages everything into a single `.tar.gz` archive.
  - Recommended for creating portable packages for server or VPS installation.

---


1. Download the latest binary from [releases](https://github.com/haxorport/haxorport-go-client/releases)
2. Extract and move it to a directory in your PATH

## 💬 Usage

### ⚙️ Configuration

Before using Haxorport Client, you need to set up the configuration:

#### 🔑 Getting an Auth Token

To obtain an auth-token, you must first register at:

**[https://haxorport.online/Register](https://haxorport.online/Register)**

After registering and logging in, you can find your auth-token in your account dashboard.

#### 📝 Setting Up Configuration

```
haxorport config set server_address control.haxorport.online
haxorport config set control_port 443
haxorport config set auth_token your-auth-token
haxorport config set tls_enabled true
```

Or use the easier method with the command:

```
./build.sh config
```

To view the current configuration:

```
haxorport config show
```

### 🌐 HTTP Tunnel

Create an HTTP tunnel for a local web service:

```
haxorport http --port 8080 --subdomain myapp
```

With basic authentication:

```
haxorport http --port 8080 --subdomain myapp --auth basic --username user --password pass
```

With header authentication:

```
haxorport http --port 8080 --subdomain myapp --auth header --header "X-API-Key" --value "secret-key"
```

### 🔒 HTTPS Tunnel

Haxorport now supports HTTPS tunnels automatically with a reverse connection architecture. When the client connects to the server, the server detects whether the request comes via HTTP or HTTPS and forwards the request to the client through a WebSocket connection. The client then makes a request to the local service and sends the response back to the server.

Advantages of the reverse connection architecture:

1. **No SSH tunnel required**: You don't need to set up an SSH tunnel to access local services
2. **Automatic URL replacement**: Local URLs in HTML responses are automatically replaced with tunnel URLs
3. **HTTPS support**: Access local services via HTTPS without configuring TLS on the local service
4. **Custom subdomains**: Use easy-to-remember subdomains to access local services

To use an HTTPS tunnel:

1. Ensure the haxorport server is correctly configured to support HTTPS
2. Run the client by specifying the local port and subdomain:
   ```
   haxorport http --port 8080 --subdomain myapp
   ```
3. Access your service via HTTPS:
   ```
   https://myapp.haxorport.online
   ```

All links and references in your web pages will be automatically modified to use the tunnel URL, ensuring that navigation on the website works correctly.

### 🔌 TCP Tunnel

Haxorport supports TCP tunnels that allow you to expose local TCP services (such as SSH, databases, or other services) to the internet. TCP tunnels work by forwarding connections from a remote port on the Haxorport server to a local port on your machine.

Create a TCP tunnel for a local TCP service:

```
haxorport tcp --port 22 --remote-port 2222
```

If `--remote-port` is not specified, the server will assign a remote port automatically.

Advantages of Haxorport TCP tunnels:

1. **Secure Access**: Access local TCP services from anywhere without opening ports in your firewall
2. **Multi-Protocol Support**: Supports all TCP-based protocols (SSH, MySQL, PostgreSQL, Redis, etc.)
3. **Integrated Authentication**: Uses the same authentication system as HTTP/HTTPS tunnels
4. **Usage Limits**: Control the number of tunnels based on user subscription

Examples of TCP tunnel usage:

- **🔑 SSH Server**:
  ```
  haxorport tcp --port 22 --remote-port 2222
  # Access: ssh user@haxorport.online -p 2222
  ```

- **💾 MySQL Database**:
  ```
  haxorport tcp --port 3306 --remote-port 3306
  # Access: mysql -h haxorport.online -P 3306 -u user -p
  ```

- **💾 PostgreSQL Database**:
  ```
  haxorport tcp --port 5432 --remote-port 5432
  # Access: psql -h haxorport.online -p 5432 -U user -d database
  ```

### 📝 Adding Tunnels to Configuration

You can add tunnels to the configuration for later use:

```
haxorport config add-tunnel --name web --type http --port 8080 --subdomain myapp
haxorport config add-tunnel --name ssh --type tcp --port 22 --remote-port 2222
```

## 👨‍💻 Development

### 📚 Prerequisites

- Go 1.21 or newer
- Git

### 🔧 Development Setup

1. Clone the repository:
   ```
   git clone https://github.com/haxorport/haxorport-go-client.git
   cd haxorport-go-client
   ```

2. Install dependencies:
   ```
   go mod download
   ```

3. Run the application in development mode:
   ```
   ./scripts/run.sh
   ```

### 🚀 Development with Go Run

For quick testing and development, you can run the application directly using `go run`:

1. Make sure all dependencies are downloaded:
   ```bash
   go mod download
   ```

2. Run with development configuration:
   ```bash
   # For HTTP tunnel
   go run main.go http http://localhost:8080 --config config.dev.yaml
   
   # For TCP tunnel
   go run main.go tcp 22 --remote-port 2222 --config config.dev.yaml
   ```

3. For production configuration:
   ```bash
   go run main.go http http://localhost:8080 --config config.yaml
   ```

Notes:
- Make sure your config file is properly configured before running
- Use `--config` flag to specify configuration file
- The application will create a default config file if not found

### 💻 Code Structure

- **Domain Layer**: Contains domain models and ports (interfaces)
- **Application Layer**: Contains services that implement use cases
- **Infrastructure Layer**: Contains concrete implementations of ports
- **CLI Layer**: Contains command-line interface using Cobra
- **DI Layer**: Contains container for dependency injection

## 🔧 Troubleshooting

### 📉 Reducing Debug Output

If you see too many INFO log messages when running the application, you can change the log level to `warn` as follows:

```bash
# Edit configuration file
sudo nano /etc/haxorport/config.yaml  # For Linux
nano ~/.haxorport/config/config.yaml  # For Windows (WSL)
nano ~/Library/Preferences/haxorport/config.yaml  # For macOS
```

Change the line `log_level: info` to `log_level: warn`, then save the file.

Or use the following command to change the log level automatically:

```bash
# For Linux
sudo sed -i 's/log_level:.*/log_level: warn/g' /etc/haxorport/config.yaml

# For macOS
sed -i '' 's/log_level:.*/log_level: warn/g' ~/Library/Preferences/haxorport/config.yaml

# For Windows (WSL)
sed -i 's/log_level:.*/log_level: warn/g' ~/.haxorport/config/config.yaml
```

## 📃 License

MIT License

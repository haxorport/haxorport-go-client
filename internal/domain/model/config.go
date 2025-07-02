package model

import (
	"os"
	"path/filepath"
)

// LogLevel defines logging levels
type LogLevel string

const (
	// LogLevelDebug is the level for debug messages
	LogLevelDebug LogLevel = "debug"
	// LogLevelInfo is the level for informational messages
	LogLevelInfo LogLevel = "info"
	// LogLevelWarn is the level for warning messages
	LogLevelWarn LogLevel = "warn"
	// LogLevelError is the level for error messages
	LogLevelError LogLevel = "error"
)

// ConnectionMode defines connection modes to the server
type ConnectionMode string

const (
	// ConnectionModeWebSocket uses WebSocket as transport layer
	ConnectionModeWebSocket ConnectionMode = "websocket"
	// ConnectionModeDirectTCP uses direct TCP connection
	ConnectionModeDirectTCP ConnectionMode = "direct_tcp"
)

// Config is the configuration structure for haxorport client
type Config struct {
	// ServerAddress is the haxorport server address
	ServerAddress string
	// ControlPort is the port for control plane
	ControlPort int
	// DataPort is the port for data plane
	DataPort int
	// ConnectionMode is the connection mode to server (websocket or direct_tcp)
	ConnectionMode ConnectionMode
	// AuthEnabled is a flag to enable authentication
	AuthEnabled bool
	// AuthToken is the token for server authentication
	AuthToken string
	// AuthValidationURL is the URL for token validation (empty to use default)
	AuthValidationURL string
	// TLSEnabled is a flag to enable TLS
	TLSEnabled bool
	// TLSCert is the path to TLS certificate file
	TLSCert string
	// TLSKey is the path to TLS key file
	TLSKey string
	// LogLevel is the logging level (debug, info, warn, error)
	LogLevel LogLevel
	// LogFile is the path to log file (empty for stdout)
	LogFile string
	// BaseDomain is the base domain for tunnel subdomains
	BaseDomain string
	// Tunnels is the list of tunnels to be created at startup
	Tunnels []TunnelConfig
}

// NewConfig creates a new Config instance with default values
func NewConfig() *Config {
	return &Config{
		ServerAddress:     "control.haxorport.online",
		ControlPort:       443,
		DataPort:          8081,
		AuthEnabled:       false,
		AuthToken:         "",
		AuthValidationURL: "https://haxorport.online/AuthToken/validate",
		TLSEnabled:        false,
		TLSCert:           "",
		TLSKey:            "",
		LogLevel:          LogLevelWarn,
		LogFile:           "",
		BaseDomain:        "haxorport.online",
		Tunnels:           []TunnelConfig{},
	}
}

// AddTunnel adds a tunnel to the configuration
func (c *Config) AddTunnel(tunnel TunnelConfig) {
	c.Tunnels = append(c.Tunnels, tunnel)
}

// RemoveTunnel removes a tunnel from configuration by name
func (c *Config) RemoveTunnel(name string) bool {
	for i, tunnel := range c.Tunnels {
		if tunnel.Name == name {
			// Remove tunnel from slice
			c.Tunnels = append(c.Tunnels[:i], c.Tunnels[i+1:]...)
			return true
		}
	}
	return false
}

// GetTunnel returns a tunnel by name
func (c *Config) GetTunnel(name string) *TunnelConfig {
	for _, tunnel := range c.Tunnels {
		if tunnel.Name == name {
			return &tunnel
		}
	}
	return nil
}

// GetConfigFilePath returns the path to configuration file
func (c *Config) GetConfigFilePath() string {
	// Determine configuration directory based on user
	configDir := "/etc/haxorport"
	
	// If not root, use home directory
	if os.Getuid() != 0 {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			configDir = filepath.Join(homeDir, ".haxorport")
		}
	}
	
	// Configuration file path
	return filepath.Join(configDir, "config.yaml")
}

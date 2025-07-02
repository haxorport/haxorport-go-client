package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/haxorport/haxorport-go-client/internal/domain/model"
	"github.com/haxorport/haxorport-go-client/internal/domain/port"
	"github.com/spf13/viper"
)

// ConfigRepository is an implementation of port.ConfigRepository
type ConfigRepository struct{}

// NewConfigRepository creates a new ConfigRepository instance
func NewConfigRepository() *ConfigRepository {
	return &ConfigRepository{}
}

// Load loads configuration from file
func (r *ConfigRepository) Load(configPath string) (*model.Config, error) {
	config := model.NewConfig()

	// If configPath is empty, look in the default location
	if configPath == "" {
		var err error
		configPath, err = r.GetDefaultPath()
		if err != nil {
			return nil, err
		}
	}

	// Check if the file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil
	}

	// Load configuration from file
	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	// Map from viper to Config struct
	config.ServerAddress = viper.GetString("server_address")
	config.ControlPort = viper.GetInt("control_port")
	config.DataPort = viper.GetInt("data_port")
	config.ConnectionMode = model.ConnectionMode(viper.GetString("connection_mode"))
	config.AuthEnabled = viper.GetBool("auth_enabled")
	config.AuthToken = viper.GetString("auth_token")
	config.AuthValidationURL = viper.GetString("auth_validation_url")
	config.TLSEnabled = viper.GetBool("tls_enabled")
	config.TLSCert = viper.GetString("tls_cert")
	config.TLSKey = viper.GetString("tls_key")
	config.BaseDomain = viper.GetString("base_domain")
	config.LogLevel = model.LogLevel(viper.GetString("log_level"))
	config.LogFile = viper.GetString("log_file")

	// Load tunnels
	var tunnelConfigs []model.TunnelConfig
	if err := viper.UnmarshalKey("tunnels", &tunnelConfigs); err != nil {
		return nil, fmt.Errorf("error parsing tunnel configuration: %v", err)
	}
	config.Tunnels = tunnelConfigs

	return config, nil
}

// Save saves configuration to file
func (r *ConfigRepository) Save(config *model.Config, configPath string) error {
	// If configPath is empty, use default location
	if configPath == "" {
		var err error
		configPath, err = r.GetDefaultPath()
		if err != nil {
			return err
		}
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating config directory: %v", err)
	}

	// Set configuration in viper
	viper.SetConfigFile(configPath)

	// Set configuration values in viper
	viper.Set("server_address", config.ServerAddress)
	viper.Set("control_port", config.ControlPort)
	viper.Set("data_port", config.DataPort)
	viper.Set("connection_mode", string(config.ConnectionMode))
	viper.Set("auth_enabled", config.AuthEnabled)
	viper.Set("auth_token", config.AuthToken)
	viper.Set("auth_validation_url", config.AuthValidationURL)
	viper.Set("tls_enabled", config.TLSEnabled)
	viper.Set("tls_cert", config.TLSCert)
	viper.Set("tls_key", config.TLSKey)
	viper.Set("base_domain", config.BaseDomain)
	viper.Set("log_level", string(config.LogLevel))
	viper.Set("log_file", config.LogFile)
	viper.Set("tunnels", config.Tunnels)

	// Save to file
	if err := viper.WriteConfig(); err != nil {
		// If file doesn't exist, create new one
		if strings.Contains(err.Error(), "no such file") {
			return viper.SafeWriteConfig()
		}
		return fmt.Errorf("error saving configuration: %v", err)
	}

	return nil
}

// GetDefaultPath returns the default path for configuration file
func (r *ConfigRepository) GetDefaultPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error getting home directory: %v", err)
	}

	return filepath.Join(homeDir, ".haxorport", "config.yaml"), nil
}

// Ensure ConfigRepository implements port.ConfigRepository
var _ port.ConfigRepository = (*ConfigRepository)(nil)

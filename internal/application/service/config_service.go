package service

import (
	"fmt"

	"github.com/haxorport/haxorport-go-client/internal/domain/model"
	"github.com/haxorport/haxorport-go-client/internal/domain/port"
)

// ConfigService is a service for managing configuration
type ConfigService struct {
	configRepo port.ConfigRepository
	logger     port.Logger
}

// NewConfigService creates a new ConfigService instance
func NewConfigService(configRepo port.ConfigRepository, logger port.Logger) *ConfigService {
	return &ConfigService{
		configRepo: configRepo,
		logger:     logger,
	}
}

// LoadConfig loads configuration from a file
func (s *ConfigService) LoadConfig(configPath string) (*model.Config, error) {
	// If configPath is empty, use the default path
	if configPath == "" {
		var err error
		configPath, err = s.configRepo.GetDefaultPath()
		if err != nil {
			return nil, fmt.Errorf("failed to get default path: %v", err)
		}
	}
	
	// Load configuration
	config, err := s.configRepo.Load(configPath)
	if err != nil {
		s.logger.Warn("Failed to load configuration from %s: %v", configPath, err)
		// Return default configuration if loading fails
		return model.NewConfig(), nil
	}
	
	s.logger.Info("Configuration loaded from %s", configPath)
	
	return config, nil
}

// SaveConfig saves configuration to a file
func (s *ConfigService) SaveConfig(config *model.Config, configPath string) error {
	// If configPath is empty, use the default path
	if configPath == "" {
		var err error
		configPath, err = s.configRepo.GetDefaultPath()
		if err != nil {
			return fmt.Errorf("failed to get default path: %v", err)
		}
	}
	
	// Save configuration
	if err := s.configRepo.Save(config, configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %v", err)
	}
	
	s.logger.Info("Configuration saved to %s", configPath)
	
	return nil
}

// SetServerAddress sets the server address
func (s *ConfigService) SetServerAddress(config *model.Config, serverAddress string) {
	config.ServerAddress = serverAddress
}

// SetControlPort sets the control plane port
func (s *ConfigService) SetControlPort(config *model.Config, controlPort int) {
	config.ControlPort = controlPort
}

// SetAuthToken sets the authentication token
func (s *ConfigService) SetAuthToken(config *model.Config, authToken string) {
	config.AuthToken = authToken
}

// SetLogLevel sets the log level
func (s *ConfigService) SetLogLevel(config *model.Config, logLevel string) {
	config.LogLevel = model.LogLevel(logLevel)
}

// SetLogFile sets the log file
func (s *ConfigService) SetLogFile(config *model.Config, logFile string) {
	config.LogFile = logFile
}

// AddTunnel adds a tunnel to the configuration
func (s *ConfigService) AddTunnel(config *model.Config, tunnel model.TunnelConfig) {
	config.AddTunnel(tunnel)
}

// RemoveTunnel removes a tunnel from the configuration
func (s *ConfigService) RemoveTunnel(config *model.Config, name string) bool {
	return config.RemoveTunnel(name)
}

// GetTunnel returns a tunnel from the configuration
func (s *ConfigService) GetTunnel(config *model.Config, name string) *model.TunnelConfig {
	return config.GetTunnel(name)
}

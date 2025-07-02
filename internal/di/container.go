package di

import (
	"fmt"
	"os"

	"github.com/haxorport/haxorport-go-client/internal/application/service"
	"github.com/haxorport/haxorport-go-client/internal/domain/model"
	"github.com/haxorport/haxorport-go-client/internal/domain/port"
	"github.com/haxorport/haxorport-go-client/internal/infrastructure/config"
	"github.com/haxorport/haxorport-go-client/internal/infrastructure/logger"
	"github.com/haxorport/haxorport-go-client/internal/infrastructure/transport"
)

// Container is a container for dependency injection
type Container struct {
	// Logger
	Logger *logger.Logger

	// Repositories
	ConfigRepository *config.ConfigRepository

	// Services
	ConfigService *service.ConfigService
	TunnelService *service.TunnelService

	// Client
	Client *transport.Client

	// TunnelRepository
	TunnelRepository port.TunnelRepository

	// Config
	Config *model.Config
}

// NewContainer creates a new Container instance
func NewContainer() *Container {
	return &Container{}
}

// Initialize initializes the container
func (c *Container) Initialize(configPath string) error {
	// Initialize logger
	c.Logger = logger.NewLogger(os.Stdout, "info")

	// Initialize config repository
	c.ConfigRepository = config.NewConfigRepository()

	// Initialize config service
	c.ConfigService = service.NewConfigService(c.ConfigRepository, c.Logger)

	// Load configuration
	var err error
	c.Config, err = c.ConfigService.LoadConfig(configPath)
	if err != nil {
		return err
	}

	// Set logger level based on configuration
	c.Logger.SetLevel(string(c.Config.LogLevel))

	// If log file is specified, use file logger but still display output to terminal
	if c.Config.LogFile != "" {
		_, err := logger.NewFileLogger(c.Config.LogFile, string(c.Config.LogLevel))
		if err != nil {
			c.Logger.Error("Failed to create file logger: %v", err)
		} else {
			// Keep using the existing logger (to stdout)
			c.Logger.Info("Logs will also be written to file: %s", c.Config.LogFile)
		}
	}

	// Initialize client and tunnel repository based on connection mode
	if c.Config.ConnectionMode == model.ConnectionModeWebSocket {
		// Initialize client for WebSocket mode
		c.Client = transport.NewClient(c.Config, c.Logger)

		// Initialize tunnel repository for WebSocket mode
		tunnelRepo, repoErr := transport.CreateTunnelRepository(c.Config, c.Client, c.Logger)
		if repoErr != nil {
			return repoErr
		}
		c.TunnelRepository = tunnelRepo
	} else if c.Config.ConnectionMode == model.ConnectionModeDirectTCP {
		// For direct TCP mode, we don't need WebSocket client
		// Initialize tunnel repository for direct TCP mode
		tunnelRepo, repoErr := transport.CreateTunnelRepository(c.Config, nil, c.Logger)
		if repoErr != nil {
			return repoErr
		}
		c.TunnelRepository = tunnelRepo
	} else {
		return fmt.Errorf("connection mode not supported: %s", c.Config.ConnectionMode)
	}

	// Initialize tunnel service
	c.TunnelService = service.NewTunnelService(c.TunnelRepository, c.Logger)

	// Register handler for HTTP request messages if using WebSocket
	if c.Config.ConnectionMode == model.ConnectionModeWebSocket && c.Client != nil {
		c.Client.RegisterHandler(model.MessageTypeHTTPRequest, c.Client.HandleHTTPRequestMessage)
	}

	return nil
}

// Close closes all resources
func (c *Container) Close() {
	// Close client if exists
	if c.Client != nil {
		c.Client.Close()
	}

	// Close logger
	if c.Logger != nil {
		c.Logger.Close()
	}
}

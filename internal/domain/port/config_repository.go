package port

import "github.com/haxorport/haxorport-go-client/internal/domain/model"

// ConfigRepository defines operations that can be performed on configuration
type ConfigRepository interface {
	// Load loads configuration from storage
	Load(path string) (*model.Config, error)
	
	// Save saves configuration to storage
	Save(config *model.Config, path string) error
	
	// GetDefaultPath returns the default path for configuration file
	GetDefaultPath() (string, error)
}

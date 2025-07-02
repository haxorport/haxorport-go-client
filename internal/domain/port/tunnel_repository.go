package port

import "github.com/haxorport/haxorport-go-client/internal/domain/model"

// TunnelRepository defines operations that can be performed on a tunnel
type TunnelRepository interface {
	// Register registers a new tunnel to the server
	Register(config model.TunnelConfig) (*model.Tunnel, error)
	
	// Unregister removes a tunnel from the server
	Unregister(tunnelID string) error
	
	// GetAll returns all active tunnels
	GetAll() []*model.Tunnel
	
	// GetByID returns a tunnel by its ID
	GetByID(tunnelID string) (*model.Tunnel, error)
	
	// SendData sends data through the tunnel
	SendData(tunnelID string, connectionID string, data []byte) error
	
	// HandleData handles data received from the server
	HandleData(tunnelID string, connectionID string, data []byte) error
}

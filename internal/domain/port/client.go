package port

import "github.com/haxorport/haxorport-go-client/internal/domain/model"

// Client is an interface for communicating with the haxorport server
type Client interface {
	// Connect establishes a connection to the haxorport server
	Connect() error
	
	// Close closes the connection to the server
	Close()
	
	// IsConnected returns the connection status
	IsConnected() bool
	
	// RunWithReconnect runs the client with automatic reconnection
	RunWithReconnect()
	
	// GetUserData returns the authenticated user data
	GetUserData() *model.AuthData
	
	// CheckTunnelLimit checks if the user has reached their tunnel limit
	// Returns: reached limit (bool), used tunnels (int), total limit (int)
	CheckTunnelLimit() (bool, int, int)
}

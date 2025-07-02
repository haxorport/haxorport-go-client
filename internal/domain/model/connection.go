package model

// Connection represents a tunnel connection
type Connection struct {
	// ID is the unique connection ID
	ID string
	// TunnelID is the associated tunnel ID
	TunnelID string
	// Data is the data sent through the connection
	Data []byte
}

// NewConnection creates a new Connection instance
func NewConnection(id string, tunnelID string) *Connection {
	return &Connection{
		ID:       id,
		TunnelID: tunnelID,
	}
}

// SetData sets the connection data
func (c *Connection) SetData(data []byte) {
	c.Data = data
}

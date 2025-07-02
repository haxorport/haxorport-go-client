package model

import (
	"encoding/json"
	"fmt"
	"time"
)

// MessageType defines message types for client-server communication
type MessageType string

const (
	// MessageTypeAuth is for authentication
	MessageTypeAuth MessageType = "auth"
	// MessageTypeRegister is for tunnel registration
	MessageTypeRegister MessageType = "register"
	// MessageTypeUnregister is for tunnel removal
	MessageTypeUnregister MessageType = "unregister"
	// MessageTypeData contains tunnel data
	MessageTypeData MessageType = "data"
	// MessageTypePing keeps the connection alive
	MessageTypePing MessageType = "ping"
	// MessageTypePong is a response to ping
	MessageTypePong MessageType = "pong"
	// MessageTypeError indicates an error message
	MessageTypeError MessageType = "error"
)

// Message represents the base structure for all client-server messages
type Message struct {
	// Type is the message type
	Type MessageType `json:"type"`
	// Version is the protocol version
	Version string `json:"version"`
	// Timestamp is when the message was created (in milliseconds since epoch)
	Timestamp int64 `json:"timestamp"`
	// Payload contains the actual message data
	Payload json.RawMessage `json:"payload,omitempty"`
}

// NewMessage creates a new message with specified type and payload
func NewMessage(msgType MessageType, payload interface{}) (*Message, error) {
	var payloadJSON json.RawMessage
	var err error

	if payload != nil {
		payloadJSON, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to convert payload to JSON: %v", err)
		}
	}

	return &Message{
		Type:      msgType,
		Version:   "1.0.0", // Current protocol version
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
		Payload:   payloadJSON,
	}, nil
}

// ParsePayload parses message payload into the provided struct
func (m *Message) ParsePayload(v interface{}) error {
	if m.Payload == nil {
		return nil
	}
	return json.Unmarshal(m.Payload, v)
}

// AuthPayload is for authentication messages
type AuthPayload struct {
	// Token is the authentication token
	Token string `json:"token"`
}

// RegisterPayload is for tunnel registration messages
type RegisterPayload struct {
	// TunnelType specifies the tunnel type (http, tcp)
	TunnelType string `json:"tunnel_type"`
	// Subdomain is the requested subdomain (optional)
	Subdomain string `json:"subdomain,omitempty"`
	// LocalAddr is the specific local address for forwarding (optional)
	LocalAddr string `json:"local_addr,omitempty"`
	// LocalPort is the local port to be tunneled
	LocalPort int `json:"local_port"`
	// RemotePort is the requested remote port (for TCP, optional)
	RemotePort int `json:"remote_port,omitempty"`
	// Auth contains tunnel authentication information (optional)
	Auth *TunnelAuth `json:"auth,omitempty"`
}

// UnregisterPayload is for tunnel removal messages
type UnregisterPayload struct {
	// TunnelID is the ID of the tunnel to be removed
	TunnelID string `json:"tunnel_id"`
}

// DataPayload is for data messages
type DataPayload struct {
	// TunnelID is the ID of the tunnel associated with the data
	TunnelID string `json:"tunnel_id"`
	// ConnectionID is the ID of the connection associated with the data
	ConnectionID string `json:"connection_id"`
	// Data is the actual data being sent
	Data []byte `json:"data"`
}

// ErrorPayload is for error messages
type ErrorPayload struct {
	// Code is the error code
	Code string `json:"code"`
	// Message contains the error details
	Message string `json:"message"`
}

// RegisterResponsePayload is the response to registration messages
type RegisterResponsePayload struct {
	// Success indicates if the registration was successful
	Success bool `json:"success"`
	// TunnelID is the ID of the created tunnel
	TunnelID string `json:"tunnel_id"`
	// URL is the public URL for HTTP tunnels
	URL string `json:"url,omitempty"`
	// RemotePort is the remote port for TCP tunnels
	RemotePort int `json:"remote_port,omitempty"`
	// Error contains the error message if registration failed
	Error string `json:"error,omitempty"`
}

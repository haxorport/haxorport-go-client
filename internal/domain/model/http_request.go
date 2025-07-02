package model

import (
	"net/http"
)

// HTTPRequest represents an HTTP request sent from server to client
type HTTPRequest struct {
	// ID is the unique request identifier
	ID string `json:"id"`
	// TunnelID is the tunnel ID associated with the request
	TunnelID string `json:"tunnel_id"`
	// Method is the HTTP method (GET, POST, etc.)
	Method string `json:"method"`
	// URL is the request URL
	URL string `json:"url"`
	// Headers are the request headers
	Headers http.Header `json:"headers"`
	// Body is the request body
	Body []byte `json:"body,omitempty"`
	// LocalPort is the local port that will be connected by the client
	LocalPort int `json:"local_port"`
	// RemoteAddr is the remote address of the HTTP client
	RemoteAddr string `json:"remote_addr"`
	// Scheme is the protocol scheme (http or https)
	Scheme string `json:"scheme,omitempty"`
}

// HTTPResponse represents an HTTP response sent from client to server
type HTTPResponse struct {
	// ID is the request ID associated with the response
	ID string `json:"id"`
	// StatusCode is the HTTP status code (200, 404, etc.)
	StatusCode int `json:"status_code"`
	// Headers are the response headers
	Headers http.Header `json:"headers"`
	// Body is the response body
	Body []byte `json:"body,omitempty"`
	// Error contains any error that occurred
	Error string `json:"error,omitempty"`
}

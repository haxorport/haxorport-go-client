package model

// No imports required

// MessageTypeHTTPRequest is the message type for HTTP requests
const MessageTypeHTTPRequest MessageType = "http_request"

// MessageTypeHTTPResponse is the message type for HTTP responses
const MessageTypeHTTPResponse MessageType = "http_response"

// HTTPRequestPayload is the payload for HTTP request messages
type HTTPRequestPayload struct {
	// Request is the HTTP request
	Request *HTTPRequest `json:"request"`
}

// HTTPResponsePayload is the payload for HTTP response messages
type HTTPResponsePayload struct {
	// Response is the HTTP response
	Response *HTTPResponse `json:"response"`
}

// NewHTTPRequestMessage creates a new HTTP request message
func NewHTTPRequestMessage(request *HTTPRequest) (*Message, error) {
	payload := HTTPRequestPayload{
		Request: request,
	}
	return NewMessage(MessageTypeHTTPRequest, payload)
}

// NewHTTPResponseMessage creates a new HTTP response message
func NewHTTPResponseMessage(response *HTTPResponse) (*Message, error) {
	payload := HTTPResponsePayload{
		Response: response,
	}
	return NewMessage(MessageTypeHTTPResponse, payload)
}

// ParseHTTPRequestPayload parses the HTTP request payload
func (m *Message) ParseHTTPRequestPayload() (*HTTPRequest, error) {
	var payload HTTPRequestPayload
	if err := m.ParsePayload(&payload); err != nil {
		return nil, err
	}
	return payload.Request, nil
}

// ParseHTTPResponsePayload parses the HTTP response payload
func (m *Message) ParseHTTPResponsePayload() (*HTTPResponse, error) {
	var payload HTTPResponsePayload
	if err := m.ParsePayload(&payload); err != nil {
		return nil, err
	}
	return payload.Response, nil
}

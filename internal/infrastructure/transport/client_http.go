package transport

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/haxorport/haxorport-go-client/internal/domain/model"
)

// HandleHTTPRequestMessage menangani pesan permintaan HTTP dari server
func (c *Client) HandleHTTPRequestMessage(msg *model.Message) error {
	// Parse payload
	request, err := msg.ParseHTTPRequestPayload()
	if err != nil {
		c.logger.Error("Gagal mengurai payload permintaan HTTP: %v", err)
		return err
	}

	c.logger.Info("Received HTTP request: %s %s", request.Method, request.URL)

	// Create HTTP request to local service on client computer
	// Always use HTTP for local connections, regardless of the scheme received from server
	// This is because local services typically only support HTTP
	scheme := "http"
	
	// Use localhost on client computer, not on server
	targetURL := fmt.Sprintf("%s://localhost:%d%s", scheme, request.LocalPort, request.URL)
	c.logger.Info("Sending request to local service: %s", targetURL)
	httpReq, err := http.NewRequest(request.Method, targetURL, bytes.NewReader(request.Body))
	if err != nil {
		c.logger.Error("Failed to create local HTTP request: %v", err)
		return c.sendHTTPErrorResponse(request.ID, err)
	}

	// Copy headers
	for key, values := range request.Headers {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}

	// Add X-Forwarded-* headers
	httpReq.Header.Set("X-Forwarded-Host", request.Headers.Get("Host"))
	httpReq.Header.Set("X-Forwarded-Proto", scheme) // Use the scheme received from the server
	httpReq.Header.Set("X-Forwarded-For", request.RemoteAddr)

	// Send request to local service via reverse connection
	c.logger.Info("Making HTTP connection to local service with method %s", request.Method)
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		c.logger.Error("Failed to send local HTTP request: %v", err)
		return c.sendHTTPErrorResponse(request.ID, err)
	}
	c.logger.Info("Successfully connected to local service, status: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("Failed to read response body: %v", err)
		return c.sendHTTPErrorResponse(request.ID, err)
	}

	// Check Content-Type to determine if it's HTML
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		// Replace local URLs with tunnel URLs in HTML response
		localURLPrefix := fmt.Sprintf("http://localhost:%d", request.LocalPort)
		localURLPrefixSecure := fmt.Sprintf("https://localhost:%d", request.LocalPort)
		
		// Create tunnel URL based on received scheme
		tunnelScheme := "http"
		if request.Scheme == "https" {
			tunnelScheme = "https"
		}
		
		// Extract hostname from Host header
		hostname := ""
		if host, ok := request.Headers["Host"]; ok && len(host) > 0 {
			hostname = host[0]
		}
		
		// If hostname is still empty, use X-Forwarded-Host
		if hostname == "" {
			if host, ok := request.Headers["X-Forwarded-Host"]; ok && len(host) > 0 {
				hostname = host[0]
			}
		}
		
		// If hostname is still empty, extract subdomain from tunnel URL
		if hostname == "" {
			// Try to get subdomain from URL provided by user
			subdomain := c.GetSubdomain()
			if subdomain != "" {
				hostname = subdomain + ".haxorport.online"
			} else {
				// Fallback to tunnel ID if subdomain is not available
				hostname = request.TunnelID + ".haxorport.online"
			}
		}
		
		c.logger.Info(fmt.Sprintf("Using hostname: %s for URL replacement", hostname))
		tunnelURLPrefix := fmt.Sprintf("%s://%s", tunnelScheme, hostname)
		
		// Replace local URLs with tunnel URLs in body
		bodyStr := string(body)
		bodyStr = strings.ReplaceAll(bodyStr, localURLPrefix, tunnelURLPrefix)
		bodyStr = strings.ReplaceAll(bodyStr, localURLPrefixSecure, tunnelURLPrefix)
		
		// Replace relative URLs in href and src
		// Example: href="/path" becomes href="https://subdomain.haxorport.online/path"
		bodyStr = strings.ReplaceAll(bodyStr, "href=\"/", "href=\""+tunnelURLPrefix+"/")
		bodyStr = strings.ReplaceAll(bodyStr, "src=\"/", "src=\""+tunnelURLPrefix+"/")
		
		// Update body with modified content
		body = []byte(bodyStr)
		
		c.logger.Info("Local URLs in HTML response replaced with tunnel URLs")
	}

	// Create HTTP response
	httpResp := &model.HTTPResponse{
		ID:         request.ID,
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       body,
	}

	// Send response to server
	return c.sendHTTPResponse(httpResp)
}

// sendHTTPResponse sends HTTP response to server
func (c *Client) sendHTTPResponse(response *model.HTTPResponse) error {
	// Create HTTP response message
	msg, err := model.NewHTTPResponseMessage(response)
	if err != nil {
		c.logger.Error("Failed to create HTTP response message: %v", err)
		return err
	}

	// Send message to server
	return c.sendMessage(msg)
}

// sendHTTPErrorResponse sends HTTP error response to server
func (c *Client) sendHTTPErrorResponse(requestID string, err error) error {
	// Create HTTP error response
	httpResp := &model.HTTPResponse{
		ID:         requestID,
		StatusCode: http.StatusInternalServerError,
		Headers:    http.Header{},
		Error:      err.Error(),
	}

	// Send response to server
	return c.sendHTTPResponse(httpResp)
}

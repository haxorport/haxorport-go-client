package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/haxorport/haxorport-go-client/internal/domain/model"
)

// AuthService is an interface for authentication services
type AuthService interface {
	// ValidateToken validates authentication token
	ValidateToken(token string) (bool, error)
	// ValidateTokenWithResponse validates authentication token and returns complete response
	ValidateTokenWithResponse(token string) (*model.AuthResponse, error)
}

// authService is an implementation of AuthService
type authService struct {
	validationURL string
}

// NewAuthService creates a new AuthService instance
func NewAuthService(validationURL string) AuthService {
	return &authService{
		validationURL: validationURL,
	}
}

// ValidateToken validates authentication token by sending request to validation API
func (s *authService) ValidateToken(token string) (bool, error) {
	// Use ValidateTokenWithResponse and only return valid status
	response, err := s.ValidateTokenWithResponse(token)
	if err != nil {
		return false, err
	}
	
	// Check if response indicates valid token
	return response.Status == "success" && response.Code == 200, nil
}

// ValidateTokenWithResponse validates authentication token and returns complete response
func (s *authService) ValidateTokenWithResponse(token string) (*model.AuthResponse, error) {
	// If token is empty, return error immediately
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	// Buat form data
	data := url.Values{}
	data.Set("token", token)

	// Buat request
	req, err := http.NewRequest("POST", s.validationURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set header Content-Type
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Add User-Agent to avoid blocking
	req.Header.Set("User-Agent", "HaxorportClient/1.0")

	// Kirim request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token validation failed with status code: %d", resp.StatusCode)
	}

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// Check if response is valid JSON
	if !json.Valid(respBody) {
		// If not valid JSON, try to see first 100 characters for debugging
		preview := string(respBody)
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		return nil, fmt.Errorf("response is not valid JSON: %s", preview)
	}

	// Parse response
	var authResponse model.AuthResponse
	err = json.Unmarshal(respBody, &authResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &authResponse, nil
}

// ValidateTokenWithURLEncoded validates authentication token by sending request with application/x-www-form-urlencoded format
func (s *authService) ValidateTokenWithURLEncoded(token string) (bool, error) {
	// If token is empty, immediately return false
	if token == "" {
		return false, fmt.Errorf("token cannot be empty")
	}

	// Buat form data
	data := url.Values{}
	data.Set("token", token)

	// Buat request
	req, err := http.NewRequest("POST", s.validationURL, strings.NewReader(data.Encode()))
	if err != nil {
		return false, fmt.Errorf("failed to create request: %v", err)
	}

	// Set header Content-Type
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Kirim request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("token validation failed with status code: %d", resp.StatusCode)
	}

	// Parse response
	var result struct {
		Valid bool `json:"valid"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return false, fmt.Errorf("failed to parse response: %v", err)
	}

	return result.Valid, nil
}

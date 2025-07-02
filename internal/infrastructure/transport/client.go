package transport

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/haxorport/haxorport-go-client/internal/domain/model"
	"github.com/haxorport/haxorport-go-client/internal/domain/port"
	"github.com/haxorport/haxorport-go-client/internal/domain/service"
)


type Client struct {
	serverAddr   string
	controlPort  int
	dataPort     int
	authEnabled  bool
	authToken    string
	tlsEnabled   bool
	tlsCert      string
	tlsKey       string
	baseDomain   string
	conn         *websocket.Conn
	isConnected  bool
	reconnecting bool
	mutex        sync.Mutex
	logger       port.Logger
	handlers     map[model.MessageType]func(*model.Message) error
	subdomain    string 
	config       *model.Config
	userData     *model.AuthData 
}


func NewClient(config *model.Config, logger port.Logger) *Client {
	return &Client{
		serverAddr:   config.ServerAddress,
		controlPort:  config.ControlPort,
		dataPort:     config.DataPort,
		authEnabled:  config.AuthEnabled,
		authToken:    config.AuthToken,
		tlsEnabled:   config.TLSEnabled,
		tlsCert:      config.TLSCert,
		tlsKey:       config.TLSKey,
		baseDomain:   config.BaseDomain,
		isConnected:  false,
		reconnecting: false,
		logger:       logger,
		handlers:     make(map[model.MessageType]func(*model.Message) error),
		config:       config,
	}
}


func (c *Client) Connect() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.isConnected {
		return nil
	}

	
	if c.config.AuthEnabled && c.config.AuthToken != "" {
		c.logger.Info("Validating authentication token...")

		
		validationURL := c.config.AuthValidationURL
		if validationURL == "" {
			
			validationURL = fmt.Sprintf("http://%s/AuthToken/validate", c.config.ServerAddress)
			if c.config.TLSEnabled {
				validationURL = fmt.Sprintf("https://%s/AuthToken/validate", c.config.ServerAddress)
			}
		}
		c.logger.Info("Using validation URL: %s", validationURL)

		// Create authentication service
		authService := service.NewAuthService(validationURL)

		// Validate token
		response, err := authService.ValidateTokenWithResponse(c.config.AuthToken)
		if err != nil {
			c.logger.Error("Failed to validate token: %v", err)
			return fmt.Errorf("failed to validate token: %v", err)
		}

		// Check response status
		if response.Status != "success" || response.Code != 200 {
			c.logger.Error("Invalid token: %s", response.Message)
			return fmt.Errorf("Authentication failed: invalid token")
		}

		// Store user data
		c.userData = &response.Data
		c.logger.Info("Token validated for user: %s (%s)", c.userData.Fullname, c.userData.Email)
		c.logger.Info("Subscription: %s, Tunnel Limit: %d/%d", c.userData.Subscription.Name, c.userData.Subscription.Limits.Tunnels.Used, c.userData.Subscription.Limits.Tunnels.Limit)
	}

	// Determine protocol (ws or wss)
	var protocol string

	// Create dialer
	dialer := websocket.DefaultDialer

	// Enable TLS if configured
	if c.tlsEnabled {
		protocol = "wss"

		// Create TLS config
		tlsConfig := &tls.Config{}

		// Load TLS certificate and key if provided
		if c.tlsCert != "" && c.tlsKey != "" {
			cert, err := tls.LoadX509KeyPair(c.tlsCert, c.tlsKey)
			if err != nil {
				return fmt.Errorf("failed to load TLS certificate: %v", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		} else {
			// Skip TLS verification if no certificate is provided
			tlsConfig.InsecureSkipVerify = true
		}

		dialer.TLSClientConfig = tlsConfig
	} else {
		protocol = "ws"
	}

	// Create server URL
	serverURL := fmt.Sprintf("%s://%s:%d/control", protocol, c.serverAddr, c.controlPort)
	c.logger.Info("Connecting to server: %s", serverURL)

	// Parse server URL
	u, err := url.Parse(serverURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}

	// Establish connection
	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %v", err)
	}

	c.conn = conn
	c.isConnected = true

	// Create authentication message
	authPayload := model.AuthPayload{
		Token: c.authToken,
	}
	authMessage, err := model.NewMessage(model.MessageTypeAuth, authPayload)
	if err != nil {
		c.Close()
		return fmt.Errorf("failed to create authentication message: %v", err)
	}

	// Marshal authentication message to JSON
	data, err := json.Marshal(authMessage)
	if err != nil {
		c.Close()
		return fmt.Errorf("failed to convert authentication message to JSON: %v", err)
	}

	// Send authentication message if authentication is enabled
	if c.authEnabled {
		if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			c.logger.Error("Failed to send authentication message: %v", err)
			c.Close()
			return fmt.Errorf("failed to send authentication: %v", err)
		}
	}

	// Start read pump
	go c.readPump()

	c.logger.Info("Connected to server: %s", serverURL)

	return nil
}

// Close closes the client connection.
func (c *Client) Close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.isConnected {
		return
	}

	c.logger.Info("Closing connection")

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	c.isConnected = false
}

// IsConnected returns whether the client is connected to the server.
func (c *Client) IsConnected() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.isConnected
}

// RunWithReconnect runs the client with automatic reconnect.
func (c *Client) RunWithReconnect() {
	c.mutex.Lock()
	if c.reconnecting {
		c.mutex.Unlock()
		return
	}
	c.reconnecting = true
	c.mutex.Unlock()

	go func() {
		for {
			if !c.IsConnected() {
				c.logger.Info("Reconnecting to server...")
				if err := c.Connect(); err != nil {
					c.logger.Error("Failed to reconnect: %v", err)
					time.Sleep(5 * time.Second)
					continue
				}
			}
			time.Sleep(1 * time.Second)
		}
	}()

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			if c.IsConnected() {
				pingMessage, err := model.NewMessage(model.MessageTypePing, nil)
				if err != nil {
					c.logger.Error("Failed to create ping message: %v", err)
					continue
				}

				if err := c.sendMessage(pingMessage); err != nil {
					c.logger.Error("Failed to send ping: %v", err)
					c.Close()
				}
			}
		}
	}()
}

// RegisterHandler registers a message handler for the given message type.
func (c *Client) RegisterHandler(msgType model.MessageType, handler func(*model.Message) error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.handlers[msgType] = handler
}

// sendMessage sends a message to the server.
func (c *Client) sendMessage(msg *model.Message) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.isConnected || c.conn == nil {
		return fmt.Errorf("not connected to server")
	}

	// Marshal message to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to convert message to JSON: %v", err)
	}


	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		c.logger.Error("Failed to send message: %v", err)
		c.isConnected = false
		return fmt.Errorf("failed to send message: %v", err)
	}

	return nil
}

// readPump reads messages from the server.
func (c *Client) readPump() {
	defer c.Close()

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			c.logger.Error("Failed to read message: %v", err)
			break
		}

		// Message received from server

		var msg model.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			c.logger.Error("Failed to parse message: %v", err)
			continue
		}

		if msg.Type == model.MessageTypePong {
			// Pong received from server
			continue
		}

		c.mutex.Lock()
		handler, exists := c.handlers[msg.Type]
		c.mutex.Unlock()

		if exists {
			if err := handler(&msg); err != nil {
				c.logger.Error("Error handling message %s: %v", msg.Type, err)
			}
		} else {
			c.logger.Error("No handler for message type: %s", msg.Type)
		}
	}
}

// SendRegisterTunnel sends a tunnel registration request to the server.
func (c *Client) SendRegisterTunnel(config model.TunnelConfig) (*model.RegisterResponsePayload, error) {
	c.subdomain = config.Subdomain

	responseCh := make(chan *model.RegisterResponsePayload, 1)
	errCh := make(chan error, 1)

	c.RegisterHandler(model.MessageTypeRegister, func(msg *model.Message) error {
		var response model.RegisterResponsePayload
		if err := msg.ParsePayload(&response); err != nil {
			errCh <- fmt.Errorf("failed to parse registration response: %v", err)
			return err
		}

		responseCh <- &response
		return nil
	})

	c.RegisterHandler(model.MessageTypeError, func(msg *model.Message) error {
		var errorPayload model.ErrorPayload
		if err := msg.ParsePayload(&errorPayload); err != nil {
			errCh <- fmt.Errorf("failed to parse error message: %v", err)
			return err
		}

		errCh <- fmt.Errorf("error from server: %s - %s", errorPayload.Code, errorPayload.Message)
		return nil
	})

	payload := model.RegisterPayload{
		TunnelType: string(config.Type),
		Subdomain:  config.Subdomain,
		LocalAddr:  config.LocalAddr,
		LocalPort:  config.LocalPort,
		RemotePort: config.RemotePort,
		Auth:       config.Auth,
	}

	msg, err := model.NewMessage(model.MessageTypeRegister, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create registration message: %v", err)
	}

	if err := c.sendMessage(msg); err != nil {
		return nil, err
	}

	select {
	case response := <-responseCh:
		if !response.Success {
			return nil, fmt.Errorf("tunnel registration failed: %s", response.Error)
		}
		return response, nil
	case err := <-errCh:
		return nil, err
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout waiting for registration response")
	}
}


func (c *Client) SendUnregisterTunnel(tunnelID string) error {
	payload := model.UnregisterPayload{
		TunnelID: tunnelID,
	}

	msg, err := model.NewMessage(model.MessageTypeUnregister, payload)
	if err != nil {
		return fmt.Errorf("failed to create message: %v", err)
	}

	return c.sendMessage(msg)
}


// SendData sends data through the tunnel with retry mechanism.
func (c *Client) SendData(tunnelID string, connectionID string, data []byte) error {
	const maxRetries = 3
	const initialBackoff = 100 * time.Millisecond
	const maxBackoff = 2 * time.Second

	payload := model.DataPayload{
		TunnelID:     tunnelID,
		ConnectionID: connectionID,
		Data:         data,
	}

	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt < maxRetries; attempt++ {
		msg, err := model.NewMessage(model.MessageTypeData, payload)
		if err != nil {
			c.logger.Error("Failed to create message (attempt %d/%d): %v", 
				attempt+1, maxRetries, err)
			return fmt.Errorf("failed to create message: %v", err)
		}

		err = c.sendMessage(msg)
		if err == nil {
			if attempt > 0 {
				// Successfully sent data after %d retries
			}
			return nil
		}

		lastErr = err
		c.logger.Warn("Failed to send data (attempt %d/%d): %v", 
			attempt+1, maxRetries, err)

		// Exponential backoff with jitter
		time.Sleep(backoff)
		backoff = time.Duration(float64(backoff) * 1.5)
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}

	return fmt.Errorf("failed to send data after %d attempts: %v", maxRetries, lastErr)
}


func (c *Client) GetSubdomain() string {
	return c.subdomain
}


func (c *Client) SetSubdomain(subdomain string) {
	c.subdomain = subdomain
}


func (c *Client) GetUserData() *model.AuthData {
	return c.userData
}


func (c *Client) CheckTunnelLimit() (bool, int, int) {

	if c.userData == nil {
		return false, 0, 0
	}
	

	limits := c.userData.Subscription.Limits.Tunnels
	
	// Check if tunnel limit has been reached
	reached := limits.Reached || limits.Used >= limits.Limit
	
	return reached, limits.Used, limits.Limit
}


var _ port.Client = (*Client)(nil)

package transport

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/haxorport/haxorport-go-client/internal/domain/model"
	"github.com/haxorport/haxorport-go-client/internal/domain/port"
)

type TunnelRepository struct {
	client      *Client
	logger      port.Logger
	tunnels     map[string]*model.Tunnel
	connections map[string]net.Conn
	mutex       sync.RWMutex
	ctx         context.Context
}

const (
	maxRetries          = 5
	readTimeout         = 30 * time.Second
	sshReadTimeout      = 60 * time.Second
	dialTimeout         = 5 * time.Second
	keepAlivePeriod     = 30 * time.Second
	bufferSize          = 128 * 1024 // 128KB
	maxConnectionErrors = 5
	writeTimeout        = 30 * time.Second
	maxMessageSize      = 32 * 1024  // 32KB
	initialBackoff      = 100 * time.Millisecond
	maxBackoff          = 5 * time.Second
	sshHandshakeTimeout = 15 * time.Second
)

// NewTunnelRepository returns a new instance of TunnelRepository.
func NewTunnelRepository(client *Client, logger port.Logger) *TunnelRepository {
	// Use background context as default
	ctx := context.Background()
	
	repo := &TunnelRepository{
		client:      client,
		logger:      logger,
		tunnels:     make(map[string]*model.Tunnel),
		connections: make(map[string]net.Conn),
		mutex:       sync.RWMutex{},
		ctx:         ctx,
	}

	// Register handler for data messages
	client.RegisterHandler(model.MessageTypeData, repo.handleDataMessage)

	return repo
}

// Register registers a new tunnel with the given configuration.
func (r *TunnelRepository) Register(config model.TunnelConfig) (*model.Tunnel, error) {
	// Ensure the client is connected
	if !r.client.IsConnected() {
		if err := r.client.Connect(); err != nil {
			return nil, fmt.Errorf("failed to connect to server: %v", err)
		}
	}

	// Send register tunnel request to server
	response, err := r.client.SendRegisterTunnel(config)
	if err != nil {
		return nil, fmt.Errorf("failed to register tunnel: %v", err)
	}

	// Check if registration was successful
	if !response.Success {
		return nil, fmt.Errorf("tunnel registration failed: %s", response.Error)
	}

	// Create a new tunnel instance
	tunnel := model.NewTunnel(response.TunnelID, config)

	// Set HTTP or TCP info based on tunnel type
	if config.Type == model.TunnelTypeHTTP {
		tunnel.SetHTTPInfo(response.URL)
	} else if config.Type == model.TunnelTypeTCP {
		tunnel.SetTCPInfo(response.RemotePort)
	}

	// Store the tunnel in the repository
	r.mutex.Lock()
	r.tunnels[response.TunnelID] = tunnel
	r.mutex.Unlock()

	// Start the tunnel listener if it's a TCP tunnel
	if config.Type == model.TunnelTypeTCP {
		go r.startTunnelListener(tunnel)
	}

	return tunnel, nil
}

// Unregister unregisters a tunnel with the given ID.
func (r *TunnelRepository) Unregister(tunnelID string) error {
	// Ensure the client is connected
	if !r.client.IsConnected() {
		if err := r.client.Connect(); err != nil {
			return fmt.Errorf("failed to connect to server: %v", err)
		}
	}

	// Send unregister tunnel request to server
	if err := r.client.SendUnregisterTunnel(tunnelID); err != nil {
		return fmt.Errorf("failed to remove tunnel: %v", err)
	}

	// Remove the tunnel from the repository
	r.mutex.Lock()
	delete(r.tunnels, tunnelID)
	r.mutex.Unlock()

	return nil
}

// GetAll returns all tunnels in the repository.
func (r *TunnelRepository) GetAll() []*model.Tunnel {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	tunnels := make([]*model.Tunnel, 0, len(r.tunnels))
	for _, tunnel := range r.tunnels {
		tunnels = append(tunnels, tunnel)
	}

	return tunnels
}

// GetByID returns a tunnel by its ID.
func (r *TunnelRepository) GetByID(tunnelID string) (*model.Tunnel, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	tunnel, exists := r.tunnels[tunnelID]
	if !exists {
		return nil, fmt.Errorf("tunnel with ID %s not found", tunnelID)
	}

	return tunnel, nil
}

// SendData sends data to a tunnel.
func (r *TunnelRepository) SendData(tunnelID string, connectionID string, data []byte) error {
	return r.client.SendData(tunnelID, connectionID, data)
}

// HandleData handles data for a tunnel.
func (r *TunnelRepository) HandleData(tunnelID string, connectionID string, data []byte) error {
	// Handling data for tunnel %s, connection %s

	r.mutex.RLock()
	conn, exists := r.connections[connectionID]
	r.mutex.RUnlock()

	if !exists {
		r.logger.Error("Connection %s not found", connectionID)
		return fmt.Errorf("connection not found")
	}

	// Set a deadline for writing to avoid blocking indefinitely
	if err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		r.logger.Warn("Failed to set write deadline for connection %s: %v", connectionID, err)
	}

	// Coba tulis data beberapa kali jika gagal
	var writeErr error
	var n int
	for attempts := 0; attempts < 3; attempts++ {
		n, writeErr = conn.Write(data)
		if writeErr == nil {
			break // Penulisan berhasil
		}
		
		r.logger.Warn("Attempt %d to write data to connection %s failed: %v, retrying...", attempts+1, connectionID, writeErr)
		time.Sleep(100 * time.Millisecond)
	}

	// Reset write deadline
	conn.SetWriteDeadline(time.Time{})

	if writeErr != nil {
		r.logger.Error("All attempts to write data to connection %s failed: %v", connectionID, writeErr)
		return fmt.Errorf("failed to write data to connection after multiple attempts: %v", writeErr)
	}

	if n < len(data) {
		r.logger.Warn("Partial write to connection %s: %d/%d bytes", connectionID, n, len(data))
		
		// Coba tulis sisa data jika ada penulisan parsial
		if n > 0 && n < len(data) {
			remaining := data[n:]
			// Attempting to write remaining %d bytes to connection %s
			
			// Set deadline again for writing remaining data
			conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			
			nRemaining, errRemaining := conn.Write(remaining)
			
			// Reset write deadline
			conn.SetWriteDeadline(time.Time{})
			
			if errRemaining != nil {
				r.logger.Error("Failed to write remaining data to connection %s: %v", connectionID, errRemaining)
			} else {
				// Wrote additional %d bytes to connection %s
				n += nRemaining
			}
		}
	}

	// Successfully wrote data to connection %s
	return nil
}

// handleDataMessage handles incoming data messages from the server.
func (r *TunnelRepository) handleDataMessage(msg *model.Message) error {
	var payload model.DataPayload
	if err := msg.ParsePayload(&payload); err != nil {
		r.logger.Error("Failed to parse data payload: %v", err)
		return fmt.Errorf("failed to parse data payload: %v", err)
	}
	
	// Detect message size for debugging
	if len(payload.Data) > 1024 {
		// Received large data message: %d bytes
	}

	// Log received data with more details for debugging
	if len(payload.Data) > 0 {
		// Display maximum of first 32 bytes for debugging
		previewLen := 32
		if len(payload.Data) < previewLen {
			previewLen = len(payload.Data)
		}
		
		// Detect SSH handshake with a simpler and more reliable approach
		if len(payload.Data) >= 4 {
			// Check first 4 bytes for "SSH-" (0x53, 0x53, 0x48, 0x2d)
			if bytes.HasPrefix(payload.Data, []byte{0x53, 0x53, 0x48, 0x2d}) {
				// Log only first 64 bytes to avoid excessive logging
				logSize := 64
				if len(payload.Data) < logSize {
					logSize = len(payload.Data)
				}
				r.logger.Info("SSH handshake detected, data length: %d bytes", logSize)
			}
		}
		r.logger.Info("Handling data for tunnel %s, connection %s, length: %d bytes", 
		payload.TunnelID, payload.ConnectionID, len(payload.Data))
	}

	r.mutex.RLock()
	_, exists := r.connections[payload.ConnectionID]
	r.mutex.RUnlock()

	if !exists {
		tunnel, err := r.GetByID(payload.TunnelID)
		if err != nil {
			r.logger.Error("Tunnel not found: %v", err)
			return fmt.Errorf("tunnel not found: %v", err)
		}

		localAddr := net.JoinHostPort(tunnel.Config.LocalAddr, fmt.Sprintf("%d", tunnel.Config.LocalPort))
		r.logger.Info("Connecting to local service at %s for connection %s...", localAddr, payload.ConnectionID)

		// Try multiple times to connect to local service
		var dialErr error
		var conn net.Conn
		
		for attempts := 0; attempts < 5; attempts++ {
			dialer := &net.Dialer{
				Timeout: dialTimeout,
				KeepAlive: keepAlivePeriod,
			}
			conn, dialErr = dialer.Dial("tcp", localAddr)
			if dialErr == nil {
				r.logger.Info("Connected to local service at %s", localAddr)
				break
			}
			r.logger.Warn("Failed to connect to %s (attempt %d): %v", localAddr, attempts+1, dialErr)
			time.Sleep(time.Duration(attempts+1) * 200 * time.Millisecond)
		}

		if dialErr != nil {
			r.logger.Error("All attempts to connect to local service at %s failed: %v", localAddr, dialErr)
			return fmt.Errorf("failed to connect to local service after multiple attempts: %v", dialErr)
		}

		// Set TCP_NODELAY to reduce latency (important for SSH)
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.SetNoDelay(true)
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(keepAlivePeriod)
			// Add larger buffer size
			if err := tcpConn.SetReadBuffer(bufferSize); err != nil {
				r.logger.Warn("Failed to set read buffer size: %v", err)
			}
			if err := tcpConn.SetWriteBuffer(bufferSize); err != nil {
				r.logger.Warn("Failed to set write buffer size: %v", err)
			}
		}

		r.logger.Info("Successfully connected to local service at %s for connection %s", localAddr, payload.ConnectionID)

		r.mutex.Lock()
		r.connections[payload.ConnectionID] = conn
		r.mutex.Unlock()

		r.logger.Info("Starting data forwarding from remote to local for connection %s...", payload.ConnectionID)

		// Start goroutine to read from local connection
		go r.handleConnection(payload.TunnelID, payload.ConnectionID, conn)

		if len(payload.Data) > 0 {
			r.logger.Debug("Forwarding initial %d bytes to local connection %s", len(payload.Data), payload.ConnectionID)
			
			// Always use longer timeout for initial data
			if err := conn.SetWriteDeadline(time.Now().Add(sshHandshakeTimeout)); err != nil {
				r.logger.Error("Failed to set extended write deadline: %v", err)
			}
			
			// Prioritaskan pengiriman data dengan TCP_NODELAY
			if tcpConn, ok := conn.(*net.TCPConn); ok {
				tcpConn.SetNoDelay(true) // Pastikan TCP_NODELAY aktif
				r.logger.Info("TCP_NODELAY set for data forwarding")
			}
			
			// Write data directly without buffering for initial data
			_, writeErr := conn.Write(payload.Data)
			if writeErr != nil {
				r.logger.Error("Failed to write initial data: %v", writeErr)
				
				// If failed to write initial data, try again with retries
				for i := 0; i < maxRetries; i++ {
					time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
					_, writeErr = conn.Write(payload.Data)
					if writeErr == nil {
						r.logger.Info("Successfully wrote initial data on retry %d", i+1)
						break
					}
					r.logger.Warn("Retry %d failed: %v", i+1, writeErr)
				}
			}
			
			// Reset deadline setelah penulisan
			if err := conn.SetWriteDeadline(time.Time{}); err != nil {
				r.logger.Error("Failed to reset write deadline: %v", err)
			}
			
			if writeErr != nil {
				r.logger.Error("Failed to write initial data after all retries: %v", writeErr)
			} else {
				r.logger.Debug("Successfully forwarded initial data to local connection")
			}
		}
	} else {
		r.logger.Debug("Using existing connection for tunnel %s, connection %s",
			payload.TunnelID, payload.ConnectionID)
		
		// Ambil koneksi yang sudah ada
		r.mutex.RLock()
		conn, ok := r.connections[payload.ConnectionID]
		r.mutex.RUnlock()
		
		if !ok {
			r.logger.Error("Connection %s exists flag is true but connection not found in map", payload.ConnectionID)
			return fmt.Errorf("connection not found in map: %s", payload.ConnectionID)
		}

		if len(payload.Data) > 0 {
			r.logger.Debug("Forwarding %d bytes to local connection %s", len(payload.Data), payload.ConnectionID)
			
			// Set deadline for writing
			if err := conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
				r.logger.Error("Failed to set write deadline: %v", err)
			}
			
			// Write data directly to avoid buffering
			_, writeErr := conn.Write(payload.Data)
			if writeErr != nil {
				r.logger.Error("Failed to write data: %v", writeErr)
				
				// Jika gagal, coba lagi dengan retry
				for i := 0; i < maxRetries; i++ {
					time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
					_, writeErr = conn.Write(payload.Data)
					if writeErr == nil {
						r.logger.Info("Successfully wrote data on retry %d", i+1)
						break
					}
					r.logger.Warn("Retry %d failed: %v", i+1, writeErr)
				}
			}
			
			// Reset deadline setelah penulisan
			if err := conn.SetWriteDeadline(time.Time{}); err != nil {
				r.logger.Error("Failed to reset write deadline: %v", err)
			}
			
			if writeErr != nil {
				r.logger.Error("Failed to write data after all retries: %v", writeErr)
			} else {
				r.logger.Debug("Successfully forwarded data to local connection")
			}
		}
	}

	return nil
}

// startTunnelListener starts a tunnel to forward connections to the local service
func (r *TunnelRepository) startTunnelListener(tunnel *model.Tunnel) {
	// Don't create a new listener, just log that we're ready to forward connections
	localAddr := fmt.Sprintf("%s:%d", tunnel.Config.LocalAddr, tunnel.Config.LocalPort)
	r.logger.Info("Ready to forward connections to local service at %s", localAddr)

	// The actual connection handling will be done in handleConnection
	// when data is received from the server
}

// handleConnection handles a connection to a tunnel.
// sendDataWithRetryFunc is a function type for sending data with retry mechanism
type sendDataWithRetryFunc func(tunnelID, connectionID string, data []byte) error

// sendDataWithRetry sends data with a retry mechanism that will try multiple times if it fails
func (r *TunnelRepository) sendDataWithRetry(tunnelID, connectionID string, data []byte, maxRetries int, initialBackoff, maxBackoff time.Duration) error {
	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := r.client.SendData(tunnelID, connectionID, data)
		if err == nil {
			if attempt > 0 {
				r.logger.Debug("Successfully sent data after %d retries", attempt)
			}
			return nil
		}

		lastErr = err
		r.logger.Warn("Failed to send data (attempt %d/%d): %v", 
			attempt+1, maxRetries, err)

		// Exponential backoff with jitter
		time.Sleep(backoff)
		backoff = time.Duration(float64(backoff) * 1.5)
		if backoff > maxBackoff {
			backoff = maxBackoff
		}

		// Add jitter between 50-150% of backoff
		jitter := time.Duration(rand.Int63n(int64(backoff/2)) + int64(backoff/2))
		if jitter > 0 {
			backoff = jitter
		}
	}

	return fmt.Errorf("failed to send data after %d attempts: %v", maxRetries, lastErr)
}

func (r *TunnelRepository) handleConnection(tunnelID, connectionID string, conn net.Conn) {
	r.mutex.Lock()
	r.connections[connectionID] = conn
	r.mutex.Unlock()

	// Setup koneksi dengan parameter yang lebih baik
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)  // Non-blocking I/O
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
		
		// Set larger buffer size for better performance
		tcpConn.SetReadBuffer(128 * 1024)  // 128KB buffer
		tcpConn.SetWriteBuffer(128 * 1024) // 128KB buffer
	}

	defer func() {
		r.logger.Info("Closing connection %s", connectionID)
		conn.Close()
		
		r.mutex.Lock()
		delete(r.connections, connectionID)
		r.mutex.Unlock()
	}()
	
	r.logger.Info("Starting data forwarding from local to remote for connection %s on tunnel %s", connectionID, tunnelID)
	
	// Read initial data with longer timeout
	conn.SetReadDeadline(time.Now().Add(10 * time.Second)) // 10 second timeout for initial read
	buffer := make([]byte, 8192) // 8KB buffer for initial data
	n, err := conn.Read(buffer)
	
	// Reset deadline after reading
	conn.SetReadDeadline(time.Time{})
	
	// If there's an error reading initial data, log and return
	if err != nil {
		r.logger.Error("Error reading initial data for connection %s: %v", connectionID, err)
		return
	}
	
	// Save initial data to send later
	initialData := make([]byte, n)
	copy(initialData, buffer[:n])
	
	// Detect SSH from initial data
	isSSH := n >= 4 && bytes.HasPrefix(initialData, []byte{0x53, 0x53, 0x48, 0x2d}) // "SSH-"
	
	// Log connection information
	if isSSH {
		r.logger.Info("SSH connection detected for %s, applying optimizations", connectionID)
		
		// If this is SSH, set more aggressive TCP parameters
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.SetNoDelay(true)
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(10 * time.Second)
			tcpConn.SetReadBuffer(256 * 1024)  // 256KB buffer untuk SSH
			tcpConn.SetWriteBuffer(256 * 1024) // 256KB buffer untuk SSH
		}
	}
	
	// Log informasi tentang data awal
	r.logger.Info("Initial data received for connection %s: %d bytes, isSSH: %v", connectionID, n, isSSH)
	
	// Tampilkan preview data awal (maksimal 16 byte) untuk debugging
	if n > 0 {
		previewSize := n
		if previewSize > 16 {
			previewSize = 16
		}
		r.logger.Info("Initial data preview (%d bytes): %v", previewSize, initialData[:previewSize])
	}
	
	// Send initial data that has been read (if any)
	if len(initialData) > 0 {
		r.logger.Info("Sending initial data (%d bytes) for connection %s", len(initialData), connectionID)
		
		// Send initial data with high priority
		var sendErr error
		maxRetries := 5
		baseDelay := 50 * time.Millisecond
		maxDelay := 2 * time.Second
		
		for attempt := 0; attempt < maxRetries; attempt++ {
			sendErr = r.client.SendData(tunnelID, connectionID, initialData)
			if sendErr == nil {
				r.logger.Info("Successfully sent initial %d bytes for connection %s", len(initialData), connectionID)
				break
			}
			
			// Calculate delay with exponential backoff
			delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt)))
			if delay > maxDelay {
				delay = maxDelay
			}
			
			// Add random jitter between 50-150% of the calculated delay
			jitter := time.Duration(rand.Int63n(int64(delay)) + int64(delay/2))
			if jitter > 0 {
				delay = jitter
			}
			
			r.logger.Warn("Attempt %d/%d to send %d bytes failed: %v, retrying in %v...", 
				attempt+1, maxRetries, len(initialData), sendErr, delay)
			
			time.Sleep(delay)
		}
		
		if sendErr != nil {
			r.logger.Error("All attempts to send initial data to server failed: %v", sendErr)
			return
		}
	}
	
	// Use a larger buffer for SSH
	actualBufferSize := bufferSize
	if isSSH {
		actualBufferSize = 32768 // 32KB for SSH
	}
	// Use a new buffer with appropriate size
	bufferMain := make([]byte, actualBufferSize)
	
	// Create ticker for keepalive and deadline management
	keepaliveInterval := 15 * time.Second
	if isSSH {
		keepaliveInterval = 30 * time.Second
	}
	keepaliveTicker := time.NewTicker(keepaliveInterval)
	defer keepaliveTicker.Stop()

	// Create channel for error signals
	errCh := make(chan error, 1)

	// Goroutine for reading data from local connection
	type readResult struct {
		n   int
		err error
	}
	readCh := make(chan readResult, 1)

	// Function to read data
	readData := func() {
		n, err := conn.Read(bufferMain)
		readCh <- readResult{n: n, err: err}
	}

	// Read first data
	go readData()

	// Variable to track activity
	lastActivityTime := time.Now()
	
	for {
		// Set read deadline for different connections
		// Use timeout variables defined in constants
		// readTimeout or sshReadTimeout according to connection type

		select {
		case result := <-readCh:
			n, err := result.n, result.err
			
			// If successfully read data, update last activity time
			if err == nil && n > 0 {
				lastActivityTime = time.Now()
			}

			// Handle error
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// Timeout, continue to next iteration
					go readData()
					continue
				} else if err == io.EOF {
					r.logger.Info("Local connection %s closed by local service (EOF)", connectionID)
					return
				} else {
					r.logger.Error("Error reading from local connection %s: %v", connectionID, err)
					return
				}
			}

			// If there is data read, send to server
			if n > 0 {
				r.logger.Debug("Read %d bytes from local connection %s", n, connectionID)
				
				// Send data asynchronously
				go func(data []byte) {
					err := r.sendDataWithRetry(tunnelID, connectionID, data, 5, 50*time.Millisecond, 2*time.Second)
					if err != nil {
						r.logger.Error("Failed to send data to server: %v", err)
						errCh <- fmt.Errorf("failed to send data: %w", err)
					}
				}(append([]byte(nil), bufferMain[:n]...))
			}

			// Start reading next data
			go readData()

		case <-keepaliveTicker.C:
			// Send keepalive if there's no activity for a certain period
			if isSSH && time.Since(lastActivityTime) > 30*time.Second {
				r.logger.Debug("Sending SSH keepalive for connection %s", connectionID)
				go func() {
					if err := r.client.SendData(tunnelID, connectionID, []byte{}); err != nil {
						r.logger.Warn("Failed to send keepalive: %v", err)
					}
				}()
			}

		case err := <-errCh:
			r.logger.Error("Error in connection handler: %v", err)
			r.logger.Info("Shutting down connection handler for %s", connectionID)
			return
		}

		// Reset read deadline
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.SetReadDeadline(time.Time{}) // Reset deadline
		}
	}
}

package transport

import (
	"fmt"
	"net"
	"time"

	"github.com/haxorport/haxorport-go-client/internal/domain/model"
	"github.com/haxorport/haxorport-go-client/internal/domain/port"
)

// CreateTunnelRepository membuat instance TunnelRepository berdasarkan mode koneksi
func CreateTunnelRepository(config *model.Config, client port.Client, logger port.Logger) (port.TunnelRepository, error) {
	switch config.ConnectionMode {
	case model.ConnectionModeWebSocket:
		// Use existing WebSocket implementation
		if client == nil {
			return nil, fmt.Errorf("client must not be nil for WebSocket connection mode")
		}
		clientImpl, ok := client.(*Client)
		if !ok {
			return nil, fmt.Errorf("client bukan implementasi *Client")
		}
		return NewTunnelRepository(clientImpl, logger), nil
	case model.ConnectionModeDirectTCP:
		// Use new Direct TCP implementation
		// For direct TCP mode, client can be nil because we use direct TCP connection
		directRepo, err := NewDirectTunnelRepository(config, logger)
		if err != nil {
			return nil, err
		}
		return directRepo, nil
	default:
		return nil, fmt.Errorf("connection mode not supported: %s", config.ConnectionMode)
	}
}

// NewDirectTunnelRepository creates a new direct tunnel repository
func NewDirectTunnelRepository(config *model.Config, logger port.Logger) (port.TunnelRepository, error) {
	if config.ServerAddress == "" || config.ControlPort == 0 {
		return nil, fmt.Errorf("server address and control port must be set")
	}

	return &directTunnelRepository{
		config: config,
		logger: logger,
		tunnels: make(map[string]interface{}),
	}, nil
}

// directTunnelRepository is an implementation of TunnelRepository using direct TCP connection
type directTunnelRepository struct {
	config *model.Config
	logger port.Logger
	tunnels map[string]interface{}
}

// CreateTunnel creates a new direct tunnel
func (r *directTunnelRepository) CreateTunnel(localPort, remotePort int, targetHost string, targetPort int) (interface{}, error) {
	targetAddr := fmt.Sprintf("%s:%d", targetHost, targetPort)

	// If remotePort is 0, generate random port
	if remotePort == 0 {
		remotePort = generateRandomInt(10000, 30000)
		r.logger.Info("Using random port %d for tunnel", remotePort)
	}

	// In reverse tunnel, localAddr is not used for listening
	// but we still store it for reference
	localAddr := fmt.Sprintf("127.0.0.1:%d", localPort)

	// Use control port from configuration and auth settings
	tunnel := NewDirectTunnel(
		localAddr, 
		r.config.ServerAddress, 
		targetAddr, 
		localPort, 
		remotePort, 
		r.config.ControlPort, 
		r.logger,
		r.config.AuthEnabled,
		r.config.AuthToken,
	)
	return tunnel, nil
}

// Register mendaftarkan tunnel baru ke server
func (r *directTunnelRepository) Register(config model.TunnelConfig) (*model.Tunnel, error) {
	// If remote port is not specified, use random port in range 10000-30000
	if config.RemotePort == 0 {
		// Use random port for direct TCP
		minPort := 10000
		maxPort := 30000
		config.RemotePort = generateRandomInt(minPort, maxPort)
		r.logger.Info("Using random port %d for tunnel", config.RemotePort)
	}

	// Create tunnel model
	tunnel := model.NewTunnel(generateID(), config)
	tunnel.SetTCPInfo(config.RemotePort)

	// Use local port with offset 6000 to avoid conflicts with ports already in use
	localListenerPort := 6000 + config.LocalPort
	if localListenerPort < 1024 || localListenerPort > 65535 {
		localListenerPort = generateRandomInt(3000, 9000)
	}

	// Create direct tunnel
	targetHost := "127.0.0.1"
	targetPort := config.LocalPort
	directTunnel, err := r.CreateTunnel(localListenerPort, config.RemotePort, targetHost, targetPort)
	if err != nil {
		return nil, err
	}

	// Mulai tunnel
	dt, ok := directTunnel.(*DirectTunnel)
	if !ok {
		return nil, fmt.Errorf("unexpected tunnel type")
	}

	err = dt.Start()
	if err != nil {
		return nil, err
	}

	// Update RemotePort in model.Tunnel if server uses alternative port
	if dt.remotePort != config.RemotePort {
		r.logger.Warn("IMPORTANT: Server using alternative port %d (requested: %d), updating tunnel", dt.remotePort, config.RemotePort)
		tunnel.RemotePort = dt.remotePort
		tunnel.Config.RemotePort = dt.remotePort
		
		// Additional log to ensure alternative port is clearly visible
		r.logger.Info("Tunnel will use alternative port %d for connection", dt.remotePort)
	} else {
		r.logger.Info("Server is using the requested port: %d", dt.remotePort)
	}

	// Simpan tunnel
	r.tunnels[tunnel.ID] = directTunnel

	return tunnel, nil
}

// Unregister removes registered tunnel
func (r *directTunnelRepository) Unregister(tunnelID string) error {
	// Stop tunnel if exists
	if tunnel, ok := r.tunnels[tunnelID]; ok {
		if dt, ok := tunnel.(*DirectTunnel); ok {
			dt.Stop()
			delete(r.tunnels, tunnelID)
			r.logger.Info("Tunnel removed: %s", tunnelID)
			return nil
		}
	}
	
	r.logger.Warn("Tunnel not found: %s", tunnelID)
	return fmt.Errorf("tunnel not found: %s", tunnelID)
}

// GetByID mengembalikan tunnel berdasarkan ID
func (r *directTunnelRepository) GetByID(tunnelID string) (*model.Tunnel, error) {
	// Cek apakah tunnel ada
	if directTunnel, ok := r.tunnels[tunnelID]; ok {
		// Konversi dari interface{} ke *DirectTunnel
		tunnel, ok := directTunnel.(*DirectTunnel)
		if !ok {
			return nil, fmt.Errorf("tunnel with ID %s is not of type *DirectTunnel", tunnelID)
		}
		
		// Parse targetAddr to get host and port
		host, _, err := net.SplitHostPort(tunnel.targetAddr)
		if err != nil {
			r.logger.Warn("Gagal memparse targetAddr %s: %v", tunnel.targetAddr, err)
			host = "127.0.0.1"
		}
		
		// Buat model.Tunnel dari DirectTunnel
		modelTunnel := &model.Tunnel{
			ID:         tunnelID,
			RemotePort: tunnel.remotePort,
			Active:     true,
			Config: model.TunnelConfig{
				LocalPort:  tunnel.localPort,
				RemotePort: tunnel.remotePort,
				LocalAddr:  host,
				Type:       model.TunnelTypeTCP,
			},
		}
		
		return modelTunnel, nil
	}

	return nil, fmt.Errorf("tunnel not found: %s", tunnelID)
}

// GetDirectTunnel mengembalikan instance DirectTunnel langsung berdasarkan ID
// This is used to access special DirectTunnel methods like SetPortChangeCallback
func (r *directTunnelRepository) GetDirectTunnel(tunnelID string) interface{ SetPortChangeCallback(func(int)) } {
	// Cek apakah tunnel ada
	if directTunnel, ok := r.tunnels[tunnelID]; ok {
		// Konversi dari interface{} ke *DirectTunnel
		tunnel, ok := directTunnel.(*DirectTunnel)
		if !ok {
			r.logger.Warn("tunnel with ID %s is not of type *DirectTunnel", tunnelID)
			return nil
		}
		return tunnel
	}
	r.logger.Warn("tunnel not found: %s", tunnelID)
	return nil
}

// GetAll returns all registered tunnels
func (r *directTunnelRepository) GetAll() []*model.Tunnel {
	tunnels := make([]*model.Tunnel, 0, len(r.tunnels))
	
	// Iterasi semua tunnel dan konversi ke model.Tunnel
	for id, t := range r.tunnels {
		directTunnel, ok := t.(*DirectTunnel)
		if !ok {
			r.logger.Warn("Tunnel with ID %s is not of type *DirectTunnel", id)
			continue
		}
		
		// Parse targetAddr to get host
		host, _, err := net.SplitHostPort(directTunnel.targetAddr)
		if err != nil {
			r.logger.Warn("Failed to parse targetAddr %s: %v", directTunnel.targetAddr, err)
			host = "127.0.0.1"
		}
		
		// Buat model.Tunnel dari DirectTunnel
		modelTunnel := &model.Tunnel{
			ID:         id,
			RemotePort: directTunnel.remotePort,
			Active:     true,
			Config: model.TunnelConfig{
				LocalPort:  directTunnel.localPort,
				RemotePort: directTunnel.remotePort,
				LocalAddr:  host,
				Type:       model.TunnelTypeTCP,
			},
		}
		
		tunnels = append(tunnels, modelTunnel)
	}
	
	return tunnels
}

// SendData sends data through the tunnel
func (r *directTunnelRepository) SendData(tunnelID string, connectionID string, data []byte) error {
	// Dalam implementasi reverse tunnel, data langsung dikirim melalui koneksi TCP
	// sehingga tidak perlu implementasi khusus di sini.
	// Data sudah ditangani oleh goroutine io.Copy di DirectTunnel
	r.logger.Debug("SendData called for tunnel %s, connection %s, %d bytes", tunnelID, connectionID, len(data))
	return nil
}

// HandleData handles data received from server
func (r *directTunnelRepository) HandleData(tunnelID string, connectionID string, data []byte) error {
	// Dalam implementasi reverse tunnel, data langsung dikirim melalui koneksi TCP
	// sehingga tidak perlu implementasi khusus di sini.
	// Data sudah ditangani oleh goroutine io.Copy di DirectTunnel
	r.logger.Debug("HandleData called for tunnel %s, connection %s, %d bytes", tunnelID, connectionID, len(data))
	return nil
}

// Helpers
func generateID() string {
	return fmt.Sprintf("tunnel-%d", time.Now().UnixNano())
}


// generateRandomInt menghasilkan angka acak dalam rentang min-max
func generateRandomInt(min, max int) int {
	return min + int(time.Now().UnixNano() % int64(max-min+1))
}

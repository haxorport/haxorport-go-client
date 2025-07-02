package transport

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/haxorport/haxorport-go-client/internal/domain/port"
)

// DirectTunnel adalah implementasi tunnel TCP langsung
type DirectTunnel struct {
	localAddr      string
	serverAddr     string
	targetAddr     string
	localPort      int
	remotePort     int
	controlPort    int
	listener       net.Listener
	// controlListener dihapus, menggunakan koneksi keluar saja
	connection     net.Conn
	stopped        bool
	mutex          sync.Mutex
	logger         port.Logger
	// Callback to update SSH Access information
	portChangeCallback func(int)
	authEnabled        bool
	authToken          string
}

// NewDirectTunnel creates a new DirectTunnel instance
func NewDirectTunnel(localAddr, serverAddr, targetAddr string, localPort, remotePort, controlPort int, logger port.Logger, authEnabled bool, authToken string) *DirectTunnel {
	return &DirectTunnel{
		localAddr:      localAddr,
		serverAddr:     serverAddr,
		targetAddr:     targetAddr,
		localPort:      localPort,
		remotePort:     remotePort,
		controlPort:    controlPort,
		logger:         logger,
		authEnabled:    authEnabled,
		authToken:      authToken,
		portChangeCallback: nil,
	}
}

// SetPortChangeCallback mengatur callback yang akan dipanggil ketika port berubah
func (t *DirectTunnel) SetPortChangeCallback(callback func(int)) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.portChangeCallback = callback
}

// Start memulai tunnel
func (t *DirectTunnel) Start() error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.listener != nil || t.connection != nil {
		return fmt.Errorf("tunnel sudah berjalan")
	}
	
	// Reset flag stopped
	t.stopped = false

	// Create local listener for connections from local application
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", t.localPort))
	if err != nil {
		return fmt.Errorf("failed to start listener on port %d: %v", t.localPort, err)
	}

	// Create connection to server to register the tunnel
	// Menggunakan net.JoinHostPort untuk mendukung IPv6
	serverAddr := net.JoinHostPort(t.serverAddr, fmt.Sprintf("%d", t.controlPort))
	serverConn, err := net.DialTimeout("tcp", serverAddr, 10*time.Second)
	if err != nil {
		listener.Close()
		return fmt.Errorf("failed to connect to server: %v", err)
	}
	t.connection = serverConn

	// Prepare data to send with auth token if enabled
	var dataToSend string
	if t.authEnabled && t.authToken != "" {
		// Format with AUTH_TOKEN prefix
		dataToSend = fmt.Sprintf("AUTH_TOKEN=%s:%s:%d:DIRECT_TCP_FORWARD", t.authToken, t.targetAddr, t.remotePort)
		t.logger.Info("Establishing authenticated outbound control connection to server on port %d", t.controlPort)
	} else {
		// Format without auth token
		dataToSend = fmt.Sprintf("%s:%d:DIRECT_TCP_FORWARD", t.targetAddr, t.remotePort)
		t.logger.Info("Establishing outbound control connection to server on port %d", t.controlPort)
	}

	// Send data to server
	_, err = t.connection.Write([]byte(dataToSend))
	if err != nil {
		return fmt.Errorf("failed to send data to server: %v", err)
	}

	// Add log for debugging

	// Read confirmation from server
	// Use larger buffer for SSH
	actualBufferSize := 4096
	isSSH := strings.Contains(t.targetAddr, ":22") // Detect SSH based on port 22
	if isSSH {
		actualBufferSize = 65536 // 64KB for SSH, increased from 32KB
	}
	buffer := make([]byte, actualBufferSize)
	t.connection.SetReadDeadline(time.Now().Add(15 * time.Second)) // Increased timeout from 10 seconds to 15 seconds
	n, err := t.connection.Read(buffer)
	t.connection.SetReadDeadline(time.Time{})
	if err != nil {
		return fmt.Errorf("failed to read response from server: %v", err)
	}

	// Log data awal dalam format hex untuk debugging

	// Periksa apakah respons mengandung error atau konfirmasi dengan port alternatif
	response := string(buffer[:n])
	
	// Jika respons mengandung error, coba dengan port baru
	if strings.Contains(response, "ERROR") {
		t.logger.Warn("Server error response: %s. Will try with a different port", strings.TrimSpace(response))

		// Tutup koneksi saat ini
		t.connection.Close()

		// Tunggu sebentar sebelum mencoba lagi
		time.Sleep(500 * time.Millisecond)

		// Coba dengan port acak yang berbeda
		rand.Seed(time.Now().UnixNano())
		// Gunakan rentang port sesuai dengan konfigurasi server relay
		t.remotePort = 10000 + rand.Intn(20000) // Port acak antara 10000-30000
		t.logger.Info("Trying with new random port: %d", t.remotePort)
		return t.Start()
	}
	
	// Periksa apakah server memberikan port alternatif
	if strings.HasPrefix(response, "CONNECTED:") {
		parts := strings.Split(response, ":")
		if len(parts) >= 2 {
			actualPort, err := strconv.Atoi(parts[1])
			if err == nil {
				if actualPort != t.remotePort {
					t.mutex.Lock()
					oldPort := t.remotePort
					t.logger.Info("Using alternative port: %d (requested: %d)", actualPort, t.remotePort)
					t.remotePort = actualPort
					
					// Panggil callback jika ada
					if t.portChangeCallback != nil {
						t.portChangeCallback(actualPort)
					}
					t.mutex.Unlock()
					
					// Log dengan level lebih tinggi untuk memastikan terlihat
					t.logger.Warn("IMPORTANT: Alternative port %d is used by server for main connection (previous: %d)", actualPort, oldPort)
				} else {
					// Server confirmed requested port: %d
				}
			} else {
				t.logger.Warn("Gagal memparse port dari respons server: %s", response)
			}
		} else {
			t.logger.Warn("Format respons server tidak valid: %s", response)
		}
	} else if !strings.Contains(response, "CONNECTED") {
		// Respons tidak mengandung CONNECTED sama sekali
		t.logger.Warn("Unexpected response from server: %s", strings.TrimSpace(response))
		t.connection.Close()
		return fmt.Errorf("unexpected response from server: %s", strings.TrimSpace(response))
	}

	// Konfirmasi sudah diperiksa di atas, jadi jika sampai di sini berarti sudah "CONNECTED"
	// Server connection established

	t.listener = listener
	t.logger.Info("Tunnel active: localhost:%d -> %s:%d", t.localPort, t.serverAddr, t.remotePort)
	
	// Tidak perlu membuat listener kontrol di client
	// Semua komunikasi akan menggunakan koneksi keluar yang sudah ada
	t.logger.Info("Using outbound connection for all server communications")

	// Terima koneksi lokal
	go func() {
		for {
			localConn, err := listener.Accept()
			if err != nil {
				if !t.isStopped() {
					t.logger.Error("Failed to accept connection: %v", err)
				}
				break
			}

			go t.handleLocalConnection(localConn)
		}
	}()
	
	// Tidak perlu menerima koneksi kontrol dari server
	// Semua komunikasi akan menggunakan koneksi keluar yang sudah ada

	// Create control connection to server for DIRECT_TCP_FORWARD mode
	go func() {
		// Gunakan koneksi keluar alih-alih listener masuk
		// This avoids the need to open ports in client firewall
		t.logger.Info("Establishing outbound control connection to server on port %d", t.controlPort)

		// Create channel to signal when tunnel is stopped
		stopCh := make(chan struct{})

		// Goroutine to detect when tunnel is stopped
		go func() {
			for {
				if t.isStopped() {
					close(stopCh)
					return
				}
				time.Sleep(1 * time.Second)
			}
		}()

		// Loop to retry connection if disconnected
		for {
			select {
			case <-stopCh:
				t.logger.Info("Stopping control connection loop")
				return
			default:
				// Create control connection to server
				// Using net.JoinHostPort to support IPv6
				serverAddr := net.JoinHostPort(t.serverAddr, fmt.Sprintf("%d", t.controlPort))
				controlConn, err := net.DialTimeout("tcp", serverAddr, 10*time.Second)
				if err != nil {
					// Jika server tidak dapat dihubungi, coba cek apakah tunnel masih berjalan
					if t.isStopped() {
						t.logger.Info("Tunnel sudah dihentikan, berhenti mencoba koneksi")
						return
					}
					
					// Use exponential backoff for retries
					backoffTime := 5 * time.Second
					t.logger.Error("Failed to establish control connection: %v. Retrying in %v", err, backoffTime)
					
					// Cek apakah error menunjukkan server tidak tersedia
					if strings.Contains(err.Error(), "connection refused") || 
					   strings.Contains(err.Error(), "no route to host") || 
					   strings.Contains(err.Error(), "network is unreachable") {
						t.logger.Warn("Server mungkin tidak tersedia, menunggu lebih lama sebelum mencoba lagi")
						backoffTime = 30 * time.Second
					}
					
					time.Sleep(backoffTime)
					continue
				}

				// Send registration message with CONTROL_CONNECTION flag
				var dataToSend string
				if t.authEnabled && t.authToken != "" {
					// Tambahkan token autentikasi jika diaktifkan
					dataToSend = fmt.Sprintf("AUTH_TOKEN=%s:%s:%d:CONTROL_CONNECTION", t.authToken, t.targetAddr, t.remotePort)
				} else {
					dataToSend = fmt.Sprintf("%s:%d:CONTROL_CONNECTION", t.targetAddr, t.remotePort)
				}
				_, err = controlConn.Write([]byte(dataToSend))
				if err != nil {
					t.logger.Error("Failed to send control registration: %v", err)
					controlConn.Close()
					time.Sleep(5 * time.Second)
					continue
				}
				
				// Baca respons dari server
				buffer := make([]byte, 1024)
				controlConn.SetReadDeadline(time.Now().Add(10 * time.Second))
				n, err := controlConn.Read(buffer)
				controlConn.SetReadDeadline(time.Time{})
				
				if err != nil {
					if strings.Contains(err.Error(), "use of closed network connection") {
						t.logger.Warn("Server closed connection, waiting before retry")
						time.Sleep(10 * time.Second)
					} else {
						t.logger.Error("Failed to read response from server: %v", err)
						time.Sleep(5 * time.Second)
					}
					controlConn.Close()
					continue
				}
				
				response := string(buffer[:n])
				if !strings.Contains(response, "CONNECTED") {
					t.logger.Warn("Unexpected response from server: %s", response)
					controlConn.Close()
					time.Sleep(5 * time.Second)
					continue
				}

				// Check if the server provides an alternative port for the control connection
				if strings.HasPrefix(response, "CONNECTED:") {
					parts := strings.Split(response, ":")
					if len(parts) >= 2 {
						actualPort, err := strconv.Atoi(parts[1])
						if err == nil && actualPort != t.remotePort {
							t.mutex.Lock()
							oldPort := t.remotePort
							t.logger.Info("Server assigned alternative port for control connection: %d (requested: %d)", actualPort, t.remotePort)
							t.remotePort = actualPort
							
							// Call the callback if it exists
							if t.portChangeCallback != nil {
								t.portChangeCallback(actualPort)
							}
							t.mutex.Unlock()
							
							// Log with higher level to ensure visibility
							t.logger.Warn("IMPORTANT: Server is using alternative port %d for control connection (previously: %d)", actualPort, oldPort)
						}
					}
				}
				
				// Control connection established successfully
				
				// Handle the control connection
				t.handleControlConnection(controlConn)
				
				// If we reach here, the connection has been closed
				t.logger.Info("Control connection lost, reconnecting...")
				time.Sleep(5 * time.Second)

				// If the connection is lost and the tunnel is still active, try to reconnect
				if !t.isStopped() {
					t.logger.Info("Reconnecting control connection...")
					time.Sleep(5 * time.Second)
				} else {
					return
				}
			}
		}
	}()

	return nil
}

// TODO: Fungsi ini akan digunakan untuk implementasi NAT traversal di masa depan
// GetOutboundIP mendapatkan alamat IP yang digunakan untuk koneksi keluar
// Ini akan mengembalikan alamat IP publik client yang dapat diakses oleh server
func (t *DirectTunnel) GetOutboundIP() string {
	// Coba dapatkan IP dari koneksi yang sudah ada ke server
	if t.connection != nil {
		localAddr := t.connection.LocalAddr().String()
		host, _, err := net.SplitHostPort(localAddr)
		if err == nil && host != "" && host != "::" && !strings.HasPrefix(host, "127.") {
			return host
		}
	}

	// Jika tidak bisa mendapatkan dari koneksi yang ada, coba buat koneksi baru
	// Kita tidak perlu benar-benar terhubung, hanya perlu mendapatkan alamat IP lokal
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err == nil {
		defer conn.Close()
		localAddr := conn.LocalAddr().String()
		var host string
		host, _, err = net.SplitHostPort(localAddr)
		if err == nil && host != "" && host != "::" && !strings.HasPrefix(host, "127.") {
			return host
		}
	}

	// Jika semua cara gagal, coba dapatkan alamat IP dari interface jaringan
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					return ipnet.IP.String()
				}
			}
		}
	}

	// Jika semua cara gagal, gunakan 127.0.0.1 sebagai fallback
	t.logger.Warn("Failed to get outbound IP, using 127.0.0.1 as fallback")
	return "127.0.0.1"
}

// Stop stops the tunnel and closes the connection
func (t *DirectTunnel) Stop() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.stopped = true

	// Tutup listener jika ada``
	if t.listener != nil {
		// Stopping local listener on port %d
		t.listener.Close()
		t.listener = nil
	}

	// Tidak perlu menutup control listener karena sudah tidak digunakan

	// Tutup koneksi ke server jika ada
	if t.connection != nil {
		t.logger.Info("Stopping tunnel to %s (remote port: %d)", t.serverAddr, t.remotePort)
		t.connection.Close()
		t.connection = nil
	}
}

// isStopped checks if the tunnel has been stopped
func (t *DirectTunnel) isStopped() bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.stopped
}

// bidirectionalCopy melakukan copy data dua arah antara dua koneksi
func (t *DirectTunnel) bidirectionalCopy(conn1, conn2 net.Conn) {
	// Gunakan WaitGroup untuk menunggu kedua goroutine selesai
	var wg sync.WaitGroup
	wg.Add(2)
	
	// Copy dari conn1 ke conn2
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				t.logger.Error("Panic in copy routine: %v", r)
			}
		}()
		
		_, err := io.Copy(conn2, conn1)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
			t.logger.Error("Error copying data: %v", err)
		}
		
		// Close the write side of conn2 to signal EOF
		if tcpConn, ok := conn2.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
	}()
	
	// Copy dari conn2 ke conn1
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				t.logger.Error("Panic in copy routine: %v", r)
			}
		}()
		
		_, err := io.Copy(conn1, conn2)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
			t.logger.Error("Error copying data: %v", err)
		}
		
		// Close the write side of conn1 to signal EOF
		if tcpConn, ok := conn1.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
	}()
	
	// Tunggu kedua goroutine selesai
	wg.Wait()
}

// handleControlConnection menangani koneksi kontrol dari server
func (t *DirectTunnel) handleControlConnection(controlConn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			t.logger.Error("Panic in handleControlConnection: %v", r)
		}
		controlConn.Close()
	}()
	
	
	// Baca permintaan dari server
	buffer := make([]byte, 1024)
	n, err := controlConn.Read(buffer)
	if err != nil {
		t.logger.Error("Failed to read from control connection: %v", err)
		return
	}
	
	// Parse permintaan
	request := string(buffer[:n])
	
	// Periksa jenis permintaan
	if strings.HasPrefix(request, "CONNECT:") {
		// Format: CONNECT:targetAddr:targetPort
		parts := strings.Split(request, ":")
		if len(parts) >= 3 {
			targetAddr := parts[1]
			targetPort := parts[2]
			
			// Create connection to target
			t.logger.Info("Connecting to target %s:%s as requested by server", targetAddr, targetPort)
			targetAddrFull := net.JoinHostPort(targetAddr, targetPort)
			targetConn, err := net.DialTimeout("tcp", targetAddrFull, 10*time.Second)
			if err != nil {
				t.logger.Error("Failed to connect to target %s: %v", targetAddrFull, err)
				controlConn.Write([]byte(fmt.Sprintf("ERROR:%v", err)))
				return
			}
			defer targetConn.Close()
			
			// Konfirmasi koneksi berhasil
			controlConn.Write([]byte("OK"))
			
			// Forward data antara koneksi kontrol dan target
			t.bidirectionalCopy(controlConn, targetConn)
		} else {
			t.logger.Error("Invalid CONNECT request format: %s", request)
			controlConn.Write([]byte("ERROR:Invalid request format"))
		}
	} else {
		t.logger.Error("Unknown control request: %s", request)
		controlConn.Write([]byte("ERROR:Unknown request"))
	}
}

// handleLocalConnection handles a connection from the local port
func (t *DirectTunnel) handleLocalConnection(localConn net.Conn) {
	defer localConn.Close()

	// Create a connection to the target (local SSH server)
	// Connecting to target %s
	targetConn, err := net.DialTimeout("tcp", t.targetAddr, 10*time.Second)
	if err != nil {
		t.logger.Error("Failed to connect to target %s: %v", t.targetAddr, err)
		return
	}
	defer targetConn.Close()

	// Optimize TCP connection if possible
	if tcpConn, ok := targetConn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
		tcpConn.SetReadBuffer(256 * 1024)
		tcpConn.SetWriteBuffer(256 * 1024)
	}

	// Connection established to target

	// Forward data between local and target connections
	done := make(chan struct{}, 2)

	// Copy data from local connection to target connection
	go func() {
		_, err := io.Copy(targetConn, localConn)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
			t.logger.Error("Error copying data from local to target: %v", err)
		}

		// Close the write side of the target connection to signal EOF
		if tcpConn, ok := targetConn.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
		done <- struct{}{}
	}()

	// Copy data from target connection to local connection
	go func() {
		_, err := io.Copy(localConn, targetConn)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
			t.logger.Error("Error copying data from target to local: %v", err)
		}

		// Close the write side of the local connection to signal EOF
		if tcpConn, ok := localConn.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
		done <- struct{}{}
	}()

	// Wait for both copy operations to complete
	<-done
	<-done
	// Connection closed
}

// handleServerInitiatedConnection menangani koneksi kontrol yang diinisiasi oleh server relay
// handleServerInitiatedConnection dihapus karena tidak digunakan lagi
// Semua komunikasi sekarang menggunakan koneksi keluar yang diinisiasi oleh client

// TODO: Fungsi ini akan digunakan untuk implementasi relay mode di masa depan
// HandleRelayConnection menangani koneksi balik dari server relay
func (t *DirectTunnel) HandleRelayConnection(relayConn net.Conn) {
	defer relayConn.Close()
	
	// Baca alamat target dari server relay dengan buffer yang lebih besar
	buffer := make([]byte, 4096) // Meningkatkan buffer dari 1024 menjadi 4096 bytes
	relayConn.SetReadDeadline(time.Now().Add(15*time.Second)) // Meningkatkan timeout dari 10 detik menjadi 15 detik
	n, err := relayConn.Read(buffer)
	relayConn.SetReadDeadline(time.Time{}) // Reset deadline
	if err != nil {
		t.logger.Error("Failed to read target address from relay: %v", err)
		return
	}
	
	// Log data awal dalam format hex untuk debugging
	
	targetAddr := string(buffer[:n])
	
	// Create connection to target (local SSH server)
	targetConn, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
	if err != nil {
		t.logger.Error("Failed to connect to target %s: %v", targetAddr, err)
		return
	}
	defer targetConn.Close()
	
	// Send confirmation to relay server - protocol simplification
	_, err = relayConn.Write([]byte("READY"))
	if err != nil {
		t.logger.Error("Failed to send confirmation to relay: %v", err)
		return
	}
	
	// Tambahkan delay kecil setelah mengirim konfirmasi untuk memastikan server menerima konfirmasi
	time.Sleep(300 * time.Millisecond) // Meningkatkan delay dari 100ms menjadi 300ms
	
	// Optimize TCP connections
	if tcpConn, ok := targetConn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
		tcpConn.SetReadBuffer(256 * 1024)
		tcpConn.SetWriteBuffer(256 * 1024)
	}
	
	if tcpConn, ok := relayConn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
		tcpConn.SetReadBuffer(256 * 1024)
		tcpConn.SetWriteBuffer(256 * 1024)
	}
	
	// Relay connection established

	// Forward data between relay and target connections
	done := make(chan struct{}, 2)
	
	// Copy data from relay connection to target connection
	go func() {
		_, err := io.Copy(targetConn, relayConn)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
			t.logger.Error("Error copying data from relay to target: %v", err)
		}
		
		// Close the write side of the target connection to signal EOF
		if tcpConn, ok := targetConn.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
		done <- struct{}{}
	}()
	
	// Copy data from target connection to relay connection
	go func() {
		_, err := io.Copy(relayConn, targetConn)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
			t.logger.Error("Error copying data from target to relay: %v", err)
		}
		
		// Close the write side of the relay connection to signal EOF
		if tcpConn, ok := relayConn.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
		done <- struct{}{}
	}()
	
	// Wait for both copy operations to complete
	<-done
	<-done
	// Relay connection closed
}

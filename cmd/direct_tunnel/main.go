package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/haxorport/haxorport-go-client/internal/infrastructure/transport"
	"github.com/spf13/viper"
)

func main() {
	// Check if this is an HTTP command with WebSocket mode
	// If yes, exit with an appropriate message
	isHTTPMode := false
	for _, arg := range os.Args {
		if arg == "http" {
			isHTTPMode = true
			break
		}
	}
	
	if isHTTPMode {
		// Try to read configuration to check connection mode
		homeDir, err := os.UserHomeDir()
		if err == nil {
			configPath := homeDir + "/.haxorport/config.yaml"
			if _, err := os.Stat(configPath); err == nil {
				// Read configuration
				v := viper.New()
				v.SetConfigFile(configPath)
				if err := v.ReadInConfig(); err == nil {
					connectionMode := v.GetString("connection_mode")
					if connectionMode == "websocket" {
						fmt.Println("HTTP tunnel with WebSocket mode detected. Direct tunnel is not required.")
						fmt.Println("Use command 'go run main.go http -p PORT' or './haxorport-client http -p PORT' instead.")
						os.Exit(0)
					}
				}
			}
		}
	}
	
	// Parse command line arguments
	localPort := flag.Int("local-port", 2222, "Local port to listen on")
	serverAddr := flag.String("server", "", "Server address (host:port)")
	remoteHost := flag.String("remote-host", "localhost", "Remote host to connect to")
	authToken := flag.String("auth-token", "", "Authentication token for server")
	authEnabled := flag.Bool("auth", true, "Enable authentication")
	remotePort := flag.Int("remote-port", 22, "Remote port to connect to (SSH port)")
	flag.Parse()

	// Create target address
	targetAddr := fmt.Sprintf("%s:%d", *remoteHost, *remotePort)

	// Create logger
	logger := &transport.DefaultLogger{}

	// Determine configuration file based on tunnel mode
	configPath := "config.yaml"
	// isHTTPMode already declared at the beginning of the function
	
	// Check command arguments to determine configuration file
	for _, arg := range os.Args {
		if arg == "tcp" {
			configPath = "config_tcp.yaml"
			break
		} else if arg == "http" {
			isHTTPMode = true
			// Keep using config.yaml for HTTP mode
		}
	}
	
	// Log tunnel mode being used (only in debug mode)
	if os.Getenv("LOG_LEVEL") == "debug" {
		if isHTTPMode {
			log.Printf("Tunnel mode: HTTP")
		} else {
			log.Printf("Tunnel mode: TCP")
		}
	}

	// Get server address from config if not provided
	serverAddress := *serverAddr
	if serverAddress == "" {
		// Try to load from config
		// Look for configuration file
		homeDir, err := os.UserHomeDir()
		if err == nil {
			// Try in .haxorport directory in home
			homeConfigPath := homeDir + "/.haxorport/" + configPath
			if _, err := os.Stat(homeConfigPath); err == nil {
				configPath = homeConfigPath
				if os.Getenv("LOG_LEVEL") == "debug" {
					log.Printf("Using configuration file: %s", configPath)
				}
			}
		}

		// Read configuration
		v := viper.New()
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			log.Printf("Error reading configuration file: %v", err)
			// Use default value if configuration file not found
			if *serverAddr == "" {
				*serverAddr = "localhost:7000"
			}
		} else {
			// Get server address and connection mode from configuration
			connectionMode := v.GetString("connection_mode")
			if os.Getenv("LOG_LEVEL") == "debug" {
				log.Printf("Connection mode from configuration: %s", connectionMode)
			}

			// For HTTP tunnel with WebSocket mode, don't run direct tunnel
			if isHTTPMode && connectionMode == "websocket" {
				log.Printf("HTTP tunnel with WebSocket mode detected. Direct tunnel is not required.")
				log.Printf("Use command 'go run main.go http -p PORT' or './haxorport-client http -p PORT' instead.")
				os.Exit(0)
			}

			// Get server address from configuration if not specified via command line
			if *serverAddr == "" {
				*serverAddr = v.GetString("server_address")
				controlPort := v.GetInt("control_port")
				if controlPort != 0 {
					*serverAddr = fmt.Sprintf("%s:%d", *serverAddr, controlPort)
				}
				if os.Getenv("LOG_LEVEL") == "debug" {
					log.Printf("Using server address from configuration: %s", *serverAddr)
				}
			}
		}
	}

	// Log connection information
	log.Printf("Creating direct tunnel with configuration:")
	log.Printf("- Local address: 0.0.0.0:%d", *localPort)
	log.Printf("- Server address: %s", *serverAddr)
	log.Printf("- Target address: %s:%d", *remoteHost, *remotePort)

	// Create and start direct tunnel
	directTunnel := transport.NewDirectTunnel(
		"0.0.0.0",
		*serverAddr,
		*remoteHost+":"+strconv.Itoa(*remotePort),
		*localPort,
		0, // Remote port will be determined by the server
		0, // Control port is no longer used
		logger,
		*authEnabled, // Use auth enabled flag from command line
		*authToken, // Use auth token from command line
	)

	// In debug mode, display IP information used for connection
	if os.Getenv("LOG_LEVEL") == "debug" {
		ip := directTunnel.GetOutboundIP()
		log.Printf("Using IP for outbound connection: %s", ip)

		// Example of using HandleRelayConnection if needed in the future
		log.Printf("HandleRelayConnection available for relay mode implementation")
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start client in a goroutine
	errChan := make(chan error)
	go func() {
		errChan <- directTunnel.Start()
	}()

	// Print usage information
	log.Printf("[INFO] Direct tunnel successfully started:")
	log.Printf("[INFO] Local port %d -> %s through %s", *localPort, targetAddr, *serverAddr)
	log.Printf("[INFO] Press Ctrl+C to stop the tunnel")

	// Wait for signal or error
	select {
	case err := <-errChan:
		if err != nil {
			log.Fatalf("Tunnel error: %v", err)
		}
	case sig := <-sigChan:
		log.Printf("Received signal: %v", sig)
		directTunnel.Stop() // Stop() does not return an error
		log.Println("Tunnel stopped")
	}
}

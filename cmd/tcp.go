package cmd

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/haxorport/haxorport-go-client/internal/domain/model"
	"github.com/haxorport/haxorport-go-client/internal/domain/port"
	"github.com/haxorport/haxorport-go-client/internal/domain/service"
	"github.com/spf13/cobra"
)

var (
	tcpLocalPort  int
	tcpRemotePort int
	tcpLocalAddr  string
)

var tcpCmd = &cobra.Command{
	Use:   "tcp",
	Short: "Create a TCP tunnel",
	Long: `Create a TCP tunnel to expose local TCP services to the internet.
Examples:
  haxorport tcp -p 22
  haxorport tcp --port 22 --remote-port 2222
  haxorport tcp --port 5432`,
	Run: func(cmd *cobra.Command, args []string) {
		if tcpLocalPort <= 0 {
			fmt.Println("Error: Local port must be greater than 0")
			os.Exit(1)
		}
		
		// Always force DirectTCP mode for TCP tunnels
		if Container.Config.ConnectionMode != model.ConnectionModeDirectTCP {
			log.Printf("Warning: Changing connection mode from %s to direct_tcp for TCP tunnel", Container.Config.ConnectionMode)
			Container.Config.ConnectionMode = model.ConnectionModeDirectTCP
		}

		// Validate auth token for all connection modes
		// For DirectTCP mode, always validate token regardless of auth_enabled setting
		if Container.Config.AuthEnabled || Container.Config.ConnectionMode == model.ConnectionModeDirectTCP {
			// Check if token is empty
			if Container.Config.AuthToken == "" {
				// Determine the correct TCP configuration file path
				configTcpPath := ""
				homeDir, err := os.UserHomeDir()
				if err == nil {
					configTcpPath = homeDir + "/.haxorport/config_tcp.yaml"
				} else {
					configTcpPath = "config_tcp.yaml"
				}
				
				fmt.Println("==================================================")
				fmt.Println("❌ ERROR: AUTHENTICATION FAILED")
				fmt.Println("==================================================")
				fmt.Println("🔑 Authentication token is required to create a TCP tunnel")
				fmt.Println("🔧 Please add your token in the configuration file:")
				// Display the TCP configuration file path being used
				fmt.Printf("   %s\n", configTcpPath)
				fmt.Println("📝 Example configuration:")
				fmt.Println("   auth_enabled: true")
				fmt.Println("   auth_token: \"hxp_your_token_here\"")
				fmt.Println("==================================================")
				fmt.Println("ℹ️ To obtain a token, please login to the Haxorport dashboard")
				fmt.Println("   or contact your system administrator")
				fmt.Println("   https://haxorport.online/dashboard")
				fmt.Println("==================================================")
				os.Exit(1)
			}
			// Create a temporary client for validation if needed
			var client port.Client
			var userData *model.AuthData
			
			if Container.Config.ConnectionMode == model.ConnectionModeWebSocket {
				// For WebSocket mode, use the existing client
				if Container.Client == nil {
					fmt.Println("Error: Client not available for WebSocket connection mode")
					os.Exit(1)
				}
				
				if !Container.Client.IsConnected() {
					if err := Container.Client.Connect(); err != nil {
						fmt.Printf("Error: Failed to connect to server: %v\n", err)
						os.Exit(1)
					}
				}
				
				client = Container.Client
				userData = client.GetUserData()
			} else {
				// For DirectTCP mode, create a temporary client just for token validation
				validationURL := Container.Config.AuthValidationURL
				if validationURL == "" {
					validationURL = fmt.Sprintf("https://%s/AuthToken/validate", Container.Config.ServerAddress)
				}
				
				// Create authentication service
				authService := service.NewAuthService(validationURL)
				
				// Validate token
				response, err := authService.ValidateTokenWithResponse(Container.Config.AuthToken)
				if err != nil {
					fmt.Printf("Error: Failed to validate token: %v\n", err)
					os.Exit(1)
				}
				
				// Check response status
				if response.Status != "success" || response.Code != 200 {
					fmt.Printf("Error: Invalid token: %s\n", response.Message)
					os.Exit(1)
				}
				
				userData = &response.Data
			}
			
			// Check if token is valid
			if userData == nil {
				fmt.Println("Error: Invalid or unvalidated authentication token")
				os.Exit(1)
			}
			
			// Check tunnel limits
			if userData.Subscription.Limits.Tunnels.Reached || 
			   userData.Subscription.Limits.Tunnels.Used >= userData.Subscription.Limits.Tunnels.Limit {
				fmt.Printf("Error: Tunnel limit reached (%d/%d). Please upgrade your subscription.\n", 
					userData.Subscription.Limits.Tunnels.Used, 
					userData.Subscription.Limits.Tunnels.Limit)
				os.Exit(1)
			}
			
			// Log user information
			if os.Getenv("LOG_LEVEL") == "debug" {
				log.Printf("Token validated for user: %s (%s)", userData.Fullname, userData.Email)
				log.Printf("Subscription: %s, Tunnel Limit: %d/%d", 
					userData.Subscription.Name, 
					userData.Subscription.Limits.Tunnels.Used, 
					userData.Subscription.Limits.Tunnels.Limit)
			}
		}
		
		// For WebSocket connection mode, ensure the client is running
		if Container.Config.ConnectionMode == model.ConnectionModeWebSocket && Container.Client != nil {
			Container.Client.RunWithReconnect()
		}
		// For DirectTCP mode, we don't need to maintain a persistent client connection

		localHost := "127.0.0.1"
		localPort := tcpLocalPort

		host, _, err := net.SplitHostPort(tcpLocalAddr)
		if err == nil {
			if host != "" {
				localHost = host
			}
		} else if !strings.Contains(tcpLocalAddr, ":") {
			localHost = tcpLocalAddr
		} else {
			fmt.Printf("Error: Invalid local address format: %s\n", tcpLocalAddr)
			os.Exit(1)
		}

		// If remote port is not specified, use 0 to request an automatic port from the server
		remotePort := tcpRemotePort
		if remotePort == 0 {
			// If direct TCP connection mode, we can use a random port
			if Container.Config.ConnectionMode == model.ConnectionModeDirectTCP {
				// Use port 0 to request an automatic port
				remotePort = 0
			} else {
				// For WebSocket connection mode, we still need a specific port
				// Use the same port as the local port as default
				remotePort = localPort
			}
		}

		tunnelConfig := model.TunnelConfig{
			Type:       model.TunnelTypeTCP,
			LocalAddr:  localHost,
			LocalPort:  localPort,
			RemotePort: remotePort,
		}

		tunnel, err := Container.TunnelService.CreateTCPTunnel(tunnelConfig)
		if err != nil {
			fmt.Printf("Error: Failed to create tunnel: %v\n", err)
			os.Exit(1)
		}

		// Debug log
		if os.Getenv("LOG_LEVEL") == "debug" {
			log.Printf("Creating TCP tunnel for %s:%d with remote port %d", tunnelConfig.LocalAddr, tunnelConfig.LocalPort, tunnelConfig.RemotePort)
		}

		// Function to display tunnel information
		printTunnelInfo := func(remotePort int) {
			// Clear screen and move cursor to top
			fmt.Print("\033[H\033[2J")
			
			fmt.Println("=================================================")
			fmt.Println("✅ TCP TUNNEL CREATED SUCCESSFULLY!")
			fmt.Println("=================================================")
			fmt.Printf("🔌 Status    : Connected\n")
			fmt.Printf("🖥️ Local     : %s:%d\n", tunnelConfig.LocalAddr, tunnelConfig.LocalPort)
			fmt.Printf("🌐 Remote    : %s:%d\n", Container.Config.ServerAddress, remotePort)
			fmt.Printf("🔄 Type      : TCP\n")
			fmt.Printf("🔑 SSH Access: ssh -p %d username@%s\n", remotePort, Container.Config.ServerAddress)
			fmt.Printf("🔌 Connection Mode: %s\n", Container.Config.ConnectionMode)
			fmt.Printf("📝 Log File: %s\n", Container.Config.LogFile)
			fmt.Println("=================================================")
			fmt.Println("📋 Press Ctrl+C to close the tunnel")
			fmt.Println("=================================================")
			
			// Debug log for alternative port
			if os.Getenv("LOG_LEVEL") == "debug" {
				log.Printf("TCP tunnel active with remote port: %d", remotePort)
			}
		}
		
		// Display tunnel information with simple format
		printTunnelInfo(tunnel.RemotePort)
		
		// Get tunnel directly from internal repository
		// We need to access the internal repository to get the DirectTunnel instance
		if repo, ok := Container.TunnelRepository.(interface{
			GetDirectTunnel(tunnelID string) interface{
				SetPortChangeCallback(func(int))
			}
		}); ok {
			if directTunnel := repo.GetDirectTunnel(tunnel.ID); directTunnel != nil {
				directTunnel.SetPortChangeCallback(func(newPort int) {
					// Update tunnel model
					tunnel.RemotePort = newPort
					tunnel.Config.RemotePort = newPort
					
					// Display updated tunnel information
					printTunnelInfo(newPort)
				})
			}
		}
		
		// Debug log for alternative port
		if os.Getenv("LOG_LEVEL") == "debug" {
			log.Printf("TCP tunnel active with remote port: %d", tunnel.RemotePort)
		}

		// Add log to ensure tunnel remains active
		log.Printf("Tunnel active and waiting for connections. Press Ctrl+C to exit.")

		// Wait for interrupt signal
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		if err := Container.TunnelService.CloseTunnel(tunnel.ID); err != nil {
			fmt.Printf("\n\033[1;31m⚠️ Error: Failed to close tunnel: %v\033[0m\n", err)
		} else {
			fmt.Print("\n\033[1;32m✓ Tunnel closed successfully!\033[0m\n")
		}
	},
}

func init() {
	RootCmd.AddCommand(tcpCmd)

	tcpCmd.Flags().IntVarP(&tcpLocalPort, "port", "p", 0, "Local port to tunnel")
	tcpCmd.Flags().IntVarP(&tcpRemotePort, "remote-port", "r", 0, "Requested remote port (optional, will be automatically selected if not specified)")
	tcpCmd.Flags().StringVarP(&tcpLocalAddr, "local-addr", "l", "127.0.0.1", "Local address to forward to (default: 127.0.0.1)")

	tcpCmd.MarkFlagRequired("port")
}

package cmd

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/haxorport/haxorport-go-client/internal/domain/model"
	"github.com/spf13/cobra"
)

var (
	// HTTP command flags
	httpLocalPort int
	httpSubdomain string
	httpAuthType  string
	httpUsername  string
	httpPassword  string
	httpHeader    string
	httpValue     string
)

// httpCmd is the command to create an HTTP tunnel
var httpCmd = &cobra.Command{
	Use:   "http [target_url]",
	Short: "Create an HTTP tunnel",
	Long: `Create an HTTP tunnel to expose local HTTP services to the internet.
Examples:
  haxorport http -p 2712
  haxorport http --port 8080 --subdomain myapp
  haxorport http --port 3000 --auth basic --username user --password pass`,
	Run: func(cmd *cobra.Command, args []string) {
		// Check if URL argument is provided
		if len(args) > 0 {
			// Parse URL from argument
			targetURL := args[0]

			// Extract port and host from URL
			u, err := url.Parse(targetURL)
			if err != nil {
				fmt.Printf("Error: Invalid URL: %v\n", err)
				os.Exit(1)
			}

			// Extract port from URL
			port := u.Port()
			if port == "" {
				// Default port based on scheme
				if u.Scheme == "https" {
					port = "443"
				} else {
					port = "80"
				}
			}

			// Convert port to integer
			portInt, err := strconv.Atoi(port)
			if err != nil {
				fmt.Printf("Error: Invalid port: %v\n", err)
				os.Exit(1)
			}

			// Set local port
			httpLocalPort = portInt

			// Generate automatic subdomain if not specified
			if httpSubdomain == "" {
				// Use timestamp to create unique subdomain without "haxor-" prefix
				timestamp := time.Now().UnixNano() / int64(time.Millisecond)
				httpSubdomain = fmt.Sprintf("%x", timestamp%0xFFFFFF)
			}
		}

		// Validate parameters
		if httpLocalPort <= 0 {
			fmt.Println("Error: Local port must be greater than 0")
			os.Exit(1)
		}

		// Create auth if needed
		var auth *model.TunnelAuth
		if httpAuthType != "" {
			auth = &model.TunnelAuth{}
			switch httpAuthType {
			case "basic":
				auth.Type = model.AuthTypeBasic
				auth.Username = httpUsername
				auth.Password = httpPassword
				if auth.Username == "" || auth.Password == "" {
					fmt.Println("Error: Username and password are required for basic auth")
					os.Exit(1)
				}
			case "header":
				auth.Type = model.AuthTypeHeader
				auth.HeaderName = httpHeader
				auth.HeaderValue = httpValue
				if auth.HeaderName == "" || auth.HeaderValue == "" {
					fmt.Println("Error: Header name and value are required for header auth")
					os.Exit(1)
				}
			default:
				fmt.Printf("Error: Invalid auth type: %s\n", httpAuthType)
				os.Exit(1)
			}
		}

		// Check token configuration first
		if Container.Config.AuthEnabled {
			if Container.Config.AuthToken == "" {
				fmt.Println("\n===================================================")
				fmt.Println("⚠️ ERROR: Auth token not found in configuration")
				fmt.Println("===================================================")
				fmt.Println("You need to add an authentication token to your configuration file:")
				fmt.Printf("  %s\n\n", Container.Config.GetConfigFilePath())
				fmt.Println("How to add a token:")
				fmt.Println("1. Edit the configuration file with a text editor")
				fmt.Println("2. Find the line 'auth_token: \"\"' and replace it with your token")
				fmt.Println("3. Save the file and run this command again")
				fmt.Println("\nYou can get a token from the Haxorport dashboard:")
				fmt.Println("  https://haxorport.online/dashboard")
				fmt.Println("===================================================")
				os.Exit(1)
			}
		}

		// Ensure client is available and connected (only for WebSocket mode)
		if Container.Config.ConnectionMode == model.ConnectionModeWebSocket {
			if Container.Client == nil {
				fmt.Println("\n===================================================")
				fmt.Printf("⚠️ ERROR: Client not available for WebSocket connection mode\n")
				fmt.Println("===================================================")
				os.Exit(1)
			}
			
			if !Container.Client.IsConnected() {
				if err := Container.Client.Connect(); err != nil {
				fmt.Println("\n===================================================")
				fmt.Printf("⚠️ ERROR: Failed to connect to server\n")
				fmt.Println("===================================================")
				fmt.Printf("Error details: %v\n", err)
				
				// Display suggestions based on error type
				if strings.Contains(err.Error(), "websocket: bad handshake") {
					fmt.Println("\nSuggestions:")
					fmt.Println("- Check your internet connection")
					fmt.Println("- Make sure the Haxorport server is running")
					fmt.Println("- Check your TLS configuration")
					fmt.Println("- Try running './setup.sh' to update your configuration")
				}
				fmt.Println("====================================================")
				os.Exit(1)
			}
		}
		}
		
		// Check token validation if auth is enabled
		if Container.Config.AuthEnabled && Container.Config.ConnectionMode == model.ConnectionModeWebSocket && Container.Client != nil {
			// Check if user data is available (means token has been validated)
			userData := Container.Client.GetUserData()
			if userData == nil {
				fmt.Println("\n===================================================")
				fmt.Println("⚠️ ERROR: Invalid authentication token")
				fmt.Println("===================================================")
				fmt.Println("The token you provided is invalid or could not be validated.")
				fmt.Println("\nSuggestions:")
				fmt.Println("1. Check if the token is correct in your configuration file:")
				fmt.Printf("   %s\n", Container.Config.GetConfigFilePath())
				fmt.Println("2. Make sure you are using a valid token from the Haxorport dashboard")
				fmt.Println("   https://haxorport.online/dashboard")
				fmt.Println("===================================================")
				os.Exit(1)
			}
			
			// Check tunnel limit
			if Container.Client != nil {
				reached, used, limit := Container.Client.CheckTunnelLimit()
				if reached {
					fmt.Println("\n===================================================")
					fmt.Printf("⚠️ ERROR: Tunnel limit reached (%d/%d)\n", used, limit)
					fmt.Println("===================================================")
					fmt.Println("You have reached the tunnel limit for your subscription.")
					fmt.Println("\nSuggestions:")
					fmt.Println("- Close some unused tunnels")
					fmt.Println("- Upgrade your subscription to get more tunnels")
					fmt.Println("  https://haxorport.online/pricing")
					fmt.Println("===================================================")
					os.Exit(1)
				}
			}
		}

		// Ensure connection mode is WebSocket for HTTP tunnel
		if Container.Config.ConnectionMode != model.ConnectionModeWebSocket {
			fmt.Println("\n===================================================")
			fmt.Println("⚠️ ERROR: Invalid connection mode for HTTP tunnel")
			fmt.Println("===================================================")
			fmt.Println("HTTP tunnel requires WebSocket connection mode.")
			fmt.Printf("Current connection mode: %s\n", Container.Config.ConnectionMode)
			fmt.Println("\nSuggestions:")
			fmt.Println("1. Edit configuration file:")
			fmt.Printf("   %s\n", Container.Config.GetConfigFilePath())
			fmt.Println("2. Add or modify the following parameter:")
			fmt.Println("   connection_mode: \"websocket\"")
			fmt.Println("===================================================")
			os.Exit(1)
		}

		// Log connection information
		Container.Logger.Info("Active connection mode: WebSocket to %s", Container.Config.ServerAddress)

		// Run client with automatic reconnection
		if Container.Client != nil {
			Container.Client.RunWithReconnect()
		} else {
			Container.Logger.Error("Client was not properly initialized")
			os.Exit(1)
		}

		// Create tunnel
		tunnel, err := Container.TunnelService.CreateHTTPTunnel(httpLocalPort, httpSubdomain, auth)
		if err != nil {
			fmt.Printf("Error: Failed to create tunnel: %v\n", err)
			os.Exit(1)
		}

		// Write to log file for debugging
		if os.Getenv("LOG_LEVEL") == "debug" {
			logFile, err := os.OpenFile("output.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err == nil {
				defer logFile.Close()
				fmt.Fprintf(logFile, "Tunnel created successfully: %s\n", tunnel.URL)
			}
		}

		// Clear screen and move cursor to top like in TCP command
		fmt.Print("\033[H\033[2J")
		
		// Use fmt.Fprintf with os.Stderr to ensure output is displayed
		fmt.Fprintf(os.Stderr, "=================================================\n")
		fmt.Fprintf(os.Stderr, "✅ HTTP TUNNEL CREATED SUCCESSFULLY!\n")
		fmt.Fprintf(os.Stderr, "=================================================\n")
		fmt.Fprintf(os.Stderr, "🌐 Tunnel URL: %s\n", tunnel.URL)
		fmt.Fprintf(os.Stderr, "🔌 Local Port: %d\n", tunnel.Config.LocalPort)
		fmt.Fprintf(os.Stderr, "🆔 Tunnel ID: %s\n", tunnel.ID)
		fmt.Fprintf(os.Stderr, "🔌 Connection Mode: %s\n", Container.Config.ConnectionMode)
		fmt.Fprintf(os.Stderr, "🖥️ Server: %s:%d\n", Container.Config.ServerAddress, Container.Config.ControlPort)
		fmt.Fprintf(os.Stderr, "📝 Log File: %s\n", Container.Config.LogFile)

		// Display additional information
		if auth != nil {
			fmt.Fprintf(os.Stderr, "🔒 Authentication: %s\n", auth.Type)
		}

		// Add instructions for accessing the URL
		fmt.Fprintf(os.Stderr, "\n📌 To access your service, open the URL above in your browser\n")
		fmt.Fprintf(os.Stderr, "   or use curl:\n")
		fmt.Fprintf(os.Stderr, "   curl %s\n", tunnel.URL)

		fmt.Fprintf(os.Stderr, "=================================================\n")
		fmt.Fprintf(os.Stderr, "📋 Press Ctrl+C to stop the tunnel\n")
		fmt.Fprintf(os.Stderr, "=================================================\n")

		// Flush stderr to ensure output is displayed
		os.Stderr.Sync()

		// Use log.Printf to display output
		log.Printf("Tunnel created successfully: %s", tunnel.URL)

		// Wait for exit signal
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		// Close tunnel
		if err := Container.TunnelService.CloseTunnel(tunnel.ID); err != nil {
			fmt.Printf("Error: Failed to close tunnel: %v\n", err)
		} else {
			fmt.Println("Tunnel closed")
		}
	},
}

func init() {
	RootCmd.AddCommand(httpCmd)

	// Add flags
	httpCmd.Flags().IntVarP(&httpLocalPort, "port", "p", 0, "Local port to tunnel")
	httpCmd.Flags().StringVarP(&httpSubdomain, "subdomain", "s", "", "Requested subdomain (optional)")
	httpCmd.Flags().StringVarP(&httpAuthType, "auth", "a", "", "Authentication type (basic, header)")
	httpCmd.Flags().StringVarP(&httpUsername, "username", "u", "", "Username for basic authentication")
	httpCmd.Flags().StringVarP(&httpPassword, "password", "w", "", "Password for basic authentication")
	httpCmd.Flags().StringVar(&httpHeader, "header", "", "Header name for header authentication")
	httpCmd.Flags().StringVar(&httpValue, "value", "", "Header value for header authentication")

	// Port is only required if URL is not provided
	// httpCmd.MarkFlagRequired("port")
}

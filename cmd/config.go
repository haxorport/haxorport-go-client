package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/haxorport/haxorport-go-client/internal/domain/model"
	"github.com/spf13/cobra"
)

var (
	// Config command flags
	configServerAddress string
	configControlPort   int
	configAuthToken     string
	configLogLevel      string
	configLogFile       string
)

// configCmd is the command to manage configuration
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Manage Haxorport Client configuration.`,
}

// configShowCmd is the command to display configuration
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show configuration",
	Long:  `Display Haxorport Client configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Display configuration
		fmt.Println("Haxorport Client Configuration:")
		fmt.Printf("Server Address: %s\n", Container.Config.ServerAddress)
		fmt.Printf("Control Port: %d\n", Container.Config.ControlPort)
		fmt.Printf("Auth Token: %s\n", maskString(Container.Config.AuthToken))
		fmt.Printf("Log Level: %s\n", Container.Config.LogLevel)
		fmt.Printf("Log File: %s\n", Container.Config.LogFile)

		// Display tunnels
		if len(Container.Config.Tunnels) > 0 {
			fmt.Println("\nTunnel:")
			for i, tunnel := range Container.Config.Tunnels {
				fmt.Printf("  %d. %s (%s)\n", i+1, tunnel.Name, tunnel.Type)
				fmt.Printf("     Local Port: %d\n", tunnel.LocalPort)
				if tunnel.Type == model.TunnelTypeHTTP {
					fmt.Printf("     Subdomain: %s\n", tunnel.Subdomain)
				} else if tunnel.Type == model.TunnelTypeTCP {
					fmt.Printf("     Remote Port: %d\n", tunnel.RemotePort)
				}
				if tunnel.Auth != nil {
					fmt.Printf("     Auth: %s\n", tunnel.Auth.Type)
				}
			}
		}
	},
}

// configSetCmd is the command to set configuration
var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set configuration",
	Long: `Set Haxorport Client configuration.
Examples:
  haxorport config set server_address example.com
  haxorport config set control_port 8080
  haxorport config set auth_token my-token
  haxorport config set log_level debug
  haxorport config set log_file /path/to/log.txt`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		value := args[1]

		// Update configuration
		switch key {
		case "server_address":
			Container.ConfigService.SetServerAddress(Container.Config, value)
		case "control_port":
			port, err := strconv.Atoi(value)
			if err != nil {
				fmt.Printf("Error: Port must be a number: %v\n", err)
				os.Exit(1)
			}
			Container.ConfigService.SetControlPort(Container.Config, port)
		case "auth_token":
			Container.ConfigService.SetAuthToken(Container.Config, value)
		case "log_level":
			Container.ConfigService.SetLogLevel(Container.Config, value)
		case "log_file":
			Container.ConfigService.SetLogFile(Container.Config, value)
		default:
			fmt.Printf("Error: Invalid configuration key: %s\n", key)
			os.Exit(1)
		}

		// Save configuration
		if err := Container.ConfigService.SaveConfig(Container.Config, ConfigPath); err != nil {
			fmt.Printf("Error: Failed to save configuration: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Configuration %s successfully changed to %s\n", key, value)
	},
}

// configAddTunnelCmd is the command to add a tunnel to configuration
var configAddTunnelCmd = &cobra.Command{
	Use:   "add-tunnel",
	Short: "Add tunnel to configuration",
	Long: `Add tunnel to Haxorport Client configuration.
Examples:
  haxorport config add-tunnel --name web --type http --port 8080 --subdomain myapp
  haxorport config add-tunnel --name ssh --type tcp --port 22 --remote-port 2222`,
	Run: func(cmd *cobra.Command, args []string) {
		// Validate parameters
		if httpLocalPort <= 0 {
			fmt.Println("Error: Local port must be greater than 0")
			os.Exit(1)
		}

		// Create auth if required
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

		// Create tunnel configuration
		tunnelConfig := model.TunnelConfig{
			Name:      httpSubdomain,
			LocalPort: httpLocalPort,
		}

		// Set tunnel type
		tunnelType := cmd.Flag("type").Value.String()
		switch tunnelType {
		case "http":
			tunnelConfig.Type = model.TunnelTypeHTTP
			tunnelConfig.Subdomain = httpSubdomain
		case "tcp":
			tunnelConfig.Type = model.TunnelTypeTCP
			tunnelConfig.RemotePort = tcpRemotePort
		default:
			fmt.Printf("Error: Invalid tunnel type: %s\n", tunnelType)
			os.Exit(1)
		}

		// Set auth
		tunnelConfig.Auth = auth

		// Add tunnel to configuration
		Container.ConfigService.AddTunnel(Container.Config, tunnelConfig)

		// Save configuration
		if err := Container.ConfigService.SaveConfig(Container.Config, ConfigPath); err != nil {
			fmt.Printf("Error: Failed to save configuration: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Tunnel successfully added to configuration")
	},
}

// configRemoveTunnelCmd is the command to remove a tunnel from configuration
var configRemoveTunnelCmd = &cobra.Command{
	Use:   "remove-tunnel [name]",
	Short: "Remove tunnel from configuration",
	Long: `Remove tunnel from Haxorport Client configuration.
Examples:
  haxorport config remove-tunnel web`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		// Remove tunnel from configuration
		if !Container.ConfigService.RemoveTunnel(Container.Config, name) {
			fmt.Printf("Error: Tunnel with name %s not found\n", name)
			os.Exit(1)
		}

		// Save configuration
		if err := Container.ConfigService.SaveConfig(Container.Config, ConfigPath); err != nil {
			fmt.Printf("Error: Failed to save configuration: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Tunnel %s successfully removed from configuration\n", name)
	},
}

// maskString hides part of a string
func maskString(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}

func init() {
	RootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configAddTunnelCmd)
	configCmd.AddCommand(configRemoveTunnelCmd)

	// Add flags for add-tunnel
	configAddTunnelCmd.Flags().StringP("name", "n", "", "Tunnel name")
	configAddTunnelCmd.Flags().StringP("type", "t", "", "Tunnel type (http, tcp)")
	configAddTunnelCmd.Flags().IntVarP(&httpLocalPort, "port", "p", 0, "Local port to tunnel")
	configAddTunnelCmd.Flags().StringVarP(&httpSubdomain, "subdomain", "s", "", "Requested subdomain (for HTTP)")
	configAddTunnelCmd.Flags().IntVarP(&tcpRemotePort, "remote-port", "r", 0, "Requested remote port (for TCP)")
	configAddTunnelCmd.Flags().StringVarP(&httpAuthType, "auth", "a", "", "Authentication type (basic, header)")
	configAddTunnelCmd.Flags().StringVarP(&httpUsername, "username", "u", "", "Username for basic authentication")
	configAddTunnelCmd.Flags().StringVarP(&httpPassword, "password", "w", "", "Password for basic authentication")
	configAddTunnelCmd.Flags().StringVar(&httpHeader, "header", "", "Header name for header authentication")
	configAddTunnelCmd.Flags().StringVar(&httpValue, "value", "", "Header value for header authentication")

	// Mark required flags
	configAddTunnelCmd.MarkFlagRequired("type")
	configAddTunnelCmd.MarkFlagRequired("port")
}

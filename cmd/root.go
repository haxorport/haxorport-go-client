package cmd

import (
	"fmt"
	"os"

	"github.com/haxorport/haxorport-go-client/internal/di"
	"github.com/spf13/cobra"
)

var (
	// Container is the dependency injection container
	Container *di.Container

	// ConfigPath is the path to the configuration file
	ConfigPath string
	
	// AutoConfigPath is a flag to determine whether to use automatic config selection
	AutoConfigPath bool
	
	// CommandType stores the type of command being run (http or tcp)
	CommandType string
	
	// LogLevel is the logging level
	LogLevel string

	// RootCmd is the root command for CLI
	RootCmd = &cobra.Command{
		Use:   "haxorport",
		Short: "Haxorport Client - HTTP and TCP Tunneling",
		Long: `Haxorport Client is a tool for creating HTTP and TCP tunnels.
With Haxorport, you can expose local services to the internet.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Initialize container
			Container = di.NewContainer()
			
			// Set log level from flag if provided
			if LogLevel != "" {
				// LogLevel will be used inside Initialize
			}
			
			// Detect command type based on the executed subcommand
			CommandType = "http" // Default to http
			if cmd.Parent() != nil && cmd.Parent().Name() == "haxorport" {
				CommandType = cmd.Name()
			}
			
			// Select configuration file automatically if not explicitly specified
			if ConfigPath == "" && AutoConfigPath {
				if CommandType == "tcp" {
					// Use config_tcp.yaml for TCP command
					homeDir, err := os.UserHomeDir()
					if err == nil {
						ConfigPath = homeDir + "/.haxorport/config_tcp.yaml"
						// If file doesn't exist, try in application directory
						if _, err := os.Stat(ConfigPath); os.IsNotExist(err) {
							ConfigPath = "config_tcp.yaml"
						}
					} else {
						ConfigPath = "config_tcp.yaml"
					}
				} else {
					// Use default config.yaml for HTTP command
					homeDir, err := os.UserHomeDir()
					if err == nil {
						ConfigPath = homeDir + "/.haxorport/config.yaml"
						// If file doesn't exist, try in application directory
						if _, err := os.Stat(ConfigPath); os.IsNotExist(err) {
							ConfigPath = "config.yaml"
						}
					} else {
						ConfigPath = "config.yaml"
					}
				}
				if os.Getenv("LOG_LEVEL") == "debug" {
					fmt.Printf("Using configuration file: %s for %s mode\n", ConfigPath, CommandType)
				}
			}
			
			if err := Container.Initialize(ConfigPath); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			
			// Set log level after container initialization
			if LogLevel != "" {
				Container.Logger.SetLevel(LogLevel)
				if os.Getenv("LOG_LEVEL") == "debug" {
					fmt.Printf("Log level set to: %s\n", LogLevel)
				}
			}
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			// Close container
			if Container != nil {
				Container.Close()
			}
		},
	}
)

// Execute runs the root command
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Add global flags
	RootCmd.PersistentFlags().StringVarP(&ConfigPath, "config", "c", "", "Path to configuration file (default: auto-detect based on command)")
	RootCmd.PersistentFlags().StringVar(&LogLevel, "log-level", "warn", "Set logging level (debug, info, warn, error)")
	RootCmd.PersistentFlags().BoolVar(&AutoConfigPath, "auto-config", true, "Automatically select config file based on command type")
}

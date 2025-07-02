package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// authTokenCmd is the command to set the authentication token
var authTokenCmd = &cobra.Command{
	Use:   "auth-token [token]",
	Short: "Set authentication token",
	Long: `Set authentication token for server connection.
Example:
  haxorport auth-token your_auth_token`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Get token from arguments
		token := args[0]

		// Ensure token is not empty
		if token == "" {
			fmt.Println("Error: Token cannot be empty")
			os.Exit(1)
		}

		// Enable authentication
		Container.Config.AuthEnabled = true
		Container.Config.AuthToken = token

		// Save configuration
		err := Container.ConfigRepository.Save(Container.Config, ConfigPath)
		if err != nil {
			fmt.Printf("Error: Failed to save configuration: %v\n", err)
			os.Exit(1)
		}

		// Validate token
		if err := Container.Client.Connect(); err != nil {
			fmt.Println("\n===================================================")
			fmt.Println("❌ TOKEN VALIDATION FAILED!")
			fmt.Println("===================================================")
			fmt.Printf("📛 Error: %v\n", err)
			fmt.Println("\nSuggestions:")
			fmt.Println("1. Make sure the token you entered is correct")
			fmt.Println("2. Check your internet connection")
			fmt.Println("3. Get a new token from the Haxorport dashboard:")
			fmt.Println("   https://haxorport.online/dashboard")
			fmt.Println("===================================================")
			os.Exit(1)
		}

		// Get user data
		userData := Container.Client.GetUserData()
		if userData == nil {
			fmt.Println("Error: Failed to get user data")
			os.Exit(1)
		}

		// Display user information
		fmt.Println("\n=================================================")
		fmt.Println("✅ TOKEN HAS BEEN SUCCESSFULLY SET AND VALIDATED!")
		fmt.Println("=================================================")
		fmt.Printf("👤 User: %s (%s)\n", userData.Fullname, userData.Email)
		fmt.Printf("\n=== Account Information ===\n")
		fmt.Printf("🔑 Subscription: %s\n", userData.Subscription.Name)
		fmt.Printf("📊 Tunnel Limit: %d/%d\n", userData.Subscription.Limits.Tunnels.Used, userData.Subscription.Limits.Tunnels.Limit)
		
		// Display subscription features
		fmt.Printf("\n=== Subscription Information ===\n")
		if userData.Subscription.Features.CustomDomains {
			fmt.Println("  ✓ Custom Domains")
		}
		if userData.Subscription.Features.Analytics {
			fmt.Println("  ✓ Analytics")
		}
		if userData.Subscription.Features.PrioritySupport {
			fmt.Println("  ✓ Priority Support")
		}
		
		fmt.Printf("\nAuthentication token has been successfully saved and validated.\n")
		fmt.Println("=================================================")
	},
}

func init() {
	RootCmd.AddCommand(authTokenCmd)
}

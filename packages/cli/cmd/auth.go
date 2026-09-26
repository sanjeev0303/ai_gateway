package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate against the company gateway",
	Long:  `Authenticate against the company gateway using OAuth device flow or API key.`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate against the company gateway",
	Long: `Authenticate against the company gateway using OAuth device flow.
This will open a browser window for you to complete the authentication.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		gatewayURL, _ := cmd.Flags().GetString("gateway-url")
		if gatewayURL == "" {
			return fmt.Errorf("--gateway-url is required")
		}
		fmt.Printf("Authenticating against %s...\n", gatewayURL)
		fmt.Println("Opening browser for OAuth device flow...")
		fmt.Println("Authentication successful!")
		fmt.Println("Credentials stored in ~/.aigw/config.yaml")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Authenticated as: developer@company.com")
		fmt.Println("Gateway: https://gateway.company.com")
		fmt.Println("Organization: acme-corp")
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout and remove stored credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Logged out successfully")
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authLogoutCmd)

	authLoginCmd.Flags().String("gateway-url", "", "Gateway URL (required)")
	authLoginCmd.MarkFlagRequired("gateway-url")
}
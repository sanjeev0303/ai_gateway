package cmd

import (
	"context"
	"fmt"
	"time"

	"ai-governance-saas/packages/cli/internal/client"

	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate and check gateway credential status",
	Long:  `Authenticate against the Sentinel gateway and verify permitted access.`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Configure and verify a gateway credential",
	Long: `Configure and verify a Sentinel gateway API key.
The key will be tested against the gateway before saving to ~/.aigw/config.yaml.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		gatewayURL, _ := cmd.Flags().GetString("gateway-url")
		credential, _ := cmd.Flags().GetString("credential")
		projectID, _ := cmd.Flags().GetString("project-id")

		cfg := LoadConfig()
		if gatewayURL != "" {
			cfg.GatewayURL = gatewayURL
		}
		if credential != "" {
			cfg.Credential = credential
		}
		if projectID != "" {
			cfg.ProjectID = projectID
		}

		if cfg.Credential == "" {
			return fmt.Errorf("--credential is required (e.g. aigw_...)")
		}

		fmt.Printf("Verifying credential against %s...\n", cfg.GatewayURL)
		c := client.New(cfg.GatewayURL, cfg.Credential, cfg.ProjectID)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		models, err := c.ListModels(ctx)
		if err != nil {
			return fmt.Errorf("authentication verification failed: %w", err)
		}

		if err := SaveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		fmt.Println("✅ Authentication successful!")
		fmt.Printf("Gateway URL:      %s\n", cfg.GatewayURL)
		fmt.Printf("Active Key:       %s\n", maskSecret(cfg.Credential))
		if cfg.ProjectID != "" {
			fmt.Printf("Active Project:   %s\n", cfg.ProjectID)
		}
		fmt.Printf("Permitted Models: %d accessible\n", len(models))
		fmt.Printf("Configuration saved to %s\n", configFilePath())
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status against the gateway",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := LoadConfig()
		if cfg.Credential == "" {
			fmt.Println("Status: Not authenticated (no credential configured)")
			fmt.Println("Run 'aigw auth login --credential <key>' to configure.")
			return nil
		}

		fmt.Printf("Gateway URL: %s\n", cfg.GatewayURL)
		fmt.Printf("Credential:  %s\n", maskSecret(cfg.Credential))
		if cfg.ProjectID != "" {
			fmt.Printf("Project ID:  %s\n", cfg.ProjectID)
		} else {
			fmt.Println("Project ID:  (unscoped / org-wide)")
		}

		c := client.New(cfg.GatewayURL, cfg.Credential, cfg.ProjectID)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		models, err := c.ListModels(ctx)
		if err != nil {
			fmt.Printf("Gateway Connection: ❌ Failed (%v)\n", err)
			return nil
		}

		fmt.Println("Gateway Connection: ✅ Active & Authorized")
		fmt.Printf("Permitted Models:   %d available\n", len(models))
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout and remove stored credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := LoadConfig()
		cfg.Credential = ""
		_ = SaveConfig(cfg)
		fmt.Println("✅ Credentials removed from local configuration.")
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authLogoutCmd)

	authLoginCmd.Flags().String("gateway-url", "http://localhost:8084", "Gateway URL")
	authLoginCmd.Flags().String("credential", "", "Gateway API Key (required)")
	authLoginCmd.Flags().String("project-id", "", "Default project ID")
}

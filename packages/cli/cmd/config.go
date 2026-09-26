package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage local gateway configuration",
	Long:  `Manage local CLI configuration for gateway access.`,
}

var configInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install local gateway configuration",
	Long: `Install local gateway configuration with credentials.
This creates ~/.aigw/config.yaml with gateway URL and credentials.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		gatewayURL, _ := cmd.Flags().GetString("gateway-url")
		credential, _ := cmd.Flags().GetString("credential")
		name, _ := cmd.Flags().GetString("name")

		if gatewayURL == "" {
			return fmt.Errorf("--gateway-url is required")
		}
		if credential == "" {
			return fmt.Errorf("--credential is required")
		}

		configDir := filepath.Join(os.Getenv("HOME"), ".aigw")
		if err := os.MkdirAll(configDir, 0700); err != nil {
			return fmt.Errorf("failed to create config dir: %w", err)
		}

		configPath := filepath.Join(configDir, "config.yaml")
		config := map[string]interface{}{
			"gateway_url": gatewayURL,
			"credential":  credential,
			"name":        name,
		}

		data, err := yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		if err := os.WriteFile(configPath, data, 0600); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		fmt.Printf("✅ Configuration installed to %s\n", configPath)
		fmt.Printf("Gateway: %s\n", gatewayURL)
		if name != "" {
			fmt.Printf("Profile: %s\n", name)
		}
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := filepath.Join(os.Getenv("HOME"), ".aigw", "config.yaml")
		data, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("no configuration found: run 'aigw config install' first")
		}
		fmt.Println(string(data))
		return nil
	},
}

var configUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove local configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := filepath.Join(os.Getenv("HOME"), ".aigw", "config.yaml")
		if err := os.Remove(configPath); err != nil {
			return fmt.Errorf("failed to remove config: %w", err)
		}
		fmt.Println("✅ Configuration removed")
		return nil
	},
}

func init() {
	configCmd.AddCommand(configInstallCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configUninstallCmd)

	configInstallCmd.Flags().String("gateway-url", "", "Gateway URL (required)")
	configInstallCmd.Flags().String("credential", "", "Raw credential key (required)")
	configInstallCmd.Flags().String("name", "", "Profile name")
	configInstallCmd.MarkFlagRequired("gateway-url")
	configInstallCmd.MarkFlagRequired("credential")
}
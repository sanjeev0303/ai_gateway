package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type Config struct {
	GatewayURL string `yaml:"gateway_url"`
	ControlURL string `yaml:"control_url"`
	Credential string `yaml:"credential"`
	ProjectID  string `yaml:"project_id"`
	Name       string `yaml:"name"`
}

func configFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, ".aigw", "config.yaml")
}

func LoadConfig() Config {
	cfg := Config{
		GatewayURL: "http://localhost:8084",
		ControlURL: "http://localhost:8081",
	}

	data, err := os.ReadFile(configFilePath())
	if err == nil {
		_ = yaml.Unmarshal(data, &cfg)
	}

	// Environment variable overrides
	if u := os.Getenv("AIGW_GATEWAY_URL"); u != "" {
		cfg.GatewayURL = u
	} else if u := os.Getenv("SENTINEL_GATEWAY_URL"); u != "" {
		cfg.GatewayURL = u
	}

	if k := os.Getenv("AIGW_API_KEY"); k != "" {
		cfg.Credential = k
	} else if k := os.Getenv("SENTINEL_API_KEY"); k != "" {
		cfg.Credential = k
	}

	if p := os.Getenv("AIGW_PROJECT_ID"); p != "" {
		cfg.ProjectID = p
	}

	return cfg
}

func SaveConfig(cfg Config) error {
	path := configFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	return nil
}

func maskSecret(s string) string {
	if len(s) <= 8 {
		return "********"
	}
	return s[:6] + "..." + s[len(s)-4:]
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage local gateway configuration",
	Long:  `Manage local CLI configuration for gateway URL, API credentials, and active project.`,
}

var configInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install or update gateway configuration",
	Long:  `Install local gateway configuration with API credentials and default project.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		gatewayURL, _ := cmd.Flags().GetString("gateway-url")
		controlURL, _ := cmd.Flags().GetString("control-url")
		credential, _ := cmd.Flags().GetString("credential")
		projectID, _ := cmd.Flags().GetString("project-id")
		name, _ := cmd.Flags().GetString("name")

		cfg := LoadConfig()
		if gatewayURL != "" {
			cfg.GatewayURL = strings.TrimRight(gatewayURL, "/")
		}
		if controlURL != "" {
			cfg.ControlURL = strings.TrimRight(controlURL, "/")
		}
		if credential != "" {
			cfg.Credential = credential
		}
		if projectID != "" {
			cfg.ProjectID = projectID
		}
		if name != "" {
			cfg.Name = name
		}

		if cfg.GatewayURL == "" {
			return fmt.Errorf("--gateway-url is required")
		}
		if cfg.Credential == "" {
			return fmt.Errorf("--credential is required")
		}

		if err := SaveConfig(cfg); err != nil {
			return err
		}

		fmt.Printf("✅ Configuration saved to %s\n", configFilePath())
		fmt.Printf("Gateway:    %s\n", cfg.GatewayURL)
		fmt.Printf("Credential: %s\n", maskSecret(cfg.Credential))
		if cfg.ProjectID != "" {
			fmt.Printf("Project:    %s\n", cfg.ProjectID)
		}
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := LoadConfig()
		if cfg.Credential == "" {
			return fmt.Errorf("no credential configured: run 'aigw config install' or set AIGW_API_KEY")
		}

		fmt.Printf("Config File: %s\n", configFilePath())
		fmt.Printf("Gateway URL: %s\n", cfg.GatewayURL)
		fmt.Printf("Control URL: %s\n", cfg.ControlURL)
		fmt.Printf("Credential:  %s\n", maskSecret(cfg.Credential))
		fmt.Printf("Project ID:  %s\n", cfg.ProjectID)
		if cfg.Name != "" {
			fmt.Printf("Profile:     %s\n", cfg.Name)
		}
		return nil
	},
}

var configUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove local configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := configFilePath()
		if err := os.Remove(path); err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No configuration file found.")
				return nil
			}
			return fmt.Errorf("remove config file: %w", err)
		}
		fmt.Println("✅ Configuration removed successfully.")
		return nil
	},
}

func init() {
	configCmd.AddCommand(configInstallCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configUninstallCmd)

	configInstallCmd.Flags().String("gateway-url", "http://localhost:8084", "Gateway data plane URL")
	configInstallCmd.Flags().String("control-url", "http://localhost:8081", "Control plane URL")
	configInstallCmd.Flags().String("credential", "", "Gateway API Key (aigw_...)")
	configInstallCmd.Flags().String("project-id", "", "Default project ID for scoping")
	configInstallCmd.Flags().String("name", "", "Profile name")
}
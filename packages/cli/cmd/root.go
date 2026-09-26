package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "aigw",
	Short: "AI Gateway CLI - manage credentials and configuration",
	Long: `AI Gateway CLI provides commands for managing gateway credentials,
service accounts, and local configuration for developers.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(credentialCmd)
	rootCmd.AddCommand(configCmd)
}
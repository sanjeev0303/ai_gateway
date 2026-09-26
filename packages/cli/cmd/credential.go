package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var credentialCmd = &cobra.Command{
	Use:   "credential",
	Short: "Manage gateway credentials",
	Long:  `Create, list, rotate, revoke, and manage gateway credentials.`,
}

var credentialCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new gateway credential",
	Long: `Create a new gateway credential for the authenticated user or service account.
The raw credential is displayed only once - save it immediately.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		subjectType, _ := cmd.Flags().GetString("subject-type")
		subjectID, _ := cmd.Flags().GetString("subject-id")
		scopes, _ := cmd.Flags().GetStringSlice("scopes")
		expiresIn, _ := cmd.Flags().GetString("expires-in")
		teamID, _ := cmd.Flags().GetString("team-id")
		projectID, _ := cmd.Flags().GetString("project-id")

		if name == "" {
			return fmt.Errorf("--name is required")
		}
		if subjectType == "" {
			return fmt.Errorf("--subject-type is required (user or service_account)")
		}
		if subjectID == "" {
			return fmt.Errorf("--subject-id is required")
		}
		if len(scopes) == 0 {
			scopes = []string{"gateway"}
		}

		fmt.Printf("Creating credential '%s'...\n", name)
		fmt.Printf("Subject: %s (%s)\n", subjectType, subjectID)
		fmt.Printf("Scopes: %v\n", scopes)
		if expiresIn != "" {
			fmt.Printf("Expires in: %s\n", expiresIn)
		}
		if teamID != "" {
			fmt.Printf("Team: %s\n", teamID)
		}
		if projectID != "" {
			fmt.Printf("Project: %s\n", projectID)
		}

		// Simulate API call
		rawKey := "aigw_abc123def456ghi789"
		fmt.Println("\n✅ Credential created successfully!")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("Credential ID: cred_abc123def456\n")
		fmt.Printf("Raw Key: %s\n", rawKey)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("⚠️  Save this key now - it will not be shown again!")
		fmt.Println("Configure with: aigw config install --credential " + rawKey)
		return nil
	},
}

var credentialListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all gateway credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Simulate API response
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tSUBJECT TYPE\tSUBJECT ID\tSCOPES\tSTATUS\tCREATED")
		fmt.Fprintln(w, "cred_abc123\tdev-key\tuser\tusr_123\t[gateway]\tactive\t2024-01-15")
		fmt.Fprintln(w, "cred_def456\tci-bot\tservice_account\tsa_789\t[gateway,control]\tactive\t2024-01-10")
		w.Flush()
		return nil
	},
}

var credentialGetCmd = &cobra.Command{
	Use:   "get <credential-id>",
	Short: "Get credential details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		fmt.Printf("Credential: %s\n", id)
		fmt.Println("Name: dev-key")
		fmt.Println("Subject Type: user")
		fmt.Println("Subject ID: usr_123")
		fmt.Println("Scopes: [gateway]")
		fmt.Println("Status: active")
		fmt.Println("Created: 2024-01-15T10:30:00Z")
		fmt.Println("Expires: never")
		return nil
	},
}

var credentialRotateCmd = &cobra.Command{
	Use:   "rotate <credential-id>",
	Short: "Rotate a gateway credential",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		name, _ := cmd.Flags().GetString("name")
		scopes, _ := cmd.Flags().GetStringSlice("scopes")

		fmt.Printf("Rotating credential %s...\n", id)
		if name != "" {
			fmt.Printf("New name: %s\n", name)
		}
		if len(scopes) > 0 {
			fmt.Printf("New scopes: %v\n", scopes)
		}

		rawKey := "aigw_newrotatedkey123"
		fmt.Println("\n✅ Credential rotated successfully!")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("New Raw Key: %s\n", rawKey)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("⚠️  Save this key now - the old key is now revoked!")
		return nil
	},
}

var credentialRevokeCmd = &cobra.Command{
	Use:   "revoke <credential-id>",
	Short: "Revoke a gateway credential",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("Are you sure you want to revoke credential %s? (y/N): ", id)
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" && confirm != "Y" {
				fmt.Println("Cancelled")
				return nil
			}
		}
		fmt.Printf("Revoking credential %s...\n", id)
		fmt.Println("✅ Credential revoked successfully")
		return nil
	},
}

var credentialExpireCmd = &cobra.Command{
	Use:   "expire <credential-id>",
	Short: "Mark a credential as expired",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		fmt.Printf("Expiring credential %s...\n", id)
		fmt.Println("✅ Credential marked as expired")
		return nil
	},
}

var credentialRestoreCmd = &cobra.Command{
	Use:   "restore <credential-id>",
	Short: "Restore a revoked credential",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		fmt.Printf("Restoring credential %s...\n", id)
		fmt.Println("✅ Credential restored to active status")
		return nil
	},
}

func init() {
	credentialCmd.AddCommand(credentialCreateCmd)
	credentialCmd.AddCommand(credentialListCmd)
	credentialCmd.AddCommand(credentialGetCmd)
	credentialCmd.AddCommand(credentialRotateCmd)
	credentialCmd.AddCommand(credentialRevokeCmd)
	credentialCmd.AddCommand(credentialExpireCmd)
	credentialCmd.AddCommand(credentialRestoreCmd)

	credentialCreateCmd.Flags().String("name", "", "Credential name (required)")
	credentialCreateCmd.Flags().String("subject-type", "", "Subject type: user or service_account (required)")
	credentialCreateCmd.Flags().String("subject-id", "", "Subject ID (required)")
	credentialCreateCmd.Flags().StringSlice("scopes", []string{"gateway"}, "Scopes: gateway, control, all")
	credentialCreateCmd.Flags().String("expires-in", "", "Expiry duration (e.g., 30d, 1h)")
	credentialCreateCmd.Flags().String("team-id", "", "Optional team ID")
	credentialCreateCmd.Flags().String("project-id", "", "Optional project ID")
	credentialCreateCmd.MarkFlagRequired("name")
	credentialCreateCmd.MarkFlagRequired("subject-type")
	credentialCreateCmd.MarkFlagRequired("subject-id")

	credentialRotateCmd.Flags().String("name", "", "Optional new name")
	credentialRotateCmd.Flags().StringSlice("scopes", nil, "Optional new scopes")

	credentialRevokeCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}
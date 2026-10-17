package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

type gatewayCredResponse struct {
	ID             string   `json:"id"`
	OrganizationID string   `json:"organization_id"`
	Name           string   `json:"name"`
	KeyPrefix      string   `json:"key_prefix"`
	SubjectType    string   `json:"subject_type"`
	SubjectID      string   `json:"subject_id"`
	TeamID         *string  `json:"team_id,omitempty"`
	ProjectID      *string  `json:"project_id,omitempty"`
	Scopes         []string `json:"scopes"`
	Status         string   `json:"status"`
	CreatedAt      string   `json:"created_at"`
	ExpiresAt      *string  `json:"expires_at,omitempty"`
}

func doControlRequest(cfg Config, method, path string, body any) (*http.Response, error) {
	if cfg.ControlURL == "" {
		return nil, fmt.Errorf("control-plane URL not configured (set AIGW_CONTROL_URL or configure ~/.aigw/config.yaml)")
	}
	url := cfg.ControlURL + path
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.Credential != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Credential)
		req.Header.Set("x-api-key", cfg.Credential)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request to control-plane (%s): %w", url, err)
	}
	return resp, nil
}

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
			return fmt.Errorf("--subject-id is required (UUID)")
		}
		if len(scopes) == 0 {
			scopes = []string{"gateway"}
		}

		payload := map[string]any{
			"name":         name,
			"subject_type": subjectType,
			"subject_id":   subjectID,
			"scopes":       scopes,
		}
		if teamID != "" {
			payload["team_id"] = teamID
		}
		if projectID != "" {
			payload["project_id"] = projectID
		}
		if expiresIn != "" {
			payload["expires_in"] = expiresIn
		}

		cfg := LoadConfig()
		resp, err := doControlRequest(cfg, http.MethodPost, "/v1/gateway-credentials", payload)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			data, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("create credential failed (HTTP %d): %s", resp.StatusCode, string(data))
		}

		var cred gatewayCredResponse
		if err := json.NewDecoder(resp.Body).Decode(&cred); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}

		rawKey := cred.KeyPrefix // On create, key_prefix holds the issued raw token
		fmt.Println("\n✅ Credential created successfully!")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("Credential ID: %s\n", cred.ID)
		fmt.Printf("Name:          %s\n", cred.Name)
		fmt.Printf("Subject:       %s (%s)\n", cred.SubjectType, cred.SubjectID)
		fmt.Printf("Scopes:        %v\n", cred.Scopes)
		fmt.Printf("Raw Key:       %s\n", rawKey)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("⚠️  Save this key now - it will not be shown again!")
		fmt.Println("Configure locally: aigw auth login --credential " + rawKey)
		return nil
	},
}

var credentialListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all gateway credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := LoadConfig()
		resp, err := doControlRequest(cfg, http.MethodGet, "/v1/gateway-credentials", nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			data, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("list credentials failed (HTTP %d): %s", resp.StatusCode, string(data))
		}

		var res struct {
			Items []gatewayCredResponse `json:"items"`
		}
		respBytes, _ := io.ReadAll(resp.Body)
		if err := json.Unmarshal(respBytes, &res); err != nil || len(res.Items) == 0 {
			var directItems []gatewayCredResponse
			if err := json.Unmarshal(respBytes, &directItems); err == nil {
				res.Items = directItems
			}
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tSUBJECT TYPE\tSUBJECT ID\tSCOPES\tSTATUS\tKEY PREFIX\tCREATED")
		for _, c := range res.Items {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%v\t%s\t%s\t%s\n",
				c.ID, c.Name, c.SubjectType, c.SubjectID, c.Scopes, c.Status, c.KeyPrefix, c.CreatedAt)
		}
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
		cfg := LoadConfig()
		resp, err := doControlRequest(cfg, http.MethodGet, "/v1/gateway-credentials/"+id, nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			data, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("get credential failed (HTTP %d): %s", resp.StatusCode, string(data))
		}

		var cred gatewayCredResponse
		if err := json.NewDecoder(resp.Body).Decode(&cred); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}

		fmt.Printf("Credential ID: %s\n", cred.ID)
		fmt.Printf("Name:          %s\n", cred.Name)
		fmt.Printf("Subject Type:  %s\n", cred.SubjectType)
		fmt.Printf("Subject ID:    %s\n", cred.SubjectID)
		fmt.Printf("Key Prefix:    %s\n", cred.KeyPrefix)
		fmt.Printf("Scopes:        %v\n", cred.Scopes)
		fmt.Printf("Status:        %s\n", cred.Status)
		fmt.Printf("Created:       %s\n", cred.CreatedAt)
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

		payload := map[string]any{}
		if name != "" {
			payload["name"] = name
		}
		if len(scopes) > 0 {
			payload["scopes"] = scopes
		}

		cfg := LoadConfig()
		resp, err := doControlRequest(cfg, http.MethodPost, "/v1/gateway-credentials/"+id+"/rotate", payload)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			data, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("rotate credential failed (HTTP %d): %s", resp.StatusCode, string(data))
		}

		var cred gatewayCredResponse
		if err := json.NewDecoder(resp.Body).Decode(&cred); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}

		rawKey := cred.KeyPrefix // On rotate, key_prefix holds the newly issued raw token
		fmt.Println("\n✅ Credential rotated successfully!")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("Credential ID: %s\n", cred.ID)
		fmt.Printf("New Raw Key:   %s\n", rawKey)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("⚠️  Save this key now - the previous key has been immediately revoked!")

		// If current configured key was rotated, offer or update configuration
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

		cfg := LoadConfig()
		resp, err := doControlRequest(cfg, http.MethodDelete, "/v1/gateway-credentials/"+id, nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNoContent {
			data, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("revoke credential failed (HTTP %d): %s", resp.StatusCode, string(data))
		}

		fmt.Printf("✅ Credential %s revoked successfully.\n", id)
		return nil
	},
}

var credentialExpireCmd = &cobra.Command{
	Use:   "expire <credential-id>",
	Short: "Mark a credential as expired",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		cfg := LoadConfig()
		resp, err := doControlRequest(cfg, http.MethodPost, "/v1/gateway-credentials/"+id+"/expire", nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNoContent {
			data, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("expire credential failed (HTTP %d): %s", resp.StatusCode, string(data))
		}

		fmt.Printf("✅ Credential %s marked as expired.\n", id)
		return nil
	},
}

var credentialRestoreCmd = &cobra.Command{
	Use:   "restore <credential-id>",
	Short: "Restore a revoked credential",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		cfg := LoadConfig()
		resp, err := doControlRequest(cfg, http.MethodPost, "/v1/gateway-credentials/"+id+"/restore", nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNoContent {
			data, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("restore credential failed (HTTP %d): %s", resp.StatusCode, string(data))
		}

		fmt.Printf("✅ Credential %s restored to active status.\n", id)
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

	credentialRotateCmd.Flags().String("name", "", "Optional new name")
	credentialRotateCmd.Flags().StringSlice("scopes", nil, "Optional new scopes")

	credentialRevokeCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}

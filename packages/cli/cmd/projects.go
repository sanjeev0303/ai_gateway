package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

func isValidUUID(u string) bool {
	if len(u) != 36 {
		return false
	}
	for i, r := range u {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if r != '-' {
				return false
			}
		} else {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
				return false
			}
		}
	}
	return true
}

var projectsCmd = &cobra.Command{
	Use:     "projects",
	Aliases: []string{"project"},
	Short:   "Manage and select active project for gateway scoping",
	Long:    `Select or list permitted projects to bind requests to a specific project scope.`,
}

var projectsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := LoadConfig()

		// Attempt to query control plane if reachable
		if cfg.ControlURL != "" && cfg.Credential != "" {
			req, err := http.NewRequest(http.MethodGet, cfg.ControlURL+"/v1/projects", nil)
			if err == nil {
				req.Header.Set("Authorization", "Bearer "+cfg.Credential)
				req.Header.Set("x-api-key", cfg.Credential)
				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Do(req)
				if err == nil && resp.StatusCode == http.StatusOK {
					defer resp.Body.Close()
					var data struct {
						Items []struct {
							ID          string `json:"id"`
							Name        string `json:"name"`
							Slug        string `json:"slug"`
							Description string `json:"description"`
						} `json:"items"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&data); err == nil && len(data.Items) > 0 {
						w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
						fmt.Fprintln(w, "PROJECT ID\tNAME\tSLUG\tACTIVE")
						for _, p := range data.Items {
							active := ""
							if p.ID == cfg.ProjectID {
								active = "✓ (selected)"
							}
							fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.ID, p.Name, p.Slug, active)
						}
						w.Flush()
						return nil
					}
				}
			}
		}

		if cfg.ProjectID != "" {
			fmt.Printf("Active Project: %s\n", cfg.ProjectID)
		} else {
			fmt.Println("No project currently selected (unscoped / org-wide).")
			fmt.Println("Use 'aigw projects select <project-id>' to select a project.")
		}
		return nil
	},
}

var projectsSelectCmd = &cobra.Command{
	Use:     "select <project-id>",
	Aliases: []string{"use", "set"},
	Short:   "Select active project for gateway requests",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectID := args[0]
		if !isValidUUID(projectID) {
			return fmt.Errorf("invalid project ID %q: must be a valid UUID", projectID)
		}

		cfg := LoadConfig()
		cfg.ProjectID = projectID
		if err := SaveConfig(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Printf("✅ Active project set to: %s\n", projectID)
		return nil
	},
}

var projectsCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show currently selected project",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := LoadConfig()
		if cfg.ProjectID == "" {
			fmt.Println("No active project configured (unscoped).")
			return nil
		}
		fmt.Printf("Active Project: %s\n", cfg.ProjectID)
		return nil
	},
}

var projectsClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear active project selection",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := LoadConfig()
		cfg.ProjectID = ""
		if err := SaveConfig(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Println("✅ Active project cleared (now unscoped).")
		return nil
	},
}

func init() {
	projectsCmd.AddCommand(projectsListCmd)
	projectsCmd.AddCommand(projectsSelectCmd)
	projectsCmd.AddCommand(projectsCurrentCmd)
	projectsCmd.AddCommand(projectsClearCmd)
}

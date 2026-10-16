package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"ai-governance-saas/packages/cli/internal/client"

	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:     "models",
	Aliases: []string{"model"},
	Short:   "List permitted models accessible to this credential",
	Long:    `Query the Sentinel Gateway data plane for all models permitted under active governance policy.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runListModels(cmd)
	},
}

var modelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List permitted models",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runListModels(cmd)
	},
}

func runListModels(cmd *cobra.Command) error {
	cfg := LoadConfig()
	if cfg.Credential == "" {
		return fmt.Errorf("no credential configured: run 'aigw auth login --credential <key>' or set AIGW_API_KEY")
	}

	c := client.New(cfg.GatewayURL, cfg.Credential, cfg.ProjectID)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	models, err := c.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}

	if len(models) == 0 {
		fmt.Println("No permitted models found for this credential.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "MODEL ID\tPROVIDER\tDISPLAY NAME")
	for _, m := range models {
		displayName := m.DisplayName
		if displayName == "" {
			displayName = m.ID
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", m.ID, m.Provider, displayName)
	}
	w.Flush()
	return nil
}

func init() {
	modelsCmd.AddCommand(modelsListCmd)
}

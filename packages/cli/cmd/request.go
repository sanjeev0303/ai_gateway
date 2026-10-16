package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"ai-governance-saas/packages/cli/internal/client"

	"github.com/spf13/cobra"
)

var requestCmd = &cobra.Command{
	Use:     "request",
	Aliases: []string{"chat", "prompt"},
	Short:   "Send an LLM request through the Sentinel Gateway",
	Long: `Send an LLM prompt through the Sentinel Gateway data plane with streaming and policy governance.
Validates credentials, rates, and budgets, preserving streaming and inspecting request telemetry.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		model, _ := cmd.Flags().GetString("model")
		prompt, _ := cmd.Flags().GetString("prompt")
		stream, _ := cmd.Flags().GetBool("stream")
		projectID, _ := cmd.Flags().GetString("project-id")
		maxTokens, _ := cmd.Flags().GetInt("max-tokens")

		// If prompt flag is empty, check positional args or stdin
		if prompt == "" && len(args) > 0 {
			prompt = strings.Join(args, " ")
		}
		if prompt == "" {
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				bytes, err := io.ReadAll(os.Stdin)
				if err == nil {
					prompt = strings.TrimSpace(string(bytes))
				}
			}
		}

		if prompt == "" {
			return fmt.Errorf("prompt is required: provide via --prompt, arguments, or stdin")
		}

		cfg := LoadConfig()
		if cfg.Credential == "" {
			return fmt.Errorf("no credential configured: run 'aigw auth login --credential <key>' or set AIGW_API_KEY")
		}

		if projectID == "" {
			projectID = cfg.ProjectID
		}

		c := client.New(cfg.GatewayURL, cfg.Credential, projectID)
		payload := client.RequestPayload{
			Model:     model,
			Prompt:    prompt,
			Stream:    stream,
			MaxTokens: maxTokens,
			ProjectID: projectID,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		var meta *client.RequestMetadata
		var err error

		if stream {
			meta, err = c.StreamRequest(ctx, payload, func(chunk string) {
				_, _ = os.Stdout.WriteString(chunk)
			})
			fmt.Println() // trailing newline after stream
		} else {
			var text string
			text, meta, err = c.SendRequest(ctx, payload)
			if err == nil {
				fmt.Println(text)
			}
		}

		if err != nil {
			return fmt.Errorf("gateway request rejected: %w", err)
		}

		if meta != nil {
			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			fmt.Printf("Status:     HTTP %d OK\n", meta.StatusCode)
			fmt.Printf("Model:      %s\n", meta.Model)
			if meta.RequestID != "" {
				fmt.Printf("Request ID: %s\n", meta.RequestID)
			}
			fmt.Printf("Latency:    %v\n", meta.Duration.Round(time.Millisecond))
			if meta.TotalTokens > 0 {
				fmt.Printf("Tokens:     %d input / %d output (%d total)\n", meta.InputTokens, meta.OutputTokens, meta.TotalTokens)
			}
			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		}

		return nil
	},
}

func init() {
	requestCmd.Flags().StringP("model", "m", "claude-sonnet-4-5", "Model identifier to request")
	requestCmd.Flags().StringP("prompt", "p", "", "Prompt text")
	requestCmd.Flags().Bool("stream", true, "Stream response tokens in real-time")
	requestCmd.Flags().String("project-id", "", "Project scope override")
	requestCmd.Flags().Int("max-tokens", 1024, "Maximum completion tokens")
}

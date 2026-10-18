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

		interactive, _ := cmd.Flags().GetBool("interactive")

		// If prompt flag is empty, check positional args or stdin
		if prompt == "" && len(args) > 0 {
			prompt = strings.Join(args, " ")
		}

		isTerminal := false
		stat, err := os.Stdin.Stat()
		if err == nil {
			if (stat.Mode() & os.ModeCharDevice) != 0 {
				isTerminal = true
			} else if prompt == "" {
				bytes, readErr := io.ReadAll(os.Stdin)
				if readErr == nil {
					prompt = strings.TrimSpace(string(bytes))
				}
			}
		}

		cfg := LoadConfig()
		if cfg.Credential == "" {
			return fmt.Errorf("no credential configured: run 'aigw auth login --credential <key>' or set AIGW_API_KEY")
		}

		if projectID == "" {
			projectID = cfg.ProjectID
		}

		session := NewSessionState(cfg, model, projectID, stream, maxTokens)
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		// Handle interactive REPL mode when no prompt is provided in a terminal or --interactive is set
		if prompt == "" {
			if isTerminal || interactive {
				return RunInteractiveSession(ctx, session, os.Stdin, os.Stdout)
			}
			return fmt.Errorf("prompt is required: provide via --prompt, arguments, stdin, or run with --interactive")
		}

		// Handle single-shot slash commands (e.g. aigw request "/provider" or aigw request "/model")
		if strings.HasPrefix(strings.TrimSpace(prompt), "/") {
			handled, _, slashErr := HandleSlashCommand(ctx, prompt, session, os.Stdout)
			if slashErr != nil {
				return slashErr
			}
			if handled {
				return nil
			}
		}

		payload := client.RequestPayload{
			Model:     session.Model,
			Prompt:    prompt,
			Stream:    session.Stream,
			MaxTokens: session.MaxTokens,
			ProjectID: session.ProjectID,
		}

		reqCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()

		var meta *client.RequestMetadata
		var reqErr error

		if session.Stream {
			meta, reqErr = session.Client.StreamRequest(reqCtx, payload, func(chunk string) {
				_, _ = os.Stdout.WriteString(chunk)
			})
			fmt.Println() // trailing newline after stream
		} else {
			var text string
			text, meta, reqErr = session.Client.SendRequest(reqCtx, payload)
			if reqErr == nil {
				fmt.Println(text)
			}
		}

		if reqErr != nil {
			return fmt.Errorf("gateway request rejected: %w", reqErr)
		}

		if meta != nil {
			callCost := EstimateCost(session.Model, meta.InputTokens, meta.OutputTokens)
			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			fmt.Printf("Status:     HTTP %d OK\n", meta.StatusCode)
			fmt.Printf("Model:      %s (Provider: %s)\n", meta.Model, session.Provider)
			if meta.RequestID != "" {
				fmt.Printf("Request ID: %s\n", meta.RequestID)
			}
			fmt.Printf("Latency:    %v\n", meta.Duration.Round(time.Millisecond))
			if meta.TotalTokens > 0 {
				sourceTag := ""
				if meta.UsageSource != "" {
					sourceTag = fmt.Sprintf(" [%s]", meta.UsageSource)
				}
				fmt.Printf("Tokens:     %d input / %d output (%d total)%s\n", meta.InputTokens, meta.OutputTokens, meta.TotalTokens, sourceTag)
				fmt.Printf("Est. Cost:  $%.6f USD\n", callCost)
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
	requestCmd.Flags().BoolP("interactive", "i", false, "Start interactive REPL session with slash command support")
}

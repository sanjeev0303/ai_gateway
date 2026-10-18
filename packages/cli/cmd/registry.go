package cmd

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"ai-governance-saas/packages/cli/internal/client"
)

// SlashCommand defines metadata, applicability, validation, and execution for a CLI slash command.
type SlashCommand struct {
	Name            string
	Aliases         []string
	Description     string
	Provider        string // "common", "anthropic", "openai", "gemini", "huggingface"
	Capability      string // "", "reasoning", "streaming", "system"
	RequiresGateway bool   // true if execution involves a gateway API call
	ArgsDescription string
	HelpText        string
	ValidateArgs    func(args []string) error
	Execute         func(ctx context.Context, session *SessionState, args []string, out io.Writer) error
}

// CommandRegistry holds all registered slash commands and provides provider-aware discovery.
type CommandRegistry struct {
	commands map[string]*SlashCommand
	order    []string
}

// NewDefaultCommandRegistry creates and initializes the default provider-aware command registry.
func NewDefaultCommandRegistry() *CommandRegistry {
	r := &CommandRegistry{
		commands: make(map[string]*SlashCommand),
	}

	// -------------------------------------------------------------
	// Common Commands (Applicable to all providers)
	// -------------------------------------------------------------

	r.Register(&SlashCommand{
		Name:            "/help",
		Aliases:         []string{"/?"},
		Description:     "Display available slash commands for the active provider and model",
		Provider:        "common",
		ArgsDescription: "[command]",
		HelpText:        "Type /help to display the command menu, or /help <cmd> for detailed usage.",
		Execute: func(ctx context.Context, session *SessionState, args []string, out io.Writer) error {
			if len(args) > 0 {
				target := strings.ToLower(args[0])
				if !strings.HasPrefix(target, "/") {
					target = "/" + target
				}
				cmd := session.Registry.Find(target)
				if cmd == nil || !session.Registry.IsCommandAvailable(cmd, session) {
					return fmt.Errorf("command %q is not available for active provider %s", target, session.Provider)
				}
				fmt.Fprintf(out, "Command:     %s %s\n", cmd.Name, cmd.ArgsDescription)
				fmt.Fprintf(out, "Provider:    %s\n", cmd.Provider)
				fmt.Fprintf(out, "Description: %s\n", cmd.Description)
				if cmd.HelpText != "" {
					fmt.Fprintf(out, "Usage:       %s\n", cmd.HelpText)
				}
				if cmd.RequiresGateway {
					fmt.Fprintln(out, "Execution:   Gateway API call")
				} else {
					fmt.Fprintln(out, "Execution:   Local client operation")
				}
				return nil
			}
			session.Registry.RenderMenu(session, out)
			return nil
		},
	})

	r.Register(&SlashCommand{
		Name:            "/provider",
		Description:     "Display current provider or switch authorized provider connection",
		Provider:        "common",
		ArgsDescription: "[name]",
		HelpText:        "/provider [openai|anthropic|gemini|huggingface] — switch provider if permitted by your credential.",
		Execute: func(ctx context.Context, session *SessionState, args []string, out io.Writer) error {
			if len(args) == 0 {
				prov := session.Provider
				if prov == "" || prov == "unknown" {
					prov = DetectProvider(session.Model)
				}
				fmt.Fprintf(out, "Current provider: %s (active model: %s)\n", prov, session.Model)
				if len(session.AuthorizedProviders) > 0 {
					fmt.Fprintf(out, "Authorized providers: %s\n", strings.Join(session.AuthorizedProviders, ", "))
				}
				return nil
			}

			targetProv := strings.ToLower(args[0])
			validProviders := map[string]bool{
				"openai":      true,
				"anthropic":   true,
				"gemini":      true,
				"huggingface": true,
			}
			if !validProviders[targetProv] {
				return fmt.Errorf("unsupported provider %q: valid providers are openai, anthropic, gemini, huggingface", targetProv)
			}

			if len(session.AuthorizedProviders) > 0 {
				authorized := false
				for _, p := range session.AuthorizedProviders {
					if strings.EqualFold(p, targetProv) {
						authorized = true
						break
					}
				}
				if !authorized {
					return fmt.Errorf("provider %q is not authorized for this credential (authorized: %s)", targetProv, strings.Join(session.AuthorizedProviders, ", "))
				}
			}

			session.Provider = targetProv
			currDetected := DetectProvider(session.Model)
			if currDetected != targetProv {
				foundPermitted := false
				for _, m := range session.PermittedModels {
					if strings.EqualFold(m.Provider, targetProv) {
						session.Model = m.ID
						foundPermitted = true
						break
					}
				}
				if !foundPermitted {
					session.Model = DefaultModelForProvider(targetProv)
				}
			}
			fmt.Fprintf(out, "Switched provider to %s (active model: %s)\n", session.Provider, session.Model)
			return nil
		},
	})

	r.Register(&SlashCommand{
		Name:            "/model",
		Description:     "Display active model, select model, or list permitted models from gateway",
		Provider:        "common",
		RequiresGateway: true,
		ArgsDescription: "[model_id]",
		HelpText:        "/model to list permitted models, or /model <id> to switch active model.",
		Execute: func(ctx context.Context, session *SessionState, args []string, out io.Writer) error {
			if len(args) == 0 {
				fmt.Fprintln(out, "Querying permitted models from Sentinel Gateway...")
				listCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				defer cancel()
				models, err := session.Client.ListModels(listCtx)
				if err != nil {
					return fmt.Errorf("list models failed: %w", err)
				}
				session.PermittedModels = models
				if len(models) == 0 {
					fmt.Fprintln(out, "No permitted models found.")
					return nil
				}

				w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
				fmt.Fprintln(w, "  MODEL ID\tPROVIDER\tDISPLAY NAME")
				for _, m := range models {
					marker := " "
					if m.ID == session.Model {
						marker = "*"
					}
					displayName := m.DisplayName
					if displayName == "" {
						displayName = m.ID
					}
					fmt.Fprintf(w, "%s %s\t%s\t%s\n", marker, m.ID, m.Provider, displayName)
				}
				w.Flush()
				fmt.Fprintf(out, "\nActive model: %s (*)\n", session.Model)
				return nil
			}

			newModel := args[0]
			if len(session.PermittedModels) > 0 {
				isPermitted := false
				for _, m := range session.PermittedModels {
					if m.ID == newModel {
						isPermitted = true
						if m.Provider != "" {
							session.Provider = m.Provider
						}
						break
					}
				}
				if !isPermitted {
					return fmt.Errorf("model %q is not permitted for this credential", newModel)
				}
			} else {
				detected := DetectProvider(newModel)
				if detected != "unknown" {
					session.Provider = detected
				}
			}
			session.Model = newModel
			fmt.Fprintf(out, "Active model set to: %s (provider: %s)\n", session.Model, session.Provider)
			return nil
		},
	})

	r.Register(&SlashCommand{
		Name:            "/status",
		Description:     "Show gateway URL, masked credential, active provider/model, and session telemetry",
		Provider:        "common",
		HelpText:        "/status shows full session environment and telemetry metrics.",
		Execute: func(ctx context.Context, session *SessionState, args []string, out io.Writer) error {
			prov := session.Provider
			if prov == "" || prov == "unknown" {
				prov = DetectProvider(session.Model)
			}
			fmt.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			fmt.Fprintln(out, "Sentinel Gateway Session Status")
			fmt.Fprintf(out, "  Gateway URL:      %s\n", session.GatewayURL)
			fmt.Fprintf(out, "  Credential:       %s\n", maskKey(session.APIKey))
			if session.ProjectID != "" {
				fmt.Fprintf(out, "  Project ID:       %s\n", session.ProjectID)
			}
			fmt.Fprintf(out, "  Active Provider:  %s\n", prov)
			fmt.Fprintf(out, "  Active Model:     %s\n", session.Model)
			fmt.Fprintf(out, "  Streaming:        %v\n", session.Stream)
			fmt.Fprintf(out, "  Max Tokens:       %d\n", session.MaxTokens)
			if session.Temperature != nil {
				fmt.Fprintf(out, "  Temperature:      %.2f\n", *session.Temperature)
			}
			if session.SystemPrompt != "" {
				preview := session.SystemPrompt
				if len(preview) > 30 {
					preview = preview[:27] + "..."
				}
				fmt.Fprintf(out, "  System Prompt:    %q\n", preview)
			}
			if session.ReasoningEffort != "" {
				fmt.Fprintf(out, "  Reasoning Effort: %s\n", session.ReasoningEffort)
			}
			fmt.Fprintf(out, "  History Turns:    %d messages\n", len(session.Messages))
			fmt.Fprintf(out, "  Session Tokens:   %d (in: %d, out: %d)\n", session.TotalTokens, session.TotalInputTokens, session.TotalOutputTokens)
			fmt.Fprintf(out, "  Session Spend:    $%.6f USD\n", session.EstimatedCost)
			fmt.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			return nil
		},
	})

	r.Register(&SlashCommand{
		Name:            "/usage",
		Aliases:         []string{"/tokens"},
		Description:     "Show cumulative session token usage breakdown",
		Provider:        "common",
		HelpText:        "/usage displays prompt, completion, and total session tokens.",
		Execute: func(ctx context.Context, session *SessionState, args []string, out io.Writer) error {
			fmt.Fprintln(out, "Session Cumulative Token Usage:")
			fmt.Fprintf(out, "  Prompt/Input:  %d tokens\n", session.TotalInputTokens)
			fmt.Fprintf(out, "  Completion:    %d tokens\n", session.TotalOutputTokens)
			fmt.Fprintf(out, "  Total:         %d tokens\n", session.TotalTokens)
			fmt.Fprintf(out, "  Messages:      %d turns in context\n", len(session.Messages))
			return nil
		},
	})

	r.Register(&SlashCommand{
		Name:            "/cost",
		Description:     "Show cumulative estimated session cost in USD",
		Provider:        "common",
		HelpText:        "/cost displays cumulative estimated cost based on token consumption.",
		Execute: func(ctx context.Context, session *SessionState, args []string, out io.Writer) error {
			fmt.Fprintf(out, "Session Cumulative Estimated Cost: $%.6f USD (based on %d total tokens)\n", session.EstimatedCost, session.TotalTokens)
			return nil
		},
	})

	r.Register(&SlashCommand{
		Name:            "/permissions",
		Aliases:         []string{"/auth"},
		Description:     "Inspect effective authorization scope, bound provider account, and limits",
		Provider:        "common",
		HelpText:        "/permissions shows authorized providers, model restrictions, and scope boundaries.",
		Execute: func(ctx context.Context, session *SessionState, args []string, out io.Writer) error {
			fmt.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			fmt.Fprintln(out, "Sentinel Credential Authorization Scope")
			fmt.Fprintf(out, "  Credential:        %s\n", maskKey(session.APIKey))
			if session.ProjectID != "" {
				fmt.Fprintf(out, "  Project Scope:     %s\n", session.ProjectID)
			} else {
				fmt.Fprintln(out, "  Project Scope:     Organization default")
			}

			if len(session.AuthorizedProviders) > 0 {
				fmt.Fprintf(out, "  Bound Providers:   %s\n", strings.Join(session.AuthorizedProviders, ", "))
			} else {
				fmt.Fprintln(out, "  Bound Providers:   All organization-connected providers")
			}

			if len(session.PermittedModels) > 0 {
				fmt.Fprintf(out, "  Permitted Models:  %d models accessible\n", len(session.PermittedModels))
				for _, m := range session.PermittedModels {
					fmt.Fprintf(out, "    - %s (%s)\n", m.ID, m.Provider)
				}
			} else {
				fmt.Fprintln(out, "  Permitted Models:  Dynamic (unrestricted within connected providers)")
			}
			fmt.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			return nil
		},
	})

	r.Register(&SlashCommand{
		Name:            "/clear",
		Description:     "Reset conversation message history and session usage counters",
		Provider:        "common",
		HelpText:        "/clear wipes local multi-turn context and resets session token counters.",
		Execute: func(ctx context.Context, session *SessionState, args []string, out io.Writer) error {
			turns := len(session.Messages)
			session.Messages = nil
			session.TotalInputTokens = 0
			session.TotalOutputTokens = 0
			session.TotalTokens = 0
			session.EstimatedCost = 0.0
			fmt.Fprintf(out, "Conversation history cleared (%d messages removed). Usage counters reset.\n", turns)
			return nil
		},
	})

	r.Register(&SlashCommand{
		Name:            "/compact",
		Description:     "Compact conversation history via gateway summarization to save context tokens",
		Provider:        "common",
		RequiresGateway: true,
		HelpText:        "/compact summarizes prior message turns, keeping essential context while freeing tokens.",
		Execute: func(ctx context.Context, session *SessionState, args []string, out io.Writer) error {
			numMsgs := len(session.Messages)
			if numMsgs <= 2 {
				fmt.Fprintf(out, "Conversation is already compact (%d messages). No compaction needed.\n", numMsgs)
				return nil
			}

			fmt.Fprintf(out, "Compacting %d messages using %s...\n", numMsgs, session.Model)

			// Build transcript of messages to compact (all except the very last turn)
			var sb strings.Builder
			for i, m := range session.Messages[:numMsgs-1] {
				sb.WriteString(fmt.Sprintf("%s: %s\n\n", strings.ToUpper(m.Role), m.Content))
				if i >= 10 { // Cap length for compaction input safety
					break
				}
			}

			compactPrompt := "Summarize the following prior conversation concisely in 2-3 sentences, preserving key decisions, constraints, context, and entities:\n\n" + sb.String()

			compactCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			summaryText, meta, err := session.Client.SendRequest(compactCtx, client.RequestPayload{
				Model:        session.Model,
				Prompt:       compactPrompt,
				MaxTokens:    512,
				ProjectID:    session.ProjectID,
				SystemPrompt: "You are a concise conversation summarizer. Distill prior turns into essential context.",
			})

			if err != nil {
				// Deterministic local compaction fallback
				truncatedMsgs := session.Messages[numMsgs-2:]
				session.Messages = append([]client.ChatMessage{
					{
						Role:      "assistant",
						Content:   "[Context compacted locally due to gateway error: prior turns preserved in summary]",
						Timestamp: time.Now(),
					},
				}, truncatedMsgs...)
				fmt.Fprintf(out, "Compacted %d messages locally to %d turns (gateway summarization was unavailable: %v)\n", numMsgs, len(session.Messages), err)
				return nil
			}

			// Successful LLM compaction
			lastMsg := session.Messages[numMsgs-1]
			summaryMsg := client.ChatMessage{
				Role:      "assistant",
				Content:   "[Compacted Conversation Context: " + strings.TrimSpace(summaryText) + "]",
				Timestamp: time.Now(),
			}
			session.Messages = []client.ChatMessage{summaryMsg, lastMsg}

			if meta != nil {
				session.TotalInputTokens += meta.InputTokens
				session.TotalOutputTokens += meta.OutputTokens
				session.TotalTokens += meta.TotalTokens
				session.EstimatedCost += EstimateCost(session.Model, meta.InputTokens, meta.OutputTokens)
			}

			fmt.Fprintf(out, "Compaction complete: %d messages compacted to %d turns.\n", numMsgs, len(session.Messages))
			fmt.Fprintf(out, "Context summary: %s\n", strings.TrimSpace(summaryText))
			return nil
		},
	})

	r.Register(&SlashCommand{
		Name:            "/stream",
		Description:     "Toggle or set streaming response mode",
		Provider:        "common",
		ArgsDescription: "[on|off]",
		HelpText:        "/stream [on|off] toggles token streaming.",
		Execute: func(ctx context.Context, session *SessionState, args []string, out io.Writer) error {
			if len(args) == 0 {
				session.Stream = !session.Stream
			} else {
				val := strings.ToLower(args[0])
				switch val {
				case "on", "true", "1", "enable", "enabled":
					session.Stream = true
				case "off", "false", "0", "disable", "disabled":
					session.Stream = false
				default:
					return fmt.Errorf("invalid stream option %q: use on/off or true/false", val)
				}
			}
			state := "disabled"
			if session.Stream {
				state = "enabled"
			}
			fmt.Fprintf(out, "Streaming is now %s\n", state)
			return nil
		},
	})

	r.Register(&SlashCommand{
		Name:            "/system",
		Description:     "Set or inspect active system prompt instructions",
		Provider:        "common",
		ArgsDescription: "[instructions]",
		HelpText:        "/system without arguments shows current system prompt; /system <prompt> sets it.",
		Execute: func(ctx context.Context, session *SessionState, args []string, out io.Writer) error {
			if len(args) == 0 {
				if session.SystemPrompt == "" {
					fmt.Fprintln(out, "No system prompt currently set.")
				} else {
					fmt.Fprintf(out, "Active system prompt: %s\n", session.SystemPrompt)
				}
				return nil
			}
			session.SystemPrompt = strings.Join(args, " ")
			fmt.Fprintf(out, "System prompt updated: %s\n", session.SystemPrompt)
			return nil
		},
	})

	r.Register(&SlashCommand{
		Name:        "/exit",
		Aliases:     []string{"/quit"},
		Description: "Exit the interactive session",
		Provider:    "common",
		HelpText:    "/exit or /quit exits the Sentinel CLI.",
		Execute: func(ctx context.Context, session *SessionState, args []string, out io.Writer) error {
			fmt.Fprintln(out, "Exiting Sentinel session. Goodbye!")
			return nil
		},
	})

	r.Register(&SlashCommand{
		Name:            "/temp",
		Description:     "Set or inspect sampling temperature (0.0 to 1.0 for Claude, 0.0 to 2.0 for others)",
		Provider:        "common",
		ArgsDescription: "[number]",
		HelpText:        "/temp <value> sets sampling temperature for the active provider.",
		Execute: func(ctx context.Context, session *SessionState, args []string, out io.Writer) error {
			maxVal := 2.0
			if strings.EqualFold(session.Provider, "anthropic") {
				maxVal = 1.0
			}
			return handleTempCommand(session, args, maxVal, out)
		},
	})

	// -------------------------------------------------------------
	// Anthropic-Specific Commands
	// -------------------------------------------------------------

	r.Register(&SlashCommand{
		Name:            "/max-tokens",
		Description:     "Set maximum token limit for Claude completion responses",
		Provider:        "anthropic",
		ArgsDescription: "<number>",
		HelpText:        "/max-tokens <int> sets max response tokens.",
		Execute: func(ctx context.Context, session *SessionState, args []string, out io.Writer) error {
			if len(args) == 0 {
				fmt.Fprintf(out, "Current max tokens: %d\n", session.MaxTokens)
				return nil
			}
			n, err := strconv.Atoi(args[0])
			if err != nil || n <= 0 {
				return fmt.Errorf("invalid max-tokens value %q: must be positive integer", args[0])
			}
			session.MaxTokens = n
			fmt.Fprintf(out, "Max tokens set to %d\n", session.MaxTokens)
			return nil
		},
	})

	// -------------------------------------------------------------
	// OpenAI-Specific Commands
	// -------------------------------------------------------------

	r.Register(&SlashCommand{
		Name:            "/reasoning",
		Description:     "Set reasoning effort level for OpenAI reasoning models (o1, o3-mini)",
		Provider:        "openai",
		Capability:      "reasoning",
		ArgsDescription: "[low|medium|high]",
		HelpText:        "/reasoning [low|medium|high] sets reasoning effort on supported OpenAI models.",
		Execute: func(ctx context.Context, session *SessionState, args []string, out io.Writer) error {
			if len(args) == 0 {
				effort := session.ReasoningEffort
				if effort == "" {
					effort = "default (not set)"
				}
				fmt.Fprintf(out, "Current reasoning effort: %s\n", effort)
				return nil
			}
			level := strings.ToLower(args[0])
			if level != "low" && level != "medium" && level != "high" && level != "off" {
				return fmt.Errorf("invalid reasoning effort %q: choose from low, medium, high, off", level)
			}
			if level == "off" {
				session.ReasoningEffort = ""
				fmt.Fprintln(out, "Reasoning effort reset to default.")
			} else {
				session.ReasoningEffort = level
				fmt.Fprintf(out, "Reasoning effort set to: %s\n", session.ReasoningEffort)
			}
			return nil
		},
	})

	// -------------------------------------------------------------
	// Gemini-Specific Commands
	// -------------------------------------------------------------

	r.Register(&SlashCommand{
		Name:            "/safety",
		Description:     "Inspect or set Google Gemini safety threshold settings",
		Provider:        "gemini",
		ArgsDescription: "[standard|strict|permissive]",
		HelpText:        "/safety shows or configures Gemini content safety threshold preset.",
		Execute: func(ctx context.Context, session *SessionState, args []string, out io.Writer) error {
			if len(args) == 0 {
				setting := session.SafetySettings
				if setting == "" {
					setting = "BLOCK_MEDIUM_AND_ABOVE (standard)"
				}
				fmt.Fprintf(out, "Active Gemini safety setting: %s\n", setting)
				return nil
			}
			val := strings.ToLower(args[0])
			switch val {
			case "standard", "default":
				session.SafetySettings = "BLOCK_MEDIUM_AND_ABOVE"
			case "strict":
				session.SafetySettings = "BLOCK_LOW_AND_ABOVE"
			case "permissive":
				session.SafetySettings = "BLOCK_ONLY_HIGH"
			default:
				return fmt.Errorf("invalid safety preset %q: choose standard, strict, or permissive", val)
			}
			fmt.Fprintf(out, "Gemini safety threshold set to %s\n", session.SafetySettings)
			return nil
		},
	})

	// -------------------------------------------------------------
	// Hugging Face-Specific Commands
	// -------------------------------------------------------------

	r.Register(&SlashCommand{
		Name:            "/endpoint",
		Description:     "View or configure dedicated Hugging Face Inference Endpoint URL",
		Provider:        "huggingface",
		ArgsDescription: "[url]",
		HelpText:        "/endpoint [url] sets custom Hugging Face dedicated inference endpoint.",
		Execute: func(ctx context.Context, session *SessionState, args []string, out io.Writer) error {
			if len(args) == 0 {
				if session.HFEndpoint == "" {
					fmt.Fprintln(out, "HF Endpoint: https://router.huggingface.co/v1 (default serverless router)")
				} else {
					fmt.Fprintf(out, "HF Endpoint: %s\n", session.HFEndpoint)
				}
				return nil
			}
			url := strings.TrimSpace(args[0])
			if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
				return fmt.Errorf("invalid endpoint URL %q: must begin with http:// or https://", url)
			}
			session.HFEndpoint = url
			fmt.Fprintf(out, "Hugging Face endpoint updated: %s\n", session.HFEndpoint)
			return nil
		},
	})

	r.Register(&SlashCommand{
		Name:            "/params",
		Description:     "Display active Hugging Face decoding and inference parameters",
		Provider:        "huggingface",
		HelpText:        "/params shows current model decoding configuration.",
		Execute: func(ctx context.Context, session *SessionState, args []string, out io.Writer) error {
			fmt.Fprintln(out, "Hugging Face Model Parameters:")
			fmt.Fprintf(out, "  Model:       %s\n", session.Model)
			fmt.Fprintf(out, "  Max Tokens:  %d\n", session.MaxTokens)
			fmt.Fprintf(out, "  Streaming:   %v\n", session.Stream)
			if session.Temperature != nil {
				fmt.Fprintf(out, "  Temperature: %.2f\n", *session.Temperature)
			} else {
				fmt.Fprintln(out, "  Temperature: default (provider standard)")
			}
			if session.HFEndpoint != "" {
				fmt.Fprintf(out, "  Endpoint:    %s\n", session.HFEndpoint)
			} else {
				fmt.Fprintln(out, "  Endpoint:    Serverless Hub Router")
			}
			return nil
		},
	})

	return r
}

func handleTempCommand(session *SessionState, args []string, maxVal float64, out io.Writer) error {
	if len(args) == 0 {
		if session.Temperature == nil {
			fmt.Fprintln(out, "Current temperature: default (model standard)")
		} else {
			fmt.Fprintf(out, "Current temperature: %.2f\n", *session.Temperature)
		}
		return nil
	}
	val, err := strconv.ParseFloat(args[0], 64)
	if err != nil || val < 0.0 || val > maxVal {
		return fmt.Errorf("invalid temperature %q: must be between 0.0 and %.1f", args[0], maxVal)
	}
	session.Temperature = &val
	fmt.Fprintf(out, "Temperature set to %.2f\n", val)
	return nil
}

// Register adds a command to the registry.
func (r *CommandRegistry) Register(cmd *SlashCommand) {
	r.commands[cmd.Name] = cmd
	r.order = append(r.order, cmd.Name)
	for _, alias := range cmd.Aliases {
		r.commands[alias] = cmd
	}
}

// Find retrieves a command by its primary name or alias.
func (r *CommandRegistry) Find(name string) *SlashCommand {
	return r.commands[strings.ToLower(name)]
}

// IsCommandAvailable reports whether a command applies to the current session provider and capabilities.
func (r *CommandRegistry) IsCommandAvailable(cmd *SlashCommand, session *SessionState) bool {
	// Provider check
	if cmd.Provider != "common" && !strings.EqualFold(cmd.Provider, session.Provider) {
		return false
	}

	// Capability check
	if cmd.Capability == "reasoning" {
		model := strings.ToLower(session.Model)
		isReasoning := strings.HasPrefix(model, "o1") || strings.HasPrefix(model, "o3") || strings.Contains(model, "deepseek-r1")
		for _, m := range session.PermittedModels {
			if m.ID == session.Model && m.SupportsReasoning {
				isReasoning = true
				break
			}
		}
		if !isReasoning {
			return false
		}
	}

	return true
}

// AvailableCommands returns all slash commands applicable to the current session.
func (r *CommandRegistry) AvailableCommands(session *SessionState) []*SlashCommand {
	var available []*SlashCommand
	seen := make(map[string]bool)

	for _, name := range r.order {
		cmd := r.commands[name]
		if cmd == nil || seen[cmd.Name] {
			continue
		}
		if r.IsCommandAvailable(cmd, session) {
			seen[cmd.Name] = true
			available = append(available, cmd)
		}
	}
	return available
}

// RenderMenu renders an interactive, formatted slash command menu for the active session.
func (r *CommandRegistry) RenderMenu(session *SessionState, out io.Writer) {
	available := r.AvailableCommands(session)

	var commonCmds []*SlashCommand
	var provCmds []*SlashCommand

	for _, cmd := range available {
		if cmd.Provider == "common" {
			commonCmds = append(commonCmds, cmd)
		} else {
			provCmds = append(provCmds, cmd)
		}
	}

	provLabel := strings.ToUpper(session.Provider)
	if provLabel == "" {
		provLabel = "UNKNOWN"
	}

	fmt.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(out, "Sentinel CLI Slash Commands:")
	fmt.Fprintf(out, "  Active Provider: %s\n", provLabel)
	fmt.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)

	if len(provCmds) > 0 {
		fmt.Fprintf(out, "\n  [%s Specific Commands]\n", provLabel)
		for _, cmd := range provCmds {
			badge := "Local"
			if cmd.RequiresGateway {
				badge = "Gateway"
			}
			syntax := cmd.Name
			if cmd.ArgsDescription != "" {
				syntax += " " + cmd.ArgsDescription
			}
			fmt.Fprintf(w, "    %-24s\t[%s]\t%s\n", syntax, badge, cmd.Description)
		}
		w.Flush()
	}

	if len(commonCmds) > 0 {
		fmt.Fprintln(out, "\n  [Common Sentinel Commands]")
		sort.Slice(commonCmds, func(i, j int) bool {
			return commonCmds[i].Name < commonCmds[j].Name
		})
		for _, cmd := range commonCmds {
			badge := "Local"
			if cmd.RequiresGateway {
				badge = "Gateway"
			}
			syntax := cmd.Name
			if cmd.ArgsDescription != "" {
				syntax += " " + cmd.ArgsDescription
			}
			fmt.Fprintf(w, "    %-24s\t[%s]\t%s\n", syntax, badge, cmd.Description)
		}
		w.Flush()
	}

	fmt.Fprintln(out, "\nType /<command> to execute, or /help <command> for details.")
	fmt.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

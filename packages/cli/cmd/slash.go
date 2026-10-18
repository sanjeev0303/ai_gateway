package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"ai-governance-saas/packages/cli/internal/client"
)

// SessionState tracks session-level configuration and usage telemetry.
type SessionState struct {
	GatewayURL          string
	APIKey              string
	ProjectID           string
	Provider            string
	Model               string
	Stream              bool
	MaxTokens           int
	SystemPrompt        string
	Temperature         *float64
	ReasoningEffort     string
	SafetySettings      string
	HFEndpoint          string
	TotalInputTokens    int
	TotalOutputTokens   int
	TotalTokens         int
	EstimatedCost       float64
	Messages            []client.ChatMessage
	Client              *client.GatewayClient
	AuthorizedProviders []string
	PermittedModels     []client.Model
	Registry            *CommandRegistry
}

// SyncPermittedScope queries the gateway to discover permitted models and providers for this credential.
func (s *SessionState) SyncPermittedScope(ctx context.Context) {
	if s.Client == nil || s.APIKey == "" {
		return
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	models, err := s.Client.ListModels(timeoutCtx)
	if err != nil || len(models) == 0 {
		return
	}
	s.PermittedModels = models
	provMap := make(map[string]bool)
	var provs []string
	for _, m := range models {
		p := strings.ToLower(m.Provider)
		if p != "" && !provMap[p] {
			provMap[p] = true
			provs = append(provs, p)
		}
	}
	s.AuthorizedProviders = provs

	// Check if current provider is permitted
	currProvAuthorized := false
	for _, p := range provs {
		if strings.EqualFold(p, s.Provider) {
			currProvAuthorized = true
			break
		}
	}
	if !currProvAuthorized && len(provs) > 0 {
		s.Provider = provs[0]
		for _, m := range models {
			if strings.EqualFold(m.Provider, s.Provider) {
				s.Model = m.ID
				break
			}
		}
	}
}

// NewSessionState initializes a session from config and overrides.
func NewSessionState(cfg Config, model, projectID string, stream bool, maxTokens int) *SessionState {
	if projectID == "" {
		projectID = cfg.ProjectID
	}
	if model == "" {
		model = "claude-sonnet-4-5"
	}
	if maxTokens <= 0 {
		maxTokens = 1024
	}

	c := client.New(cfg.GatewayURL, cfg.Credential, projectID)

	sess := &SessionState{
		GatewayURL: cfg.GatewayURL,
		APIKey:     cfg.Credential,
		ProjectID:  projectID,
		Model:      model,
		Stream:     stream,
		MaxTokens:  maxTokens,
		Client:     c,
		Registry:   NewDefaultCommandRegistry(),
		Messages:   make([]client.ChatMessage, 0),
	}
	sess.Provider = DetectProvider(model)
	sess.SyncPermittedScope(context.Background())
	return sess
}

// DetectProvider infers provider kind from model identifier.
func DetectProvider(model string) string {
	m := strings.ToLower(model)
	switch {
	case strings.HasPrefix(m, "hf-") || strings.Contains(m, "/"):
		return "huggingface"
	case strings.HasPrefix(m, "gpt-") || strings.HasPrefix(m, "o1") || strings.HasPrefix(m, "o3") || strings.HasPrefix(m, "text-embedding"):
		return "openai"
	case strings.HasPrefix(m, "claude-"):
		return "anthropic"
	case strings.HasPrefix(m, "gemini-"):
		return "gemini"
	default:
		return "unknown"
	}
}

// DefaultModelForProvider returns the recommended canonical model for a provider.
func DefaultModelForProvider(provider string) string {
	switch strings.ToLower(provider) {
	case "huggingface":
		return "meta-llama/Llama-3.3-70B-Instruct"
	case "openai":
		return "gpt-4o"
	case "anthropic":
		return "claude-sonnet-4-5"
	case "gemini":
		return "gemini-2.0-flash"
	default:
		return "claude-sonnet-4-5"
	}
}

// EstimateCost calculates approximate micro-dollar cost for prompt and completion tokens.
func EstimateCost(model string, inTokens, outTokens int) float64 {
	m := strings.ToLower(model)
	var inRate, outRate float64
	switch {
	case strings.Contains(m, "gemini-2.0-flash"):
		inRate = 0.10
		outRate = 0.40
	case strings.Contains(m, "gemini"):
		inRate = 1.25
		outRate = 5.00
	case strings.Contains(m, "gpt-4o-mini"):
		inRate = 0.15
		outRate = 0.60
	case strings.Contains(m, "gpt-4o"):
		inRate = 2.50
		outRate = 10.00
	case strings.Contains(m, "claude-3-5-sonnet") || strings.Contains(m, "claude-sonnet-4-5"):
		inRate = 3.00
		outRate = 15.00
	case strings.Contains(m, "claude-3-5-haiku"):
		inRate = 0.80
		outRate = 4.00
	case strings.Contains(m, "llama-3.3-70b") || strings.Contains(m, "meta-llama"):
		inRate = 0.50
		outRate = 0.80
	default:
		inRate = 1.00
		outRate = 2.00
	}
	return (float64(inTokens)*inRate + float64(outTokens)*outRate) / 1_000_000.0
}

func maskKey(k string) string {
	if len(k) <= 8 {
		return "********"
	}
	return k[:4] + "..." + k[len(k)-4:]
}

// HandleSlashCommand processes interactive or single-shot slash commands through the command registry.
func HandleSlashCommand(ctx context.Context, cmdLine string, session *SessionState, out io.Writer) (handled bool, shouldExit bool, err error) {
	trimmed := strings.TrimSpace(cmdLine)
	if !strings.HasPrefix(trimmed, "/") {
		return false, false, nil
	}

	if session.Registry == nil {
		session.Registry = NewDefaultCommandRegistry()
	}

	// Typing "/" or "/?" alone displays the provider-aware interactive menu
	if trimmed == "/" || trimmed == "/?" {
		session.Registry.RenderMenu(session, out)
		return true, false, nil
	}

	parts := strings.Fields(trimmed)
	cmdName := strings.ToLower(parts[0])
	args := parts[1:]

	cmd := session.Registry.Find(cmdName)
	if cmd == nil {
		return true, false, fmt.Errorf("unknown slash command %q: type /help for available commands", cmdName)
	}

	if !session.Registry.IsCommandAvailable(cmd, session) {
		return true, false, fmt.Errorf("command %q is not available for active provider %s or model %s", cmdName, session.Provider, session.Model)
	}

	if cmd.ValidateArgs != nil {
		if err := cmd.ValidateArgs(args); err != nil {
			return true, false, err
		}
	}

	if err := cmd.Execute(ctx, session, args, out); err != nil {
		return true, false, err
	}

	shouldExit = (cmd.Name == "/exit" || cmd.Name == "/quit")
	return true, shouldExit, nil
}

// RunInteractiveSession runs a continuous REPL with slash command and prompt support.
func RunInteractiveSession(ctx context.Context, session *SessionState, in io.Reader, out io.Writer) error {
	fmt.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(out, "Sentinel AI Gateway — Interactive Session")
	fmt.Fprintf(out, "Active Model: %s | Provider: %s | Streaming: %v\n", session.Model, session.Provider, session.Stream)
	fmt.Fprintln(out, "Type / or /help for slash commands (/provider, /model, /stream, /tokens, /cost, /exit)")
	fmt.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	scanner := bufio.NewScanner(in)
	for {
		promptPrefix := fmt.Sprintf("sentinel [%s/%s]> ", session.Provider, session.Model)
		fmt.Fprint(out, promptPrefix)

		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "/") {
			handled, shouldExit, err := HandleSlashCommand(ctx, line, session, out)
			if err != nil {
				fmt.Fprintf(out, "Error: %v\n", err)
			}
			if handled && shouldExit {
				return nil
			}
			continue
		}

		// Append user message to conversation history
		session.Messages = append(session.Messages, client.ChatMessage{
			Role:      "user",
			Content:   line,
			Timestamp: time.Now(),
		})

		// Regular prompt
		payload := client.RequestPayload{
			Model:           session.Model,
			Prompt:          line,
			Messages:        session.Messages,
			Stream:          session.Stream,
			MaxTokens:       session.MaxTokens,
			ProjectID:       session.ProjectID,
			SystemPrompt:    session.SystemPrompt,
			Temperature:     session.Temperature,
			ReasoningEffort: session.ReasoningEffort,
		}

		reqCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		var meta *client.RequestMetadata
		var reqErr error
		var assistantResp string

		if session.Stream {
			var sb strings.Builder
			meta, reqErr = session.Client.StreamRequest(reqCtx, payload, func(chunk string) {
				sb.WriteString(chunk)
				_, _ = io.WriteString(out, chunk)
			})
			fmt.Fprintln(out)
			assistantResp = sb.String()
		} else {
			assistantResp, meta, reqErr = session.Client.SendRequest(reqCtx, payload)
			if reqErr == nil {
				fmt.Fprintln(out, assistantResp)
			}
		}
		cancel()

		if reqErr != nil {
			fmt.Fprintf(out, "Request failed: %v\n", reqErr)
			if len(session.Messages) > 0 {
				session.Messages = session.Messages[:len(session.Messages)-1]
			}
			continue
		}

		if assistantResp != "" {
			session.Messages = append(session.Messages, client.ChatMessage{
				Role:      "assistant",
				Content:   assistantResp,
				Timestamp: time.Now(),
			})
		}

		if meta != nil {
			session.TotalInputTokens += meta.InputTokens
			session.TotalOutputTokens += meta.OutputTokens
			session.TotalTokens += meta.TotalTokens
			callCost := EstimateCost(session.Model, meta.InputTokens, meta.OutputTokens)
			session.EstimatedCost += callCost

			fmt.Fprintln(out, "────────────────────────────────────────────────────")
			fmt.Fprintf(out, "HTTP %d OK | %v | %d tokens (in: %d, out: %d) | est. $%.6f\n",
				meta.StatusCode, meta.Duration.Round(time.Millisecond), meta.TotalTokens, meta.InputTokens, meta.OutputTokens, callCost)
			fmt.Fprintln(out, "────────────────────────────────────────────────────")
		}
	}

	return scanner.Err()
}

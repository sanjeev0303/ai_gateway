package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// GatewayClient interacts with the Sentinel Gateway data plane.
type GatewayClient struct {
	GatewayURL string
	APIKey     string
	ProjectID  string
	HTTPClient *http.Client
	MaxRetries int
}

// Model represents a permitted model returned by the gateway.
type Model struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Provider          string   `json:"provider"`
	DisplayName       string   `json:"display_name,omitempty"`
	Capabilities      []string `json:"capabilities,omitempty"`
	ContextLimit      int      `json:"context_limit,omitempty"`
	SupportsStreaming bool     `json:"supports_streaming"`
	SupportsTools     bool     `json:"supports_tools"`
	SupportsReasoning bool     `json:"supports_reasoning"`
}

// ChatMessage represents a single message in a multi-turn conversation.
type ChatMessage struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// RequestMetadata contains execution statistics for a gateway call.
type RequestMetadata struct {
	RequestID    string
	StatusCode   int
	Duration     time.Duration
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	Model        string
	IsEstimated  bool
	UsageSource  string
}

// RequestPayload specifies the parameters for an LLM prompt.
type RequestPayload struct {
	Model           string
	Prompt          string
	Messages        []ChatMessage
	Stream          bool
	MaxTokens       int
	ProjectID       string
	SystemPrompt    string
	Temperature     *float64
	ReasoningEffort string
	SafetySettings  string
	HFEndpoint      string
}

// New creates a new GatewayClient.
func New(gatewayURL, apiKey, projectID string) *GatewayClient {
	gatewayURL = strings.TrimRight(gatewayURL, "/")
	return &GatewayClient{
		GatewayURL: gatewayURL,
		APIKey:     apiKey,
		ProjectID:  projectID,
		HTTPClient: &http.Client{Timeout: 0}, // streaming requests need no overall timeout
		MaxRetries: 2,
	}
}

// ListModels fetches the permitted models from the gateway with safe retry for transient errors.
func (c *GatewayClient) ListModels(ctx context.Context) ([]Model, error) {
	url := c.GatewayURL + "/v1/models"

	var lastErr error
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt*100) * time.Millisecond):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

		c.setHeaders(req)

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("send request: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
			lastErr = parseHTTPError(resp)
			resp.Body.Close()
			continue
		}

		if resp.StatusCode != http.StatusOK {
			defer resp.Body.Close()
			return nil, parseHTTPError(resp)
		}

		var data struct {
			Data  []Model `json:"data"`
			Items []Model `json:"items"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode response: %w", err)
		}
		resp.Body.Close()

		if len(data.Data) > 0 {
			return data.Data, nil
		}
		return data.Items, nil
	}

	return nil, lastErr
}

// StreamRequest sends a streaming prompt to the gateway and invokes onChunk for each text delta.
func (c *GatewayClient) StreamRequest(ctx context.Context, payload RequestPayload, onChunk func(string)) (*RequestMetadata, error) {
	start := time.Now()
	isOpenAI := strings.HasPrefix(payload.Model, "gpt-") || strings.HasPrefix(payload.Model, "o1") || strings.HasPrefix(payload.Model, "o3") || strings.HasPrefix(payload.Model, "gemini-")

	endpoint := c.GatewayURL + "/v1/messages"
	if isOpenAI {
		endpoint = c.GatewayURL + "/v1/chat/completions"
	}

	bodyMap := c.buildRequestBody(payload, true, isOpenAI)
	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if payload.HFEndpoint != "" {
		req.Header.Set("x-hf-endpoint", payload.HFEndpoint)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, parseHTTPError(resp)
	}

	meta := &RequestMetadata{
		RequestID:  resp.Header.Get("x-request-id"),
		StatusCode: resp.StatusCode,
		Model:      payload.Model,
	}

	tokensReported := false
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}

		// Gemini SSE format
		var geminiEvt struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
			UsageMetadata struct {
				PromptTokenCount     int `json:"promptTokenCount"`
				CandidatesTokenCount int `json:"candidatesTokenCount"`
				TotalTokenCount      int `json:"totalTokenCount"`
			} `json:"usageMetadata"`
		}
		if err := json.Unmarshal([]byte(data), &geminiEvt); err == nil && len(geminiEvt.Candidates) > 0 {
			for _, part := range geminiEvt.Candidates[0].Content.Parts {
				if part.Text != "" {
					onChunk(part.Text)
				}
			}
			if geminiEvt.UsageMetadata.PromptTokenCount > 0 {
				tokensReported = true
				meta.InputTokens = geminiEvt.UsageMetadata.PromptTokenCount
			}
			if geminiEvt.UsageMetadata.CandidatesTokenCount > 0 {
				tokensReported = true
				meta.OutputTokens = geminiEvt.UsageMetadata.CandidatesTokenCount
			}
			if geminiEvt.UsageMetadata.TotalTokenCount > 0 {
				tokensReported = true
				meta.TotalTokens = geminiEvt.UsageMetadata.TotalTokenCount
			}
			continue
		}

		if isOpenAI {
			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
				Usage *struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
					TotalTokens      int `json:"total_tokens"`
				} `json:"usage"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err == nil {
				if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
					onChunk(chunk.Choices[0].Delta.Content)
				}
				if chunk.Usage != nil {
					tokensReported = true
					meta.InputTokens = chunk.Usage.PromptTokens
					meta.OutputTokens = chunk.Usage.CompletionTokens
					meta.TotalTokens = chunk.Usage.TotalTokens
				}
			}
		} else {
			// Anthropic SSE format
			var evt struct {
				Type  string `json:"type"`
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
				Message struct {
					Usage struct {
						InputTokens int `json:"input_tokens"`
					} `json:"usage"`
				} `json:"message"`
				Usage struct {
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
			}
			if err := json.Unmarshal([]byte(data), &evt); err == nil {
				if evt.Type == "content_block_delta" && evt.Delta.Text != "" {
					onChunk(evt.Delta.Text)
				}
				if evt.Type == "message_start" && evt.Message.Usage.InputTokens > 0 {
					tokensReported = true
					meta.InputTokens = evt.Message.Usage.InputTokens
				}
				if evt.Type == "message_delta" && evt.Usage.OutputTokens > 0 {
					tokensReported = true
					meta.OutputTokens = evt.Usage.OutputTokens
				}
			}
		}
	}

	meta.Duration = time.Since(start)
	if meta.TotalTokens == 0 {
		meta.TotalTokens = meta.InputTokens + meta.OutputTokens
	}
	meta.IsEstimated = !tokensReported
	if tokensReported {
		meta.UsageSource = "reported"
	} else if meta.TotalTokens > 0 {
		meta.UsageSource = "estimated"
	} else {
		meta.UsageSource = "none"
	}

	if err := scanner.Err(); err != nil {
		return meta, fmt.Errorf("stream interrupted: %w", err)
	}
	return meta, nil
}

// SendRequest sends a non-streaming prompt to the gateway.
func (c *GatewayClient) SendRequest(ctx context.Context, payload RequestPayload) (string, *RequestMetadata, error) {
	start := time.Now()
	isOpenAI := strings.HasPrefix(payload.Model, "gpt-") || strings.HasPrefix(payload.Model, "o1") || strings.HasPrefix(payload.Model, "o3") || strings.HasPrefix(payload.Model, "gemini-")

	endpoint := c.GatewayURL + "/v1/messages"
	if isOpenAI {
		endpoint = c.GatewayURL + "/v1/chat/completions"
	}

	bodyMap := c.buildRequestBody(payload, false, isOpenAI)
	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return "", nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	if payload.HFEndpoint != "" {
		req.Header.Set("x-hf-endpoint", payload.HFEndpoint)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", nil, parseHTTPError(resp)
	}

	meta := &RequestMetadata{
		RequestID:  resp.Header.Get("x-request-id"),
		StatusCode: resp.StatusCode,
		Duration:   time.Since(start),
		Model:      payload.Model,
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("read response body: %w", err)
	}

	tokensReported := false
	var outputText string
	if isOpenAI {
		var chatResp struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
			Usage struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(respBody, &chatResp); err == nil && len(chatResp.Choices) > 0 {
			outputText = chatResp.Choices[0].Message.Content
			if chatResp.Usage.TotalTokens > 0 || chatResp.Usage.PromptTokens > 0 {
				tokensReported = true
				meta.InputTokens = chatResp.Usage.PromptTokens
				meta.OutputTokens = chatResp.Usage.CompletionTokens
				meta.TotalTokens = chatResp.Usage.TotalTokens
			}
		} else {
			var geminiResp struct {
				Candidates []struct {
					Content struct {
						Parts []struct {
							Text string `json:"text"`
						} `json:"parts"`
					} `json:"content"`
				} `json:"candidates"`
				UsageMetadata struct {
					PromptTokenCount     int `json:"promptTokenCount"`
					CandidatesTokenCount int `json:"candidatesTokenCount"`
					TotalTokenCount      int `json:"totalTokenCount"`
				} `json:"usageMetadata"`
			}
			if err := json.Unmarshal(respBody, &geminiResp); err == nil && len(geminiResp.Candidates) > 0 {
				for _, part := range geminiResp.Candidates[0].Content.Parts {
					outputText += part.Text
				}
				if geminiResp.UsageMetadata.TotalTokenCount > 0 || geminiResp.UsageMetadata.PromptTokenCount > 0 {
					tokensReported = true
					meta.InputTokens = geminiResp.UsageMetadata.PromptTokenCount
					meta.OutputTokens = geminiResp.UsageMetadata.CandidatesTokenCount
					meta.TotalTokens = geminiResp.UsageMetadata.TotalTokenCount
				}
			}
		}
	} else {
		var geminiResp struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
			UsageMetadata struct {
				PromptTokenCount     int `json:"promptTokenCount"`
				CandidatesTokenCount int `json:"candidatesTokenCount"`
				TotalTokenCount      int `json:"totalTokenCount"`
			} `json:"usageMetadata"`
		}
		if err := json.Unmarshal(respBody, &geminiResp); err == nil && len(geminiResp.Candidates) > 0 {
			for _, part := range geminiResp.Candidates[0].Content.Parts {
				outputText += part.Text
			}
			if geminiResp.UsageMetadata.TotalTokenCount > 0 || geminiResp.UsageMetadata.PromptTokenCount > 0 {
				tokensReported = true
				meta.InputTokens = geminiResp.UsageMetadata.PromptTokenCount
				meta.OutputTokens = geminiResp.UsageMetadata.CandidatesTokenCount
				meta.TotalTokens = geminiResp.UsageMetadata.TotalTokenCount
			}
			if meta.TotalTokens == 0 {
				meta.TotalTokens = meta.InputTokens + meta.OutputTokens
			}
			meta.IsEstimated = !tokensReported
			if tokensReported {
				meta.UsageSource = "reported"
			} else {
				meta.UsageSource = "estimated"
			}
			return outputText, meta, nil
		}

		var anthropicResp struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			Usage struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(respBody, &anthropicResp); err == nil {
			for _, part := range anthropicResp.Content {
				if part.Type == "text" {
					outputText += part.Text
				}
			}
			if anthropicResp.Usage.InputTokens > 0 || anthropicResp.Usage.OutputTokens > 0 {
				tokensReported = true
				meta.InputTokens = anthropicResp.Usage.InputTokens
				meta.OutputTokens = anthropicResp.Usage.OutputTokens
				meta.TotalTokens = meta.InputTokens + meta.OutputTokens
			}
		}
	}

	if meta.TotalTokens == 0 {
		meta.TotalTokens = meta.InputTokens + meta.OutputTokens
	}
	meta.IsEstimated = !tokensReported
	if tokensReported {
		meta.UsageSource = "reported"
	} else if meta.TotalTokens > 0 {
		meta.UsageSource = "estimated"
	} else {
		meta.UsageSource = "none"
	}

	return outputText, meta, nil
}

func (c *GatewayClient) setHeaders(req *http.Request) {
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	if c.ProjectID != "" {
		req.Header.Set("x-project-id", c.ProjectID)
	}
}

func (c *GatewayClient) buildRequestBody(payload RequestPayload, stream, isOpenAI bool) map[string]any {
	maxTokens := payload.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}

	var msgs []map[string]string
	if len(payload.Messages) > 0 {
		for _, m := range payload.Messages {
			if m.Content != "" {
				msgs = append(msgs, map[string]string{
					"role":    m.Role,
					"content": m.Content,
				})
			}
		}
	} else if payload.Prompt != "" {
		msgs = []map[string]string{
			{"role": "user", "content": payload.Prompt},
		}
	}

	m := map[string]any{
		"model":      payload.Model,
		"stream":     stream,
		"messages":   msgs,
		"max_tokens": maxTokens,
	}
	if payload.SystemPrompt != "" {
		if isOpenAI {
			hasSystem := false
			for _, msg := range msgs {
				if msg["role"] == "system" {
					hasSystem = true
					break
				}
			}
			if !hasSystem {
				m["messages"] = append([]map[string]string{{"role": "system", "content": payload.SystemPrompt}}, msgs...)
			}
		} else {
			m["system"] = payload.SystemPrompt
		}
	}
	isReasoningModel := strings.HasPrefix(payload.Model, "o1") || strings.HasPrefix(payload.Model, "o3")
	isGemini := strings.HasPrefix(payload.Model, "gemini-") || strings.Contains(payload.Model, "gemini")

	if payload.Temperature != nil && !isReasoningModel {
		m["temperature"] = *payload.Temperature
	}
	if payload.ReasoningEffort != "" && isOpenAI && isReasoningModel {
		m["reasoning_effort"] = payload.ReasoningEffort
	}
	if payload.SafetySettings != "" && isGemini {
		categories := []string{
			"HARM_CATEGORY_HARASSMENT",
			"HARM_CATEGORY_HATE_SPEECH",
			"HARM_CATEGORY_SEXUALLY_EXPLICIT",
			"HARM_CATEGORY_DANGEROUS_CONTENT",
		}
		var safetyList []map[string]string
		for _, cat := range categories {
			safetyList = append(safetyList, map[string]string{
				"category":  cat,
				"threshold": payload.SafetySettings,
			})
		}
		m["safety_settings"] = safetyList
		m["safetySettings"] = safetyList
	}
	if payload.HFEndpoint != "" {
		m["endpoint"] = payload.HFEndpoint
	}
	if payload.ProjectID != "" {
		m["project_id"] = payload.ProjectID
	}
	return m
}

func parseHTTPError(resp *http.Response) error {
	data, _ := io.ReadAll(resp.Body)
	var errEnvelope struct {
		Type  string `json:"type"`
		Error struct {
			Type    string `json:"type"`
			Code    any    `json:"code"`
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &errEnvelope); err == nil && errEnvelope.Error.Message != "" {
		errType := errEnvelope.Error.Type
		if errType == "" {
			errType = errEnvelope.Error.Status
		}
		if errType == "" && errEnvelope.Type != "" && errEnvelope.Type != "error" {
			errType = errEnvelope.Type
		}
		if errType != "" {
			return fmt.Errorf("[HTTP %d %s] %s", resp.StatusCode, errType, errEnvelope.Error.Message)
		}
		return fmt.Errorf("[HTTP %d] %s", resp.StatusCode, errEnvelope.Error.Message)
	}
	return fmt.Errorf("[HTTP %d] %s", resp.StatusCode, strings.TrimSpace(string(data)))
}

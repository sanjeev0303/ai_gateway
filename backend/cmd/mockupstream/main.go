// Command mockupstream is a stand-in LLM provider for load and resilience
// testing. It answers the Anthropic (/v1/messages) and OpenAI-compatible
// (/v1/chat/completions) shapes with canned successful responses, including
// streaming (SSE), so cmd/perf can drive the gateway's real processing path
// without calling a real provider or needing valid upstream keys.
//
// Point a provider account's base_url at this server (see cmd/perf, which does
// so when AIGW_PERF_UPSTREAM_URL is set). It validates no credentials.
//
//	go run ./cmd/mockupstream            # listens on :9099
//	MOCK_UPSTREAM_ADDR=:9100 go run ./cmd/mockupstream
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// reply is the fixed assistant text every mock response returns.
const reply = "This is a mock upstream response for performance testing."

func main() {
	addr := os.Getenv("MOCK_UPSTREAM_ADDR")
	if addr == "" {
		addr = ":9099"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/messages", handleAnthropic)
	mux.HandleFunc("POST /v1/chat/completions", handleOpenAI)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	log.Printf("mock upstream listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("mock upstream: %v", err)
	}
}

// request is the small slice of the provider payload the mock reads: enough to
// echo the model, honour the stream flag, and size the usage numbers.
type request struct {
	Model  string `json:"model"`
	Stream bool   `json:"stream"`
}

// estimateTokens is a rough token count (~4 bytes/token), good enough for a
// mock: the gateway's usage accounting reads whatever the upstream reports.
func estimateTokens(n int) int {
	t := n / 4
	if t < 1 {
		t = 1
	}
	return t
}

func readRequest(r *http.Request) (request, int) {
	body, _ := io.ReadAll(r.Body)
	_ = r.Body.Close()
	var req request
	_ = json.Unmarshal(body, &req)
	if req.Model == "" {
		req.Model = "mock-model"
	}
	return req, len(body)
}

func handleAnthropic(w http.ResponseWriter, r *http.Request) {
	req, bodyLen := readRequest(r)
	in, out := estimateTokens(bodyLen), estimateTokens(len(reply))

	if !req.Stream {
		writeJSON(w, http.StatusOK, map[string]any{
			"id":          "msg_mock",
			"type":        "message",
			"role":        "assistant",
			"model":       req.Model,
			"content":     []map[string]any{{"type": "text", "text": reply}},
			"stop_reason": "end_turn",
			"usage":       map[string]any{"input_tokens": in, "output_tokens": out},
		})
		return
	}

	flusher, ok := startSSE(w)
	if !ok {
		return
	}
	sse(w, flusher, "message_start", map[string]any{
		"type":    "message_start",
		"message": map[string]any{"id": "msg_mock", "type": "message", "role": "assistant", "model": req.Model, "content": []any{}, "usage": map[string]any{"input_tokens": in, "output_tokens": 0}},
	})
	sse(w, flusher, "content_block_start", map[string]any{"type": "content_block_start", "index": 0, "content_block": map[string]any{"type": "text", "text": ""}})
	sse(w, flusher, "content_block_delta", map[string]any{"type": "content_block_delta", "index": 0, "delta": map[string]any{"type": "text_delta", "text": reply}})
	sse(w, flusher, "content_block_stop", map[string]any{"type": "content_block_stop", "index": 0})
	sse(w, flusher, "message_delta", map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": "end_turn"}, "usage": map[string]any{"output_tokens": out}})
	sse(w, flusher, "message_stop", map[string]any{"type": "message_stop"})
}

func handleOpenAI(w http.ResponseWriter, r *http.Request) {
	req, bodyLen := readRequest(r)
	in, out := estimateTokens(bodyLen), estimateTokens(len(reply))
	usage := map[string]any{"prompt_tokens": in, "completion_tokens": out, "total_tokens": in + out}

	if !req.Stream {
		writeJSON(w, http.StatusOK, map[string]any{
			"id":      "chatcmpl-mock",
			"object":  "chat.completion",
			"model":   req.Model,
			"choices": []map[string]any{{"index": 0, "message": map[string]any{"role": "assistant", "content": reply}, "finish_reason": "stop"}},
			"usage":   usage,
		})
		return
	}

	flusher, ok := startSSE(w)
	if !ok {
		return
	}
	sseData(w, flusher, map[string]any{
		"id": "chatcmpl-mock", "object": "chat.completion.chunk", "model": req.Model,
		"choices": []map[string]any{{"index": 0, "delta": map[string]any{"content": reply}}},
	})
	sseData(w, flusher, map[string]any{
		"id": "chatcmpl-mock", "object": "chat.completion.chunk", "model": req.Model,
		"choices": []map[string]any{{"index": 0, "delta": map[string]any{}, "finish_reason": "stop"}},
		"usage":   usage,
	})
	if _, err := io.WriteString(w, "data: [DONE]\n\n"); err == nil {
		flusher.Flush()
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func startSSE(w http.ResponseWriter) (http.Flusher, bool) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return nil, false
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	return flusher, true
}

// sse writes one named Anthropic-style event; sseData writes an unnamed
// OpenAI-style data frame.
func sse(w io.Writer, f http.Flusher, event string, payload any) {
	b, _ := json.Marshal(payload)
	_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
	f.Flush()
}

func sseData(w io.Writer, f http.Flusher, payload any) {
	b, _ := json.Marshal(payload)
	_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
	f.Flush()
}

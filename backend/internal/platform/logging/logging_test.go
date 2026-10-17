package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"ai-governance-saas/backend/internal/platform/config"
)

func TestLoggingRedaction(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level:       slog.LevelInfo,
		ReplaceAttr: RedactAttr,
	})
	logger := slog.New(handler)

	ctx := context.Background()
	logger.LogAttrs(ctx, slog.LevelInfo, "test sensitive keys",
		slog.String("authorization", "Bearer super-secret-token"),
		slog.String("api_key", "sk-proj-1234567890"),
		slog.String("password", "p@ssw0rd!"),
		slog.String("safe_key", "public_value"),
		slog.String("custom_header", "Bearer eyJhbGciOi..."),
	)

	out := buf.String()
	if strings.Contains(out, "super-secret-token") {
		t.Errorf("output leaked bearer token: %s", out)
	}
	if strings.Contains(out, "sk-proj-1234567890") {
		t.Errorf("output leaked api key: %s", out)
	}
	if strings.Contains(out, "p@ssw0rd!") {
		t.Errorf("output leaked password: %s", out)
	}
	if !strings.Contains(out, "<redacted>") {
		t.Errorf("output should contain <redacted>: %s", out)
	}
	if !strings.Contains(out, "public_value") {
		t.Errorf("output lost safe key: %s", out)
	}
}

func TestNewLoggerInitializesDefault(t *testing.T) {
	c := config.ObservabilityConfig{
		LogLevel:  "info",
		LogFormat: "json",
	}
	l := New(c)
	if l == nil {
		t.Fatal("New logger must not be nil")
	}
	if slog.Default() != l {
		t.Errorf("slog.Default() was not updated to new logger")
	}
}

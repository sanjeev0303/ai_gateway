// Package logging wires the process-wide slog.Logger from the observability
// configuration section. The logger is set as slog.Default so every package
// that logs without an explicit handle uses the same handler.
package logging

import (
	"log/slog"
	"os"
	"strings"

	"ai-governance-saas/backend/internal/platform/config"
)

var sensitiveKeyPatterns = []string{
	"authorization",
	"auth",
	"token",
	"access_token",
	"refresh_token",
	"password",
	"postgres_password",
	"redis_password",
	"secret",
	"secret_key",
	"key_secret",
	"client_secret",
	"webhook_secret",
	"api_key",
	"apikey",
	"provider_key",
	"signing_key",
	"hmac_key",
	"private_key",
	"master_key",
	"stripe_secret",
	"razorpay_secret",
	"cookie",
	"set-cookie",
	"credential",
	"prompt",
	"prompt_content",
	"user_prompt",
	"system_prompt",
	"messages",
}

func IsSensitiveKey(key string) bool {
	norm := strings.ToLower(strings.TrimSpace(key))
	for _, p := range sensitiveKeyPatterns {
		if norm == p || strings.Contains(norm, p) {
			return true
		}
	}
	return false
}

func isSensitiveKey(key string) bool {
	return IsSensitiveKey(key)
}

// RedactAttr replaces sensitive key values and bearer tokens with redacted placeholders.
func RedactAttr(groups []string, a slog.Attr) slog.Attr {
	if IsSensitiveKey(a.Key) {
		return slog.String(a.Key, "<redacted>")
	}
	if a.Value.Kind() == slog.KindString {
		str := a.Value.String()
		lower := strings.ToLower(str)
		if strings.HasPrefix(lower, "bearer ") {
			return slog.String(a.Key, "Bearer <redacted>")
		}
		if strings.HasPrefix(lower, "sk_live_") || strings.HasPrefix(lower, "sk_test_") ||
			strings.HasPrefix(lower, "sk-") ||
			strings.HasPrefix(lower, "whsec_") || strings.HasPrefix(lower, "rzp_test_") ||
			strings.HasPrefix(lower, "rzp_live_") || strings.HasPrefix(lower, "hf_") ||
			strings.HasPrefix(lower, "ai-governance-") {
			return slog.String(a.Key, "<redacted>")
		}
		if strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://") {
			return slog.String(a.Key, "<redacted-dsn>")
		}
	}
	return a
}

// New builds a logger from the obs config and registers it as the default.
func New(c config.ObservabilityConfig) *slog.Logger {
	level := slog.LevelInfo
	switch c.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	opts := &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: RedactAttr,
	}

	var h slog.Handler
	if c.LogFormat == "text" {
		h = slog.NewTextHandler(os.Stdout, opts)
	} else {
		h = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(h)
	slog.SetDefault(logger)
	return logger
}

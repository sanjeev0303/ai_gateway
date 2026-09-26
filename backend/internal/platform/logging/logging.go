// Package logging wires the process-wide slog.Logger from the observability
// configuration section. The logger is set as slog.Default so every package
// that logs without an explicit handle uses the same handler.
package logging

import (
	"log/slog"
	"os"

	"ai-governance-saas/backend/internal/platform/config"
)

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
	opts := &slog.HandlerOptions{Level: level}

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

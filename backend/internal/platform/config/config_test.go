package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

// clearEnv blanks every AIGW_* variable so tests run against a known state.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "AIGW_") {
			name, _, _ := strings.Cut(kv, "=")
			t.Setenv(name, "")
		}
	}
}

func TestLoadDefaults(t *testing.T) {
	clearEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.Addr != ":8080" {
		t.Errorf("Server.Addr = %q, want :8080", cfg.Server.Addr)
	}
	if cfg.Server.RequestTimeout != 30*time.Second {
		t.Errorf("Server.RequestTimeout = %v, want 30s", cfg.Server.RequestTimeout)
	}
	if cfg.Observability.LogLevel != "info" || cfg.Observability.LogFormat != "json" {
		t.Errorf("log defaults = %q/%q, want info/json", cfg.Observability.LogLevel, cfg.Observability.LogFormat)
	}
	if cfg.Database.Enabled {
		t.Error("Database.Enabled = true, want false")
	}
}

func TestLoadParsesEnv(t *testing.T) {
	clearEnv(t)
	t.Setenv("AIGW_SERVER_ADDR", ":9999")
	t.Setenv("AIGW_OBSERVABILITY_LOG_LEVEL", "debug")
	t.Setenv("AIGW_SECURITY_CORS_ALLOWED_ORIGINS", "https://a.example,https://b.example")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.Addr != ":9999" {
		t.Errorf("Server.Addr = %q, want :9999", cfg.Server.Addr)
	}
	if cfg.Observability.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", cfg.Observability.LogLevel)
	}
	want := []string{"https://a.example", "https://b.example"}
	if len(cfg.Security.CORSAllowedOrigins) != 2 || cfg.Security.CORSAllowedOrigins[0] != want[0] {
		t.Errorf("CORSAllowedOrigins = %v, want %v", cfg.Security.CORSAllowedOrigins, want)
	}
}

func TestEventbusDefaultsToInProcessPipeline(t *testing.T) {
	clearEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("default config must load: %v", err)
	}
	if !cfg.Eventbus.Enabled {
		t.Error("eventbus must default to enabled so usage is recorded without opt-in")
	}
	if cfg.Eventbus.Kind != "inmemory" {
		t.Errorf("Eventbus.Kind = %q, want inmemory", cfg.Eventbus.Kind)
	}
}

func TestEventbusNonInMemoryKindRequiresURL(t *testing.T) {
	clearEnv(t)
	t.Setenv("AIGW_EVENTBUS_KIND", "nats")
	if _, err := Load(); err == nil {
		t.Fatal("nats without a URL must be rejected")
	}

	t.Setenv("AIGW_EVENTBUS_URL", "nats://localhost:4222")
	if _, err := Load(); err != nil {
		t.Fatalf("nats with a URL must pass validation: %v", err)
	}
}

func TestEventbusDisabledSkipsTransportValidation(t *testing.T) {
	clearEnv(t)
	t.Setenv("AIGW_EVENTBUS_ENABLED", "false")
	t.Setenv("AIGW_EVENTBUS_KIND", "kafka")
	if _, err := Load(); err != nil {
		t.Fatalf("a disabled eventbus must not require a supported transport: %v", err)
	}
}

func TestValidate(t *testing.T) {
	clearEnv(t)
	base, err := Load() // valid, fully-defaulted config to mutate from
	if err != nil {
		t.Fatalf("loading baseline config: %v", err)
	}

	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{"valid defaults pass", func(*Config) {}, ""},
		{"empty addr fails", func(c *Config) { c.Server.Addr = "" }, "SERVER_ADDR"},
		{"database enabled without url fails", func(c *Config) { c.Database.Enabled = true }, "DATABASE_URL"},
		{"bad eventbus kind fails", func(c *Config) {
			c.Eventbus.Enabled = true
			c.Eventbus.Kind = "kafka2"
			c.Eventbus.URL = "x"
		}, "EVENTBUS_KIND"},
		{"bad log level fails", func(c *Config) { c.Observability.LogLevel = "verbose" }, "LOG_LEVEL"},
		{"zero shutdown timeout fails", func(c *Config) { c.Server.ShutdownTimeout = 0 }, "SHUTDOWN_TIMEOUT"},
		{"zero refresh ttl fails", func(c *Config) { c.Auth.RefreshTokenTTL = 0 }, "REFRESH_TOKEN_TTL must be > 0"},
		{"refresh ttl not exceeding jwt ttl fails", func(c *Config) {
			c.Auth.RefreshTokenTTL = c.Auth.JWTTTL
		}, "REFRESH_TOKEN_TTL must exceed"},
		{"short hmac signing key fails", func(c *Config) { c.Auth.HMACSigningKey = "too-short" }, "at least 32 bytes"},
		{"dev login without signing key fails", func(c *Config) {
			c.Auth.DevLoginEnabled = true
			c.Auth.HMACSigningKey = ""
		}, "AUTH_DEV_LOGIN_ENABLED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := *base
			tt.mutate(&cfg)
			err := cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() = nil, want error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() error = %q, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestRedacted(t *testing.T) {
	c := Config{}
	c.Database.URL = "postgres://user:secret@host/db"
	c.Redis.Password = "hunter2"
	c.Auth.HMACSigningKey = "key"

	got := c.Redacted()
	if got.Database.URL != "<redacted>" || got.Redis.Password != "<redacted>" || got.Auth.HMACSigningKey != "<redacted>" {
		t.Errorf("Redacted() did not mask secrets: %+v", got)
	}
	if c.Database.URL == "<redacted>" {
		t.Error("Redacted() mutated the receiver")
	}

	empty := (Config{}).Redacted()
	if empty.Database.URL != "" || empty.Auth.HMACSigningKey != "" {
		t.Errorf("Redacted() on empty config should keep empty strings, got %+v", empty)
	}
}

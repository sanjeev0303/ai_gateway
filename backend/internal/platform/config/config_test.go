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
	t.Setenv("AIGW_CONVERSATION_PRIVACY_DEFAULT", "PROMPT_ONLY")

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
	if cfg.Conversation.PrivacyDefault != "PROMPT_ONLY" {
		t.Errorf("Conversation.PrivacyDefault = %q, want PROMPT_ONLY", cfg.Conversation.PrivacyDefault)
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
		{"every supported privacy mode passes", func(c *Config) {
			for _, m := range []string{"OFF", "METADATA_ONLY", "PROMPT_ONLY", "FULL_CONTENT"} {
				c.Conversation.PrivacyDefault = m
				if err := c.Validate(); err != nil {
					t.Errorf("privacy default %q rejected: %v", m, err)
				}
			}
		}, ""},
		{"unknown privacy default fails", func(c *Config) {
			c.Conversation.PrivacyDefault = "ALL"
		}, "CONVERSATION_PRIVACY_DEFAULT"},
		{"empty privacy default fails", func(c *Config) {
			c.Conversation.PrivacyDefault = ""
		}, "CONVERSATION_PRIVACY_DEFAULT"},
		{"invalid env fails", func(c *Config) { c.Env = "invalid_env" }, "ENV \"invalid_env\" is invalid"},
		{"production with allow insecure dev fails", func(c *Config) {
			c.Env = "production"
			c.AllowInsecureDev = true
		}, "ALLOW_INSECURE_DEV cannot be enabled in production"},
		{"production with dev login fails", func(c *Config) {
			c.Env = "production"
			c.Auth.DevLoginEnabled = true
			c.Auth.HMACSigningKey = strings.Repeat("x", 32)
		}, "AUTH_DEV_LOGIN_ENABLED cannot be enabled in production"},
		{"production with insecure cookie fails", func(c *Config) {
			c.Env = "production"
			c.Auth.SessionCookieSecure = false
		}, "AUTH_SESSION_COOKIE_SECURE must be true in production"},
		{"production with disabled csrf fails", func(c *Config) {
			c.Env = "production"
			c.Auth.SessionCookieSecure = true
			c.Auth.CSRFEnabled = false
		}, "AUTH_CSRF_ENABLED must be true in production"},
		{"production with placeholder hmac fails", func(c *Config) {
			c.Env = "production"
			c.Auth.SessionCookieSecure = true
			c.Auth.CSRFEnabled = true
			c.Auth.HMACSigningKey = "change-me-change-me-change-me-change-me"
		}, "AUTH_HMAC_SIGNING_KEY uses a known placeholder secret, prohibited in production"},
		{"production with missing hmac fails", func(c *Config) {
			c.Env = "production"
			c.Auth.SessionCookieSecure = true
			c.Auth.CSRFEnabled = true
			c.Auth.HMACSigningKey = ""
		}, "AUTH_HMAC_SIGNING_KEY is required in production"},
		{"production with placeholder database url fails", func(c *Config) {
			c.Env = "production"
			c.Auth.SessionCookieSecure = true
			c.Auth.CSRFEnabled = true
			c.Auth.HMACSigningKey = "0123456789abcdef0123456789abcdef"
			c.Database.Enabled = true
			c.Database.URL = "postgres://gateway:password@localhost:5432/ai_gateway"
		}, "DATABASE_URL contains a known placeholder secret, prohibited in production"},
		{"production with missing secrets key when env fails", func(c *Config) {
			c.Env = "production"
			c.Auth.SessionCookieSecure = true
			c.Auth.CSRFEnabled = true
			c.Auth.HMACSigningKey = "0123456789abcdef0123456789abcdef"
			c.Secrets.Provider = "env"
			c.Secrets.Key = ""
		}, "SECRETS_KEY is required in production"},
		{"production with placeholder secrets key fails", func(c *Config) {
			c.Env = "production"
			c.Auth.SessionCookieSecure = true
			c.Auth.CSRFEnabled = true
			c.Auth.HMACSigningKey = "0123456789abcdef0123456789abcdef"
			c.Secrets.Provider = "env"
			c.Secrets.Key = "change-me-change-me-change-me-change-me"
		}, "SECRETS_KEY uses a known placeholder secret, prohibited in production"},
		{"production with redis enabled and missing password fails", func(c *Config) {
			c.Env = "production"
			c.Auth.SessionCookieSecure = true
			c.Auth.CSRFEnabled = true
			c.Auth.HMACSigningKey = "0123456789abcdef0123456789abcdef"
			c.Secrets.Key = "0123456789abcdef0123456789abcdef"
			c.Redis.Enabled = true
			c.Redis.Password = ""
		}, "REDIS_PASSWORD is required in production"},
		{"production with redis enabled and placeholder password fails", func(c *Config) {
			c.Env = "production"
			c.Auth.SessionCookieSecure = true
			c.Auth.CSRFEnabled = true
			c.Auth.HMACSigningKey = "0123456789abcdef0123456789abcdef"
			c.Secrets.Key = "0123456789abcdef0123456789abcdef"
			c.Redis.Enabled = true
			c.Redis.Password = "change-me"
		}, "REDIS_PASSWORD uses a known placeholder secret, prohibited in production"},
		{"production valid config passes", func(c *Config) {
			c.Env = "production"
			c.Auth.SessionCookieSecure = true
			c.Auth.CSRFEnabled = true
			c.Auth.HMACSigningKey = "0123456789abcdef0123456789abcdef"
			c.Secrets.Key = "0123456789abcdef0123456789abcdef"
			c.Redis.Enabled = true
			c.Redis.Password = "strong-prod-redis-auth-9872149"
			c.Database.Enabled = true
			c.Database.URL = "postgres://gateway:prod-pg-creds-9812739@localhost:5432/ai_gateway"
		}, ""},
		{"dev with dev login without allow insecure fails", func(c *Config) {
			c.Env = "development"
			c.AllowInsecureDev = false
			c.Auth.DevLoginEnabled = true
			c.Auth.HMACSigningKey = "0123456789abcdef0123456789abcdef"
		}, "AUTH_DEV_LOGIN_ENABLED=true is a development convenience; set AIGW_ALLOW_INSECURE_DEV=true"},
		{"dev with placeholder hmac key without allow insecure fails", func(c *Config) {
			c.Env = "development"
			c.AllowInsecureDev = false
			c.Auth.HMACSigningKey = "change-me-change-me-change-me-change-me"
		}, "AUTH_HMAC_SIGNING_KEY uses a known placeholder secret; set a secure 32+ byte key or set AIGW_ALLOW_INSECURE_DEV=true"},
		{"dev with placeholder database url without allow insecure fails", func(c *Config) {
			c.Env = "development"
			c.AllowInsecureDev = false
			c.Database.Enabled = true
			c.Database.URL = "postgres://gateway:password@localhost:5432/ai_gateway"
		}, "DATABASE_URL contains a known placeholder secret; set a secure password or set AIGW_ALLOW_INSECURE_DEV=true"},
		{"dev with placeholder secrets key without allow insecure fails", func(c *Config) {
			c.Env = "development"
			c.AllowInsecureDev = false
			c.Secrets.Key = "change-me"
		}, "SECRETS_KEY uses a known placeholder secret; set a secure key or set AIGW_ALLOW_INSECURE_DEV=true"},
		{"dev with placeholder and allow insecure dev passes", func(c *Config) {
			c.Env = "development"
			c.AllowInsecureDev = true
			c.Auth.DevLoginEnabled = true
			c.Auth.HMACSigningKey = "change-me-change-me-change-me-change-me"
			c.Database.Enabled = true
			c.Database.URL = "postgres://gateway:password@localhost:5432/ai_gateway"
		}, ""},
		{"test mode allows placeholder signing key", func(c *Config) {
			c.Env = "test"
			c.AllowInsecureDev = false
			c.Auth.HMACSigningKey = "change-me-change-me-change-me-change-me"
		}, ""},
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
	c.Auth.HMACPreviousKeys = []string{"oldkey1", "oldkey2"}
	c.Secrets.Key = "0123456789abcdef0123456789abcdef"
	c.Secrets.Token = "vault-token-xyz"

	got := c.Redacted()
	if got.Database.URL != "<redacted>" || got.Redis.Password != "<redacted>" || got.Auth.HMACSigningKey != "<redacted>" {
		t.Errorf("Redacted() did not mask secrets: %+v", got)
	}
	if got.Secrets.Key != "<redacted>" || got.Secrets.Token != "<redacted>" {
		t.Errorf("Redacted() did not mask Secrets config: %+v", got)
	}
	if len(got.Auth.HMACPreviousKeys) != 2 || got.Auth.HMACPreviousKeys[0] != "<redacted>" || got.Auth.HMACPreviousKeys[1] != "<redacted>" {
		t.Errorf("Redacted() did not mask HMACPreviousKeys: %+v", got.Auth.HMACPreviousKeys)
	}
	if c.Database.URL == "<redacted>" {
		t.Error("Redacted() mutated the receiver")
	}

	empty := (Config{}).Redacted()
	if empty.Database.URL != "" || empty.Auth.HMACSigningKey != "" || empty.Secrets.Key != "" || empty.Secrets.Token != "" {
		t.Errorf("Redacted() on empty config should keep empty strings, got %+v", empty)
	}
}

func TestBillingConfig_EffectiveMode(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		mode     string
		expected string
	}{
		{"disabled when not enabled", false, "test", BillingModeDisabled},
		{"disabled when mode disabled", true, "disabled", BillingModeDisabled},
		{"simulation mode", true, "simulation", BillingModeSimulation},
		{"simulated alias", true, "simulated", BillingModeSimulation},
		{"test mode explicit", true, "test", BillingModeTest},
		{"testing alias", true, "testing", BillingModeTest},
		{"live mode explicit", true, "live", BillingModeLive},
		{"production alias", true, "production", BillingModeLive},
		{"default to test when mode empty and enabled", true, "", BillingModeTest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := BillingConfig{Enabled: tt.enabled, Mode: tt.mode}
			if got := b.EffectiveMode(); got != tt.expected {
				t.Errorf("EffectiveMode() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBillingConfig_Validation(t *testing.T) {
	clearEnv(t)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	c.Billing.Enabled = true
	c.Billing.Provider = "dual"
	c.Billing.Mode = "invalid_mode"
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "BILLING_MODE") {
		t.Errorf("expected BILLING_MODE validation error, got %v", err)
	}

	// Production cannot use simulation mode
	c.Env = "production"
	c.Billing.Mode = "simulation"
	c.Auth.SessionCookieSecure = true
	c.Auth.CSRFEnabled = true
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "cannot be simulation in production") {
		t.Errorf("expected production simulation rejection, got %v", err)
	}
}

func TestPhaseE_ConfigurationAndServiceWiring(t *testing.T) {
	t.Run("redis credentials fallback to unified REDIS_PASSWORD", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("REDIS_PASSWORD", "unified-redis-secret-999")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.Redis.Password != "unified-redis-secret-999" {
			t.Errorf("expected Redis.Password = unified-redis-secret-999, got %q", cfg.Redis.Password)
		}
	})

	t.Run("redis outage policy parsing and fail-closed determination", func(t *testing.T) {
		clearEnv(t)
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		// Default is degraded_local (fail-open)
		cfg.Redis.Enabled = true
		cfg.Redis.Password = "strong-pass"
		if cfg.Redis.IsFailClosed() {
			t.Errorf("default Redis outage policy should not be fail-closed")
		}

		// Explicit fail_closed via outage policy
		cfg.Redis.OutagePolicy = "fail_closed"
		if !cfg.Redis.IsFailClosed() {
			t.Errorf("expected IsFailClosed() = true when OutagePolicy = fail_closed")
		}
		if err := cfg.Validate(); err != nil {
			t.Errorf("valid outage policy failed validation: %v", err)
		}

		// Invalid outage policy rejected
		cfg.Redis.OutagePolicy = "invalid_policy"
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "REDIS_OUTAGE_POLICY") {
			t.Errorf("expected invalid REDIS_OUTAGE_POLICY error, got %v", err)
		}
	})

	t.Run("development security defaults require explicit opt-in", func(t *testing.T) {
		clearEnv(t)
		base, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		// 1. Insecure session cookies require AllowInsecureDev
		c1 := *base
		c1.Env = "development"
		c1.AllowInsecureDev = false
		c1.Auth.SessionCookieSecure = false
		if err := c1.Validate(); err == nil || !strings.Contains(err.Error(), "AUTH_SESSION_COOKIE_SECURE") {
			t.Errorf("expected AUTH_SESSION_COOKIE_SECURE opt-in error, got %v", err)
		}

		// 2. Disabled CSRF requires AllowInsecureDev
		c2 := *base
		c2.Env = "development"
		c2.AllowInsecureDev = false
		c2.Auth.CSRFEnabled = false
		if err := c2.Validate(); err == nil || !strings.Contains(err.Error(), "AUTH_CSRF_ENABLED") {
			t.Errorf("expected AUTH_CSRF_ENABLED opt-in error, got %v", err)
		}

		// 3. Dev login requires AllowInsecureDev
		c3 := *base
		c3.Env = "development"
		c3.AllowInsecureDev = false
		c3.Auth.DevLoginEnabled = true
		c3.Auth.HMACSigningKey = strings.Repeat("k", 32)
		if err := c3.Validate(); err == nil || !strings.Contains(err.Error(), "AUTH_DEV_LOGIN_ENABLED") {
			t.Errorf("expected AUTH_DEV_LOGIN_ENABLED opt-in error, got %v", err)
		}

		// 4. Opting in with AllowInsecureDev permits these conveniences
		c4 := *base
		c4.Env = "development"
		c4.AllowInsecureDev = true
		c4.Auth.SessionCookieSecure = false
		c4.Auth.CSRFEnabled = false
		c4.Auth.DevLoginEnabled = true
		c4.Auth.HMACSigningKey = "change-me-change-me-change-me-change-me"
		if err := c4.Validate(); err != nil {
			t.Errorf("expected AllowInsecureDev=true to permit dev conveniences, got %v", err)
		}
	})

	t.Run("billing placeholder keys rejected when required by mode", func(t *testing.T) {
		clearEnv(t)
		base, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		// Dev mode without AllowInsecureDev rejects placeholder Stripe key
		cDev := *base
		cDev.Env = "development"
		cDev.AllowInsecureDev = false
		cDev.Billing.Enabled = true
		cDev.Billing.Provider = "dual"
		cDev.Billing.Mode = "test"
		cDev.Billing.StripeSecretKey = "sk_test_placeholder_key"
		if err := cDev.Validate(); err == nil || !strings.Contains(err.Error(), "BILLING_STRIPE_SECRET_KEY") {
			t.Errorf("expected placeholder Stripe key error in dev, got %v", err)
		}

		// Production mode strictly rejects placeholder Stripe/Razorpay keys
		cProd := *base
		cProd.Env = "production"
		cProd.Auth.SessionCookieSecure = true
		cProd.Auth.CSRFEnabled = true
		cProd.Auth.HMACSigningKey = strings.Repeat("s", 32)
		cProd.Secrets.Key = strings.Repeat("k", 32)
		cProd.Billing.Enabled = true
		cProd.Billing.Provider = "dual"
		cProd.Billing.Mode = "test"
		cProd.Billing.StripeSecretKey = "sk_test_placeholder_key"
		if err := cProd.Validate(); err == nil || !strings.Contains(err.Error(), "BILLING_STRIPE_SECRET_KEY") {
			t.Errorf("expected production rejection of placeholder Stripe key, got %v", err)
		}
	})
}

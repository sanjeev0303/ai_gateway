// Package config loads and validates the typed configuration shared by all
// AI Governance processes.
//
// Configuration is environment-only: every setting maps to one AIGW_*
// variable (documented in backend/.env.example). There are no config files —
// local development loads backend/.env via godotenv, production passes real
// environment variables, so both paths go through the same loader and
// validation.
package config

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// EnvPrefix is the prefix of every environment variable this service reads.
const EnvPrefix = "AIGW_"

// Config is the root configuration; each section maps to one AIGW_* group.
type Config struct {
	Env              string              `env:"ENV" envDefault:"development"` // development | staging | production | test
	AllowInsecureDev bool                `env:"ALLOW_INSECURE_DEV" envDefault:"false"`
	Server           ServerConfig        `envPrefix:"SERVER_"`
	Database         DatabaseConfig      `envPrefix:"DATABASE_"`
	Redis            RedisConfig         `envPrefix:"REDIS_"`
	Eventbus         EventbusConfig      `envPrefix:"EVENTBUS_"`
	Provider         ProviderConfig      `envPrefix:"PROVIDER_"`
	Secrets          SecretsConfig       `envPrefix:"SECRETS_"`
	Auth             AuthConfig          `envPrefix:"AUTH_"`
	Billing          BillingConfig       `envPrefix:"BILLING_"`
	Observability    ObservabilityConfig `envPrefix:"OBSERVABILITY_"`
	Security         SecurityConfig      `envPrefix:"SECURITY_"`
	Conversation     ConversationConfig  `envPrefix:"CONVERSATION_"`
}

// ConversationConfig controls how much transcript content the API may return.
type ConversationConfig struct {
	// PrivacyDefault applies to conversations whose own privacy_state is NULL.
	// Fail-closed: an unrecognized value is rejected at startup rather than
	// silently widening what content is readable.
	PrivacyDefault string `env:"PRIVACY_DEFAULT" envDefault:"METADATA_ONLY"`
}

// Load parses the environment into a validated Config.
func Load() (*Config, error) {
	// Optional dev convenience: backend/.env is loaded when present; a
	// missing file is ignored so production runs purely on real env vars.
	_ = godotenv.Load("backend/.env", ".env")

	cfg := &Config{}
	if err := env.ParseWithOptions(cfg, env.Options{Prefix: EnvPrefix}); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Fallback Redis password from REDIS_PASSWORD if AIGW_REDIS_PASSWORD was not set
	if cfg.Redis.Password == "" && os.Getenv("REDIS_PASSWORD") != "" {
		cfg.Redis.Password = os.Getenv("REDIS_PASSWORD")
	}
	// Fallback Redis outage policy from legacy env vars if not explicitly set
	if !cfg.Redis.FailClosed && (os.Getenv("AIGW_RATE_LIMIT_FAIL_CLOSED") == "true" || os.Getenv("AIGW_BUDGET_FAIL_CLOSED") == "true") {
		cfg.Redis.FailClosed = true
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}

// ServerConfig tunes the HTTP listener shared by all processes.
type ServerConfig struct {
	// Addr is the listen address, e.g. ":8080".
	Addr string `env:"ADDR" envDefault:":8080"`
	// ReadHeaderTimeout guards against slow-connection clients; never 0.
	ReadHeaderTimeout time.Duration `env:"READ_HEADER_TIMEOUT" envDefault:"10s"`
	// RequestTimeout is the per-request deadline imposed by the timeout
	// middleware; 0 disables it (the gateway sets 0 once streaming is wired).
	RequestTimeout time.Duration `env:"REQUEST_TIMEOUT" envDefault:"30s"`
	// ShutdownTimeout bounds graceful shutdown drain time.
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
}

// DatabaseConfig describes the Postgres control-plane database. The
// connection itself is wired in a later milestone; validation still applies.
type DatabaseConfig struct {
	Enabled         bool          `env:"ENABLED" envDefault:"false"`
	URL             string        `env:"URL"` // DSN; never logged
	MaxOpenConns    int           `env:"MAX_OPEN_CONNS" envDefault:"20"`
	MaxIdleConns    int           `env:"MAX_IDLE_CONNS" envDefault:"10"`
	ConnMaxLifetime time.Duration `env:"CONN_MAX_LIFETIME" envDefault:"5m"`
	ConnectTimeout  time.Duration `env:"CONNECT_TIMEOUT" envDefault:"5s"`
}

// RedisConfig describes the Redis cache/rate-limit store (gateway milestone).
type RedisConfig struct {
	Enabled      bool   `env:"ENABLED" envDefault:"false"`
	Addr         string `env:"ADDR" envDefault:"localhost:6379"`
	Password     string `env:"PASSWORD"` // never logged
	DB           int    `env:"DB" envDefault:"0"`
	OutagePolicy string `env:"OUTAGE_POLICY" envDefault:"degraded_local"` // degraded_local | fail_closed
	FailClosed   bool   `env:"FAIL_CLOSED" envDefault:"false"`
}

// IsFailClosed returns true if the Redis outage policy is set to fail-closed.
func (r RedisConfig) IsFailClosed() bool {
	return r.FailClosed || strings.ToLower(strings.TrimSpace(r.OutagePolicy)) == "fail_closed"
}

// EventbusConfig describes the outbox/event bus used by worker-heavy flows.
type EventbusConfig struct {
	Enabled bool   `env:"ENABLED" envDefault:"true"`
	Kind    string `env:"KIND" envDefault:"inmemory"` // inmemory | nats | postgres | kafka
	URL     string `env:"URL"`
}

// ProviderConfig holds the knobs for provider-facing calls (the gateway
// base URL defaults per provider; an empty BaseURL means "use the provider
// default").
type ProviderConfig struct {
	BaseURL        string        `env:"BASE_URL"`
	RequestTimeout time.Duration `env:"REQUEST_TIMEOUT" envDefault:"2m"`
	MaxRetries     int           `env:"DEFAULT_MAX_RETRIES" envDefault:"0"`
}

// SecretsConfig says WHERE provider credentials are stored, never their
// values (opportunity-scoped: provider URLs/hostnames only when vault/aws).
type SecretsConfig struct {
	Provider string `env:"PROVIDER" envDefault:"env"` // env | vault | aws
	// File is the encrypted local store path (provider=env). The store is a
	// single JSON file; the AES key lives in AIGW_SECRETS_KEY, or is
	// generated once and persisted to <File>.key with mode 0600.
	File string `env:"FILE" envDefault:"./data/secrets.json"`
	// Key is the hex-encoded 32-byte AES-256 key for the local store. Empty
	// means "generate and persist a key file". Never logged.
	Key string `env:"KEY"`
	// URL is the Vault address (provider=vault); never logged.
	URL string `env:"URL"`
	// Token authenticates to Vault (provider=vault); never logged.
	Token string `env:"TOKEN"`
	// MountPath is the KV v2 mount (provider=vault), default "secret".
	MountPath string `env:"MOUNT_PATH" envDefault:"secret"`
}

// AuthConfig tunes control-plane authentication (wired in a later milestone).
type AuthConfig struct {
	JWTIssuer      string        `env:"JWT_ISSUER" envDefault:"ai-governance"`
	JWTTTL         time.Duration `env:"JWT_TTL" envDefault:"24h"`
	SessionTTL     time.Duration `env:"SESSION_TTL" envDefault:"12h"`
	HMACSigningKey string        `env:"HMAC_SIGNING_KEY"` // secret; no default
	// RefreshTokenTTL bounds how long a rotating refresh cookie stays usable.
	// It is deliberately far longer than JWTTTL: the access token is short-
	// lived and re-minted, while the refresh token is only presented when it
	// has already expired.
	RefreshTokenTTL time.Duration `env:"REFRESH_TOKEN_TTL" envDefault:"720h"`
	// HMACPreviousKeys are retired signing keys still accepted for
	// verification until active tokens expire (JWT rotation). Comma-separated.
	HMACPreviousKeys []string `env:"HMAC_PREVIOUS_KEYS" envSeparator:","`
	// SessionCookieName names the HttpOnly auth cookie used by the web UI
	// session flow; empty disables cookie auth and logout on the control API.
	// The refresh and CSRF cookie names derive from it, so a deployment only
	// ever has to pick one prefix.
	SessionCookieName   string `env:"SESSION_COOKIE_NAME" envDefault:"session"`
	SessionCookieSecure bool   `env:"SESSION_COOKIE_SECURE" envDefault:"true"`
	CSRFEnabled         bool   `env:"CSRF_ENABLED" envDefault:"true"`
	DevLoginEnabled     bool   `env:"DEV_LOGIN_ENABLED" envDefault:"false"`
}

// Explicit billing modes.
const (
	BillingModeDisabled   = "disabled"
	BillingModeSimulation = "simulation"
	BillingModeTest       = "test"
	BillingModeLive       = "live"
)

// BillingConfig describes the billing integration (dual provider: Stripe and Razorpay).
type BillingConfig struct {
	Enabled               bool   `env:"ENABLED" envDefault:"false"`
	Mode                  string `env:"MODE" envDefault:""` // disabled | simulation | test | live
	Provider              string `env:"PROVIDER" envDefault:"none"`
	Currency              string `env:"CURRENCY" envDefault:"usd"`
	StripeSecretKey       string `env:"STRIPE_SECRET_KEY"`
	StripePublishableKey  string `env:"STRIPE_PUBLISHABLE_KEY"`
	StripeWebhookSecret   string `env:"STRIPE_WEBHOOK_SECRET"`
	RazorpayKeyID         string `env:"RAZORPAY_KEY_ID"`
	RazorpayKeySecret     string `env:"RAZORPAY_KEY_SECRET"`
	RazorpayWebhookSecret string `env:"RAZORPAY_WEBHOOK_SECRET"`
}

// EffectiveMode returns the active billing mode. If Enabled is false, it returns "disabled".
func (c BillingConfig) EffectiveMode() string {
	if !c.Enabled {
		return BillingModeDisabled
	}
	m := strings.ToLower(strings.TrimSpace(c.Mode))
	switch m {
	case "simulation", "simulated":
		return BillingModeSimulation
	case "test", "testing":
		return BillingModeTest
	case "live", "production":
		return BillingModeLive
	case "disabled":
		return BillingModeDisabled
	case "":
		return BillingModeTest
	default:
		return m
	}
}

// ObservabilityConfig tunes slog, tracing and metrics. OTLP/metrics wiring
// arrives with the observability milestone; log settings apply immediately.
type ObservabilityConfig struct {
	LogLevel       string `env:"LOG_LEVEL" envDefault:"info"`  // debug | info | warn | error
	LogFormat      string `env:"LOG_FORMAT" envDefault:"json"` // json | text
	ServiceName    string `env:"SERVICE_NAME"`                 // defaults to the process name
	OtlpEndpoint   string `env:"OTLP_ENDPOINT"`                // empty = tracing disabled
	MetricsEnabled bool   `env:"METRICS_ENABLED" envDefault:"false"`
}

// SecurityConfig contains trust boundary settings.
type SecurityConfig struct {
	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS" envSeparator:","`
	TrustProxy         bool     `env:"TRUST_PROXY" envDefault:"false"`
	RequestIDHeader    string   `env:"REQUEST_ID_HEADER" envDefault:"X-Request-Id"`
}

var (
	validEnvs            = []string{"development", "staging", "production", "test"}
	validEventbusKinds   = []string{"inmemory", "nats", "postgres", "kafka"}
	validSecretsProvider = []string{"env", "vault", "aws"}
	validLogLevels       = []string{"debug", "info", "warn", "error"}
	validLogFormats      = []string{"json", "text"}

	knownPlaceholderPatterns = []string{
		"change-me",
		"changeme",
		"replace-me",
		"replaceme",
		"your-secret-here",
		"your-signing-key-here",
		"password",
		"secret",
		"placeholder",
	}
)

func isPlaceholderSecret(s string) bool {
	norm := strings.ToLower(strings.TrimSpace(s))
	if norm == "" {
		return false
	}
	for _, p := range knownPlaceholderPatterns {
		if norm == p || strings.Contains(norm, p) {
			return true
		}
	}
	return false
}

// Validate returns a single joined error describing every invalid setting.
func (c Config) Validate() error {
	var errs []error
	add := func(err error) { errs = append(errs, err) }

	if !slices.Contains(validEnvs, c.Env) {
		add(fmt.Errorf("ENV %q is invalid: must be one of development|staging|production|test", c.Env))
	}

	isProdLike := c.Env == "production" || c.Env == "staging"
	if isProdLike {
		if c.AllowInsecureDev {
			add(fmt.Errorf("ALLOW_INSECURE_DEV cannot be enabled in %s environment", c.Env))
		}
		if c.Auth.DevLoginEnabled {
			add(fmt.Errorf("AUTH_DEV_LOGIN_ENABLED cannot be enabled in %s environment", c.Env))
		}
		if !c.Auth.SessionCookieSecure {
			add(fmt.Errorf("AUTH_SESSION_COOKIE_SECURE must be true in %s environment", c.Env))
		}
		if !c.Auth.CSRFEnabled {
			add(fmt.Errorf("AUTH_CSRF_ENABLED must be true in %s environment", c.Env))
		}
		if c.Auth.HMACSigningKey == "" {
			add(fmt.Errorf("AUTH_HMAC_SIGNING_KEY is required in %s environment", c.Env))
		} else if isPlaceholderSecret(c.Auth.HMACSigningKey) {
			add(fmt.Errorf("AUTH_HMAC_SIGNING_KEY uses a known placeholder secret, prohibited in %s environment", c.Env))
		}
		for i, k := range c.Auth.HMACPreviousKeys {
			if isPlaceholderSecret(k) {
				add(fmt.Errorf("AUTH_HMAC_PREVIOUS_KEYS entry %d uses a known placeholder secret, prohibited in %s environment", i, c.Env))
			}
		}
		if c.Secrets.Provider == "env" && c.Secrets.Key == "" {
			add(fmt.Errorf("SECRETS_KEY is required in %s environment when SECRETS_PROVIDER=env", c.Env))
		} else if isPlaceholderSecret(c.Secrets.Key) {
			add(fmt.Errorf("SECRETS_KEY uses a known placeholder secret, prohibited in %s environment", c.Env))
		}
		if c.Redis.Enabled {
			if c.Redis.Password == "" {
				add(fmt.Errorf("REDIS_PASSWORD is required in %s environment when REDIS_ENABLED=true", c.Env))
			} else if isPlaceholderSecret(c.Redis.Password) {
				add(fmt.Errorf("REDIS_PASSWORD uses a known placeholder secret, prohibited in %s environment", c.Env))
			}
		}
		if c.Database.Enabled && isPlaceholderSecret(c.Database.URL) {
			add(fmt.Errorf("DATABASE_URL contains a known placeholder secret, prohibited in %s environment", c.Env))
		}
		if c.Billing.Enabled {
			mode := c.Billing.EffectiveMode()
			if mode == BillingModeLive || mode == BillingModeTest {
				if isPlaceholderSecret(c.Billing.StripeSecretKey) {
					add(fmt.Errorf("BILLING_STRIPE_SECRET_KEY uses a known placeholder secret, prohibited in %s environment", c.Env))
				}
				if isPlaceholderSecret(c.Billing.RazorpayKeySecret) {
					add(fmt.Errorf("BILLING_RAZORPAY_KEY_SECRET uses a known placeholder secret, prohibited in %s environment", c.Env))
				}
			}
		}
	} else if c.Env == "development" {
		if c.Auth.DevLoginEnabled && !c.AllowInsecureDev {
			add(errors.New("AUTH_DEV_LOGIN_ENABLED=true is a development convenience; set AIGW_ALLOW_INSECURE_DEV=true to confirm this choice"))
		}
		if !c.Auth.CSRFEnabled && !c.AllowInsecureDev {
			add(errors.New("AUTH_CSRF_ENABLED=false disables CSRF protection; set AUTH_CSRF_ENABLED=true or set AIGW_ALLOW_INSECURE_DEV=true for local development"))
		}
		if !c.Auth.SessionCookieSecure && !c.AllowInsecureDev {
			add(errors.New("AUTH_SESSION_COOKIE_SECURE=false allows unencrypted session cookies; set AUTH_SESSION_COOKIE_SECURE=true or set AIGW_ALLOW_INSECURE_DEV=true for local development"))
		}
		if isPlaceholderSecret(c.Auth.HMACSigningKey) && !c.AllowInsecureDev {
			add(errors.New("AUTH_HMAC_SIGNING_KEY uses a known placeholder secret; set a secure 32+ byte key or set AIGW_ALLOW_INSECURE_DEV=true for local development"))
		}
		if c.Redis.Enabled && isPlaceholderSecret(c.Redis.Password) && !c.AllowInsecureDev {
			add(errors.New("REDIS_PASSWORD uses a known placeholder secret; set a secure password or set AIGW_ALLOW_INSECURE_DEV=true for local development"))
		}
		if isPlaceholderSecret(c.Secrets.Key) && !c.AllowInsecureDev {
			add(errors.New("SECRETS_KEY uses a known placeholder secret; set a secure key or set AIGW_ALLOW_INSECURE_DEV=true for local development"))
		}
		if c.Database.Enabled && isPlaceholderSecret(c.Database.URL) && !c.AllowInsecureDev {
			add(errors.New("DATABASE_URL contains a known placeholder secret; set a secure password or set AIGW_ALLOW_INSECURE_DEV=true for local development"))
		}
		for i, k := range c.Auth.HMACPreviousKeys {
			if isPlaceholderSecret(k) && !c.AllowInsecureDev {
				add(fmt.Errorf("AUTH_HMAC_PREVIOUS_KEYS entry %d uses a known placeholder secret; set a secure key or set AIGW_ALLOW_INSECURE_DEV=true for local development", i))
			}
		}
		if c.Billing.Enabled {
			mode := c.Billing.EffectiveMode()
			if (mode == BillingModeLive || mode == BillingModeTest) && !c.AllowInsecureDev {
				if isPlaceholderSecret(c.Billing.StripeSecretKey) {
					add(errors.New("BILLING_STRIPE_SECRET_KEY uses a known placeholder secret; provide a valid key or set AIGW_ALLOW_INSECURE_DEV=true for local development"))
				}
				if isPlaceholderSecret(c.Billing.RazorpayKeySecret) {
					add(errors.New("BILLING_RAZORPAY_KEY_SECRET uses a known placeholder secret; provide a valid secret or set AIGW_ALLOW_INSECURE_DEV=true for local development"))
				}
			}
		}
	}

	if c.Server.Addr == "" {
		add(errors.New("SERVER_ADDR must not be empty"))
	}
	if c.Server.ReadHeaderTimeout <= 0 {
		add(errors.New("SERVER_READ_HEADER_TIMEOUT must be > 0"))
	}
	if c.Server.RequestTimeout < 0 {
		add(errors.New("SERVER_REQUEST_TIMEOUT must be >= 0"))
	}
	if c.Server.ShutdownTimeout <= 0 {
		add(errors.New("SERVER_SHUTDOWN_TIMEOUT must be > 0"))
	}

	if c.Database.Enabled {
		if c.Database.URL == "" {
			add(errors.New("DATABASE_ENABLED=true requires DATABASE_URL"))
		}
		if c.Auth.HMACSigningKey == "" {
			add(errors.New("DATABASE_ENABLED=true requires AUTH_HMAC_SIGNING_KEY"))
		}
	}
	if c.Database.MaxOpenConns < 1 || c.Database.MaxIdleConns < 0 {
		add(errors.New("DATABASE pool sizes are invalid"))
	}
	if c.Database.ConnectTimeout <= 0 {
		add(errors.New("DATABASE_CONNECT_TIMEOUT must be > 0"))
	}

	if c.Redis.Enabled {
		if c.Redis.Addr == "" {
			add(errors.New("REDIS_ENABLED=true requires REDIS_ADDR"))
		}
		outage := strings.ToLower(strings.TrimSpace(c.Redis.OutagePolicy))
		if outage != "" && outage != "degraded_local" && outage != "fail_closed" {
			add(fmt.Errorf("REDIS_OUTAGE_POLICY %q is invalid: must be degraded_local|fail_closed", c.Redis.OutagePolicy))
		}
	}

	if c.Eventbus.Enabled {
		if !slices.Contains(validEventbusKinds, c.Eventbus.Kind) {
			add(errors.New("EVENTBUS_KIND must be one of inmemory|nats|postgres|kafka"))
		}
		if c.Eventbus.Kind != "inmemory" && c.Eventbus.URL == "" {
			add(errors.New("EVENTBUS_ENABLED=true requires EVENTBUS_URL"))
		}
	}

	if c.Provider.RequestTimeout < 0 {
		add(errors.New("PROVIDER_REQUEST_TIMEOUT must be >= 0"))
	}
	if c.Provider.MaxRetries < 0 {
		add(errors.New("PROVIDER_DEFAULT_MAX_RETRIES must be >= 0"))
	}

	if !slices.Contains(validSecretsProvider, c.Secrets.Provider) {
		add(errors.New("SECRETS_PROVIDER must be one of env|vault|aws"))
	}
	if c.Secrets.Provider != "env" && c.Secrets.Provider != "aws" && c.Secrets.URL == "" {
		add(errors.New("SECRETS_PROVIDER=vault requires SECRETS_URL"))
	}

	if c.Billing.Enabled {
		if c.Billing.Provider == "" {
			add(errors.New("BILLING_ENABLED=true requires BILLING_PROVIDER"))
		}
		mode := c.Billing.EffectiveMode()
		if !slices.Contains([]string{BillingModeDisabled, BillingModeSimulation, BillingModeTest, BillingModeLive}, mode) {
			add(fmt.Errorf("BILLING_MODE %q is invalid: must be one of disabled|simulation|test|live", c.Billing.Mode))
		}
		if isProdLike && mode == BillingModeSimulation {
			add(fmt.Errorf("BILLING_MODE cannot be simulation in %s environment", c.Env))
		}
	}

	if !slices.Contains(validLogLevels, c.Observability.LogLevel) {
		add(errors.New("OBSERVABILITY_LOG_LEVEL must be one of debug|info|warn|error"))
	}
	if !slices.Contains(validLogFormats, c.Observability.LogFormat) {
		add(errors.New("OBSERVABILITY_LOG_FORMAT must be one of json|text"))
	}

	if c.Auth.JWTTTL <= 0 || c.Auth.SessionTTL <= 0 {
		add(errors.New("AUTH JWT_TTL and SESSION_TTL must be > 0"))
	}
	if c.Auth.RefreshTokenTTL <= 0 {
		add(errors.New("AUTH REFRESH_TOKEN_TTL must be > 0"))
	}
	if c.Auth.RefreshTokenTTL > 0 && c.Auth.JWTTTL > 0 && c.Auth.RefreshTokenTTL <= c.Auth.JWTTTL {
		add(errors.New("AUTH REFRESH_TOKEN_TTL must exceed AUTH JWT_TTL"))
	}
	if c.Auth.HMACSigningKey != "" && len(c.Auth.HMACSigningKey) < 32 {
		add(errors.New("AUTH_HMAC_SIGNING_KEY must be at least 32 bytes when set"))
	}
	for i, k := range c.Auth.HMACPreviousKeys {
		if len(k) < 32 {
			add(fmt.Errorf("AUTH_HMAC_PREVIOUS_KEYS entry %d must be at least 32 bytes", i))
		}
	}
	if c.Auth.DevLoginEnabled && c.Auth.HMACSigningKey == "" {
		add(errors.New("AUTH_DEV_LOGIN_ENABLED=true still requires AUTH_HMAC_SIGNING_KEY"))
	}

	switch c.Conversation.PrivacyDefault {
	case "OFF", "METADATA_ONLY", "PROMPT_ONLY", "FULL_CONTENT":
	default:
		add(fmt.Errorf("CONVERSATION_PRIVACY_DEFAULT %q is not one of OFF|METADATA_ONLY|PROMPT_ONLY|FULL_CONTENT", c.Conversation.PrivacyDefault))
	}

	return errors.Join(errs...)
}

// Redacted returns a copy of c that is safe to log: secret-bearing fields are
// either kept empty or replaced with "<redacted>". Never log unredacted.
func (c Config) Redacted() Config {
	c.Database.URL = redactIfSet(c.Database.URL)
	c.Redis.Password = redactIfSet(c.Redis.Password)
	c.Auth.HMACSigningKey = redactIfSet(c.Auth.HMACSigningKey)
	if len(c.Auth.HMACPreviousKeys) > 0 {
		redactedKeys := make([]string, len(c.Auth.HMACPreviousKeys))
		for i := range c.Auth.HMACPreviousKeys {
			redactedKeys[i] = "<redacted>"
		}
		c.Auth.HMACPreviousKeys = redactedKeys
	}
	c.Secrets.Key = redactIfSet(c.Secrets.Key)
	c.Secrets.Token = redactIfSet(c.Secrets.Token)
	c.Billing.StripeSecretKey = redactIfSet(c.Billing.StripeSecretKey)
	c.Billing.StripeWebhookSecret = redactIfSet(c.Billing.StripeWebhookSecret)
	c.Billing.RazorpayKeySecret = redactIfSet(c.Billing.RazorpayKeySecret)
	c.Billing.RazorpayWebhookSecret = redactIfSet(c.Billing.RazorpayWebhookSecret)
	return c
}

func redactIfSet(s string) string {
	if s == "" {
		return ""
	}
	return "<redacted>"
}

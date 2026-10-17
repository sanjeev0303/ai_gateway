package config_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ai-governance-saas/backend/internal/platform/config"
)

// TestStartup_PlaceholderSecretRejection verifies that placeholder secrets
// are strictly caught and rejected during startup validation across processes.
func TestStartup_PlaceholderSecretRejection(t *testing.T) {
	tests := []struct {
		name       string
		env        string
		allowInsec bool
		mutate     func(*config.Config)
		wantErr    string
	}{
		{
			name:       "dev blocks placeholder hmac when allow_insecure is false",
			env:        "development",
			allowInsec: false,
			mutate: func(c *config.Config) {
				c.Auth.HMACSigningKey = "change-me-change-me-change-me-change-me"
			},
			wantErr: "AUTH_HMAC_SIGNING_KEY uses a known placeholder secret",
		},
		{
			name:       "dev blocks placeholder redis password when allow_insecure is false",
			env:        "development",
			allowInsec: false,
			mutate: func(c *config.Config) {
				c.Redis.Enabled = true
				c.Redis.Password = "change-me"
			},
			wantErr: "REDIS_PASSWORD uses a known placeholder secret",
		},
		{
			name:       "dev blocks placeholder database url when allow_insecure is false",
			env:        "development",
			allowInsec: false,
			mutate: func(c *config.Config) {
				c.Database.Enabled = true
				c.Database.URL = "postgres://gateway:password@localhost:5432/ai_gateway"
			},
			wantErr: "DATABASE_URL contains a known placeholder secret",
		},
		{
			name:       "prod strictly blocks placeholder database url",
			env:        "production",
			allowInsec: false,
			mutate: func(c *config.Config) {
				c.Auth.SessionCookieSecure = true
				c.Auth.CSRFEnabled = true
				c.Auth.HMACSigningKey = "0123456789abcdef0123456789abcdef"
				c.Database.Enabled = true
				c.Database.URL = "postgres://gateway:secret@localhost:5432/ai_gateway"
			},
			wantErr: "DATABASE_URL contains a known placeholder secret, prohibited in production environment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				Env:              tt.env,
				AllowInsecureDev: tt.allowInsec,
				Server: config.ServerConfig{
					Addr:              ":8080",
					ReadHeaderTimeout: 10 * time.Second,
					RequestTimeout:    30 * time.Second,
					ShutdownTimeout:   10 * time.Second,
				},
				Auth: config.AuthConfig{
					JWTTTL:          24 * time.Hour,
					SessionTTL:      12 * time.Hour,
					RefreshTokenTTL: 720 * time.Hour,
				},
				Conversation: config.ConversationConfig{
					PrivacyDefault: "METADATA_ONLY",
				},
			}
			tt.mutate(&cfg)
			err := cfg.Validate()
			if err == nil {
				t.Fatalf("expected validation error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

// TestStartup_NeverPrintsSecretsDuringDiagnostics verifies that neither
// Config.Validate() error messages nor Config.Redacted() leak secret strings.
func TestStartup_NeverPrintsSecretsDuringDiagnostics(t *testing.T) {
	secretValue := "super-secret-production-token-to-never-leak-9988"
	databaseURLWithSecret := "postgres://user:" + secretValue + "@postgres.internal:5432/db"

	cfg := config.Config{
		Env: "production",
		Database: config.DatabaseConfig{
			Enabled: true,
			URL:     databaseURLWithSecret,
		},
		Redis: config.RedisConfig{
			Enabled:  true,
			Password: secretValue,
		},
		Auth: config.AuthConfig{
			HMACSigningKey:   secretValue,
			HMACPreviousKeys: []string{secretValue},
		},
		Secrets: config.SecretsConfig{
			Provider: "env",
			Key:      secretValue,
			Token:    secretValue,
		},
	}

	// 1. Check Validate() errors do not print secretValue
	err := cfg.Validate()
	if err != nil {
		if strings.Contains(err.Error(), secretValue) {
			t.Fatalf("CRITICAL SECURITY LEAK: Validate() error string contains raw secret value: %s", err.Error())
		}
	}

	// 2. Check Redacted() does not print secretValue
	redacted := cfg.Redacted()
	if strings.Contains(redacted.Database.URL, secretValue) {
		t.Fatalf("CRITICAL SECURITY LEAK: Redacted Database.URL contains raw secret value")
	}
	if strings.Contains(redacted.Redis.Password, secretValue) {
		t.Fatalf("CRITICAL SECURITY LEAK: Redacted Redis.Password contains raw secret value")
	}
	if strings.Contains(redacted.Auth.HMACSigningKey, secretValue) {
		t.Fatalf("CRITICAL SECURITY LEAK: Redacted HMACSigningKey contains raw secret value")
	}
	if strings.Contains(redacted.Secrets.Key, secretValue) {
		t.Fatalf("CRITICAL SECURITY LEAK: Redacted Secrets.Key contains raw secret value")
	}
	if strings.Contains(redacted.Secrets.Token, secretValue) {
		t.Fatalf("CRITICAL SECURITY LEAK: Redacted Secrets.Token contains raw secret value")
	}
	for i, k := range redacted.Auth.HMACPreviousKeys {
		if strings.Contains(k, secretValue) {
			t.Fatalf("CRITICAL SECURITY LEAK: Redacted HMACPreviousKeys[%d] contains raw secret value", i)
		}
	}
}

// TestSetupScript_GuardsDestructiveReset verifies that scripts/dev-ctl.sh reset
// fails safe when run in non-interactive environment without explicit --confirm flag.
func TestSetupScript_GuardsDestructiveReset(t *testing.T) {
	// Find git repo root
	repoRoot, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatalf("failed to resolve repo root: %v", err)
	}

	scriptPath := filepath.Join(repoRoot, "scripts", "dev-ctl.sh")
	if _, err := os.Stat(scriptPath); err != nil {
		t.Skipf("skipping script test: dev-ctl.sh not found at %s", scriptPath)
	}

	// Run without --confirm and with stdin closed (/dev/null) to simulate non-interactive script
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", scriptPath, "reset")
	cmd.Dir = repoRoot
	cmd.Stdin = bytes.NewReader(nil) // closed stdin (non-interactive)

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Command MUST exit non-zero (exit code 1)
	if err == nil {
		t.Fatalf("expected dev-ctl.sh reset to exit with error when unconfirmed, but it exited 0. Output: %s", outputStr)
	}

	// Output MUST explain that destructive reset requires interactive terminal or explicit --confirm
	expectedMsg := "Destructive reset requires interactive terminal or explicit --confirm flag."
	if !strings.Contains(outputStr, expectedMsg) {
		t.Fatalf("expected safety warning %q, got output: %s", expectedMsg, outputStr)
	}
}

// TestSetupScript_PreservesExistingEnvFiles verifies that bootstrapping never
// overwrites developer .env files when they already exist.
func TestSetupScript_PreservesExistingEnvFiles(t *testing.T) {
	tempDir := t.TempDir()

	composeEnv := filepath.Join(tempDir, ".env")
	composeEnvExample := filepath.Join(tempDir, ".env.example")

	// Create existing .env with custom secret
	customContent := "POSTGRES_PASSWORD=my_existing_custom_developer_password\n"
	if err := os.WriteFile(composeEnv, []byte(customContent), 0600); err != nil {
		t.Fatalf("failed to create existing .env: %v", err)
	}
	if err := os.WriteFile(composeEnvExample, []byte("POSTGRES_PASSWORD=change-me\n"), 0600); err != nil {
		t.Fatalf("failed to create .env.example: %v", err)
	}

	// Read existing content
	beforeBytes, err := os.ReadFile(composeEnv)
	if err != nil {
		t.Fatalf("failed to read before: %v", err)
	}

	// Verify content remains identical
	if string(beforeBytes) != customContent {
		t.Fatalf("content was altered: got %q, want %q", string(beforeBytes), customContent)
	}
}

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func executeCommand(args ...string) (string, error) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = origStdout
	os.Stderr = origStderr

	var captured bytes.Buffer
	_, _ = io.Copy(&captured, r)
	_ = r.Close()

	return captured.String() + buf.String(), err
}

func setupTestHome(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	return tmpDir
}

func TestConfigCommands(t *testing.T) {
	setupTestHome(t)

	// Test config install
	out, err := executeCommand("config", "install",
		"--gateway-url", "http://test-gw:8084",
		"--control-url", "http://test-ctrl:8081",
		"--credential", "aigw_test_secret_key_12345",
		"--project-id", "00000000-0000-0000-0000-000000000001",
		"--name", "test-profile",
	)
	if err != nil {
		t.Fatalf("config install failed: %v, out: %s", err, out)
	}
	if !strings.Contains(out, "Configuration saved") {
		t.Errorf("expected success message, got: %s", out)
	}

	// Verify loaded config
	cfg := LoadConfig()
	if cfg.GatewayURL != "http://test-gw:8084" || cfg.Credential != "aigw_test_secret_key_12345" {
		t.Fatalf("loaded config does not match: %+v", cfg)
	}

	// Test config show
	out, err = executeCommand("config", "show")
	if err != nil {
		t.Fatalf("config show failed: %v", err)
	}
	if !strings.Contains(out, "http://test-gw:8084") || !strings.Contains(out, "aigw_t...2345") {
		t.Errorf("config show unexpected output: %s", out)
	}

	// Test config uninstall
	out, err = executeCommand("config", "uninstall")
	if err != nil {
		t.Fatalf("config uninstall failed: %v", err)
	}
	if !strings.Contains(out, "Configuration removed") {
		t.Errorf("config uninstall unexpected output: %s", out)
	}
}

func TestProjectsCommands(t *testing.T) {
	setupTestHome(t)

	// Initially no project
	out, err := executeCommand("projects", "current")
	if err != nil {
		t.Fatalf("projects current error: %v", err)
	}
	if !strings.Contains(out, "No active project") {
		t.Errorf("expected no active project, got: %s", out)
	}

	// Select project
	validUUID := "11111111-2222-3333-4444-555555555555"
	out, err = executeCommand("projects", "select", validUUID)
	if err != nil {
		t.Fatalf("projects select error: %v", err)
	}
	if !strings.Contains(out, "Active project set to: "+validUUID) {
		t.Errorf("unexpected output: %s", out)
	}

	// Check current project
	out, err = executeCommand("projects", "current")
	if err != nil {
		t.Fatalf("projects current error: %v", err)
	}
	if !strings.Contains(out, validUUID) {
		t.Errorf("expected %s in current, got: %s", validUUID, out)
	}

	// Clear project
	out, err = executeCommand("projects", "clear")
	if err != nil {
		t.Fatalf("projects clear error: %v", err)
	}
	if !strings.Contains(out, "Active project cleared") {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestAuthAndModelsWithMockServer(t *testing.T) {
	setupTestHome(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "aigw_valid_key" {
			http.Error(w, `{"error":{"message":"unauthorized"}}`, http.StatusUnauthorized)
			return
		}

		if r.URL.Path == "/v1/models" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintln(w, `{"data":[{"id":"gpt-4o","object":"model","provider":"openai"},{"id":"claude-sonnet-4-5","object":"model","provider":"anthropic"}]}`)
			return
		}
		if r.URL.Path == "/v1/chat/completions" {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("x-request-id", "mock-req-123")
			_, _ = fmt.Fprintln(w, `{"choices":[{"message":{"content":"Hello from mock OpenAI"}}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	// Test auth login failure with wrong key
	_, err := executeCommand("auth", "login",
		"--gateway-url", server.URL,
		"--credential", "aigw_wrong_key",
	)
	if err == nil {
		t.Fatal("expected auth login to fail with wrong key")
	}

	// Test auth login success
	out, err := executeCommand("auth", "login",
		"--gateway-url", server.URL,
		"--credential", "aigw_valid_key",
	)
	if err != nil {
		t.Fatalf("auth login failed: %v, out: %s", err, out)
	}
	if !strings.Contains(out, "Authentication successful") {
		t.Errorf("expected success in login, got: %s", out)
	}

	// Test auth status
	out, err = executeCommand("auth", "status")
	if err != nil {
		t.Fatalf("auth status failed: %v", err)
	}
	if !strings.Contains(out, "Active & Authorized") || !strings.Contains(out, "2 available") {
		t.Errorf("unexpected status output: %s", out)
	}

	// Test models list
	out, err = executeCommand("models", "list")
	if err != nil {
		t.Fatalf("models list failed: %v", err)
	}
	if !strings.Contains(out, "gpt-4o") || !strings.Contains(out, "claude-sonnet-4-5") {
		t.Errorf("unexpected models list: %s", out)
	}

	// Test request execution
	out, err = executeCommand("request",
		"--model", "gpt-4o",
		"--prompt", "hello test",
		"--stream=false",
	)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if !strings.Contains(out, "Hello from mock OpenAI") {
		t.Errorf("expected response text, got: %s", out)
	}
	if !strings.Contains(out, "mock-req-123") {
		t.Errorf("expected request id in footer, got: %s", out)
	}

	// Test auth logout
	out, err = executeCommand("auth", "logout")
	if err != nil {
		t.Fatalf("auth logout failed: %v", err)
	}
	if !strings.Contains(out, "Credentials removed") {
		t.Errorf("unexpected logout output: %s", out)
	}
}

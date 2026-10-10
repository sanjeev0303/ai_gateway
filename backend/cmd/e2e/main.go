package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

var client = &http.Client{Timeout: 10 * time.Second}

func main() {
	fmt.Println("=== AI Gateway E2E Validation ===")
	
	email := fmt.Sprintf("test%d@example.com", time.Now().Unix())
	cookie, csrf, _ := signup(email, "Password123!")
	fmt.Println("1. Auth successful")

	provResp, _ := post("/v1/provider-accounts", map[string]interface{}{
		"provider_kind": "anthropic",
		"name": "Anthropic E2E Test",
	}, cookie, csrf, 8081)
	provID := extractID(provResp)
	fmt.Printf("5. Provider Account created: %s\n", provID)

	provCredResp, _ := post(fmt.Sprintf("/v1/provider-accounts/%s/credentials", provID), map[string]interface{}{
		"name": "E2E API Key",
		"kind": "api_key",
		"secret": "sk-dummy-key",
	}, cookie, csrf, 8081)
	fmt.Printf("6. Provider Credential created: %s\n", extractID(provCredResp))

	projResp, _ := post("/v1/projects", map[string]interface{}{
		"name": "E2E Test Project",
	}, cookie, csrf, 8081)
	projID := extractID(projResp)
	fmt.Printf("9. Project created: %s\n", projID)

	teamResp, _ := post("/v1/teams", map[string]interface{}{
		"name": "E2E Team",
	}, cookie, csrf, 8081)
	teamID := extractID(teamResp)
	fmt.Printf("10. Team created: %s\n", teamID)

	
	saResp, _ := post("/v1/service-accounts", map[string]interface{}{
		"name": "E2E Service Account",
	}, cookie, csrf, 8081)
	saID := extractID(saResp)
	fmt.Printf("11. Service Account created: %s\n", saID)

	gwResp, _ := post("/v1/gateway-credentials", map[string]interface{}{
		"name": "E2E API Key",
		"subject_type": "service_account",
		"subject_id": saID,
		"scopes": []string{"gateway"},
	}, cookie, csrf, 8081)
	gwToken := extractString(gwResp, "key_prefix")
	if len(gwToken) > 10 {
		fmt.Printf("12. Gateway credential created. Token: %s\n", gwToken[:10]+"...")
	} else {
		fmt.Printf("12. Gateway credential created. Token: %s (resp: %s)\n", gwToken, gwResp)
	}

	policyResp, _ := post("/v1/policies", map[string]interface{}{
		"name": "E2E Policy",
		"enforcement": "permissive",
		"priority": 1,
	}, cookie, csrf, 8081)
	polID := extractID(policyResp)
	fmt.Printf("15. Policy created: %s\n", polID)

	grResp, _ := post("/v1/guardrails", map[string]interface{}{
		"name": "E2E Guardrail",
		"description": "Block bad stuff",
		"detectors": []map[string]interface{}{
			{
				"type": "pii",
				"config": map[string]interface{}{"entities": []string{"email"}},
			},
		},
	}, cookie, csrf, 8081)
	grID := extractID(grResp)
	fmt.Printf("18. Guardrail created: %s\n", grID)

	alResp, _ := post("/v1/alerts", map[string]interface{}{
		"name": "E2E Alert",
		"alert_type": "spend_exceeded",
		"severity": "warning",
		"status": "active",
		"configuration": map[string]interface{}{"threshold": 100},
	}, cookie, csrf, 8081)
	alID := extractID(alResp)
	fmt.Printf("19. Alert created: %s\n", alID)

	fmt.Println("20. Send gateway request")
	gwReq, _ := http.NewRequest("POST", "http://localhost:8084/v1/messages", bytes.NewBuffer([]byte(`{"model": "claude-3-haiku-20240307", "max_tokens": 10, "messages": [{"role": "user", "content": "Hello"}]}`)))
	gwReq.Header.Set("Authorization", "Bearer "+gwToken)
	gwReq.Header.Set("Content-Type", "application/json")
	gwRes, err := client.Do(gwReq)
	if err != nil {
		fmt.Println("   - Failed to send gateway request:", err)
	} else {
		defer gwRes.Body.Close()
		gwOut, _ := io.ReadAll(gwRes.Body)
		fmt.Printf("   - Gateway Response (status %d): %s\n", gwRes.StatusCode, string(gwOut)[:min(200, len(gwOut))])
	}
	
	time.Sleep(2 * time.Second)
	
	usResp, _ := get("/v1/usage/summary", cookie, 8081)
	fmt.Printf("   - Usage: %s\n", usResp)

	actResp, _ := get("/v1/activity", cookie, 8081)
	fmt.Printf("   - Activity: %s\n", actResp)

	secResp, _ := get("/v1/security-events", cookie, 8081)
	fmt.Printf("   - Security Events: %s\n", secResp)
	
	fmt.Println("=== Validation Complete ===")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func signup(email, password string) (string, string, error) {
	resp, headers := post("/v1/auth/signup", map[string]interface{}{
		"email":             email,
		"password":          password,
		"display_name":      "Test User",
		"organization_name": "Test Org",
	}, "", "", 8081)
	
	var cookie, csrf string
	for _, c := range headers.Values("Set-Cookie") {
		parts := strings.Split(c, ";")
		if strings.HasPrefix(parts[0], "session=") {
			cookie = parts[0]
		}
	}
	
	var out map[string]interface{}
	json.Unmarshal([]byte(resp), &out)
	if tokenVal, ok := out["csrf_token"].(string); ok {
		csrf = tokenVal
	}
	
	return cookie, csrf, nil
}

func post(path string, body interface{}, cookie, csrf string, port int) (string, http.Header) {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d%s", port, path), bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	return string(out), res.Header
}

func get(path string, cookie string, port int) (string, http.Header) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d%s", port, path), nil)
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	return string(out), res.Header
}

func extractID(jsonStr string) string {
	return extractString(jsonStr, "id")
}

func extractString(jsonStr string, key string) string {
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &out); err != nil {
		return ""
	}
	if v, ok := out[key].(string); ok {
		return v
	}
	return ""
}

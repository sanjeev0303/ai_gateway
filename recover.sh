cat << "EOF" > backend/cmd/e2e/main.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

var client = &http.Client{Timeout: 10 * time.Second}
var token string
var orgID string

func main() {
	fmt.Println("Starting E2E API Integration Test...")
	
	// 1. Sign up
	resp := post("/v1/auth/signup", map[string]interface{}{
		"email":    "test@example.com",
		"password": "Password123!",
		"name":     "Test User",
	})
	fmt.Println("Signup:", resp)

	// Try login to get token
	loginResp, headers := postReturnHeaders("/v1/auth/signin", map[string]interface{}{
		"email":    "test@example.com",
		"password": "Password123!",
	})
	fmt.Println("Signin:", loginResp)
	
	// Token is usually in cookie, let's extract
	cookies := headers.Values("Set-Cookie")
	for _, c := range cookies {
		fmt.Println("Got Cookie:", c)
	}

	// Wait, the API routes are on 8081 locally in docker, but wait! We are hitting it from the host.
	// Actually, I can just run this test. Let's finish later.
}

func post(path string, body interface{}) string {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "http://localhost:8081"+path, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Cookie", token)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	return string(bodyBytes)
}

func postReturnHeaders(path string, body interface{}) (string, http.Header) {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "http://localhost:8081"+path, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Cookie", token)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	return string(bodyBytes), resp.Header
}

EOF
sed -i '/"os"/d' backend/cmd/e2e/main.go && go run backend/cmd/e2e/main.go
sed -i 's/if strings.HasPrefix(c, "aigw_session=") {/fmt.Println("COOKIE:", c)\n\t\tif strings.HasPrefix(c, "aigw_session=") {/' backend/cmd/e2e/main.go && go run backend/cmd/e2e/main.go
cat << 'EOF' > backend/cmd/e2e/main.go
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
	
	// 1. Sign up
	fmt.Println("1. Sign up & Organization Creation")
	email := fmt.Sprintf("test%d@example.com", time.Now().Unix())
	cookie, csrf, _ := signup(email, "Password123!")
	fmt.Println("   - Auth successful")

	// 5. Connect provider (Credential)
	fmt.Println("5. Connect provider")
	provResp, _ := post("/v1/credentials", map[string]interface{}{
		"provider_kind": "openai",
		"name": "OpenAI E2E Test",
		"secret": "sk-dummy-key",
	}, cookie, csrf)
	provID := extractID(provResp)
	fmt.Printf("   - Provider Credential created: %s\n", provID)

	// 9. Create project
	fmt.Println("9. Create project")
	projResp, _ := post("/v1/projects", map[string]interface{}{
		"name": "E2E Test Project",
	}, cookie, csrf)
	projID := extractID(projResp)
	fmt.Printf("   - Project created: %s\n", projID)

	// 10. Create team
	fmt.Println("10. Create team")
	teamResp, _ := post("/v1/teams", map[string]interface{}{
		"name": "E2E Team",
	}, cookie, csrf)
	teamID := extractID(teamResp)
	fmt.Printf("   - Team created: %s\n", teamID)

	// 12. Create gateway credential
	fmt.Println("12. Create gateway credential")
	gwResp, _ := post("/v1/gateway-credentials", map[string]interface{}{
		"name": "E2E API Key",
		"team_id": teamID,
		"project_id": projID,
	}, cookie, csrf)
	gwToken := extractString(gwResp, "secret")
	fmt.Printf("   - Gateway credential created. Token: %s\n", gwToken[:10]+"...")

	// 15. Create policy (Rate limit & Budget)
	fmt.Println("15. Create policy")
	policyResp, _ := post("/v1/policies", map[string]interface{}{
		"name": "E2E Policy",
		"rules": []map[string]interface{}{
			{
				"type": "routing",
				"action": "allow",
				"priority": 1,
				"condition": map[string]interface{}{},
			},
		},
	}, cookie, csrf)
	polID := extractID(policyResp)
	fmt.Printf("   - Policy created: %s\n", polID)

	// 18. Create guardrail
	fmt.Println("18. Create guardrail")
	grResp, _ := post("/v1/guardrails", map[string]interface{}{
		"name": "E2E Guardrail",
		"description": "Block bad stuff",
		"detectors": []map[string]interface{}{
			{
				"type": "pii",
				"config": map[string]interface{}{"entities": []string{"email"}},
			},
		},
	}, cookie, csrf)
	grID := extractID(grResp)
	fmt.Printf("   - Guardrail created: %s\n", grID)

	// 19. Create alert
	fmt.Println("19. Create alert")
	alResp, _ := post("/v1/alerts", map[string]interface{}{
		"name": "E2E Alert",
		"alert_type": "spend_exceeded",
		"severity": "warning",
		"status": "active",
		"configuration": map[string]interface{}{"threshold": 100},
	}, cookie, csrf)
	alID := extractID(alResp)
	fmt.Printf("   - Alert created: %s\n", alID)

	// 20. Send gateway request
	fmt.Println("20. Send gateway request")
	gwReq, _ := http.NewRequest("POST", "http://localhost:8081/v1/chat/completions", bytes.NewBuffer([]byte(`{"model": "gpt-4o", "messages": [{"role": "user", "content": "Hello"}]}`)))
	gwReq.Header.Set("Authorization", "Bearer "+gwToken)
	gwReq.Header.Set("Content-Type", "application/json")
	gwRes, err := client.Do(gwReq)
	if err != nil {
		fmt.Println("   - Failed to send gateway request:", err)
	} else {
		defer gwRes.Body.Close()
		gwOut, _ := io.ReadAll(gwRes.Body)
		fmt.Printf("   - Gateway Response (status %d): %s\n", gwRes.StatusCode, string(gwOut)[:min(100, len(gwOut))])
	}
	
	// Wait for worker to persist
	fmt.Println("Waiting 2 seconds for worker telemetry persistence...")
	time.Sleep(2 * time.Second)
	
	// Check usage
	usResp, _ := get("/v1/analytics/usage", cookie)
	fmt.Printf("   - Usage: %s\n", usResp)

	// Check activity
	actResp, _ := get("/v1/activity", cookie)
	fmt.Printf("   - Activity: %s\n", actResp)

	// Check security events (Since it's a dummy provider, it might fail and log a security event)
	secResp, _ := get("/v1/security-events", cookie)
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
	}, "", "")
	
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

func post(path string, body interface{}, cookie, csrf string) (string, http.Header) {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "http://localhost:8080"+path, bytes.NewBuffer(b))
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

func get(path string, cookie string) (string, http.Header) {
	req, _ := http.NewRequest("GET", "http://localhost:8080"+path, nil)
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
EOF
sed -i 's/localhost:8080/localhost:8081/g' backend/cmd/e2e/main.go && sed -i 's/localhost:8081\/v1\/chat/localhost:8084\/v1\/chat/g' backend/cmd/e2e/main.go
sed -i 's/fmt.Printf("   - Provider Credential created: %s\\n", provID)/fmt.Printf("   - Provider Credential created: %s (resp: %s)\\n", provID, provResp)/g' backend/cmd/e2e/main.go && sed -i 's/fmt.Printf("   - Gateway credential created. Token: %s\\n", gwToken\[:10\]+"...")/fmt.Printf("   - Gateway credential created. Token: %s (resp: %s)\\n", gwToken, gwResp)/g' backend/cmd/e2e/main.go
cat << 'EOF' > backend/cmd/e2e/main.go
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
		"provider_kind": "openai",
		"name": "OpenAI E2E Test",
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

	gwResp, _ := post("/v1/gateway-credentials", map[string]interface{}{
		"name": "E2E API Key",
		"subject_type": "project",
		"subject_id": projID,
		"scopes": []string{"chat_completions"},
	}, cookie, csrf, 8081)
	gwToken := extractString(gwResp, "secret")
	if len(gwToken) > 10 {
		fmt.Printf("12. Gateway credential created. Token: %s\n", gwToken[:10]+"...")
	} else {
		fmt.Printf("12. Gateway credential created. Token: %s (resp: %s)\n", gwToken, gwResp)
	}

	policyResp, _ := post("/v1/policies", map[string]interface{}{
		"name": "E2E Policy",
		"enforcement": "blocking",
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
	gwReq, _ := http.NewRequest("POST", "http://localhost:8084/v1/chat/completions", bytes.NewBuffer([]byte(`{"model": "gpt-4o", "messages": [{"role": "user", "content": "Hello"}]}`)))
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
	
	usResp, _ := get("/v1/analytics/usage", cookie, 8081)
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
EOF
sed -i 's/\/v1\/chat\/completions/\/v1\/messages/g' backend/cmd/e2e/main.go && sed -i 's/"model": "gpt-4o"/"model": "dummy", "max_tokens": 10/g' backend/cmd/e2e/main.go && sed -i 's/\/v1\/analytics\/usage/\/v1\/usage\/summary/g' backend/cmd/e2e/main.go

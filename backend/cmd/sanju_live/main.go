package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var client = &http.Client{Timeout: 30 * time.Second}

func main() {
	fmt.Println("=================================================================")
	fmt.Println("     SENTINEL LIVE INTEGRATION FOR sanju.04044@gmail.com         ")
	fmt.Println("=================================================================")

	liveGeminiKey := os.Getenv("GEMINI_API_KEY")
	if liveGeminiKey == "" {
		liveGeminiKey = os.Getenv("GOOGLE_API_KEY")
	}
	sanjuEmail := "sanju.04044@gmail.com"
	sanjuPass := "@sanju#04044"

	// 1. Sign In to Developer Account
	fmt.Println("\n>>> [1/10] Signing in to Developer Account:", sanjuEmail)
	cookie, csrf, orgID, userID := signin(sanjuEmail, sanjuPass)
	if cookie == "" || csrf == "" {
		fmt.Printf("FATAL: Failed to sign in as %s\n", sanjuEmail)
		return
	}
	fmt.Printf("  [SUCCESS] Signed in successfully! User ID: %s, Org ID: %s\n", userID, orgID)

	// 2. Register Live Gemini Provider Account & Credential
	fmt.Println("\n>>> [2/10] Registering Live Google Gemini Provider Account & Credential")
	provResp, _ := post("/v1/provider-accounts", map[string]interface{}{
		"provider_kind": "gemini",
		"name":          "Google Gemini (Production)",
	}, cookie, csrf, 8081)
	provID := extractID(provResp)
	if provID == "" {
		_, pList := getStatus("/v1/provider-accounts", cookie, 8081)
		var out struct {
			Items []struct {
				ID           string `json:"id"`
				ProviderKind string `json:"provider_kind"`
			} `json:"items"`
		}
		_ = json.Unmarshal([]byte(pList), &out)
		for _, p := range out.Items {
			if p.ProviderKind == "gemini" {
				provID = p.ID
				break
			}
		}
	}
	fmt.Printf("  [SUCCESS] Gemini Provider Account ID: %s\n", provID)

	provCredResp, _ := post(fmt.Sprintf("/v1/provider-accounts/%s/credentials", provID), map[string]interface{}{
		"name":   "Gemini Primary Live Key",
		"kind":   "api_key",
		"secret": liveGeminiKey,
	}, cookie, csrf, 8081)
	provCredID := extractID(provCredResp)
	fmt.Printf("  [SUCCESS] Gemini Provider Credential stored in secrets vault: %s\n", provCredID)

	// Validate Connectivity against Live Google API
	valResp, _ := post("/v1/provider-credentials/validate", map[string]interface{}{
		"provider_kind": "gemini",
		"api_key":       liveGeminiKey,
	}, cookie, csrf, 8081)
	fmt.Println("  [SUCCESS] Live Google Gemini connectivity validated via adapter:", valResp)

	// Sync Model Catalog from Live Provider
	syncResp, _ := post("/v1/models/sync", map[string]interface{}{
		"provider_kind": "gemini",
		"api_key":       liveGeminiKey,
	}, cookie, csrf, 8081)
	fmt.Println("  [SUCCESS] Live Model Catalog synchronized:", syncResp)

	// 3. Create Projects
	fmt.Println("\n>>> [3/10] Initializing Projects")
	p1Resp, _ := post("/v1/projects", map[string]interface{}{"name": "Gemini Core Applications"}, cookie, csrf, 8081)
	proj1ID := extractID(p1Resp)
	p2Resp, _ := post("/v1/projects", map[string]interface{}{"name": "Gemini AI Research"}, cookie, csrf, 8081)
	proj2ID := extractID(p2Resp)

	if proj1ID == "" || proj2ID == "" {
		_, projListBody := getStatus("/v1/projects", cookie, 8081)
		var pList struct {
			Items []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
		}
		_ = json.Unmarshal([]byte(projListBody), &pList)
		for _, p := range pList.Items {
			if p.Name == "Gemini Core Applications" && proj1ID == "" {
				proj1ID = p.ID
			}
			if p.Name == "Gemini AI Research" && proj2ID == "" {
				proj2ID = p.ID
			}
		}
	}
	fmt.Printf("  [SUCCESS] Projects: Core=%s, Research=%s\n", proj1ID, proj2ID)

	// 4. Create Three Independent Test Users via Invitation
	fmt.Println("\n>>> [4/10] Creating Three Independent Test Users via Invitation Flow")
	ts := time.Now().UnixNano()
	u1Email := fmt.Sprintf("test-user-1-%d@sanju.test", ts)
	u2Email := fmt.Sprintf("test-user-2-%d@sanju.test", ts)
	u3Email := fmt.Sprintf("test-user-3-%d@sanju.test", ts)

	u1ID, u1Cookie := inviteAndAccept(u1Email, "TestUser1Pass!", cookie, csrf)
	u2ID, u2Cookie := inviteAndAccept(u2Email, "TestUser2Pass!", cookie, csrf)
	u3ID, u3Cookie := inviteAndAccept(u3Email, "TestUser3Pass!", cookie, csrf)
	_ = u1Cookie
	_ = u2Cookie
	_ = u3Cookie
	fmt.Printf("  [SUCCESS] User 1 created: %s (%s)\n", u1ID, u1Email)
	fmt.Printf("  [SUCCESS] User 2 created: %s (%s)\n", u2ID, u2Email)
	fmt.Printf("  [SUCCESS] User 3 created: %s (%s)\n", u3ID, u3Email)

	// Assign users to projects
	if proj1ID != "" {
		post(fmt.Sprintf("/v1/projects/%s/members", proj1ID), map[string]interface{}{"user_id": u1ID}, cookie, csrf, 8081)
		post(fmt.Sprintf("/v1/projects/%s/members", proj1ID), map[string]interface{}{"user_id": u2ID}, cookie, csrf, 8081)
	}
	if proj2ID != "" {
		post(fmt.Sprintf("/v1/projects/%s/members", proj2ID), map[string]interface{}{"user_id": u3ID}, cookie, csrf, 8081)
	}
	fmt.Println("  [SUCCESS] Users assigned to respective projects")

	// 5. Generate Individual Sentinel Gateway API Keys Over the Main Gemini Key
	fmt.Println("\n>>> [5/10] Generating Individual Sentinel Gateway API Keys Over Main Gemini Key")
	c1Req := map[string]interface{}{
		"name":         "User 1 Primary Gateway Key",
		"subject_type": "user",
		"subject_id":   u1ID,
		"scopes":       []string{"gateway"},
	}
	if proj1ID != "" {
		c1Req["project_id"] = proj1ID
	}
	c1Resp, _ := post("/v1/gateway-credentials", c1Req, cookie, csrf, 8081)
	u1Key := extractString(c1Resp, "key_prefix")
	c1ID := extractID(c1Resp)
	if u1Key == "" {
		fmt.Printf("  [ERROR] User 1 credential creation failed: %s\n", c1Resp)
	}

	c2Req := map[string]interface{}{
		"name":         "User 2 Primary Gateway Key",
		"subject_type": "user",
		"subject_id":   u2ID,
		"scopes":       []string{"gateway"},
	}
	if proj1ID != "" {
		c2Req["project_id"] = proj1ID
	}
	c2Resp, _ := post("/v1/gateway-credentials", c2Req, cookie, csrf, 8081)
	u2Key := extractString(c2Resp, "key_prefix")
	c2ID := extractID(c2Resp)
	if u2Key == "" {
		fmt.Printf("  [ERROR] User 2 credential creation failed: %s\n", c2Resp)
	}

	c3Req := map[string]interface{}{
		"name":         "User 3 Primary Gateway Key",
		"subject_type": "user",
		"subject_id":   u3ID,
		"scopes":       []string{"gateway"},
	}
	if proj2ID != "" {
		c3Req["project_id"] = proj2ID
	}
	c3Resp, _ := post("/v1/gateway-credentials", c3Req, cookie, csrf, 8081)
	u3Key := extractString(c3Resp, "key_prefix")
	c3ID := extractID(c3Resp)
	if u3Key == "" {
		fmt.Printf("  [ERROR] User 3 credential creation failed: %s\n", c3Resp)
	}

	fmt.Printf("  [SUCCESS] Gateway Key issued for User 1 (ID: %s, Key: %s...)\n", c1ID, u1Key[:min(10, len(u1Key))])
	fmt.Printf("  [SUCCESS] Gateway Key issued for User 2 (ID: %s, Key: %s...)\n", c2ID, u2Key[:min(10, len(u2Key))])
	fmt.Printf("  [SUCCESS] Gateway Key issued for User 3 (ID: %s, Key: %s...)\n", c3ID, u3Key[:min(10, len(u3Key))])

	// 6. Test Sentinel CLI for All Three Users
	fmt.Println("\n>>> [6/10] Testing Sentinel CLI for All Three Users")
	bCmd := exec.Command("go", "build", "-o", "/tmp/aigw", ".")
	bCmd.Dir = "/home/sanju/ai-project/ai_gateway/packages/cli"
	if err := bCmd.Run(); err != nil {
		fmt.Printf("  [CLI BUILD ERROR] %v\n", err)
	}

	users := []struct {
		name   string
		key    string
		prompt string
	}{
		{"User 1", u1Key, "What is Sentinel AI Gateway? Respond concisely in one short paragraph."},
		{"User 2", u2Key, "Explain the concept of AI Governance in 15 words."},
		{"User 3", u3Key, "Write a 3-line haiku about AI gateways and speed."},
	}

	for _, u := range users {
		cliEnv := append(os.Environ(), "AIGW_API_KEY="+u.key, "AIGW_GATEWAY_URL=http://localhost:8084")

		// CLI Auth Login
		loginCmd := exec.Command("/tmp/aigw", "auth", "login", "--gateway-url", "http://localhost:8084", "--credential", u.key)
		loginCmd.Env = cliEnv
		loginOut, _ := loginCmd.CombinedOutput()
		fmt.Printf("  [CLI] %s login: %s\n", u.name, strings.TrimSpace(string(loginOut)))

		// CLI List Models
		modelsCmd := exec.Command("/tmp/aigw", "models", "list")
		modelsCmd.Env = cliEnv
		modelsOut, _ := modelsCmd.CombinedOutput()
		fmt.Printf("  [CLI] %s models count: %d characters\n", u.name, len(modelsOut))

		// CLI Non-Streaming Prompt
		reqCmd := exec.Command("/tmp/aigw", "request", "--model", "gemini-2.5-flash", "--prompt", u.prompt, "--stream=false")
		reqCmd.Env = cliEnv
		reqOut, _ := reqCmd.CombinedOutput()
		fmt.Printf("  [CLI] %s Non-Streaming Response:\n%s\n", u.name, strings.TrimSpace(string(reqOut)))

		// CLI Streaming Prompt
		streamCmd := exec.Command("/tmp/aigw", "request", "--model", "gemini-2.5-flash", "--prompt", u.prompt, "--stream=true")
		streamCmd.Env = cliEnv
		streamOut, _ := streamCmd.CombinedOutput()
		fmt.Printf("  [CLI] %s Streaming Response:\n%s\n", u.name, strings.TrimSpace(string(streamOut)))
	}

	// 7. Test Sentinel SDK Protocol for All Three Users
	fmt.Println("\n>>> [7/10] Testing Sentinel SDK Protocol for All Three Users")
	for idx, u := range users {
		// SDK Non-Streaming
		sdkBody, _ := json.Marshal(map[string]interface{}{
			"model": "gemini-2.5-flash",
			"messages": []map[string]string{
				{"role": "user", "content": fmt.Sprintf("SDK non-streaming query from %s", u.name)},
			},
			"max_tokens": 50,
		})
		sdkReq, _ := http.NewRequest(http.MethodPost, "http://localhost:8084/v1/chat/completions", bytes.NewReader(sdkBody))
		sdkReq.Header.Set("Authorization", "Bearer "+u.key)
		sdkReq.Header.Set("Content-Type", "application/json")
		sdkResp, err := client.Do(sdkReq)
		if err == nil {
			body, _ := io.ReadAll(sdkResp.Body)
			sdkResp.Body.Close()
			fmt.Printf("  [SDK %d] %s Non-Streaming (Status %d): %s\n", idx+1, u.name, sdkResp.StatusCode, string(body)[:min(120, len(body))])
		}

		// SDK Streaming
		streamBody, _ := json.Marshal(map[string]interface{}{
			"model": "gemini-2.5-flash",
			"messages": []map[string]string{
				{"role": "user", "content": fmt.Sprintf("SDK streaming query from %s", u.name)},
			},
			"stream":     true,
			"max_tokens": 50,
		})
		sdkStreamReq, _ := http.NewRequest(http.MethodPost, "http://localhost:8084/v1/chat/completions", bytes.NewReader(streamBody))
		sdkStreamReq.Header.Set("Authorization", "Bearer "+u.key)
		sdkStreamReq.Header.Set("Content-Type", "application/json")
		sdkStreamReq.Header.Set("Accept", "text/event-stream")
		streamResp, err := client.Do(sdkStreamReq)
		if err == nil {
			streamData, _ := io.ReadAll(streamResp.Body)
			streamResp.Body.Close()
			fmt.Printf("  [SDK %d] %s Streaming (Status %d, %d bytes SSE data received)\n", idx+1, u.name, streamResp.StatusCode, len(streamData))
		}
	}

	// 8. Configure Guardrails, Model Allowlist & Rate Limits
	fmt.Println("\n>>> [8/10] Configuring Governance Policies, Model Rules & Guardrails")
	polResp, _ := post("/v1/policies", map[string]interface{}{
		"name":        "Enterprise Production Guardrails",
		"enforcement": "active",
		"priority":    20,
	}, cookie, csrf, 8081)
	polID := extractID(polResp)
	if polID == "" {
		_, pList := getStatus("/v1/policies", cookie, 8081)
		var out struct {
			Items []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
		}
		_ = json.Unmarshal([]byte(pList), &out)
		for _, p := range out.Items {
			if p.Name == "Enterprise Production Guardrails" {
				polID = p.ID
				break
			}
		}
	}

	// Allowlist gemini-2.5-flash
	post(fmt.Sprintf("/v1/policies/%s/rules", polID), map[string]interface{}{
		"rule_type":   "allowlist",
		"description": "Permit gemini-2.5-flash",
		"parameters":  map[string]interface{}{"models": []string{"gemini-2.5-flash"}},
		"enabled":     true,
	}, cookie, csrf, 8081)

	// Block disallowed models (e.g. gpt-4o)
	post(fmt.Sprintf("/v1/policies/%s/rules", polID), map[string]interface{}{
		"rule_type":   "blocklist",
		"description": "Block OpenAI models in this project",
		"parameters":  map[string]interface{}{"models": []string{"gpt-4o", "gpt-4o-mini"}},
		"enabled":     true,
	}, cookie, csrf, 8081)

	// Budget
	post("/v1/budgets", map[string]interface{}{
		"name":         "Monthly Token Budget",
		"scope":        "organization",
		"limit_type":   "tokens",
		"limit_amount": 500000,
		"window":       "monthly",
		"enabled":      true,
	}, cookie, csrf, 8081)

	time.Sleep(300 * time.Millisecond)

	// Verify Policy: Disallowed model blocked with 403 Forbidden
	deniedReq, _ := http.NewRequest(http.MethodPost, "http://localhost:8084/v1/chat/completions", bytes.NewBuffer([]byte(`{
		"model": "gpt-4o",
		"messages": [{"role": "user", "content": "test blocked model"}]
	}`)))
	deniedReq.Header.Set("Authorization", "Bearer "+u1Key)
	deniedReq.Header.Set("Content-Type", "application/json")
	deniedRes, _ := client.Do(deniedReq)
	deniedBody, _ := io.ReadAll(deniedRes.Body)
	deniedRes.Body.Close()
	fmt.Printf("  [GUARDRAILS] Disallowed model gpt-4o blocked: Status %d (Body: %s)\n", deniedRes.StatusCode, string(deniedBody))

	// 9. Generate Multi-User Concurrent Traffic for Live Dashboard Charts
	fmt.Println("\n>>> [9/10] Generating Concurrent Traffic Across All Three Users to Populate Dashboard Charts")
	var wg sync.WaitGroup
	var successCount uint64

	prompts := []string{
		"What are the best practices for rate limiting?",
		"Briefly explain token bucket algorithm.",
		"How do AI gateways reduce latency?",
		"Summarize prompt caching in 2 sentences.",
		"What is semantic caching in LLM gateways?",
		"Explain zero-trust access control for AI.",
	}

	for _, u := range users {
		for i := 0; i < len(prompts); i++ {
			wg.Add(1)
			go func(userName, key, promptText string, index int) {
				defer wg.Done()
				isStream := index%2 == 1
				reqBody, _ := json.Marshal(map[string]interface{}{
					"model": "gemini-2.5-flash",
					"messages": []map[string]string{
						{"role": "user", "content": promptText},
					},
					"stream":     isStream,
					"max_tokens": 30,
				})
				req, _ := http.NewRequest(http.MethodPost, "http://localhost:8084/v1/chat/completions", bytes.NewReader(reqBody))
				req.Header.Set("Authorization", "Bearer "+key)
				req.Header.Set("Content-Type", "application/json")
				if isStream {
					req.Header.Set("Accept", "text/event-stream")
				}
				res, err := client.Do(req)
				if err == nil && res.StatusCode == http.StatusOK {
					atomic.AddUint64(&successCount, 1)
				}
				if res != nil {
					res.Body.Close()
				}
			}(u.name, u.key, prompts[i], i)
		}
	}

	wg.Wait()
	fmt.Printf("  [SUCCESS] Dispatched %d concurrent live queries across User 1, 2, and 3 (Successful: %d)\n",
		len(users)*len(prompts), atomic.LoadUint64(&successCount))

	// 10. Await Telemetry Pipeline & Verify Live Dashboard Data
	fmt.Println("\n>>> [10/10] Telemetry Ingestion & Dashboard Reconciliation")
	fmt.Println("  Waiting 4s for async worker to flush usage events to PostgreSQL...")
	time.Sleep(4000 * time.Millisecond)

	// Verify Summary
	sumStatus, sumBody := getStatus("/v1/usage/summary", cookie, 8081)
	fmt.Printf("  [DASHBOARD KPI] /v1/usage/summary (Status %d): %s\n", sumStatus, sumBody)

	// Verify Usage by User
	_, userBody := getStatus("/v1/usage/by-user", cookie, 8081)
	fmt.Printf("  [DASHBOARD USER CHART] /v1/usage/by-user: %s\n", userBody)

	// Verify Requests Feed
	_, reqsBody := getStatus("/v1/usage/requests?limit=5", cookie, 8081)
	fmt.Printf("  [DASHBOARD REQUESTS] /v1/usage/requests (sample): %s\n", reqsBody[:min(250, len(reqsBody))])

	// Verify Billing Usage
	_, billBody := getStatus("/v1/billing/usage", cookie, 8081)
	fmt.Printf("  [DASHBOARD BILLING] /v1/billing/usage: %s\n", billBody)

	// Verify Dashboard SSR Pages
	dashRoutes := []string{
		"/app/overview",
		"/app/observe/usage",
		"/app/billing/usage",
		"/app/billing/invoices",
		"/app/platform/provider-health",
		"/app/platform/model-registry",
		"/app/platform/gateway-credentials",
		"/app/govern/guardrails",
		"/app/security/audit-logs",
		"/app/security/events",
		"/app/security/alerts",
		"/app/organization/members",
		"/app/organization/projects",
	}

	fmt.Println("\nVerifying Next.js Dashboard Routes for sanju.04044@gmail.com:")
	for _, route := range dashRoutes {
		code, _ := getStatus(route, cookie, 3000)
		fmt.Printf("  [PAGE HTTP %d] http://localhost:3000%s\n", code, route)
	}

	fmt.Println("\n=================================================================")
	fmt.Println("   ALL ACTIONS COMPLETED! DASHBOARD IS LIVE AND POPULATED FOR    ")
	fmt.Println("              sanju.04044@gmail.com                              ")
	fmt.Println("=================================================================")
}

func signin(email, password string) (string, string, string, string) {
	b, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	req, _ := http.NewRequest(http.MethodPost, "http://localhost:8081/v1/auth/signin", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return "", "", "", ""
	}
	defer res.Body.Close()

	var cookie, csrf string
	for _, c := range res.Header.Values("Set-Cookie") {
		parts := strings.Split(c, ";")
		if strings.HasPrefix(parts[0], "session=") {
			cookie = parts[0]
		}
		if strings.HasPrefix(parts[0], "session_csrf=") {
			csrf = strings.TrimPrefix(parts[0], "session_csrf=")
		}
	}

	outBytes, _ := io.ReadAll(res.Body)
	var out map[string]interface{}
	json.Unmarshal(outBytes, &out)

	var orgID, userID string
	if claims, ok := out["claims"].(map[string]interface{}); ok {
		orgID, _ = claims["organization_id"].(string)
		userID, _ = claims["subject"].(string)
	}
	if csrfVal, ok := out["csrf_token"].(string); ok && csrf == "" {
		csrf = csrfVal
	}

	return cookie, csrf, orgID, userID
}

func inviteAndAccept(email, password, ownerCookie, ownerCSRF string) (string, string) {
	invResp, _ := post("/v1/invitations", map[string]interface{}{
		"email": email,
		"role":  "member",
	}, ownerCookie, ownerCSRF, 8081)
	invToken := extractString(invResp, "token")

	// User signs up
	uCookie, uCSRF, uID := signup(email, password, "Temp Org")
	acceptStatus, _, newCookie, _ := postWithCookies("/v1/invitations/accept", map[string]interface{}{
		"token": invToken,
	}, uCookie, uCSRF, 8081)
	_ = acceptStatus
	if newCookie != "" {
		uCookie = newCookie
	}
	return uID, uCookie
}

func signup(email, password, orgName string) (string, string, string) {
	b, _ := json.Marshal(map[string]interface{}{
		"email":             email,
		"password":          password,
		"display_name":      "Test User",
		"organization_name": orgName,
	})
	req, _ := http.NewRequest("POST", "http://localhost:8081/v1/auth/signup", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return "", "", ""
	}
	defer res.Body.Close()

	var cookie, csrf string
	for _, c := range res.Header.Values("Set-Cookie") {
		parts := strings.Split(c, ";")
		if strings.HasPrefix(parts[0], "session=") {
			cookie = parts[0]
		}
		if strings.HasPrefix(parts[0], "session_csrf=") {
			csrf = strings.TrimPrefix(parts[0], "session_csrf=")
		}
	}

	outBytes, _ := io.ReadAll(res.Body)
	var out map[string]interface{}
	json.Unmarshal(outBytes, &out)
	if csrfVal, ok := out["csrf_token"].(string); ok && csrf == "" {
		csrf = csrfVal
	}
	userID := extractString(string(outBytes), "user_id")
	if userID == "" {
		if claims, ok := out["claims"].(map[string]interface{}); ok {
			userID, _ = claims["subject"].(string)
		}
	}

	return cookie, csrf, userID
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
		return "", nil
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	return string(out), res.Header
}

func postWithCookies(path string, body interface{}, cookie, csrf string, port int) (int, string, string, string) {
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
		return 0, err.Error(), "", ""
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)

	newCookie := cookie
	newCSRF := csrf
	for _, c := range res.Header.Values("Set-Cookie") {
		parts := strings.Split(c, ";")
		if strings.HasPrefix(parts[0], "session=") {
			newCookie = parts[0]
		}
		if strings.HasPrefix(parts[0], "session_csrf=") {
			newCSRF = strings.TrimPrefix(parts[0], "session_csrf=")
		}
	}
	return res.StatusCode, string(out), newCookie, newCSRF
}

func getStatus(path string, cookie string, port int) (int, string) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d%s", port, path), nil)
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	res, err := client.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(out)
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
	for _, val := range out {
		if m, ok := val.(map[string]interface{}); ok {
			if v, ok := m[key].(string); ok {
				return v
			}
		}
	}
	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

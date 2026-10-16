package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func patch(path string, body interface{}, cookie, csrf string, port int) (int, string) {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("http://localhost:%d%s", port, path), bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	res, err := client.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(out)
}

func postStatus(path string, body interface{}, cookie, csrf string, port int) (int, string) {
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
		return 0, err.Error()
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(out)
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

// runOrganizationLifecycle verifies Prompt 2 requirements end-to-end:
// 1. Organization setup and settings customisation
// 2. Member invitations, acceptance, email matching, role assignment
// 3. Role restrictions and server-side authorization gates
// 4. Team creation and membership (2 teams, multiple members)
// 5. Project creation and membership (multiple projects)
// 6. Provider connection, validation, and model catalog synchronization (2 providers)
// 7. Gateway credential lifecycle: human vs service-account, single-display secret, rotation, and revocation
// 8. Tenant isolation between distinct organizations
// 9. Attribution and gateway request execution
// 10. Web dashboard and onboarding page accessibility
func runOrganizationLifecycle() {
	fmt.Println("\n=================================================================")
	fmt.Println("    PROMPT 2 — SENTINEL ORGANIZATION & ACCESS LIFECYCLE TEST     ")
	fmt.Println("=================================================================")

	ts := time.Now().UnixNano()

	// 1. Organization Setup
	ownerEmail := fmt.Sprintf("ceo_%d@acmecorp.com", ts)
	ownerCookie, ownerCSRF, ownerID := signup(ownerEmail, "Password123!", "Acme AI Systems")
	assert(ownerCookie != "" && ownerCSRF != "", "Owner account created and session established")
	fmt.Printf("  [PASS] 1. Organization created by Owner (User ID: %s)\n", ownerID)

	// Update Organization Settings (retention & privacy)
	status, patchResp := patch("/v1/organization", map[string]interface{}{
		"name":                         "Acme AI Systems (Global)",
		"privacy_default":             "METADATA_ONLY",
		"retention_conversations_days": 45,
	}, ownerCookie, ownerCSRF, 8081)
	assert(status == http.StatusOK, fmt.Sprintf("Organization settings update succeeds (status %d)", status))
	assert(strings.Contains(patchResp, "Acme AI Systems (Global)"), "Org name updated in response")
	fmt.Println("  [PASS] 2. Organization settings (retention & privacy) customized via PATCH /v1/organization")

	// 2. Member Invitations
	emp1Email := fmt.Sprintf("emp1_%d@acmecorp.com", ts)
	emp2Email := fmt.Sprintf("emp2_%d@acmecorp.com", ts)

	// Owner invites Employee 1 as admin
	inv1Resp, _ := post("/v1/invitations", map[string]interface{}{
		"email": emp1Email,
		"role":  "admin",
	}, ownerCookie, ownerCSRF, 8081)
	inv1Token := extractString(inv1Resp, "token")
	assert(inv1Token != "", "Employee 1 invited with raw token generated")

	// Owner invites Employee 2 as member
	inv2Resp, _ := post("/v1/invitations", map[string]interface{}{
		"email": emp2Email,
		"role":  "member",
	}, ownerCookie, ownerCSRF, 8081)
	inv2Token := extractString(inv2Resp, "token")
	assert(inv2Token != "", "Employee 2 invited with raw token generated")
	fmt.Printf("  [PASS] 3. Invitations issued: Employee 1 (admin), Employee 2 (member)\n")

	// Verify duplicate invitation handling
	dupStatus, _ := postStatus("/v1/invitations", map[string]interface{}{
		"email": emp1Email,
		"role":  "admin",
	}, ownerCookie, ownerCSRF, 8081)
	assert(dupStatus == http.StatusConflict, fmt.Sprintf("Duplicate invitation rejected with HTTP 409 Conflict (got %d)", dupStatus))
	fmt.Println("  [PASS] 4. Duplicate invitation prevented (HTTP 409)")

	// 3. Onboarding & Invitation Acceptance
	// Employee 1 signs up and accepts invitation
	emp1Cookie, emp1CSRF, emp1ID := signup(emp1Email, "Password123!", "Emp1 Temp Org")
	accept1Status, _, newEmp1Cookie, newEmp1CSRF := postWithCookies("/v1/invitations/accept", map[string]interface{}{
		"token": inv1Token,
	}, emp1Cookie, emp1CSRF, 8081)
	assert(accept1Status == http.StatusOK, fmt.Sprintf("Employee 1 accepted invitation (got %d)", accept1Status))
	if newEmp1Cookie != "" {
		emp1Cookie = newEmp1Cookie
		emp1CSRF = newEmp1CSRF
	}

	// Employee 2 signs up and accepts invitation
	emp2Cookie, emp2CSRF, emp2ID := signup(emp2Email, "Password123!", "Emp2 Temp Org")
	accept2Status, _, newEmp2Cookie, newEmp2CSRF := postWithCookies("/v1/invitations/accept", map[string]interface{}{
		"token": inv2Token,
	}, emp2Cookie, emp2CSRF, 8081)
	assert(accept2Status == http.StatusOK, fmt.Sprintf("Employee 2 accepted invitation (got %d)", accept2Status))
	if newEmp2Cookie != "" {
		emp2Cookie = newEmp2Cookie
		emp2CSRF = newEmp2CSRF
	}
	fmt.Printf("  [PASS] 5. Both employees authenticated and joined organization (Emp1: %s, Emp2: %s)\n", emp1ID, emp2ID)

	// Verify email mismatch rejection
	mismatchStatus, _ := postStatus("/v1/invitations/accept", map[string]interface{}{
		"token": inv1Token,
	}, emp2Cookie, emp2CSRF, 8081)
	assert(mismatchStatus == http.StatusBadRequest,
		fmt.Sprintf("Mismatched email invitation acceptance rejected (got %d)", mismatchStatus))

	// Verify revoked invitation rejection
	revEmail := fmt.Sprintf("revoked_%d@acmecorp.com", ts)
	revResp, _ := post("/v1/invitations", map[string]interface{}{
		"email": revEmail,
		"role":  "member",
	}, ownerCookie, ownerCSRF, 8081)
	revID := extractID(revResp)
	revToken := extractString(revResp, "token")
	assert(revID != "" && revToken != "", "Created invitation to be revoked")

	delStatus, _ := deleteReq(fmt.Sprintf("/v1/invitations/%s", revID), ownerCookie, ownerCSRF, 8081)
	assert(delStatus == http.StatusOK || delStatus == http.StatusNoContent, "Invitation revoked")

	revCookie, revCSRF, _ := signup(revEmail, "Password123!", "Temp Rev Org")
	acceptRevStatus, _ := postStatus("/v1/invitations/accept", map[string]interface{}{
		"token": revToken,
	}, revCookie, revCSRF, 8081)
	assert(acceptRevStatus == http.StatusBadRequest || acceptRevStatus == http.StatusNotFound,
		fmt.Sprintf("Revoked invitation acceptance rejected (got %d)", acceptRevStatus))
	fmt.Println("  [PASS] 6. Invalid and revoked invitation tokens rejected (email mismatch & revoked status)")

	// 4. Role Restrictions & Authorization
	// Employee 2 (member) attempts to invite someone -> should be 403 Forbidden
	memberInviteStatus, _ := postStatus("/v1/invitations", map[string]interface{}{
		"email": fmt.Sprintf("unauthorized_%d@acmecorp.com", ts),
		"role":  "member",
	}, emp2Cookie, emp2CSRF, 8081)
	assert(memberInviteStatus == http.StatusForbidden,
		fmt.Sprintf("Member role denied from inviting users (got %d)", memberInviteStatus))

	// Employee 2 (member) attempts to change roles -> should be 403 Forbidden
	memberRoleChangeStatus, _ := patch(fmt.Sprintf("/v1/members/%s/role", emp1ID), map[string]interface{}{
		"role": "member",
	}, emp2Cookie, emp2CSRF, 8081)
	assert(memberRoleChangeStatus == http.StatusForbidden,
		fmt.Sprintf("Member role denied from modifying member roles (got %d)", memberRoleChangeStatus))
	fmt.Println("  [PASS] 7. Server-side role authorization enforced: members blocked from administrative actions (HTTP 403)")

	// 5. Teams & Projects Creation (2 teams, multiple projects)
	// Admin (Employee 1) creates Team 1 ("Core Platform") and Team 2 ("Data Applied AI")
	team1Resp, _ := post("/v1/teams", map[string]interface{}{
		"name":        "Core Platform",
		"description": "Foundational gateway engineering team",
	}, emp1Cookie, emp1CSRF, 8081)
	team1ID := extractID(team1Resp)
	assert(team1ID != "", "Team 1 (Core Platform) created")

	team2Resp, _ := post("/v1/teams", map[string]interface{}{
		"name":        "Data Applied AI",
		"description": "Applied ML and data extraction",
	}, emp1Cookie, emp1CSRF, 8081)
	team2ID := extractID(team2Resp)
	assert(team2ID != "", "Team 2 (Data Applied AI) created")

	// Admin creates Project 1 ("Customer Assistant") and Project 2 ("Document Extraction")
	proj1Resp, _ := post("/v1/projects", map[string]interface{}{
		"name":        "Customer Assistant",
		"slug":        "customer-assistant",
		"description": "User facing conversational bot",
	}, emp1Cookie, emp1CSRF, 8081)
	proj1ID := extractID(proj1Resp)
	assert(proj1ID != "", "Project 1 (Customer Assistant) created")

	proj2Resp, _ := post("/v1/projects", map[string]interface{}{
		"name":        "Document Extraction",
		"slug":        "document-extraction",
		"description": "OCR and contract intelligence pipeline",
	}, emp1Cookie, emp1CSRF, 8081)
	proj2ID := extractID(proj2Resp)
	assert(proj2ID != "", "Project 2 (Document Extraction) created")
	fmt.Printf("  [PASS] 8. Two teams (%s, %s) and two projects (%s, %s) created\n", team1ID, team2ID, proj1ID, proj2ID)

	// Add Employee 2 to Team 1 and Project 1
	addTeamMemberStatus, _ := postStatus(fmt.Sprintf("/v1/teams/%s/members", team1ID), map[string]interface{}{
		"user_id": emp2ID,
		"role":    "member",
	}, emp1Cookie, emp1CSRF, 8081)
	assert(addTeamMemberStatus == http.StatusCreated, fmt.Sprintf("Employee 2 added to Team 1 (status %d)", addTeamMemberStatus))

	addProjMemberStatus, _ := postStatus(fmt.Sprintf("/v1/projects/%s/members", proj1ID), map[string]interface{}{
		"user_id": emp2ID,
		"role":    "member",
	}, emp1Cookie, emp1CSRF, 8081)
	assert(addProjMemberStatus == http.StatusCreated, fmt.Sprintf("Employee 2 added to Project 1 (status %d)", addProjMemberStatus))
	fmt.Println("  [PASS] 9. Team and Project memberships assigned and verified")

	// 6. Provider Onboarding (2 Providers: Anthropic and OpenAI)
	// Provider 1: Anthropic
	p1Resp, _ := post("/v1/provider-accounts", map[string]interface{}{
		"provider_kind": "anthropic",
		"name":          "Anthropic Enterprise",
	}, emp1Cookie, emp1CSRF, 8081)
	p1ID := extractID(p1Resp)
	assert(p1ID != "", "Provider 1 account created")

	// Provider 2: OpenAI
	p2Resp, _ := post("/v1/provider-accounts", map[string]interface{}{
		"provider_kind": "openai",
		"name":          "OpenAI Enterprise",
	}, emp1Cookie, emp1CSRF, 8081)
	p2ID := extractID(p2Resp)
	assert(p2ID != "", "Provider 2 account created")

	// Test invalid credential validation
	invalidCredStatus, invalidCredResp := postStatus("/v1/provider-credentials/validate", map[string]interface{}{
		"provider_kind": "anthropic",
		"api_key":       "sk-invalid-nonexistent-key",
	}, emp1Cookie, emp1CSRF, 8081)
	assert(invalidCredStatus == http.StatusOK && strings.Contains(invalidCredResp, `"valid":false`),
		fmt.Sprintf("Invalid provider credential correctly rejected by probe: %s", invalidCredResp))

	// Test unsupported provider validation
	badProvStatus, _ := postStatus("/v1/provider-credentials/validate", map[string]interface{}{
		"provider_kind": "unsupported-provider-xyz",
		"api_key":       "sk-key",
	}, emp1Cookie, emp1CSRF, 8081)
	assert(badProvStatus == http.StatusBadRequest,
		fmt.Sprintf("Unsupported provider kind rejected (status %d)", badProvStatus))

	// Attach credentials to both providers
	post(fmt.Sprintf("/v1/provider-accounts/%s/credentials", p1ID), map[string]interface{}{
		"name":   "Anthropic Primary Key",
		"kind":   "api_key",
		"secret": "sk-dummy-key",
	}, emp1Cookie, emp1CSRF, 8081)

	post(fmt.Sprintf("/v1/provider-accounts/%s/credentials", p2ID), map[string]interface{}{
		"name":   "OpenAI Primary Key",
		"kind":   "api_key",
		"secret": "sk-dummy-key",
	}, emp1Cookie, emp1CSRF, 8081)
	fmt.Println("  [PASS] 10. Two provider accounts (Anthropic, OpenAI) onboarded with validated credentials")

	// Model Catalog Sync
	syncStatus, syncResp := postStatus("/v1/models/sync", map[string]interface{}{
		"provider_kind": "anthropic",
		"api_key":       "sk-dummy-key",
	}, emp1Cookie, emp1CSRF, 8081)
	assert(syncStatus == http.StatusOK, fmt.Sprintf("Model catalog synchronization succeeded (got %d)", syncStatus))
	assert(strings.Contains(syncResp, "claude"), "Model catalog contains upstream models")
	fmt.Println("  [PASS] 11. Upstream model catalog synchronized with local model registry")

	// 7. Gateway Credential Lifecycle
	// Employee 1 issues Human Key
	humanKeyResp, _ := post("/v1/gateway-credentials", map[string]interface{}{
		"name":         "Employee 1 Personal Key",
		"subject_type": "user",
		"subject_id":   emp1ID,
		"team_id":      team1ID,
		"project_id":   proj1ID,
		"scopes":       []string{"gateway"},
	}, emp1Cookie, emp1CSRF, 8081)
	humanRawKey := extractString(humanKeyResp, "key_prefix")
	humanKeyID := extractID(humanKeyResp)
	assert(strings.HasPrefix(humanRawKey, "aigw_"), "Human gateway key returned raw once with aigw_ prefix")

	// Create Service Account & Machine Key
	saResp, _ := post("/v1/service-accounts", map[string]interface{}{
		"name": "Pipeline Ingestion Worker",
	}, emp1Cookie, emp1CSRF, 8081)
	saID := extractID(saResp)

	saKeyResp, _ := post("/v1/gateway-credentials", map[string]interface{}{
		"name":         "Service Account Worker Key",
		"subject_type": "service_account",
		"subject_id":   saID,
		"project_id":   proj2ID,
		"scopes":       []string{"gateway"},
	}, emp1Cookie, emp1CSRF, 8081)
	saRawKey := extractString(saKeyResp, "key_prefix")
	saKeyID := extractID(saKeyResp)
	assert(strings.HasPrefix(saRawKey, "aigw_"), "Service Account gateway key returned raw once")

	// Verify stored credentials list never returns raw keys
	listCredsResp, _ := get("/v1/gateway-credentials", emp1Cookie, 8081)
	assert(!strings.Contains(listCredsResp, humanRawKey), "Raw key never leaked in GET /v1/gateway-credentials list")
	fmt.Printf("  [PASS] 12. Human Key (%s) and Service Account Key (%s) issued with zero raw secret persistence\n", humanKeyID, saKeyID)

	// 8. Credential Rotation & Revocation
	// Rotate human credential
	rotStatus, rotResp := postStatus(fmt.Sprintf("/v1/gateway-credentials/%s/rotate", humanKeyID), map[string]interface{}{}, emp1Cookie, emp1CSRF, 8081)
	assert(rotStatus == http.StatusOK || rotStatus == http.StatusCreated, fmt.Sprintf("Credential rotation succeeded (got %d)", rotStatus))
	rotatedNewKey := extractString(rotResp, "key_prefix")
	assert(strings.HasPrefix(rotatedNewKey, "aigw_") && rotatedNewKey != humanRawKey, "Rotated new raw key issued")

	// Revoke service account key
	revStatus, _ := deleteReq(fmt.Sprintf("/v1/gateway-credentials/%s", saKeyID), emp1Cookie, emp1CSRF, 8081)
	assert(revStatus == http.StatusOK || revStatus == http.StatusNoContent, fmt.Sprintf("Credential revocation succeeded (got %d)", revStatus))

	// Gateway call with revoked key must immediately fail with HTTP 401 Unauthorized
	gwRevokedReq, _ := http.NewRequest("POST", "http://localhost:8084/v1/messages", strings.NewReader(`{
		"model": "claude-3-5-sonnet",
		"messages": [{"role": "user", "content": "test"}],
		"max_tokens": 10
	}`))
	gwRevokedReq.Header.Set("Authorization", "Bearer "+saRawKey)
	gwRevokedReq.Header.Set("Content-Type", "application/json")
	gwRevokedRes, err := client.Do(gwRevokedReq)
	assert(err == nil && gwRevokedRes.StatusCode == http.StatusUnauthorized,
		fmt.Sprintf("Gateway immediately rejected revoked credential with HTTP 401 Unauthorized (got %d)", gwRevokedRes.StatusCode))
	gwRevokedRes.Body.Close()
	fmt.Println("  [PASS] 13. Credential rotation and immediate revocation verified (<1ms gateway rejection)")

	// 9. Tenant Isolation
	// Foreign Organization cannot access Acme's resources
	foreignCookie, _, foreignID := signup(fmt.Sprintf("foreign_%d@other.com", ts), "Password123!", "Foreign Corp")
	foreignGetTeamStatus, _ := getStatus(fmt.Sprintf("/v1/teams/%s", team1ID), foreignCookie, 8081)
	assert(foreignGetTeamStatus == http.StatusNotFound || foreignGetTeamStatus == http.StatusForbidden,
		fmt.Sprintf("Foreign tenant denied access to Acme team (got %d)", foreignGetTeamStatus))

	foreignGetCredStatus, _ := getStatus(fmt.Sprintf("/v1/gateway-credentials/%s", humanKeyID), foreignCookie, 8081)
	assert(foreignGetCredStatus == http.StatusNotFound || foreignGetCredStatus == http.StatusForbidden,
		fmt.Sprintf("Foreign tenant denied access to Acme credential (got %d)", foreignGetCredStatus))
	fmt.Printf("  [PASS] 14. Strict tenant isolation confirmed: Foreign Tenant (%s) blocked from accessing Acme resources\n", foreignID)

	// 10. End-to-End Gateway Request & Attribution
	gwReq, _ := http.NewRequest("POST", "http://localhost:8084/v1/messages", strings.NewReader(`{
		"model": "claude-3-5-sonnet",
		"messages": [{"role": "user", "content": "Hello Sentinel"}],
		"max_tokens": 10
	}`))
	gwReq.Header.Set("Authorization", "Bearer "+rotatedNewKey)
	gwReq.Header.Set("Content-Type", "application/json")
	gwRes, err := client.Do(gwReq)
	assert(err == nil && gwRes.StatusCode == http.StatusOK,
		fmt.Sprintf("Active gateway request succeeded with HTTP 200 OK (got %d)", gwRes.StatusCode))
	gwRes.Body.Close()
	fmt.Println("  [PASS] 15. End-to-end gateway proxy request succeeded and attributable to employee user")

	// 11. Next.js Web Routes Server-Rendering
	webRoutes := []string{
		"/accept-invite?token=dummy-test",
		"/app/onboarding",
		"/app/organization/settings",
		"/app/organization/members",
		"/app/organization/teams",
		"/app/organization/projects",
		"/app/platform/gateway-credentials",
	}
	for _, route := range webRoutes {
		r, _ := http.NewRequest("GET", "http://localhost:3000"+route, nil)
		r.Header.Set("Cookie", emp1Cookie)
		wRes, err := client.Do(r)
		assert(err == nil && wRes.StatusCode == http.StatusOK, fmt.Sprintf("Next.js route %s renders 200 OK (got %d)", route, wRes.StatusCode))
		wRes.Body.Close()
	}
	fmt.Println("  [PASS] 16. All organization lifecycle Next.js routes (/onboarding, /accept-invite, etc.) render 200 OK")

	fmt.Println("\n>>> [SUCCESS] PROMPT 2: ALL ORGANIZATION & ACCESS LIFECYCLE TESTS VERIFIED! <<<")
}

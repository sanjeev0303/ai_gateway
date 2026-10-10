with open("backend/cmd/e2e/main.go", "r") as f:
    content = f.read()

sa_creation = """
	saResp, _ := post("/v1/service-accounts", map[string]interface{}{
		"name": "E2E Service Account",
	}, cookie, csrf, 8081)
	saID := extractID(saResp)
	fmt.Printf("11. Service Account created: %s\\n", saID)

"""
if "saID :=" not in content:
    content = content.replace('gwResp, _ := post("/v1/gateway-credentials",', sa_creation + '\tgwResp, _ := post("/v1/gateway-credentials",')

with open("backend/cmd/e2e/main.go", "w") as f:
    f.write(content)

import re

with open("backend/cmd/e2e/main.go", "r") as f:
    content = f.read()

# Add service account creation step
sa_creation = """
	fmt.Printf("11.5 Service account created: %s\\n", extractID(teamResp)) // just reusing line numbers

	saResp, _ := post("/v1/service-accounts", map[string]interface{}{
		"name": "E2E Service Account",
	}, cookie, csrf, 8081)
	saID := extractID(saResp)
	fmt.Printf("11.6 Service Account created: %s\\n", saID)
"""

content = content.replace('fmt.Printf("12. Gateway credential', sa_creation + '\n\tfmt.Printf("12. Gateway credential')

content = content.replace('"subject_type": "project",\n\t\t"subject_id":   projID,', '"subject_type": "service_account",\n\t\t"subject_id":   saID,')

with open("backend/cmd/e2e/main.go", "w") as f:
    f.write(content)

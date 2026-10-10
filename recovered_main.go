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

// Command busbench is a dedicated, database-free latency/throughput benchmarker
// for the gateway hot path. Unlike cmd/perf it opens NO connection to the
// application database, so the system-under-test's Postgres serves only the
// system — removing the shared-DB contention that inflated variance in the
// earlier bus comparison. It does its own minimal tenant setup via the control
// API, then runs a closed-loop load against the gateway over many iterations
// (with warm-up) and reports trimmed median / IQR per metric.
//
//	go run ./cmd/busbench -control http://localhost:8081 -gateway http://localhost:8084 \
//	    -upstream http://localhost:9098 -workers 8 -requests 400 -iterations 10 -warmup 2
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	controlURL = flag.String("control", "http://localhost:8081", "control-api base URL")
	gatewayURL = flag.String("gateway", "http://localhost:8084", "gateway-v2 base URL")
	upstream   = flag.String("upstream", "http://localhost:9098", "mock upstream URL (provider base_url)")
	model      = flag.String("model", "claude-3-haiku-20240307", "model to send (must be allowlisted by the policy busbench creates)")
	workers    = flag.Int("workers", 8, "concurrent workers")
	requests   = flag.Int("requests", 400, "requests per iteration")
	iterations = flag.Int("iterations", 10, "measured iterations")
	warmup     = flag.Int("warmup", 2, "warm-up iterations (discarded)")
	label      = flag.String("label", "", "label for the output (e.g. outbox / nats)")
)

// client reuses connections so we measure the gateway, not TCP/TLS setup.
var client = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        256,
		MaxIdleConnsPerHost: 256,
		IdleConnTimeout:     90 * time.Second,
	},
}

func main() {
	flag.Parse()
	token := setup()
	body := []byte(fmt.Sprintf(`{"model":%q,"max_tokens":16,"messages":[{"role":"user","content":"busbench"}]}`, *model))

	fmt.Printf("# busbench %s  workers=%d requests=%d iterations=%d (warmup=%d)\n", *label, *workers, *requests, *iterations, *warmup)
	fmt.Printf("%-5s %10s %8s %8s %8s\n", "iter", "rps", "p50ms", "p95ms", "p99ms")

	var rps, p50, p95, p99 []float64
	total := *warmup + *iterations
	for it := 1; it <= total; it++ {
		r, l50, l95, l99 := runIteration(token, body)
		tag := ""
		if it <= *warmup {
			tag = " (warmup, discarded)"
		} else {
			rps = append(rps, r)
			p50 = append(p50, l50)
			p95 = append(p95, l95)
			p99 = append(p99, l99)
		}
		fmt.Printf("%-5d %10.1f %8.2f %8.2f %8.2f%s\n", it, r, l50, l95, l99, tag)
	}

	fmt.Printf("\n# %s — trimmed summary over %d iterations (drop min+max)\n", *label, *iterations)
	report("throughput_rps", rps)
	report("p50_ms", p50)
	report("p95_ms", p95)
	report("p99_ms", p99)
}

// runIteration drives `requests` requests across `workers` and returns
// throughput and the latency percentiles of the successful requests.
func runIteration(token string, body []byte) (rps, p50, p95, p99 float64) {
	work := make(chan int, *requests)
	for i := 0; i < *requests; i++ {
		work <- i
	}
	close(work)

	var mu sync.Mutex
	lat := make([]float64, 0, *requests)
	var ok int

	start := time.Now()
	var wg sync.WaitGroup
	for w := 0; w < *workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range work {
				t0 := time.Now()
				req, _ := http.NewRequest("POST", *gatewayURL+"/v1/messages", bytes.NewReader(body))
				req.Header.Set("Authorization", "Bearer "+token)
				req.Header.Set("Content-Type", "application/json")
				res, err := client.Do(req)
				ms := float64(time.Since(t0).Microseconds()) / 1000.0
				if err != nil {
					continue
				}
				io.Copy(io.Discard, res.Body)
				res.Body.Close()
				if res.StatusCode == http.StatusOK {
					mu.Lock()
					lat = append(lat, ms)
					ok++
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start).Seconds()

	sort.Float64s(lat)
	return float64(ok) / elapsed, percentile(lat, 50), percentile(lat, 95), percentile(lat, 99)
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// report prints median, IQR, and min/max after trimming the single best and
// worst iteration — a simple guard against one-off scheduler stalls.
func report(name string, xs []float64) {
	if len(xs) == 0 {
		fmt.Printf("  %-16s (no data)\n", name)
		return
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	trimmed := s
	if len(s) >= 4 {
		trimmed = s[1 : len(s)-1] // drop min + max
	}
	fmt.Printf("  %-16s median=%9.2f  IQR=[%.2f..%.2f]  min=%.2f  max=%.2f  (trimmed n=%d)\n",
		name, median(trimmed), percentile(trimmed, 25), percentile(trimmed, 75), s[0], s[len(s)-1], len(trimmed))
}

func median(s []float64) float64 {
	if len(s) == 0 {
		return 0
	}
	c := append([]float64(nil), s...)
	sort.Float64s(c)
	m := len(c) / 2
	if len(c)%2 == 1 {
		return c[m]
	}
	return (c[m-1] + c[m]) / 2
}

// ---- minimal control-api setup (no database connection) --------------------

func setup() string {
	email := fmt.Sprintf("busbench_%d@example.com", time.Now().UnixNano())
	cookie, csrf := signup(email)

	provPayload := map[string]any{"provider_kind": "anthropic", "name": "busbench"}
	if *upstream != "" {
		provPayload["base_url"] = *upstream
	}
	provID := extractID(post("/v1/provider-accounts", provPayload, cookie, csrf))
	post(fmt.Sprintf("/v1/provider-accounts/%s/credentials", provID), map[string]any{"name": "k", "kind": "api_key", "secret": "sk-dummy-key"}, cookie, csrf)

	projID := extractID(post("/v1/projects", map[string]any{"name": "busbench"}, cookie, csrf))
	teamID := extractID(post("/v1/teams", map[string]any{"name": "busbench"}, cookie, csrf))
	saID := extractID(post("/v1/service-accounts", map[string]any{"name": "busbench"}, cookie, csrf))

	gw := post("/v1/gateway-credentials", map[string]any{
		"name": "busbench", "subject_type": "service_account", "subject_id": saID,
		"project_id": projID, "team_id": teamID, "scopes": []string{"gateway"},
	}, cookie, csrf)
	token := extractString(gw, "key_prefix")
	if token == "" {
		log.Fatalf("busbench: no gateway token in %s", gw)
	}

	polID := extractID(post("/v1/policies", map[string]any{"name": "busbench", "enforcement": "active", "priority": 10}, cookie, csrf))
	post(fmt.Sprintf("/v1/policies/%s/rules", polID), map[string]any{
		"rule_type": "allowlist", "description": "allow bench model",
		"parameters": map[string]any{"models": []string{*model}}, "enabled": true,
	}, cookie, csrf)
	// A high rate limit so the benchmark measures the hot path, not throttling
	// (429s short-circuit before the upstream forward and the usage publish, so
	// they would never exercise the event bus we are comparing).
	post(fmt.Sprintf("/v1/policies/%s/rules", polID), map[string]any{
		"rule_type": "rate_limit", "description": "high limit for bench",
		"parameters": map[string]any{"limit": 100000000, "window_seconds": 60}, "enabled": true,
	}, cookie, csrf)
	time.Sleep(400 * time.Millisecond) // let the policy snapshot compile
	return token
}

func signup(email string) (cookie, csrf string) {
	resp, headers := postH("/v1/auth/signup", map[string]any{
		"email": email, "password": "Password123!", "display_name": "Bus Bench", "organization_name": "busbench",
	}, "", "")
	for _, c := range headers.Values("Set-Cookie") {
		if parts := strings.Split(c, ";"); strings.HasPrefix(parts[0], "session=") {
			cookie = parts[0]
		}
	}
	var out map[string]any
	_ = json.Unmarshal([]byte(resp), &out)
	csrf, _ = out["csrf_token"].(string)
	return cookie, csrf
}

func post(path string, body any, cookie, csrf string) string {
	resp, _ := postH(path, body, cookie, csrf)
	return resp
}

func postH(path string, body any, cookie, csrf string) (string, http.Header) {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", *controlURL+path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	res, err := client.Do(req)
	if err != nil {
		log.Fatalf("busbench: control-api %s: %v", path, err)
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	return string(out), res.Header
}

func extractID(resp string) string { return extractString(resp, "id") }

func extractString(resp, key string) string {
	var m map[string]any
	if json.Unmarshal([]byte(resp), &m) != nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	for _, v := range m {
		if nested, ok := v.(map[string]any); ok {
			if nv, ok := nested[key].(string); ok {
				return nv
			}
		}
	}
	return ""
}

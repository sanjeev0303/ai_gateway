package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	orgIDStr    = "cb4f2b7b-78cf-4c7e-b376-6be86f484d59" // Sanju's organization
	sanjuUserID = "f003ed8c-52d0-4360-8789-8c7f013affb2"
	user1ID     = "72246b21-712d-4778-925d-e9dabe92f31b"
	user2ID     = "ab6ebdbb-4afe-48c4-b246-651c5f641d09"
	user3ID     = "e7e038a2-848a-4132-8680-e6fc8c97a03d"

	projCoreID = "e5ba505e-06fb-4698-99fa-3fa9ef829070" // Gemini Core Applications
	projAIID   = "9f3dfc81-08b9-4ab1-82ef-df4718b07a8b" // Gemini AI Research

	dbURL = "postgres://gateway:gateway@localhost:5432/ai_gateway?sslmode=disable"
)

type turn struct {
	role    string
	content string
	tokens  int
}

type convSeed struct {
	title     string
	userID    string
	projID    string
	privacy   string
	startedAt time.Time
	messages  []turn
}

func main() {
	ctx := context.Background()
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Println("Connected to PostgreSQL successfully.")

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Fatalf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// 1. Bind Tenant Context for RLS
	if _, err := tx.ExecContext(ctx, "SELECT set_config('app.org_id', $1, true)", orgIDStr); err != nil {
		log.Fatalf("failed to set tenant context: %v", err)
	}
	log.Println("Bound tenant context to organization:", orgIDStr)

	// 2. Create Teams if they do not exist
	team1ID := uuid.New()
	team2ID := uuid.New()

	row := tx.QueryRowContext(ctx, "SELECT id FROM teams WHERE organization_id = $1 AND name = $2", orgIDStr, "Platform Engineering")
	if err := row.Scan(&team1ID); err != nil {
		team1ID = uuid.New()
		_, err := tx.ExecContext(ctx, `
			INSERT INTO teams (id, organization_id, name, description, created_at, updated_at)
			VALUES ($1, $2, 'Platform Engineering', 'Core platform infrastructure, networking, and gateway operations', NOW() - INTERVAL '14 days', NOW())
		`, team1ID, orgIDStr)
		if err != nil {
			log.Fatalf("failed to create team 1: %v", err)
		}
		log.Printf("Created team Platform Engineering: %s\n", team1ID)
	}

	row = tx.QueryRowContext(ctx, "SELECT id FROM teams WHERE organization_id = $1 AND name = $2", orgIDStr, "Core AI Research")
	if err := row.Scan(&team2ID); err != nil {
		team2ID = uuid.New()
		_, err := tx.ExecContext(ctx, `
			INSERT INTO teams (id, organization_id, name, description, created_at, updated_at)
			VALUES ($1, $2, 'Core AI Research', 'Advanced AI agent orchestration, reasoning evaluations, and fine-tuning', NOW() - INTERVAL '14 days', NOW())
		`, team2ID, orgIDStr)
		if err != nil {
			log.Fatalf("failed to create team 2: %v", err)
		}
		log.Printf("Created team Core AI Research: %s\n", team2ID)
	}

	// 3. Add Members to Teams
	memberships := []struct {
		teamID string
		userID string
		role   string
	}{
		{team1ID.String(), sanjuUserID, "admin"},
		{team1ID.String(), user1ID, "member"},
		{team1ID.String(), user2ID, "member"},
		{team2ID.String(), sanjuUserID, "admin"},
		{team2ID.String(), user2ID, "member"},
		{team2ID.String(), user3ID, "member"},
	}

	for _, m := range memberships {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO team_members (team_id, organization_id, user_id, role, created_at)
			VALUES ($1, $2, $3, $4, NOW() - INTERVAL '10 days')
			ON CONFLICT (team_id, user_id) DO NOTHING
		`, m.teamID, orgIDStr, m.userID, m.role)
		if err != nil {
			log.Printf("warning inserting team member %s/%s: %v\n", m.teamID, m.userID, err)
		}
	}
	log.Println("Populated team memberships.")

	// 4. Distribute llm_requests across past 7 days, projects, and teams
	now := time.Now().UTC()
	rows, err := tx.QueryContext(ctx, "SELECT id FROM llm_requests WHERE organization_id = $1 ORDER BY created_at ASC", orgIDStr)
	if err == nil {
		var reqIDs []uuid.UUID
		for rows.Next() {
			var rid uuid.UUID
			if err := rows.Scan(&rid); err == nil {
				reqIDs = append(reqIDs, rid)
			}
		}
		rows.Close()

		log.Printf("Distributing %d llm_requests across 7-day timeline and teams...\n", len(reqIDs))
		for i, rid := range reqIDs {
			// Spread across past 6 days to today
			dayOffset := (i % 7)
			reqTime := now.AddDate(0, 0, -dayOffset).Add(-time.Duration((i*17)%1440) * time.Minute)
			compTime := reqTime.Add(1200 * time.Millisecond)

			projID := projCoreID
			teamID := team1ID.String()
			if i%2 == 1 {
				projID = projAIID
				teamID = team2ID.String()
			}

			if _, err := tx.ExecContext(ctx, `
				UPDATE llm_requests
				SET project_id = $1, team_id = $2, created_at = $3, completed_at = $4
				WHERE id = $5
			`, projID, teamID, reqTime, compTime, rid); err != nil {
				log.Fatalf("failed updating llm_request %s: %v", rid, err)
			}
		}
	}

	// 5. Refresh usage_hourly rollups for the organization
	if _, err := tx.ExecContext(ctx, "DELETE FROM usage_hourly WHERE organization_id = $1", orgIDStr); err != nil {
		log.Fatalf("failed deleting from usage_hourly: %v", err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO usage_hourly (organization_id, project_id, model, bucket_start, requests, prompt_tokens, completion_tokens, cached_input_tokens, total_tokens, cost_micros, updated_at)
		SELECT r.organization_id,
		       r.project_id,
		       r.model,
		       date_trunc('hour', r.created_at, 'UTC') AS bucket_start,
		       COUNT(*) AS requests,
		       COALESCE(SUM(r.prompt_tokens), 0) AS prompt_tokens,
		       COALESCE(SUM(r.completion_tokens), 0) AS completion_tokens,
		       COALESCE(SUM(r.cached_input_tokens), 0) AS cached_input_tokens,
		       COALESCE(SUM(r.total_tokens), 0) AS total_tokens,
		       COALESCE(SUM(r.cost_micros), 0) AS cost_micros,
		       now() AS updated_at
		FROM llm_requests r
		WHERE r.organization_id = $1 AND r.status = 'succeeded' AND r.project_id IS NOT NULL
		GROUP BY r.organization_id, r.project_id, r.model, date_trunc('hour', r.created_at, 'UTC')
	`, orgIDStr)
	if err != nil {
		log.Fatalf("failed rebuilding usage_hourly: %v\n", err)
	}
	log.Println("Recomputed usage_hourly rollups.")

	// 6. Delete previous seeded conversations for clean idempotent refresh
	_, _ = tx.ExecContext(ctx, "DELETE FROM conversations WHERE organization_id = $1", orgIDStr)

	// 7. Seed 8 Rich Conversations & Turns
	seeds := []convSeed{
		{
			title:     "Distributed Consensus & Raft Implementation",
			userID:    sanjuUserID,
			projID:    projCoreID,
			privacy:   "FULL_CONTENT",
			startedAt: now.Add(-2 * time.Hour),
			messages: []turn{
				{role: "user", content: "Can you explain how log replication safety is maintained in the Raft consensus algorithm?", tokens: 24},
				{role: "assistant", content: "In Raft, log replication safety is guaranteed by the Leader Completeness property: if a log entry is committed in a given term, that entry will be present in the logs of the leaders for all higher-numbered terms.\n\nKey mechanisms include:\n1. Election Restriction: A candidate can only be elected if its log is at least as up-to-date as any other voter.\n2. Overwrite Protection: Leaders never overwrite or truncate their own logs; they only append entries and force followers to synchronize.\n3. Majority Commit: An entry is only considered committed once safely replicated to a quorum.", tokens: 185},
				{role: "user", content: "How does the Leader handle discrepancies when a follower's log diverges?", tokens: 18},
				{role: "assistant", content: "When logs diverge, the Leader maintains a `nextIndex` for each follower. If an `AppendEntries` RPC fails due to an index or term mismatch, the Leader decrements `nextIndex` and retries until a matching point is found. Once found, all conflicting entries in the follower's log are overwritten with the Leader's entries.", tokens: 142},
			},
		},
		{
			title:     "Golang High Concurrency Worker Pool Design",
			userID:    user1ID,
			projID:    projCoreID,
			privacy:   "FULL_CONTENT",
			startedAt: now.Add(-5 * time.Hour),
			messages: []turn{
				{role: "user", content: "Provide an idiomatic Golang worker pool pattern that processes jobs from a channel with graceful shutdown context.", tokens: 28},
				{role: "assistant", content: "```go\ntype Pool struct {\n\tjobs    chan Job\n\twg      sync.WaitGroup\n\tctx     context.Context\n\tcancel  context.CancelFunc\n}\n\nfunc NewPool(workers, queueSize int) *Pool {\n\tctx, cancel := context.WithCancel(context.Background())\n\tp := &Pool{jobs: make(chan Job, queueSize), ctx: ctx, cancel: cancel}\n\tfor i := 0; i < workers; i++ {\n\t\tp.wg.Add(1)\n\t\tgo p.worker()\n\t}\n\treturn p\n}\n```", tokens: 210},
				{role: "user", content: "How do you prevent goroutine leaks if the caller stops submitting jobs?", tokens: 19},
				{role: "assistant", content: "To prevent goroutine leaks, always close the jobs channel or cancel the context. Workers should select on both `ctx.Done()` and the incoming `jobs` channel so that in-flight jobs complete cleanly without dangling goroutines.", tokens: 98},
			},
		},
		{
			title:     "Enterprise AI Security & Data Redaction Strategy",
			userID:    user2ID,
			projID:    projAIID,
			privacy:   "PROMPT_ONLY",
			startedAt: now.Add(-12 * time.Hour),
			messages: []turn{
				{role: "user", content: "How should an enterprise architecture handle PII redaction before sending prompt data to third-party LLMs under HIPAA and GDPR?", tokens: 32},
				{role: "assistant", content: "A robust enterprise pattern employs client-side or gateway proxy tokenization (such as Microsoft Presidio or AWS Comprehend). Sensitive entities like SSNs, emails, and medical record numbers are replaced with encrypted surrogate tokens before upstream dispatch and rehydrated upon downstream reception.", tokens: 165},
				{role: "user", content: "What is the computational latency overhead of local regex vs named entity recognition models?", tokens: 22},
				{role: "assistant", content: "Compiled regex benchmarks at <1ms per 1KB payload, while local transformer NER models (e.g. RoBERTa-based) introduce 15-45ms per request on CPU or 3-8ms on GPU accelerators.", tokens: 110},
			},
		},
		{
			title:     "PostgreSQL Indexing & Partitioning Strategy",
			userID:    user3ID,
			projID:    projCoreID,
			privacy:   "METADATA_ONLY",
			startedAt: now.Add(-18 * time.Hour),
			messages: []turn{
				{role: "user", content: "Analyze declarative range partitioning vs hash partitioning for an append-only telemetric database accumulating 50M rows daily.", tokens: 29},
				{role: "assistant", content: "Range partitioning by timestamp (`created_at`) with daily or weekly intervals is optimal for time-series and append-only ledgers because it enables instantaneous partition drops for data retention and partition pruning during temporal range queries.", tokens: 145},
			},
		},
		{
			title:     "Multi-Agent Workflow & Function Calling Schema",
			userID:    sanjuUserID,
			projID:    projAIID,
			privacy:   "FULL_CONTENT",
			startedAt: now.Add(-24 * time.Hour),
			messages: []turn{
				{role: "user", content: "Compare OpenAI function calling specification with Google Gemini tool declarations for autonomous multi-agent planning.", tokens: 26},
				{role: "assistant", content: "Both models adhere to JSON Schema definitions, but Gemini's `FunctionDeclaration` structures parameters under OpenAPI 3.0 types directly and natively supports mode control (`AUTO`, `ANY`, `NONE`). OpenAI utilizes the nested `tools: [{ type: 'function', function: {...} }]` envelope with `tool_choice` modifiers.", tokens: 175},
				{role: "user", content: "Which format produces fewer hallucinations during tool selection?", tokens: 14},
				{role: "assistant", content: "Gemini 2.5 Flash and Pro exhibit higher semantic fidelity when strict schema validation is enforced, while OpenAI GPT-4o performs consistently when provided with few-shot tool examples in the system prompt.", tokens: 112},
			},
		},
		{
			title:     "Real-time Telemetry & Stream Ingestion Pipeline",
			userID:    user1ID,
			projID:    projCoreID,
			privacy:   "FULL_CONTENT",
			startedAt: now.Add(-36 * time.Hour),
			messages: []turn{
				{role: "user", content: "Design a high-throughput event pipeline using NATS JetStream and Go to handle 100,000 telemetry events per second with exactly-once delivery guarantees.", tokens: 34},
				{role: "assistant", content: "Utilize NATS JetStream deduplication windows (`Nats-Msg-Id` header keyed to idempotency tokens). Producers batch events into stream subjects, while horizontally scaled consumer groups pull in batches with explicit acknowledgments (`AckSync`).", tokens: 190},
			},
		},
		{
			title:     "Prompt Injection Vulnerability Audit & Defense",
			userID:    user2ID,
			projID:    projAIID,
			privacy:   "FULL_CONTENT",
			startedAt: now.Add(-48 * time.Hour),
			messages: []turn{
				{role: "user", content: "Evaluate the risk of indirect prompt injection when ingesting untrusted markdown files from web scrapes.", tokens: 23},
				{role: "assistant", content: "Indirect prompt injection occurs when adversarial instructions embedded in retrieved content override system instructions. Effective defenses include XML delimiters (`<untrusted_content>`), secondary LLM moderation classifiers, and strict tool execution permission gates.", tokens: 155},
			},
		},
		{
			title:     "Multimodal Vision & Document Extraction",
			userID:    user3ID,
			projID:    projAIID,
			privacy:   "OFF",
			startedAt: now.Add(-60 * time.Hour),
			messages: []turn{
				{role: "user", content: "How does Gemini 2.5 process high-resolution PDF invoices containing nested financial tables?", tokens: 21},
				{role: "assistant", content: "Gemini natively renders PDF pages into high-fidelity image tiles, analyzing both spatial layout and OCR text simultaneously. It outputs structured JSON schemas mapping line items, taxes, totals, and invoice metadata with high spatial extraction precision.", tokens: 168},
			},
		},
	}

	for _, s := range seeds {
		convID := uuid.New()
		updatedAt := s.startedAt.Add(10 * time.Minute)
		_, err := tx.ExecContext(ctx, `
			INSERT INTO conversations (id, organization_id, user_id, project_id, title, status, model, provider, privacy_state, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, 'active', 'gemini-2.5-flash', 'gemini', $6, $7, $8)
		`, convID, orgIDStr, s.userID, s.projID, s.title, s.privacy, s.startedAt, updatedAt)
		if err != nil {
			log.Fatalf("failed to insert conversation %q: %v", s.title, err)
		}

		turnTime := s.startedAt
		for _, m := range s.messages {
			msgID := uuid.New()
			_, err := tx.ExecContext(ctx, `
				INSERT INTO conversation_messages (id, conversation_id, organization_id, user_id, role, content, tokens, created_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			`, msgID, convID, orgIDStr, s.userID, m.role, m.content, m.tokens, turnTime)
			if err != nil {
				log.Fatalf("failed to insert message for %q: %v", s.title, err)
			}
			turnTime = turnTime.Add(45 * time.Second)
		}
		log.Printf("Seeded conversation %q with %d messages (Privacy: %s)\n", s.title, len(s.messages), s.privacy)
	}

	// 8. Seed Security Events for Sanju's organization
	_, _ = tx.ExecContext(ctx, "DELETE FROM security_events WHERE organization_id = $1", orgIDStr)
	secEvents := []struct {
		category string
		severity string
		title    string
		source   string
		detail   map[string]any
		tOffset  time.Duration
	}{
		{
			category: "provider_incident",
			severity: "info",
			title:    "Google Gemini Health Check Operational",
			source:   "health_probe",
			detail:   map[string]any{"provider": "gemini", "status": "healthy", "latency_ms": 284},
			tOffset:  2 * time.Hour,
		},
		{
			category: "policy_violation",
			severity: "warning",
			title:    "Model Access Blocklist Enforced",
			source:   "gateway",
			detail:   map[string]any{"model": "claude-3-5-sonnet", "action": "blocked", "policy": "Model Access Control"},
			tOffset:  8 * time.Hour,
		},
		{
			category: "policy_violation",
			severity: "warning",
			title:    "Prompt Injection Pattern Intercepted",
			source:   "guardrail_evaluator",
			detail:   map[string]any{"guardrail": "Prompt Injection Defense", "score": 0.94, "action": "blocked"},
			tOffset:  16 * time.Hour,
		},
		{
			category: "provider_incident",
			severity: "info",
			title:    "Upstream Latency Target SLA Compliant",
			source:   "telemetry_worker",
			detail:   map[string]any{"provider": "gemini", "p95_ms": 1611, "slo_target_ms": 3000},
			tOffset:  24 * time.Hour,
		},
	}

	for _, se := range secEvents {
		detailJSON, _ := json.Marshal(se.detail)
		_, err := tx.ExecContext(ctx, `
			INSERT INTO security_events (id, organization_id, category, severity, title, source, detail, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, uuid.New(), orgIDStr, se.category, se.severity, se.title, se.source, detailJSON, now.Add(-se.tOffset))
		if err != nil {
			log.Printf("warning inserting security event %q: %v\n", se.title, err)
		}
	}
	log.Println("Seeded security events.")

	// 9. Seed Alert Events for Sanju's organization
	_, _ = tx.ExecContext(ctx, "DELETE FROM alert_events WHERE organization_id = $1", orgIDStr)
	// Find alerts in org
	alertRows, err := tx.QueryContext(ctx, "SELECT id, alert_type, name, severity FROM alerts WHERE organization_id = $1", orgIDStr)
	if err == nil {
		type alertRecord struct {
			id       uuid.UUID
			aType    string
			name     string
			severity string
		}
		var alertsList []alertRecord
		for alertRows.Next() {
			var ar alertRecord
			if err := alertRows.Scan(&ar.id, &ar.aType, &ar.name, &ar.severity); err == nil {
				alertsList = append(alertsList, ar)
			}
		}
		alertRows.Close()

		for _, a := range alertsList {
			_, err := tx.ExecContext(ctx, `
				INSERT INTO alert_events (id, organization_id, alert_id, alert_type, severity, status, message, context, triggered_at, created_at)
				VALUES ($1, $2, $3, $4, $5, 'triggered', $6, '{}'::jsonb, $7, $7)
			`, uuid.New(), orgIDStr, a.id, a.aType, a.severity, fmt.Sprintf("%s active condition verified on live telemetry", a.name), now.Add(-30*time.Minute))
			if err != nil {
				log.Printf("warning inserting alert event for %s: %v\n", a.name, err)
			}
		}
		log.Printf("Seeded %d alert events.\n", len(alertsList))
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("failed to commit transaction: %v", err)
	}

	fmt.Println("=================================================================")
	fmt.Println("  OBSERVABILITY SUITE SEED COMPLETED SUCCESSFULLY FOR SANJU ORG  ")
	fmt.Println("=================================================================")
}

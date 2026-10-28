package gatewayv2

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"ai-governance-saas/backend/internal/application/guardrails"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"

	"github.com/google/uuid"
)

type GuardrailEvaluator struct {
	repo       repositories.GuardrailsRepository
	eval       *guardrails.Evaluator
	events     chan domain.GuardrailEvent
	stopSignal chan struct{}
	closeOnce  sync.Once
}

func NewGuardrailEvaluator(repo repositories.GuardrailsRepository) *GuardrailEvaluator {
	g := &GuardrailEvaluator{
		repo:       repo,
		eval:       guardrails.NewEvaluator(),
		events:     make(chan domain.GuardrailEvent, 1024),
		stopSignal: make(chan struct{}),
	}
	if repo != nil {
		for i := 0; i < 2; i++ {
			go g.worker()
		}
	}
	return g
}

func (g *GuardrailEvaluator) worker() {
	for {
		select {
		case <-g.stopSignal:
			for {
				select {
				case ev := <-g.events:
					g.persist(ev)
				default:
					return
				}
			}
		case ev := <-g.events:
			g.persist(ev)
		}
	}
}

func (g *GuardrailEvaluator) persist(ev domain.GuardrailEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = g.repo.CreateEvent(ctx, ev)
}

// Close gracefully stops the background event persisters.
func (g *GuardrailEvaluator) Close() {
	g.closeOnce.Do(func() {
		close(g.stopSignal)
	})
}

type GuardrailEvalResult struct {
	Triggered       bool
	Action          domain.GuardrailAction
	MatchedCategory string
	GuardrailID     string
	Detector        domain.GuardrailDetector
}

func (g *GuardrailEvaluator) CheckInput(ctx context.Context, payload []byte, orgID, reqID string, cred GatewayCredential) *GuardrailEvalResult {
	if g.repo == nil {
		return &GuardrailEvalResult{Triggered: false}
	}

	// Fetch guardrails for all applicable target levels
	var allGuardrails []domain.Guardrail

	globals, err := g.repo.ListForEvaluation(ctx, orgID, domain.GuardrailTargetGlobal, "")
	if err != nil {
		slog.Error("failed to list global guardrails", "error", err)
	} else {
		allGuardrails = append(allGuardrails, globals...)
	}

	if cred.ProjectID != nil {
		projectGuardrails, err := g.repo.ListForEvaluation(ctx, orgID, domain.GuardrailTargetProject, cred.ProjectID.String())
		if err != nil {
			slog.Error("failed to list project guardrails", "error", err, "project_id", cred.ProjectID)
		} else {
			allGuardrails = append(allGuardrails, projectGuardrails...)
		}
	}

	if cred.TeamID != nil {
		teamGuardrails, err := g.repo.ListForEvaluation(ctx, orgID, domain.GuardrailTargetTeam, cred.TeamID.String())
		if err != nil {
			slog.Error("failed to list team guardrails", "error", err, "team_id", cred.TeamID)
		} else {
			allGuardrails = append(allGuardrails, teamGuardrails...)
		}
	}

	if cred.ID != uuid.Nil {
		credGuardrails, err := g.repo.ListForEvaluation(ctx, orgID, domain.GuardrailTargetGatewayCredential, cred.ID.String())
		if err != nil {
			slog.Error("failed to list credential guardrails", "error", err, "credential_id", cred.ID)
		} else {
			allGuardrails = append(allGuardrails, credGuardrails...)
		}
	}

	if len(allGuardrails) == 0 {
		return &GuardrailEvalResult{Triggered: false}
	}

	// Lightweight token approximation (4 characters per token heuristic) without JSON parse overhead
	tokenCount := len(payload) / 4
	if tokenCount < 1 && len(payload) > 0 {
		tokenCount = 1
	}

	res := g.eval.Evaluate(ctx, string(payload), tokenCount, allGuardrails)
	if res != nil {
		ev := domain.GuardrailEvent{
			ID:              uuid.New().String(),
			OrganizationID:  orgID,
			GuardrailID:     res.GuardrailID,
			RequestID:       reqID,
			Direction:       domain.GuardrailEventDirectionInput,
			Detector:        res.Detector,
			Action:          res.Action,
			MatchedCategory: res.MatchedCategory,
			Timestamp:       time.Now(),
		}
		select {
		case g.events <- ev:
		default:
			slog.Warn("guardrail event queue full, dropping event", "request_id", reqID)
		}

		return &GuardrailEvalResult{
			Triggered:       true,
			Action:          res.Action,
			MatchedCategory: res.MatchedCategory,
			GuardrailID:     res.GuardrailID,
			Detector:        res.Detector,
		}
	}

	return &GuardrailEvalResult{Triggered: false}
}


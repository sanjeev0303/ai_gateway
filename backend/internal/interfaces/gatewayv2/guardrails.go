package gatewayv2

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"ai-governance-saas/backend/internal/application/guardrails"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"

	"github.com/google/uuid"
)

type GuardrailEvaluator struct {
	repo repositories.GuardrailsRepository
	eval *guardrails.Evaluator
}

func NewGuardrailEvaluator(repo repositories.GuardrailsRepository) *GuardrailEvaluator {
	return &GuardrailEvaluator{
		repo: repo,
		eval: guardrails.NewEvaluator(),
	}
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

	var tokenCount int
	var asMap map[string]any
	if err := json.Unmarshal(payload, &asMap); err == nil {
		tokenCount = len(payload) / 4
	} else {
		tokenCount = len(payload) / 4
	}

	res := g.eval.Evaluate(ctx, string(payload), tokenCount, allGuardrails)
	if res != nil {
		go func() {
			eventCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = g.repo.CreateEvent(eventCtx, domain.GuardrailEvent{
				ID:              uuid.New().String(),
				OrganizationID:  orgID,
				GuardrailID:     res.GuardrailID,
				RequestID:       reqID,
				Direction:       domain.GuardrailEventDirectionInput,
				Detector:        res.Detector,
				Action:          res.Action,
				MatchedCategory: res.MatchedCategory,
				Timestamp:       time.Now(),
			})
		}()

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

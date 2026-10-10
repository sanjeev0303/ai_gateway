package guardrails

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"ai-governance-saas/backend/internal/domain"
)

type Evaluator struct{}

func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

type EvaluationResult struct {
	Action          domain.GuardrailAction
	MatchedCategory string
	GuardrailID     string
	Detector        domain.GuardrailDetector
}

func (e *Evaluator) Evaluate(ctx context.Context, payload string, tokenCount int, guardrails []domain.Guardrail) *EvaluationResult {
	for _, g := range guardrails {
		if !g.Enabled {
			continue
		}

		matched, category := e.evaluateDetector(payload, tokenCount, g.Detector, g.Configuration)
		if matched {
			return &EvaluationResult{
				Action:          g.Action,
				MatchedCategory: category,
				GuardrailID:     g.ID,
				Detector:        g.Detector,
			}
		}
	}
	return nil
}

func (e *Evaluator) evaluateDetector(payload string, tokenCount int, detector domain.GuardrailDetector, config json.RawMessage) (bool, string) {
	switch detector {
	case domain.GuardrailDetectorPII:
		return e.evaluatePII(payload, config)
	case domain.GuardrailDetectorSecrets:
		return e.evaluateSecrets(payload, config)
	case domain.GuardrailDetectorUnsafeKeywords:
		return e.evaluateUnsafeKeywords(payload, config)
	case domain.GuardrailDetectorRequestConstraints:
		return e.evaluateConstraints(tokenCount, config)
	}
	return false, ""
}

func (e *Evaluator) evaluatePII(payload string, config json.RawMessage) (bool, string) {
	var piiConfig domain.PIIConfiguration
	if err := json.Unmarshal(config, &piiConfig); err != nil {
		return false, ""
	}

	for _, t := range piiConfig.Types {
		switch t {
		case "email":
			// extremely simple heuristic
			if strings.Contains(payload, "@") && strings.Contains(payload, ".") {
				// regex for better matching could go here
				matched, _ := regexp.MatchString(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`, payload)
				if matched {
					return true, "email"
				}
			}
		case "phone":
			// very naive check for phone numbers (just for POC)
			matched, _ := regexp.MatchString(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`, payload)
			if matched {
				return true, "phone"
			}
		case "ssn":
			matched, _ := regexp.MatchString(`\b\d{3}-\d{2}-\d{4}\b`, payload)
			if matched {
				return true, "ssn"
			}
		}
	}
	return false, ""
}

func (e *Evaluator) evaluateSecrets(payload string, config json.RawMessage) (bool, string) {
	var secretsConfig domain.SecretsConfiguration
	if err := json.Unmarshal(config, &secretsConfig); err != nil {
		return false, ""
	}

	if secretsConfig.DetectBearerTokens {
		if strings.Contains(strings.ToLower(payload), "bearer ") {
			// naive
			matched, _ := regexp.MatchString(`(?i)bearer\s+[a-zA-Z0-9\-\._~+/]+=*`, payload)
			if matched {
				return true, "bearer_token"
			}
		}
	}
	if secretsConfig.DetectApiKeys {
		// naive AWS or generic key regex
		matched, _ := regexp.MatchString(`(?i)(api[_-]?key|secret|password|passwd|pwd)[\s:=]+["']?[a-zA-Z0-9\-_]{16,}["']?`, payload)
		if matched {
			return true, "api_key"
		}
	}
	return false, ""
}

func (e *Evaluator) evaluateUnsafeKeywords(payload string, config json.RawMessage) (bool, string) {
	var kwConfig domain.UnsafeKeywordsConfiguration
	if err := json.Unmarshal(config, &kwConfig); err != nil {
		return false, ""
	}

	lowerPayload := strings.ToLower(payload)
	for _, kw := range kwConfig.Keywords {
		if strings.Contains(lowerPayload, strings.ToLower(kw)) {
			return true, kw
		}
	}
	return false, ""
}

func (e *Evaluator) evaluateConstraints(tokenCount int, config json.RawMessage) (bool, string) {
	var reqConfig domain.RequestConstraintsConfiguration
	if err := json.Unmarshal(config, &reqConfig); err != nil {
		return false, ""
	}

	if reqConfig.MaxInputTokens != nil && tokenCount > *reqConfig.MaxInputTokens {
		return true, "max_input_tokens"
	}
	// Output tokens typically checked post-generation
	if reqConfig.MaxOutputTokens != nil && tokenCount > *reqConfig.MaxOutputTokens {
		return true, "max_output_tokens"
	}

	return false, ""
}

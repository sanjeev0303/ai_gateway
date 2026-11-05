package guardrails

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"ai-governance-saas/backend/internal/domain"
)

// Compiled once: the gateway evaluates these on every request.
var (
	// 415-555-0134, (415) 555 0134, +1 415 555 0134, +91 98765 43210 style numbers,
	// standing on their own: digits inside a key, hash or id are not a phone number.
	phonePattern = regexp.MustCompile(`(?:^|[^0-9A-Za-z_])(?:\+\d{1,3}[\s.-]?)?(?:(?:\(\d{3}\)|\d{3,5})[\s.-]?\d{3,5}[\s.-]?\d{4}|\d{5}[\s.-]\d{5})(?:[^0-9A-Za-z_]|$)`)
	// Provider and gateway keys that are recognisable without a label in front of them.
	providerKeyPattern = regexp.MustCompile(`\b(?:sk-(?:ant-|proj-)?[A-Za-z0-9_\-]{20,}|AKIA[0-9A-Z]{16}|AIza[0-9A-Za-z_\-]{35}|hf_[A-Za-z0-9]{30,}|gh[pousr]_[A-Za-z0-9]{36,}|xox[abprs]-[A-Za-z0-9-]{10,}|aigw_[A-Za-z0-9_\-]{16,})`)
	privateKeyPattern  = regexp.MustCompile(`-----BEGIN (?:RSA |EC |DSA |OPENSSH |ENCRYPTED )?PRIVATE KEY-----`)

	emailPattern       = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	ssnPattern         = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	bearerPattern      = regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9\-\._~+/]+=*`)
	labelledKeyPattern = regexp.MustCompile(`(?i)(api[_-]?key|secret|password|passwd|pwd)[\s:=]+["']?[a-zA-Z0-9\-_]{16,}["']?`)
	privateKeyBlock    = regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?(?:-----END [A-Z ]*PRIVATE KEY-----|$)`)
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

// Evaluate returns the strongest match: a block from one guardrail wins over a
// redact or warn from another, whatever order the guardrails are listed in.
func (e *Evaluator) Evaluate(ctx context.Context, payload string, tokenCount int, guardrails []domain.Guardrail) *EvaluationResult {
	var best *EvaluationResult
	for _, g := range guardrails {
		if !g.Enabled {
			continue
		}

		matched, category := e.evaluateDetector(payload, tokenCount, g.Detector, g.Configuration)
		if !matched {
			continue
		}
		if best == nil || actionRank(g.Action) > actionRank(best.Action) {
			best = &EvaluationResult{
				Action:          g.Action,
				MatchedCategory: category,
				GuardrailID:     g.ID,
				Detector:        g.Detector,
			}
		}
	}
	return best
}

func actionRank(action domain.GuardrailAction) int {
	switch action {
	case domain.GuardrailActionBlock:
		return 3
	case domain.GuardrailActionRedact:
		return 2
	case domain.GuardrailActionWarn:
		return 1
	}
	return 0
}

// Redact masks everything the enabled redact guardrails detect in text.
func (e *Evaluator) Redact(text string, guardrails []domain.Guardrail) string {
	for _, g := range guardrails {
		if !g.Enabled || g.Action != domain.GuardrailActionRedact {
			continue
		}
		switch g.Detector {
		case domain.GuardrailDetectorPII:
			var cfg domain.PIIConfiguration
			if json.Unmarshal(g.Configuration, &cfg) != nil {
				continue
			}
			for _, t := range cfg.Types {
				switch t {
				case "email":
					text = emailPattern.ReplaceAllString(text, "[REDACTED_EMAIL]")
				case "ssn":
					text = ssnPattern.ReplaceAllString(text, "[REDACTED_SSN]")
				case "phone":
					text = redactPhones(text)
				}
			}
		case domain.GuardrailDetectorSecrets:
			var cfg domain.SecretsConfiguration
			if json.Unmarshal(g.Configuration, &cfg) != nil {
				continue
			}
			if cfg.DetectPrivateKeys {
				text = privateKeyBlock.ReplaceAllString(text, "[REDACTED_PRIVATE_KEY]")
			}
			if cfg.DetectBearerTokens {
				text = bearerPattern.ReplaceAllString(text, "Bearer [REDACTED]")
			}
			if cfg.DetectApiKeys {
				text = labelledKeyPattern.ReplaceAllString(text, "$1: [REDACTED]")
				text = providerKeyPattern.ReplaceAllString(text, "[REDACTED_KEY]")
			}
		case domain.GuardrailDetectorUnsafeKeywords:
			var cfg domain.UnsafeKeywordsConfiguration
			if json.Unmarshal(g.Configuration, &cfg) != nil {
				continue
			}
			for _, kw := range cfg.Keywords {
				if kw == "" {
					continue
				}
				text = regexp.MustCompile(`(?i)`+regexp.QuoteMeta(kw)).ReplaceAllString(text, "[REDACTED]")
			}
		}
	}
	return text
}

// redactPhones masks phone numbers but keeps the character on either side,
// which phonePattern matches to tell a phone number from digits inside a key.
// Two numbers one space apart share that space, so it repeats until nothing
// is left; the placeholder has no digits, so the loop always ends.
func redactPhones(text string) string {
	for {
		next := phonePattern.ReplaceAllStringFunc(text, func(m string) string {
			start, end := 0, len(m)
			if !isPhoneChar(m[start]) {
				start++
			}
			if end > start && !isPhoneChar(m[end-1]) {
				end--
			}
			return m[:start] + "[REDACTED_PHONE]" + m[end:]
		})
		if next == text {
			return text
		}
		text = next
	}
}

func isPhoneChar(c byte) bool {
	return (c >= '0' && c <= '9') || c == '+' || c == '(' || c == ')'
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
			if emailPattern.MatchString(payload) {
				return true, "email"
			}
		case "phone":
			if phonePattern.MatchString(payload) {
				return true, "phone"
			}
		case "ssn":
			if ssnPattern.MatchString(payload) {
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
		if matched || providerKeyPattern.MatchString(payload) {
			return true, "api_key"
		}
	}
	if secretsConfig.DetectPrivateKeys && privateKeyPattern.MatchString(payload) {
		return true, "private_key"
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
	if reqConfig.MaxTokens != nil && tokenCount > *reqConfig.MaxTokens {
		return true, "max_input_tokens"
	}
	// Output tokens typically checked post-generation
	if reqConfig.MaxOutputTokens != nil && tokenCount > *reqConfig.MaxOutputTokens {
		return true, "max_output_tokens"
	}

	return false, ""
}

package domain

import (
	"encoding/json"
	"time"
)

type GuardrailAction string

const (
	GuardrailActionAllow  GuardrailAction = "allow"
	GuardrailActionWarn   GuardrailAction = "warn"
	GuardrailActionBlock  GuardrailAction = "block"
	GuardrailActionRedact GuardrailAction = "redact"
)

type GuardrailDetector string

const (
	GuardrailDetectorPII                GuardrailDetector = "pii"
	GuardrailDetectorSecrets            GuardrailDetector = "secrets"
	GuardrailDetectorUnsafeKeywords     GuardrailDetector = "unsafe_keywords"
	GuardrailDetectorRequestConstraints GuardrailDetector = "request_constraints"
)

type GuardrailTargetType string

const (
	GuardrailTargetGlobal            GuardrailTargetType = "global"
	GuardrailTargetTeam              GuardrailTargetType = "team"
	GuardrailTargetProject           GuardrailTargetType = "project"
	GuardrailTargetGatewayCredential GuardrailTargetType = "gateway_credential"
)

type Guardrail struct {
	ID             string
	OrganizationID string
	Name           string
	Description    string
	TargetType     GuardrailTargetType
	TargetID       *string // Nil if TargetType is global
	Detector       GuardrailDetector
	Action         GuardrailAction
	Configuration  json.RawMessage
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type GuardrailEventDirection string

const (
	GuardrailEventDirectionInput  GuardrailEventDirection = "input"
	GuardrailEventDirectionOutput GuardrailEventDirection = "output"
)

type GuardrailEvent struct {
	ID              string
	OrganizationID  string
	GuardrailID     string
	RequestID       string
	Direction       GuardrailEventDirection
	Detector        GuardrailDetector
	Action          GuardrailAction
	MatchedCategory string
	Timestamp       time.Time
}

// Detector Configurations

type PIIConfiguration struct {
	Types []string `json:"types"` // "email", "phone", "ssn"
}

type SecretsConfiguration struct {
	DetectApiKeys      bool `json:"detect_api_keys"`
	DetectBearerTokens bool `json:"detect_bearer_tokens"`
	DetectPasswords    bool `json:"detect_passwords"`
}

type UnsafeKeywordsConfiguration struct {
	Keywords []string `json:"keywords"`
}

type RequestConstraintsConfiguration struct {
	MaxInputTokens  *int `json:"max_input_tokens"`
	MaxOutputTokens *int `json:"max_output_tokens"`
}

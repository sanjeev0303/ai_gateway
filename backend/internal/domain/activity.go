package domain

import (
	"time"

	"github.com/google/uuid"
)

// ActivityState represents the current state of a gateway request.
type ActivityState string

const (
	ActivityStateActive    ActivityState = "active"
	ActivityStateIdle      ActivityState = "idle"
	ActivityStateCompleted ActivityState = "completed"
	ActivityStateFailed    ActivityState = "failed"
)

// ActivityEvent represents real-time activity occurring on the gateway.
type ActivityEvent struct {
	RequestID      string        `json:"request_id"`
	OrganizationID uuid.UUID     `json:"organization_id"`
	UserID         *uuid.UUID    `json:"user_id,omitempty"`
	UserName       string        `json:"user_name,omitempty"`
	ServiceAccount *uuid.UUID    `json:"service_account,omitempty"`
	ProjectID      *uuid.UUID    `json:"project_id,omitempty"`
	ProjectName    string        `json:"project_name,omitempty"`
	TeamID         *uuid.UUID    `json:"team_id,omitempty"`
	Provider       string        `json:"provider"`
	Model          string        `json:"model"`
	Status         string        `json:"status"` // e.g. success, error
	StartedAt      time.Time     `json:"started_at"`
	CurrentState   ActivityState `json:"current_state"`
}

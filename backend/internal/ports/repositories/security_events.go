package repositories

import (
	"context"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

// SecurityEventRepository appends platform-originated events and reads them
// back. Append is the only mutation this interface exposes, and the table also
// rejects UPDATE and DELETE, so the ledger stays append-only even for a caller
// that reaches past the interface.
type SecurityEventRepository interface {
	Append(ctx context.Context, orgID uuid.UUID, input domain.NewSecurityEvent) (domain.SecurityEvent, error)
	List(ctx context.Context, orgID uuid.UUID, filter domain.SecurityEventFilter, limit, offset int32) ([]domain.SecurityEvent, error)
}

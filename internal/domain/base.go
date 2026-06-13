package domain

import (
	"time"

	"github.com/google/uuid"
)

// Base carries the common audit and soft-delete columns shared by most entities.
// Identifiers and timestamps are assigned by the repository layer before insert.
type Base struct {
	ID        uuid.UUID  `json:"id"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"-"`
	CreatedBy *uuid.UUID `json:"createdBy,omitempty"`
	UpdatedBy *uuid.UUID `json:"updatedBy,omitempty"`
}

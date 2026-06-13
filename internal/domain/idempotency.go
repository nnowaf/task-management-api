package domain

import (
	"time"

	"github.com/google/uuid"
)

type IdempotencyStatus string

const (
	IdempotencyProcessing IdempotencyStatus = "processing"
	IdempotencyCompleted  IdempotencyStatus = "completed"
)

// IdempotencyRecord stores the outcome of an idempotent request so a replay with
// the same key returns the original response. Uniqueness is enforced per user via
// the composite unique index on (user_id, idempotency_key). The request/response
// bodies are stored as text (not jsonb) so a replayed response is byte-for-byte
// identical to the first — jsonb would normalize and reorder object keys.
type IdempotencyRecord struct {
	ID             uuid.UUID         `json:"id"`
	UserID         uuid.UUID         `json:"userId"`
	IdempotencyKey uuid.UUID         `json:"idempotencyKey"`
	RequestHash    string            `json:"-"`
	RequestBody    string            `json:"-"`
	Status         IdempotencyStatus `json:"status"`
	ResponseCode   int               `json:"responseCode"`
	ResponseBody   string            `json:"-"`
	CreatedAt      time.Time         `json:"createdAt"`
}

func (IdempotencyRecord) TableName() string { return "idempotency_records" }

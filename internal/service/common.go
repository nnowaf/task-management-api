package service

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/google/uuid"
)

// Actor is the authenticated user performing an operation, used for ownership,
// audit logging, and human-readable activity messages.
type Actor struct {
	ID       uuid.UUID
	Username string
}

// successEnvelope mirrors response.Success so the idempotency layer can produce a
// stored response body byte-identical to what the handler would normally return.
type successEnvelope struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// allowedSort whitelists sortable columns to keep the raw ORDER BY injection-safe.
var allowedSort = map[string]string{
	"created_at": "created_at",
	"updated_at": "updated_at",
	"due_date":   "due_date",
	"start_date": "start_date",
	"priority":   "priority",
	"title":      "title",
	"status":     "status",
}

func sanitizeSort(input string) string {
	if input == "" {
		return "created_at desc"
	}
	parts := strings.Fields(strings.ToLower(input))
	col, ok := allowedSort[parts[0]]
	if !ok {
		return "created_at desc"
	}
	dir := "asc"
	if len(parts) > 1 && parts[1] == "desc" {
		dir = "desc"
	}
	return col + " " + dir
}

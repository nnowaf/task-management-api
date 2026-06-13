package domain

import (
	"context"

	"github.com/google/uuid"
)

// TaskFilter describes the filtering, search, and pagination options for listing tasks.
type TaskFilter struct {
	OwnerID    uuid.UUID   // tasks created by this user are always visible
	TeamIDs    []uuid.UUID // plus tasks belonging to these teams
	Status     string
	Priority   string
	Search     string // case-insensitive match on title
	AssignedTo *uuid.UUID
	Page       int
	Limit      int
	Sort       string // e.g. "created_at desc"
}

type UserRepository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByLogin(ctx context.Context, login string) (*User, error) // username or email
	ExistsByUsernameOrEmail(ctx context.Context, username, email string) (bool, error)
}

type TaskRepository interface {
	Create(ctx context.Context, t *Task) error
	GetByID(ctx context.Context, id uuid.UUID) (*Task, error)
	Update(ctx context.Context, t *Task) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, f TaskFilter) ([]Task, int64, error)
}

type TaskLogRepository interface {
	Create(ctx context.Context, l *TaskLog) error
	ListByTask(ctx context.Context, taskID uuid.UUID) ([]TaskLog, error)
}

type TeamRepository interface {
	CreateMaster(ctx context.Context, t *TeamMaster) error
	GetMaster(ctx context.Context, id uuid.UUID) (*TeamMaster, error)
	ListMastersForUser(ctx context.Context, userID uuid.UUID) ([]TeamMaster, error)
	AddMember(ctx context.Context, d *TeamDetail) error
	GetMember(ctx context.Context, teamID, userID uuid.UUID) (*TeamDetail, error)
	ListMembers(ctx context.Context, teamID uuid.UUID) ([]TeamDetail, error)
	UpdateMember(ctx context.Context, d *TeamDetail) error
	RemoveMember(ctx context.Context, teamID, userID uuid.UUID) error
	TeamIDsForUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

type CommentRepository interface {
	Create(ctx context.Context, c *CommentTask) error
	GetByID(ctx context.Context, id uuid.UUID) (*CommentTask, error)
	ListByTask(ctx context.Context, taskID uuid.UUID) ([]CommentTask, error)
	Update(ctx context.Context, c *CommentTask) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	SoftDeleteByTask(ctx context.Context, taskID uuid.UUID) error
}

// IdempotencyResult reports the outcome of an atomic insert-if-absent.
type IdempotencyResult struct {
	Inserted bool               // true when this caller won the insert race
	Record   *IdempotencyRecord // the existing record when Inserted == false
}

type IdempotencyRepository interface {
	// InsertIfAbsent atomically inserts a fresh "processing" record, or returns the
	// existing non-expired record. Records older than the configured TTL are treated
	// as absent (the key may be reused).
	InsertIfAbsent(ctx context.Context, rec *IdempotencyRecord) (IdempotencyResult, error)
	// Complete finalizes a record with the response code and body.
	Complete(ctx context.Context, id uuid.UUID, code int, body string) error
	// Get returns the non-expired record for (userID, key), or nil when none exists.
	Get(ctx context.Context, userID, key uuid.UUID) (*IdempotencyRecord, error)
}

// Store aggregates all repositories and provides a transactional boundary via
// Atomic. Services depend only on this interface, never on the concrete database,
// which keeps the business logic fully unit-testable with an in-memory fake.
type Store interface {
	Users() UserRepository
	Tasks() TaskRepository
	TaskLogs() TaskLogRepository
	Teams() TeamRepository
	Comments() CommentRepository
	Idempotency() IdempotencyRepository

	// Atomic runs fn inside a single transaction. Returning a non-nil error rolls
	// back every change made through the Store passed to fn.
	Atomic(ctx context.Context, fn func(s Store) error) error
}

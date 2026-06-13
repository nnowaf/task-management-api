package repository

import (
	"context"
	"time"

	"github.com/gdcpay/task-api/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX is the subset of pgx used by the repositories. Both *pgxpool.Pool and
// pgx.Tx satisfy it, so a repository runs identically inside or outside a
// transaction depending on which the Store hands it.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Store is the pgx-backed implementation of domain.Store. Inside Atomic, db is
// the active transaction, so every repository obtained from the nested Store
// participates in that transaction.
type Store struct {
	pool *pgxpool.Pool
	db   DBTX
	ttl  time.Duration
}

var _ domain.Store = (*Store)(nil)

func NewStore(pool *pgxpool.Pool, idempotencyTTL time.Duration) *Store {
	return &Store{pool: pool, db: pool, ttl: idempotencyTTL}
}

func (s *Store) Users() domain.UserRepository       { return &userRepo{s.db} }
func (s *Store) Tasks() domain.TaskRepository       { return &taskRepo{s.db} }
func (s *Store) TaskLogs() domain.TaskLogRepository { return &taskLogRepo{s.db} }
func (s *Store) Teams() domain.TeamRepository       { return &teamRepo{s.db} }
func (s *Store) Comments() domain.CommentRepository { return &commentRepo{s.db} }

func (s *Store) Idempotency() domain.IdempotencyRepository {
	return &idempotencyRepo{db: s.db, ttl: s.ttl}
}

// Atomic runs fn inside a single transaction. Returning a non-nil error (or a
// panic) rolls back every change made through the Store passed to fn.
func (s *Store) Atomic(ctx context.Context, fn func(domain.Store) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	nested := &Store{pool: s.pool, db: tx, ttl: s.ttl}
	if err := fn(nested); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

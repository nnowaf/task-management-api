package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect opens a pooled pgx connection to PostgreSQL.
func Connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	cfg.MaxConns = 25
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}

// Migrate creates the schema and the indexes/partial-unique constraints (soft-delete
// aware uniqueness, trigram search index) with plain DDL. Every statement is
// idempotent so startup is safe to repeat.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	for _, stmt := range schemaStatements {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("migrate %q: %w", firstLine(stmt), err)
		}
	}
	return nil
}

// PruneExpiredIdempotency deletes idempotency records older than retention (past
// their TTL, hence already ignored by the application) and returns how many rows
// were removed. Intended to be called periodically by a background pruner.
func PruneExpiredIdempotency(ctx context.Context, pool *pgxpool.Pool, retention time.Duration) (int64, error) {
	tag, err := pool.Exec(ctx,
		`DELETE FROM idempotency_records WHERE created_at < $1`, time.Now().Add(-retention))
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func firstLine(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return s[:i]
		}
	}
	return s
}

var schemaStatements = []string{
	`CREATE EXTENSION IF NOT EXISTS pg_trgm`,

	`CREATE TABLE IF NOT EXISTS users (
		id          uuid PRIMARY KEY,
		name        varchar(150) NOT NULL,
		username    varchar(100) NOT NULL,
		email       varchar(150) NOT NULL,
		password    varchar(255) NOT NULL,
		created_at  timestamptz NOT NULL,
		updated_at  timestamptz NOT NULL,
		deleted_at  timestamptz,
		created_by  uuid,
		updated_by  uuid
	)`,

	`CREATE TABLE IF NOT EXISTS team_masters (
		id          uuid PRIMARY KEY,
		name        varchar(150) NOT NULL,
		created_at  timestamptz NOT NULL,
		updated_at  timestamptz NOT NULL,
		deleted_at  timestamptz,
		created_by  uuid,
		updated_by  uuid
	)`,

	`CREATE TABLE IF NOT EXISTS team_details (
		id             uuid PRIMARY KEY,
		id_team_master uuid NOT NULL,
		id_user        uuid NOT NULL,
		expired_at     timestamptz,
		status         varchar(20) NOT NULL DEFAULT 'active',
		created_at     timestamptz NOT NULL,
		updated_at     timestamptz NOT NULL,
		deleted_at     timestamptz,
		created_by     uuid,
		updated_by     uuid
	)`,

	`CREATE TABLE IF NOT EXISTS tasks (
		id          uuid PRIMARY KEY,
		title       varchar(255) NOT NULL,
		description text NOT NULL DEFAULT '',
		assigned_to uuid,
		id_team     uuid,
		priority    varchar(20) NOT NULL DEFAULT 'MEDIUM',
		status      varchar(20) NOT NULL DEFAULT 'TODO',
		start_date  timestamptz,
		due_date    timestamptz,
		created_at  timestamptz NOT NULL,
		updated_at  timestamptz NOT NULL,
		deleted_at  timestamptz,
		created_by  uuid,
		updated_by  uuid
	)`,

	`CREATE TABLE IF NOT EXISTS task_logs (
		id          uuid PRIMARY KEY,
		id_task     uuid NOT NULL,
		action_type varchar(20) NOT NULL,
		activity    text NOT NULL,
		created_at  timestamptz NOT NULL,
		created_by  uuid
	)`,

	`CREATE TABLE IF NOT EXISTS comment_tasks (
		id          uuid PRIMARY KEY,
		id_task     uuid NOT NULL,
		comment     text NOT NULL,
		created_at  timestamptz NOT NULL,
		updated_at  timestamptz NOT NULL,
		deleted_at  timestamptz,
		created_by  uuid,
		updated_by  uuid
	)`,

	`CREATE TABLE IF NOT EXISTS idempotency_records (
		id              uuid PRIMARY KEY,
		user_id         uuid NOT NULL,
		idempotency_key uuid NOT NULL,
		request_hash    varchar(64) NOT NULL,
		request_body    text NOT NULL DEFAULT '',
		status          varchar(20) NOT NULL DEFAULT 'processing',
		response_code   int NOT NULL DEFAULT 0,
		response_body   text NOT NULL DEFAULT '',
		created_at      timestamptz NOT NULL
	)`,

	// Soft-delete-aware uniqueness and lookup indexes.
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_users_username ON users (username) WHERE deleted_at IS NULL`,
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_users_email ON users (email) WHERE deleted_at IS NULL`,
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_team_member ON team_details (id_team_master, id_user) WHERE deleted_at IS NULL`,
	`CREATE INDEX IF NOT EXISTS idx_team_details_user ON team_details (id_user) WHERE deleted_at IS NULL`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_created_by ON tasks (created_by) WHERE deleted_at IS NULL`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_assigned_to ON tasks (assigned_to) WHERE deleted_at IS NULL`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_id_team ON tasks (id_team) WHERE deleted_at IS NULL`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks (status) WHERE deleted_at IS NULL`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_title_trgm ON tasks USING gin (lower(title) gin_trgm_ops)`,
	`CREATE INDEX IF NOT EXISTS idx_task_logs_id_task ON task_logs (id_task)`,
	`CREATE INDEX IF NOT EXISTS idx_comment_tasks_id_task ON comment_tasks (id_task) WHERE deleted_at IS NULL`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_idem_user_key ON idempotency_records (user_id, idempotency_key)`,
	`CREATE INDEX IF NOT EXISTS idx_idem_created_at ON idempotency_records (created_at)`,
}

package repository

import (
	"context"
	"errors"
	"time"

	"github.com/gdcpay/task-api/internal/domain"
	"github.com/google/uuid"
)

type idempotencyRepo struct {
	db  DBTX
	ttl time.Duration
}

const idemCols = `id, user_id, idempotency_key, request_hash, request_body, status, response_code, response_body, created_at`

func scanIdem(s scannable) (*domain.IdempotencyRecord, error) {
	var (
		rec    domain.IdempotencyRecord
		status string
	)
	err := s.Scan(&rec.ID, &rec.UserID, &rec.IdempotencyKey, &rec.RequestHash, &rec.RequestBody,
		&status, &rec.ResponseCode, &rec.ResponseBody, &rec.CreatedAt)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	rec.Status = domain.IdempotencyStatus(status)
	return &rec, nil
}

// InsertIfAbsent relies on the unique index (user_id, idempotency_key) for the
// cross-process exactly-once guarantee: only one INSERT wins (RowsAffected == 1),
// every other caller gets 0 and reads the existing record. A record older than the
// TTL is purged and the key treated as fresh.
func (r *idempotencyRepo) InsertIfAbsent(ctx context.Context, rec *domain.IdempotencyRecord) (domain.IdempotencyResult, error) {
	cutoff := time.Now().Add(-r.ttl)

	for attempt := 0; attempt < 2; attempt++ {
		rec.ID = uuid.New()
		rec.CreatedAt = nowUTC()
		if rec.Status == "" {
			rec.Status = domain.IdempotencyProcessing
		}

		tag, err := r.db.Exec(ctx,
			`INSERT INTO idempotency_records
			   (id, user_id, idempotency_key, request_hash, request_body, status, response_code, response_body, created_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			 ON CONFLICT (user_id, idempotency_key) DO NOTHING`,
			rec.ID, rec.UserID, rec.IdempotencyKey, rec.RequestHash, rec.RequestBody,
			string(rec.Status), rec.ResponseCode, rec.ResponseBody, rec.CreatedAt)
		if err != nil {
			return domain.IdempotencyResult{}, err
		}
		if tag.RowsAffected() == 1 {
			return domain.IdempotencyResult{Inserted: true}, nil
		}

		existing, err := scanIdem(r.db.QueryRow(ctx,
			`SELECT `+idemCols+` FROM idempotency_records WHERE user_id = $1 AND idempotency_key = $2`,
			rec.UserID, rec.IdempotencyKey))
		if err != nil {
			return domain.IdempotencyResult{}, err
		}
		if existing == nil {
			continue // lost a delete race; retry the insert
		}
		if existing.CreatedAt.Before(cutoff) {
			if _, err := r.db.Exec(ctx,
				`DELETE FROM idempotency_records WHERE id = $1`, existing.ID); err != nil {
				return domain.IdempotencyResult{}, err
			}
			continue // expired record removed; retry as fresh
		}
		return domain.IdempotencyResult{Inserted: false, Record: existing}, nil
	}

	return domain.IdempotencyResult{}, errors.New("idempotency: could not settle insert after retry")
}

func (r *idempotencyRepo) Complete(ctx context.Context, id uuid.UUID, code int, body string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE idempotency_records SET status = $2, response_code = $3, response_body = $4 WHERE id = $1`,
		id, string(domain.IdempotencyCompleted), code, body)
	return err
}

func (r *idempotencyRepo) Get(ctx context.Context, userID, key uuid.UUID) (*domain.IdempotencyRecord, error) {
	cutoff := time.Now().Add(-r.ttl)
	return scanIdem(r.db.QueryRow(ctx,
		`SELECT `+idemCols+` FROM idempotency_records
		 WHERE user_id = $1 AND idempotency_key = $2 AND created_at > $3`,
		userID, key, cutoff))
}

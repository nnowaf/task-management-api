package repository

import (
	"context"
	"time"

	"github.com/gdcpay/task-api/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type userRepo struct{ db DBTX }

const userCols = `id, name, username, email, password, created_at, updated_at, created_by, updated_by`

func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Name, &u.Username, &u.Email, &u.Password,
		&u.CreatedAt, &u.UpdatedAt, &u.CreatedBy, &u.UpdatedBy)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) Create(ctx context.Context, u *domain.User) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	now := time.Now().UTC()
	u.CreatedAt, u.UpdatedAt = now, now

	_, err := r.db.Exec(ctx,
		`INSERT INTO users (id, name, username, email, password, created_at, updated_at, created_by, updated_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		u.ID, u.Name, u.Username, u.Email, u.Password, u.CreatedAt, u.UpdatedAt, u.CreatedBy, u.UpdatedBy)
	if err != nil && isUniqueViolation(err) {
		return domain.NewConflict(domain.CodeConflict, "username or email already in use")
	}
	return err
}

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return scanUser(r.db.QueryRow(ctx,
		`SELECT `+userCols+` FROM users WHERE id = $1 AND deleted_at IS NULL`, id))
}

func (r *userRepo) GetByLogin(ctx context.Context, login string) (*domain.User, error) {
	return scanUser(r.db.QueryRow(ctx,
		`SELECT `+userCols+` FROM users
		 WHERE (username = $1 OR email = $1) AND deleted_at IS NULL
		 LIMIT 1`, login))
}

func (r *userRepo) ExistsByUsernameOrEmail(ctx context.Context, username, email string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users
		 WHERE (username = $1 OR email = $2) AND deleted_at IS NULL)`, username, email).Scan(&exists)
	return exists, err
}

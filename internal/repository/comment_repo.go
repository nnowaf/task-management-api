package repository

import (
	"context"

	"github.com/gdcpay/task-api/internal/domain"
	"github.com/google/uuid"
)

type commentRepo struct{ db DBTX }

const commentCols = `id, id_task, comment, created_at, updated_at, created_by, updated_by`

func scanComment(s scannable) (*domain.CommentTask, error) {
	var c domain.CommentTask
	err := s.Scan(&c.ID, &c.IDTask, &c.Comment, &c.CreatedAt, &c.UpdatedAt, &c.CreatedBy, &c.UpdatedBy)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *commentRepo) Create(ctx context.Context, c *domain.CommentTask) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	now := nowUTC()
	c.CreatedAt, c.UpdatedAt = now, now
	_, err := r.db.Exec(ctx,
		`INSERT INTO comment_tasks (id, id_task, comment, created_at, updated_at, created_by, updated_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		c.ID, c.IDTask, c.Comment, c.CreatedAt, c.UpdatedAt, c.CreatedBy, c.UpdatedBy)
	return err
}

func (r *commentRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.CommentTask, error) {
	return scanComment(r.db.QueryRow(ctx,
		`SELECT `+commentCols+` FROM comment_tasks WHERE id = $1 AND deleted_at IS NULL`, id))
}

func (r *commentRepo) ListByTask(ctx context.Context, taskID uuid.UUID) ([]domain.CommentTask, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+commentCols+` FROM comment_tasks
		 WHERE id_task = $1 AND deleted_at IS NULL ORDER BY created_at ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := make([]domain.CommentTask, 0)
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		comments = append(comments, *c)
	}
	return comments, rows.Err()
}

func (r *commentRepo) Update(ctx context.Context, c *domain.CommentTask) error {
	c.UpdatedAt = nowUTC()
	_, err := r.db.Exec(ctx,
		`UPDATE comment_tasks SET comment=$2, updated_at=$3, updated_by=$4
		 WHERE id=$1 AND deleted_at IS NULL`,
		c.ID, c.Comment, c.UpdatedAt, c.UpdatedBy)
	return err
}

func (r *commentRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := nowUTC()
	_, err := r.db.Exec(ctx,
		`UPDATE comment_tasks SET deleted_at=$2, updated_at=$2 WHERE id=$1 AND deleted_at IS NULL`, id, now)
	return err
}

func (r *commentRepo) SoftDeleteByTask(ctx context.Context, taskID uuid.UUID) error {
	now := nowUTC()
	_, err := r.db.Exec(ctx,
		`UPDATE comment_tasks SET deleted_at=$2, updated_at=$2 WHERE id_task=$1 AND deleted_at IS NULL`, taskID, now)
	return err
}

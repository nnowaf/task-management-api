package repository

import (
	"context"

	"github.com/gdcpay/task-api/internal/domain"
	"github.com/google/uuid"
)

type taskLogRepo struct{ db DBTX }

func (r *taskLogRepo) Create(ctx context.Context, l *domain.TaskLog) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	if l.CreatedAt.IsZero() {
		l.CreatedAt = nowUTC()
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO task_logs (id, id_task, action_type, activity, created_at, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		l.ID, l.IDTask, string(l.ActionType), l.Activity, l.CreatedAt, l.CreatedBy)
	return err
}

func (r *taskLogRepo) ListByTask(ctx context.Context, taskID uuid.UUID) ([]domain.TaskLog, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, id_task, action_type, activity, created_at, created_by
		 FROM task_logs WHERE id_task = $1 ORDER BY created_at ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]domain.TaskLog, 0)
	for rows.Next() {
		var l domain.TaskLog
		var action string
		if err := rows.Scan(&l.ID, &l.IDTask, &action, &l.Activity, &l.CreatedAt, &l.CreatedBy); err != nil {
			return nil, err
		}
		l.ActionType = domain.TaskLogAction(action)
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

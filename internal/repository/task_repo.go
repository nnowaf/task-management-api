package repository

import (
	"context"
	"strconv"
	"strings"

	"github.com/gdcpay/task-api/internal/domain"
	"github.com/google/uuid"
)

type taskRepo struct{ db DBTX }

// taskSelect joins the assignee so the response can carry a brief user summary.
const taskSelect = `
	SELECT t.id, t.title, t.description, t.assigned_to, t.id_team, t.priority, t.status,
	       t.start_date, t.due_date, t.created_at, t.updated_at, t.created_by, t.updated_by,
	       au.id, au.name, au.username
	FROM tasks t
	LEFT JOIN users au ON au.id = t.assigned_to AND au.deleted_at IS NULL`

func scanTask(s scannable) (*domain.Task, error) {
	var (
		t                domain.Task
		priority, status string
		aID              *uuid.UUID
		aName, aUsername *string
	)
	err := s.Scan(&t.ID, &t.Title, &t.Description, &t.AssignedTo, &t.IDTeam,
		&priority, &status, &t.StartDate, &t.DueDate,
		&t.CreatedAt, &t.UpdatedAt, &t.CreatedBy, &t.UpdatedBy,
		&aID, &aName, &aUsername)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	t.Priority = domain.TaskPriority(priority)
	t.Status = domain.TaskStatus(status)
	if aID != nil {
		t.Assignee = &domain.User{Base: domain.Base{ID: *aID}, Name: derefStr(aName), Username: derefStr(aUsername)}
	}
	return &t, nil
}

func (r *taskRepo) Create(ctx context.Context, t *domain.Task) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	if t.Priority == "" {
		t.Priority = domain.PriorityMedium
	}
	if t.Status == "" {
		t.Status = domain.TaskTODO
	}
	now := nowUTC()
	t.CreatedAt, t.UpdatedAt = now, now

	_, err := r.db.Exec(ctx,
		`INSERT INTO tasks (id, title, description, assigned_to, id_team, priority, status,
		                    start_date, due_date, created_at, updated_at, created_by, updated_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		t.ID, t.Title, t.Description, t.AssignedTo, t.IDTeam, string(t.Priority), string(t.Status),
		t.StartDate, t.DueDate, t.CreatedAt, t.UpdatedAt, t.CreatedBy, t.UpdatedBy)
	return err
}

func (r *taskRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	return scanTask(r.db.QueryRow(ctx, taskSelect+` WHERE t.id = $1 AND t.deleted_at IS NULL`, id))
}

func (r *taskRepo) Update(ctx context.Context, t *domain.Task) error {
	t.UpdatedAt = nowUTC()
	_, err := r.db.Exec(ctx,
		`UPDATE tasks SET title=$2, description=$3, assigned_to=$4, id_team=$5, priority=$6,
		        status=$7, start_date=$8, due_date=$9, updated_at=$10, updated_by=$11
		 WHERE id=$1 AND deleted_at IS NULL`,
		t.ID, t.Title, t.Description, t.AssignedTo, t.IDTeam, string(t.Priority), string(t.Status),
		t.StartDate, t.DueDate, t.UpdatedAt, t.UpdatedBy)
	return err
}

func (r *taskRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := nowUTC()
	_, err := r.db.Exec(ctx,
		`UPDATE tasks SET deleted_at=$2, updated_at=$2 WHERE id=$1 AND deleted_at IS NULL`, id, now)
	return err
}

// List applies visibility (own tasks plus team tasks), filters, search, and pagination.
func (r *taskRepo) List(ctx context.Context, f domain.TaskFilter) ([]domain.Task, int64, error) {
	var args []any
	idx := 0
	next := func(v any) string {
		args = append(args, v)
		idx++
		return "$" + strconv.Itoa(idx)
	}

	owner := next(f.OwnerID)
	var visibility string
	if len(f.TeamIDs) > 0 {
		ph := make([]string, len(f.TeamIDs))
		for i, id := range f.TeamIDs {
			ph[i] = next(id)
		}
		visibility = "(t.created_by = " + owner + " OR t.id_team IN (" + strings.Join(ph, ",") + "))"
	} else {
		visibility = "t.created_by = " + owner
	}

	conds := []string{"t.deleted_at IS NULL", visibility}
	if f.Status != "" {
		conds = append(conds, "t.status = "+next(f.Status))
	}
	if f.Priority != "" {
		conds = append(conds, "t.priority = "+next(f.Priority))
	}
	if f.AssignedTo != nil {
		conds = append(conds, "t.assigned_to = "+next(*f.AssignedTo))
	}
	if f.Search != "" {
		conds = append(conds, "lower(t.title) LIKE "+next("%"+strings.ToLower(f.Search)+"%"))
	}
	where := strings.Join(conds, " AND ")

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM tasks t WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []domain.Task{}, 0, nil
	}

	sort := f.Sort
	if sort == "" {
		sort = "created_at desc"
	}
	query := taskSelect + ` WHERE ` + where + ` ORDER BY t.` + sort +
		` LIMIT ` + next(f.Limit) + ` OFFSET ` + next((f.Page-1)*f.Limit)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	tasks := make([]domain.Task, 0, f.Limit)
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, *t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

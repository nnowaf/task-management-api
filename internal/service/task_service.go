package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gdcpay/task-api/internal/domain"
	"github.com/gdcpay/task-api/internal/dto"
	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

type TaskService struct {
	store    domain.Store
	notifier Notifier
	sf       singleflight.Group
}

func NewTaskService(store domain.Store, notifier Notifier) *TaskService {
	return &TaskService{store: store, notifier: notifier}
}

// CreateResult carries the HTTP status and serialized body so the handler can
// reproduce (or replay) the exact response of the first request.
type CreateResult struct {
	StatusCode int
	Body       []byte
	Replayed   bool
}

// ListTasksParams are the filter/search/pagination inputs for listing tasks.
type ListTasksParams struct {
	Status     string
	Priority   string
	Search     string
	AssignedTo *uuid.UUID
	Page       int
	Limit      int
	Sort       string
}

func (s *TaskService) CreateTaskIdempotent(ctx context.Context, actor Actor, key uuid.UUID, req dto.CreateTaskRequest) (*CreateResult, error) {
	canonical, err := json.Marshal(req)
	if err != nil {
		return nil, domain.NewInternal(err)
	}
	reqHash := sha256Hex(canonical)

	if existing, err := s.store.Idempotency().Get(ctx, actor.ID, key); err != nil {
		return nil, domain.NewInternal(err)
	} else if existing != nil {
		if existing.RequestHash != reqHash {
			return nil, domain.NewConflict(domain.CodeIdempotencyKeyReused, "Idempotency-Key was already used with a different request body")
		}
		if existing.Status == domain.IdempotencyCompleted {
			return &CreateResult{StatusCode: existing.ResponseCode, Body: []byte(existing.ResponseBody), Replayed: true}, nil
		}
	}

	v, err, _ := s.sf.Do(actor.ID.String()+":"+key.String(), func() (interface{}, error) {
		return s.createWithIdempotency(ctx, actor, key, reqHash, canonical, req)
	})
	if err != nil {
		return nil, err
	}
	return v.(*CreateResult), nil
}

func (s *TaskService) createWithIdempotency(ctx context.Context, actor Actor, key uuid.UUID, reqHash string, canonical []byte, req dto.CreateTaskRequest) (*CreateResult, error) {
	var result *CreateResult

	err := s.store.Atomic(ctx, func(tx domain.Store) error {
		rec := &domain.IdempotencyRecord{
			UserID:         actor.ID,
			IdempotencyKey: key,
			RequestHash:    reqHash,
			RequestBody:    string(canonical),
			Status:         domain.IdempotencyProcessing,
			ResponseBody:   "null",
		}

		res, err := tx.Idempotency().InsertIfAbsent(ctx, rec)
		if err != nil {
			return domain.NewInternal(err)
		}
		if !res.Inserted {
			r, rErr := resultFromExisting(res.Record, reqHash)
			if rErr != nil {
				return rErr
			}
			result = r
			return nil
		}

		task, err := s.buildAndCreateTask(ctx, tx, actor, req)
		if err != nil {
			return err
		}

		body, err := json.Marshal(successEnvelope{Status: "success", Data: dto.NewTaskResponse(task)})
		if err != nil {
			return domain.NewInternal(err)
		}
		if err := tx.Idempotency().Complete(ctx, rec.ID, http.StatusCreated, string(body)); err != nil {
			return domain.NewInternal(err)
		}
		result = &CreateResult{StatusCode: http.StatusCreated, Body: body, Replayed: false}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// resultFromExisting turns a stored record into a replayed response, or the proper
// conflict when the key is reused with a different body or is still processing.
func resultFromExisting(rec *domain.IdempotencyRecord, reqHash string) (*CreateResult, error) {
	if rec.RequestHash != reqHash {
		return nil, domain.NewConflict(domain.CodeIdempotencyKeyReused, "Idempotency-Key was already used with a different request body")
	}
	if rec.Status == domain.IdempotencyCompleted {
		return &CreateResult{StatusCode: rec.ResponseCode, Body: []byte(rec.ResponseBody), Replayed: true}, nil
	}
	return nil, domain.NewConflict(domain.CodeIdempotencyInProgress, "a request with this Idempotency-Key is still being processed")
}

// buildAndCreateTask validates assignment, inserts the task, and writes a CREATE log
// inside the caller's transaction.
func (s *TaskService) buildAndCreateTask(ctx context.Context, tx domain.Store, actor Actor, req dto.CreateTaskRequest) (*domain.Task, error) {
	priority := domain.TaskPriority(req.Priority)
	if priority == "" {
		priority = domain.PriorityMedium
	}
	status := domain.TaskStatus(req.Status)
	if status == "" {
		status = domain.TaskTODO
	}

	assignedTo := req.AssignedTo
	if req.IDTeam != nil {
		if err := s.ensureActorInTeam(ctx, tx, actor.ID, *req.IDTeam); err != nil {
			return nil, err
		}
		if assignedTo != nil {
			member, err := tx.Teams().GetMember(ctx, *req.IDTeam, *assignedTo)
			if err != nil {
				return nil, domain.NewInternal(err)
			}
			if member == nil || member.Status != domain.TeamMemberActive {
				return nil, domain.NewUnprocessable("assignee must be an active member of the team")
			}
		}
	} else if assignedTo != nil && *assignedTo != actor.ID {
		return nil, domain.NewUnprocessable("cannot assign a personal task to another user without a team")
	}
	if assignedTo == nil {
		self := actor.ID
		assignedTo = &self
	}

	task := &domain.Task{
		Title:       req.Title,
		Description: req.Description,
		AssignedTo:  assignedTo,
		IDTeam:      req.IDTeam,
		Priority:    priority,
		Status:      status,
		StartDate:   req.StartDate,
		DueDate:     req.DueDate,
		Base:        domain.Base{CreatedBy: &actor.ID},
	}
	if err := tx.Tasks().Create(ctx, task); err != nil {
		return nil, domain.NewInternal(err)
	}

	logEntry := &domain.TaskLog{
		IDTask:     task.ID,
		ActionType: domain.ActionCreate,
		Activity:   fmt.Sprintf("%s membuat task '%s'", actor.Username, task.Title),
		CreatedBy:  &actor.ID,
	}
	if err := tx.TaskLogs().Create(ctx, logEntry); err != nil {
		return nil, domain.NewInternal(err)
	}

	created, err := tx.Tasks().GetByID(ctx, task.ID)
	if err != nil {
		return nil, domain.NewInternal(err)
	}
	if created == nil {
		return task, nil
	}
	return created, nil
}

func (s *TaskService) Get(ctx context.Context, actor Actor, taskID uuid.UUID) (*domain.Task, error) {
	task, err := s.store.Tasks().GetByID(ctx, taskID)
	if err != nil {
		return nil, domain.NewInternal(err)
	}
	if task == nil {
		return nil, domain.NewNotFound("task not found")
	}
	if err := s.ensureCanAccess(ctx, s.store, actor.ID, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *TaskService) List(ctx context.Context, actor Actor, params ListTasksParams) ([]domain.Task, int64, error) {
	if params.Status != "" && !domain.ValidTaskStatus(params.Status) {
		return nil, 0, domain.NewValidation("invalid status filter")
	}
	if params.Priority != "" && !domain.ValidTaskPriority(params.Priority) {
		return nil, 0, domain.NewValidation("invalid priority filter")
	}

	teamIDs, err := s.store.Teams().TeamIDsForUser(ctx, actor.ID)
	if err != nil {
		return nil, 0, domain.NewInternal(err)
	}

	filter := domain.TaskFilter{
		OwnerID:    actor.ID,
		TeamIDs:    teamIDs,
		Status:     params.Status,
		Priority:   params.Priority,
		Search:     params.Search,
		AssignedTo: params.AssignedTo,
		Page:       params.Page,
		Limit:      params.Limit,
		Sort:       sanitizeSort(params.Sort),
	}
	tasks, total, err := s.store.Tasks().List(ctx, filter)
	if err != nil {
		return nil, 0, domain.NewInternal(err)
	}
	return tasks, total, nil
}

func (s *TaskService) Update(ctx context.Context, actor Actor, taskID uuid.UUID, req dto.UpdateTaskRequest) (*domain.Task, error) {
	var updated *domain.Task
	err := s.store.Atomic(ctx, func(tx domain.Store) error {
		task, err := tx.Tasks().GetByID(ctx, taskID)
		if err != nil {
			return domain.NewInternal(err)
		}
		if task == nil {
			return domain.NewNotFound("task not found")
		}
		if err := s.ensureCanAccess(ctx, tx, actor.ID, task); err != nil {
			return err
		}

		oldStatus := task.Status
		statusChanged := false
		if req.Title != nil {
			task.Title = *req.Title
		}
		if req.Description != nil {
			task.Description = *req.Description
		}
		if req.Priority != nil {
			task.Priority = domain.TaskPriority(*req.Priority)
		}
		if req.Status != nil && domain.TaskStatus(*req.Status) != task.Status {
			task.Status = domain.TaskStatus(*req.Status)
			statusChanged = true
		}
		if req.StartDate != nil {
			task.StartDate = req.StartDate
		}
		if req.DueDate != nil {
			task.DueDate = req.DueDate
		}
		// Moving the task into (or between) a team requires the actor to belong to
		// the target team, mirroring the create-time rule.
		if req.IDTeam != nil {
			if err := s.ensureActorInTeam(ctx, tx, actor.ID, *req.IDTeam); err != nil {
				return err
			}
			task.IDTeam = req.IDTeam
		}
		task.UpdatedBy = &actor.ID

		if err := tx.Tasks().Update(ctx, task); err != nil {
			return domain.NewInternal(err)
		}

		action := domain.ActionUpdateTask
		activity := fmt.Sprintf("%s memperbarui task '%s'", actor.Username, task.Title)
		if statusChanged {
			action = domain.ActionUpdateStatus
			activity = fmt.Sprintf("%s mengubah status task '%s' dari '%s' ke '%s'", actor.Username, task.Title, oldStatus, task.Status)
		}
		logEntry := &domain.TaskLog{IDTask: task.ID, ActionType: action, Activity: activity, CreatedBy: &actor.ID}
		if err := tx.TaskLogs().Create(ctx, logEntry); err != nil {
			return domain.NewInternal(err)
		}

		updated, err = tx.Tasks().GetByID(ctx, task.ID)
		if err != nil {
			return domain.NewInternal(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// Delete soft-deletes the task and cascades to its comments, while preserving the
// task logs (an audit DELETE entry is written first) — all in one transaction.
func (s *TaskService) Delete(ctx context.Context, actor Actor, taskID uuid.UUID) error {
	return s.store.Atomic(ctx, func(tx domain.Store) error {
		task, err := tx.Tasks().GetByID(ctx, taskID)
		if err != nil {
			return domain.NewInternal(err)
		}
		if task == nil {
			return domain.NewNotFound("task not found")
		}
		if err := s.ensureCanDelete(ctx, tx, actor.ID, task); err != nil {
			return err
		}

		logEntry := &domain.TaskLog{
			IDTask:     task.ID,
			ActionType: domain.ActionDelete,
			Activity:   fmt.Sprintf("%s menghapus task '%s'", actor.Username, task.Title),
			CreatedBy:  &actor.ID,
		}
		if err := tx.TaskLogs().Create(ctx, logEntry); err != nil {
			return domain.NewInternal(err)
		}
		if err := tx.Comments().SoftDeleteByTask(ctx, task.ID); err != nil {
			return domain.NewInternal(err)
		}
		if err := tx.Tasks().SoftDelete(ctx, task.ID); err != nil {
			return domain.NewInternal(err)
		}
		return nil
	})
}

// Assign reassigns a task to another active member of the same team.
func (s *TaskService) Assign(ctx context.Context, actor Actor, taskID, targetUserID uuid.UUID) (*domain.Task, error) {
	var updated *domain.Task
	err := s.store.Atomic(ctx, func(tx domain.Store) error {
		task, err := tx.Tasks().GetByID(ctx, taskID)
		if err != nil {
			return domain.NewInternal(err)
		}
		if task == nil {
			return domain.NewNotFound("task not found")
		}
		if task.IDTeam == nil {
			return domain.NewUnprocessable("task has no team; assignment requires a team")
		}

		team, err := tx.Teams().GetMaster(ctx, *task.IDTeam)
		if err != nil {
			return domain.NewInternal(err)
		}
		if team == nil {
			return domain.NewUnprocessable("team not found")
		}
		isCreator := task.CreatedBy != nil && *task.CreatedBy == actor.ID
		isOwner := team.CreatedBy != nil && *team.CreatedBy == actor.ID
		if !isCreator && !isOwner {
			return domain.NewForbidden("only the task creator or team owner can assign this task")
		}

		member, err := tx.Teams().GetMember(ctx, *task.IDTeam, targetUserID)
		if err != nil {
			return domain.NewInternal(err)
		}
		if member == nil || member.Status != domain.TeamMemberActive {
			return domain.NewUnprocessable("target user is not an active member of the team")
		}
		target, err := tx.Users().GetByID(ctx, targetUserID)
		if err != nil {
			return domain.NewInternal(err)
		}
		if target == nil {
			return domain.NewUnprocessable("target user not found")
		}

		task.AssignedTo = &targetUserID
		task.UpdatedBy = &actor.ID
		if err := tx.Tasks().Update(ctx, task); err != nil {
			return domain.NewInternal(err)
		}

		logEntry := &domain.TaskLog{
			IDTask:     task.ID,
			ActionType: domain.ActionAssignUser,
			Activity:   fmt.Sprintf("%s assign task '%s' ke %s", actor.Username, task.Title, target.Username),
			CreatedBy:  &actor.ID,
		}
		if err := tx.TaskLogs().Create(ctx, logEntry); err != nil {
			return domain.NewInternal(err)
		}

		if err := s.notifier.Notify(ctx, NotificationEvent{
			Type:      "TASK_ASSIGNED",
			Recipient: targetUserID,
			TaskID:    task.ID,
			Message:   fmt.Sprintf("You have been assigned to task '%s'", task.Title),
		}); err != nil {
			return domain.NewInternal(err) // rolls back assignee + log
		}

		updated, err = tx.Tasks().GetByID(ctx, task.ID)
		if err != nil {
			return domain.NewInternal(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// Logs returns the audit trail for a task the actor can access.
func (s *TaskService) Logs(ctx context.Context, actor Actor, taskID uuid.UUID) ([]domain.TaskLog, error) {
	task, err := s.store.Tasks().GetByID(ctx, taskID)
	if err != nil {
		return nil, domain.NewInternal(err)
	}
	if task == nil {
		return nil, domain.NewNotFound("task not found")
	}
	if err := s.ensureCanAccess(ctx, s.store, actor.ID, task); err != nil {
		return nil, err
	}
	logs, err := s.store.TaskLogs().ListByTask(ctx, taskID)
	if err != nil {
		return nil, domain.NewInternal(err)
	}
	return logs, nil
}

func (s *TaskService) ensureActorInTeam(ctx context.Context, st domain.Store, userID, teamID uuid.UUID) error {
	team, err := st.Teams().GetMaster(ctx, teamID)
	if err != nil {
		return domain.NewInternal(err)
	}
	if team == nil {
		return domain.NewUnprocessable("team not found")
	}
	if team.CreatedBy != nil && *team.CreatedBy == userID {
		return nil
	}
	member, err := st.Teams().GetMember(ctx, teamID, userID)
	if err != nil {
		return domain.NewInternal(err)
	}
	if member != nil && member.Status == domain.TeamMemberActive {
		return nil
	}
	return domain.NewForbidden("you are not a member of this team")
}

// ensureCanAccess permits the creator, the assignee, or an active team member.
func (s *TaskService) ensureCanAccess(ctx context.Context, st domain.Store, userID uuid.UUID, task *domain.Task) error {
	if task.CreatedBy != nil && *task.CreatedBy == userID {
		return nil
	}
	if task.AssignedTo != nil && *task.AssignedTo == userID {
		return nil
	}
	if task.IDTeam != nil {
		if err := s.ensureActorInTeam(ctx, st, userID, *task.IDTeam); err == nil {
			return nil
		}
	}
	return domain.NewForbidden("you do not have access to this task")
}

// ensureCanDelete permits only the creator or the team owner.
func (s *TaskService) ensureCanDelete(ctx context.Context, st domain.Store, userID uuid.UUID, task *domain.Task) error {
	if task.CreatedBy != nil && *task.CreatedBy == userID {
		return nil
	}
	if task.IDTeam != nil {
		team, err := st.Teams().GetMaster(ctx, *task.IDTeam)
		if err != nil {
			return domain.NewInternal(err)
		}
		if team != nil && team.CreatedBy != nil && *team.CreatedBy == userID {
			return nil
		}
	}
	return domain.NewForbidden("only the task creator or team owner can delete this task")
}

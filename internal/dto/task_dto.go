package dto

import (
	"time"

	"github.com/gdcpay/task-api/internal/domain"
	"github.com/google/uuid"
)

// CreateTaskRequest is the body of POST /tasks.
type CreateTaskRequest struct {
	Title       string     `json:"title" validate:"required,min=1,max=255" example:"Ship the release"`                                                                                   // required
	Description string     `json:"description,omitempty" validate:"max=5000" example:"Cut v1.0 and tag it"`                                                                              // optional
	Priority    string     `json:"priority,omitempty" validate:"omitempty,oneof=LOW MEDIUM HIGH URGENT" enums:"LOW,MEDIUM,HIGH,URGENT" example:"HIGH"`                                   // optional, default MEDIUM
	Status      string     `json:"status,omitempty" validate:"omitempty,oneof=TODO IN_PROGRESS DONE CANCELED IN_REVIEW" enums:"TODO,IN_PROGRESS,DONE,CANCELED,IN_REVIEW" example:"TODO"` // optional, default TODO
	AssignedTo  *uuid.UUID `json:"assignedTo,omitempty" example:"6f8d3c2a-1b4e-4f9a-9c2d-0a1b2c3d4e5f"`                                                                                  // optional, defaults to creator; if set with idTeam must be an active member
	IDTeam      *uuid.UUID `json:"idTeam,omitempty" example:"2a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d"`                                                                                      // optional; you must belong to this team
	StartDate   *time.Time `json:"startDate,omitempty" example:"2026-06-13T09:00:00Z"`                                                                                                   // optional
	DueDate     *time.Time `json:"dueDate,omitempty" example:"2026-06-20T17:00:00Z"`                                                                                                     // optional
}

// UpdateTaskRequest is the body of PUT /tasks/{id}.
type UpdateTaskRequest struct {
	Title       *string    `json:"title,omitempty" validate:"omitempty,min=1,max=255" example:"Ship the release (v1.0.1)"`                                                                      // optional
	Description *string    `json:"description,omitempty" validate:"omitempty,max=5000" example:"Patch release"`                                                                                 // optional
	Priority    *string    `json:"priority,omitempty" validate:"omitempty,oneof=LOW MEDIUM HIGH URGENT" enums:"LOW,MEDIUM,HIGH,URGENT" example:"URGENT"`                                        // optional
	Status      *string    `json:"status,omitempty" validate:"omitempty,oneof=TODO IN_PROGRESS DONE CANCELED IN_REVIEW" enums:"TODO,IN_PROGRESS,DONE,CANCELED,IN_REVIEW" example:"IN_PROGRESS"` // optional
	IDTeam      *uuid.UUID `json:"idTeam,omitempty" example:"2a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d"`                                                                                             // optional; move the task into a team you belong to
	StartDate   *time.Time `json:"startDate,omitempty" example:"2026-06-13T09:00:00Z"`                                                                                                          // optional
	DueDate     *time.Time `json:"dueDate,omitempty" example:"2026-06-25T17:00:00Z"`                                                                                                            // optional
}

type AssignTaskRequest struct {
	AssignedTo uuid.UUID `json:"assignedTo" validate:"required"`
}

type TaskResponse struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Priority    string     `json:"priority"`
	Status      string     `json:"status"`
	AssignedTo  *uuid.UUID `json:"assignedTo,omitempty"`
	Assignee    *UserBrief `json:"assignee,omitempty"`
	IDTeam      *uuid.UUID `json:"idTeam,omitempty"`
	StartDate   *time.Time `json:"startDate,omitempty"`
	DueDate     *time.Time `json:"dueDate,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	CreatedBy   *uuid.UUID `json:"createdBy,omitempty"`
}

func NewTaskResponse(t *domain.Task) TaskResponse {
	return TaskResponse{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		Priority:    string(t.Priority),
		Status:      string(t.Status),
		AssignedTo:  t.AssignedTo,
		Assignee:    NewUserBrief(t.Assignee),
		IDTeam:      t.IDTeam,
		StartDate:   t.StartDate,
		DueDate:     t.DueDate,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		CreatedBy:   t.CreatedBy,
	}
}

func NewTaskResponses(tasks []domain.Task) []TaskResponse {
	out := make([]TaskResponse, 0, len(tasks))
	for i := range tasks {
		out = append(out, NewTaskResponse(&tasks[i]))
	}
	return out
}

type TaskLogResponse struct {
	ID         uuid.UUID  `json:"id"`
	ActionType string     `json:"actionType"`
	Activity   string     `json:"activity"`
	CreatedAt  time.Time  `json:"createdAt"`
	CreatedBy  *uuid.UUID `json:"createdBy,omitempty"`
}

func NewTaskLogResponses(logs []domain.TaskLog) []TaskLogResponse {
	out := make([]TaskLogResponse, 0, len(logs))
	for _, l := range logs {
		out = append(out, TaskLogResponse{
			ID:         l.ID,
			ActionType: string(l.ActionType),
			Activity:   l.Activity,
			CreatedAt:  l.CreatedAt,
			CreatedBy:  l.CreatedBy,
		})
	}
	return out
}

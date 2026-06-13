package domain

import (
	"time"

	"github.com/google/uuid"
)

type TaskLogAction string

const (
	ActionCreate       TaskLogAction = "CREATE"
	ActionUpdateStatus TaskLogAction = "UPDATE_STATUS"
	ActionAssignUser   TaskLogAction = "ASSIGN_USER"
	ActionDelete       TaskLogAction = "DELETE"
	ActionUpdateTask   TaskLogAction = "UPDATE_TASK"
)

// TaskLog is an append-only audit record of a change to a task. It has no
// soft-delete column and is intentionally preserved when a task is deleted.
type TaskLog struct {
	ID         uuid.UUID     `json:"id"`
	IDTask     uuid.UUID     `json:"idTask"`
	ActionType TaskLogAction `json:"actionType"`
	Activity   string        `json:"activity"`
	CreatedAt  time.Time     `json:"createdAt"`
	CreatedBy  *uuid.UUID    `json:"createdBy,omitempty"`
}

func (TaskLog) TableName() string { return "task_logs" }

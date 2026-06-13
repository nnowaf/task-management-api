package domain

import (
	"time"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	TaskTODO       TaskStatus = "TODO"
	TaskInProgress TaskStatus = "IN_PROGRESS"
	TaskDone       TaskStatus = "DONE"
	TaskCanceled   TaskStatus = "CANCELED"
	TaskInReview   TaskStatus = "IN_REVIEW"
)

func ValidTaskStatus(s string) bool {
	switch TaskStatus(s) {
	case TaskTODO, TaskInProgress, TaskDone, TaskCanceled, TaskInReview:
		return true
	}
	return false
}

type TaskPriority string

const (
	PriorityLow    TaskPriority = "LOW"
	PriorityMedium TaskPriority = "MEDIUM"
	PriorityHigh   TaskPriority = "HIGH"
	PriorityUrgent TaskPriority = "URGENT"
)

func ValidTaskPriority(s string) bool {
	switch TaskPriority(s) {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityUrgent:
		return true
	}
	return false
}

// Task is a unit of work. AssignedTo and IDTeam are nullable so a personal task
// (no team) is a first-class concept. Assignee is populated by the repository via
// a join; it is not a stored column.
type Task struct {
	Base
	Title       string       `json:"title"`
	Description string       `json:"description"`
	AssignedTo  *uuid.UUID   `json:"assignedTo,omitempty"`
	IDTeam      *uuid.UUID   `json:"idTeam,omitempty"`
	Priority    TaskPriority `json:"priority"`
	Status      TaskStatus   `json:"status"`
	StartDate   *time.Time   `json:"startDate,omitempty"`
	DueDate     *time.Time   `json:"dueDate,omitempty"`

	Assignee *User       `json:"assignee,omitempty"`
	Team     *TeamMaster `json:"team,omitempty"`
}

func (Task) TableName() string { return "tasks" }

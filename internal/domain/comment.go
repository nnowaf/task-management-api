package domain

import "github.com/google/uuid"

// CommentTask is a comment on a task. It is soft-deleted and cascades when its
type CommentTask struct {
	Base
	IDTask  uuid.UUID `json:"idTask"`
	Comment string    `json:"comment"`
}

func (CommentTask) TableName() string { return "comment_tasks" }

package dto

import (
	"time"

	"github.com/gdcpay/task-api/internal/domain"
	"github.com/google/uuid"
)

type CreateCommentRequest struct {
	Comment string `json:"comment" validate:"required,min=1,max=2000" example:"Looks good, merging."` // required
}

type UpdateCommentRequest struct {
	Comment *string `json:"comment,omitempty" validate:"omitempty,min=1,max=2000" example:"Edited: merged and deployed."` // optional
}

type CommentResponse struct {
	ID        uuid.UUID  `json:"id"`
	IDTask    uuid.UUID  `json:"idTask"`
	Comment   string     `json:"comment"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	CreatedBy *uuid.UUID `json:"createdBy,omitempty"`
}

func NewCommentResponse(c *domain.CommentTask) CommentResponse {
	return CommentResponse{
		ID:        c.ID,
		IDTask:    c.IDTask,
		Comment:   c.Comment,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		CreatedBy: c.CreatedBy,
	}
}

func NewCommentResponses(comments []domain.CommentTask) []CommentResponse {
	out := make([]CommentResponse, 0, len(comments))
	for i := range comments {
		out = append(out, NewCommentResponse(&comments[i]))
	}
	return out
}

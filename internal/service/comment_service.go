package service

import (
	"context"

	"github.com/gdcpay/task-api/internal/domain"
	"github.com/gdcpay/task-api/internal/dto"
	"github.com/google/uuid"
)

type CommentService struct {
	store domain.Store
}

func NewCommentService(store domain.Store) *CommentService {
	return &CommentService{store: store}
}

func (s *CommentService) Create(ctx context.Context, actor Actor, taskID uuid.UUID, req dto.CreateCommentRequest) (*domain.CommentTask, error) {
	if _, err := s.requireTaskAccess(ctx, actor.ID, taskID); err != nil {
		return nil, err
	}
	comment := &domain.CommentTask{
		IDTask:  taskID,
		Comment: req.Comment,
		Base:    domain.Base{CreatedBy: &actor.ID},
	}
	if err := s.store.Comments().Create(ctx, comment); err != nil {
		return nil, domain.NewInternal(err)
	}
	return comment, nil
}

func (s *CommentService) ListByTask(ctx context.Context, actor Actor, taskID uuid.UUID) ([]domain.CommentTask, error) {
	if _, err := s.requireTaskAccess(ctx, actor.ID, taskID); err != nil {
		return nil, err
	}
	comments, err := s.store.Comments().ListByTask(ctx, taskID)
	if err != nil {
		return nil, domain.NewInternal(err)
	}
	return comments, nil
}

func (s *CommentService) Update(ctx context.Context, actor Actor, commentID uuid.UUID, req dto.UpdateCommentRequest) (*domain.CommentTask, error) {
	comment, err := s.store.Comments().GetByID(ctx, commentID)
	if err != nil {
		return nil, domain.NewInternal(err)
	}
	if comment == nil {
		return nil, domain.NewNotFound("comment not found")
	}
	if comment.CreatedBy == nil || *comment.CreatedBy != actor.ID {
		return nil, domain.NewForbidden("only the comment author can edit it")
	}

	if req.Comment != nil {
		comment.Comment = *req.Comment
	}
	comment.UpdatedBy = &actor.ID
	if err := s.store.Comments().Update(ctx, comment); err != nil {
		return nil, domain.NewInternal(err)
	}
	return comment, nil
}

func (s *CommentService) Delete(ctx context.Context, actor Actor, commentID uuid.UUID) error {
	comment, err := s.store.Comments().GetByID(ctx, commentID)
	if err != nil {
		return domain.NewInternal(err)
	}
	if comment == nil {
		return domain.NewNotFound("comment not found")
	}

	// The comment author may delete it; so may the owner of the parent task.
	if comment.CreatedBy == nil || *comment.CreatedBy != actor.ID {
		task, err := s.store.Tasks().GetByID(ctx, comment.IDTask)
		if err != nil {
			return domain.NewInternal(err)
		}
		if task == nil || task.CreatedBy == nil || *task.CreatedBy != actor.ID {
			return domain.NewForbidden("you cannot delete this comment")
		}
	}
	if err := s.store.Comments().SoftDelete(ctx, commentID); err != nil {
		return domain.NewInternal(err)
	}
	return nil
}

// requireTaskAccess loads the task and asserts the actor is the creator, the
// assignee, or an active member of the task's team.
func (s *CommentService) requireTaskAccess(ctx context.Context, userID, taskID uuid.UUID) (*domain.Task, error) {
	task, err := s.store.Tasks().GetByID(ctx, taskID)
	if err != nil {
		return nil, domain.NewInternal(err)
	}
	if task == nil {
		return nil, domain.NewNotFound("task not found")
	}
	if task.CreatedBy != nil && *task.CreatedBy == userID {
		return task, nil
	}
	if task.AssignedTo != nil && *task.AssignedTo == userID {
		return task, nil
	}
	if task.IDTeam != nil {
		member, err := s.store.Teams().GetMember(ctx, *task.IDTeam, userID)
		if err != nil {
			return nil, domain.NewInternal(err)
		}
		if member != nil && member.Status == domain.TeamMemberActive {
			return task, nil
		}
		team, err := s.store.Teams().GetMaster(ctx, *task.IDTeam)
		if err != nil {
			return nil, domain.NewInternal(err)
		}
		if team != nil && team.CreatedBy != nil && *team.CreatedBy == userID {
			return task, nil
		}
	}
	return nil, domain.NewForbidden("you do not have access to this task")
}

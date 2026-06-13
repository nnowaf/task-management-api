package service

import (
	"context"

	"github.com/gdcpay/task-api/internal/domain"
	"github.com/gdcpay/task-api/internal/dto"
	"github.com/google/uuid"
)

type TeamService struct {
	store domain.Store
}

func NewTeamService(store domain.Store) *TeamService {
	return &TeamService{store: store}
}

// Create makes a team and adds the creator as the first active member, atomically.
func (s *TeamService) Create(ctx context.Context, actor Actor, req dto.CreateTeamRequest) (*domain.TeamMaster, error) {
	team := &domain.TeamMaster{
		Name: req.Name,
		Base: domain.Base{CreatedBy: &actor.ID},
	}

	err := s.store.Atomic(ctx, func(tx domain.Store) error {
		if err := tx.Teams().CreateMaster(ctx, team); err != nil {
			return err
		}
		owner := &domain.TeamDetail{
			IDTeamMaster: team.ID,
			IDUser:       actor.ID,
			Status:       domain.TeamMemberActive,
			Base:         domain.Base{CreatedBy: &actor.ID},
		}
		return tx.Teams().AddMember(ctx, owner)
	})
	if err != nil {
		return nil, err
	}
	return s.store.Teams().GetMaster(ctx, team.ID)
}

func (s *TeamService) Get(ctx context.Context, actor Actor, teamID uuid.UUID) (*domain.TeamMaster, error) {
	team, err := s.store.Teams().GetMaster(ctx, teamID)
	if err != nil {
		return nil, domain.NewInternal(err)
	}
	if team == nil {
		return nil, domain.NewNotFound("team not found")
	}
	if !s.canView(ctx, actor.ID, team) {
		return nil, domain.NewForbidden("you do not have access to this team")
	}
	return team, nil
}

func (s *TeamService) List(ctx context.Context, actor Actor) ([]domain.TeamMaster, error) {
	teams, err := s.store.Teams().ListMastersForUser(ctx, actor.ID)
	if err != nil {
		return nil, domain.NewInternal(err)
	}
	return teams, nil
}

func (s *TeamService) AddMember(ctx context.Context, actor Actor, teamID uuid.UUID, req dto.AddMemberRequest) (*domain.TeamDetail, error) {
	if _, err := s.requireOwner(ctx, actor.ID, teamID); err != nil {
		return nil, err
	}

	user, err := s.store.Users().GetByID(ctx, req.UserID)
	if err != nil {
		return nil, domain.NewInternal(err)
	}
	if user == nil {
		return nil, domain.NewUnprocessable("target user not found")
	}

	status := domain.TeamMemberStatus(req.Status)
	if status == "" {
		status = domain.TeamMemberActive
	}

	member := &domain.TeamDetail{
		IDTeamMaster: teamID,
		IDUser:       req.UserID,
		Status:       status,
		ExpiredAt:    req.ExpiredAt,
		Base:         domain.Base{CreatedBy: &actor.ID},
	}
	if err := s.store.Teams().AddMember(ctx, member); err != nil {
		return nil, err
	}
	member.User = user
	return member, nil
}

func (s *TeamService) ListMembers(ctx context.Context, actor Actor, teamID uuid.UUID) ([]domain.TeamDetail, error) {
	team, err := s.store.Teams().GetMaster(ctx, teamID)
	if err != nil {
		return nil, domain.NewInternal(err)
	}
	if team == nil {
		return nil, domain.NewNotFound("team not found")
	}
	if !s.canView(ctx, actor.ID, team) {
		return nil, domain.NewForbidden("you do not have access to this team")
	}
	return s.store.Teams().ListMembers(ctx, teamID)
}

func (s *TeamService) UpdateMember(ctx context.Context, actor Actor, teamID, userID uuid.UUID, req dto.UpdateMemberRequest) (*domain.TeamDetail, error) {
	if _, err := s.requireOwner(ctx, actor.ID, teamID); err != nil {
		return nil, err
	}

	member, err := s.store.Teams().GetMember(ctx, teamID, userID)
	if err != nil {
		return nil, domain.NewInternal(err)
	}
	if member == nil {
		return nil, domain.NewNotFound("team member not found")
	}

	member.Status = domain.TeamMemberStatus(req.Status)
	member.ExpiredAt = req.ExpiredAt
	member.UpdatedBy = &actor.ID
	if err := s.store.Teams().UpdateMember(ctx, member); err != nil {
		return nil, err
	}
	return member, nil
}

func (s *TeamService) RemoveMember(ctx context.Context, actor Actor, teamID, userID uuid.UUID) error {
	team, err := s.requireOwner(ctx, actor.ID, teamID)
	if err != nil {
		return err
	}
	// Prevent the owner from removing themselves and orphaning the team.
	if team.CreatedBy != nil && *team.CreatedBy == userID {
		return domain.NewUnprocessable("the team owner cannot be removed")
	}
	return s.store.Teams().RemoveMember(ctx, teamID, userID)
}

// requireOwner loads the team and asserts the actor is its owner.
func (s *TeamService) requireOwner(ctx context.Context, userID, teamID uuid.UUID) (*domain.TeamMaster, error) {
	team, err := s.store.Teams().GetMaster(ctx, teamID)
	if err != nil {
		return nil, domain.NewInternal(err)
	}
	if team == nil {
		return nil, domain.NewNotFound("team not found")
	}
	if team.CreatedBy == nil || *team.CreatedBy != userID {
		return nil, domain.NewForbidden("only the team owner can manage members")
	}
	return team, nil
}

// canView reports whether the user owns the team or is one of its members.
func (s *TeamService) canView(_ context.Context, userID uuid.UUID, team *domain.TeamMaster) bool {
	if team.CreatedBy != nil && *team.CreatedBy == userID {
		return true
	}
	for _, m := range team.Members {
		if m.IDUser == userID {
			return true
		}
	}
	return false
}

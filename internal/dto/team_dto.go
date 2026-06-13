package dto

import (
	"time"

	"github.com/gdcpay/task-api/internal/domain"
	"github.com/google/uuid"
)

type CreateTeamRequest struct {
	Name string `json:"name" validate:"required,min=2,max=150" example:"Platform Team"` // required
}

type AddMemberRequest struct {
	UserID    uuid.UUID  `json:"userId" validate:"required" example:"6f8d3c2a-1b4e-4f9a-9c2d-0a1b2c3d4e5f"`                                      // required
	Status    string     `json:"status,omitempty" validate:"omitempty,oneof=active expired block" enums:"active,expired,block" example:"active"` // optional, default active
	ExpiredAt *time.Time `json:"expiredAt,omitempty" example:"2026-12-31T23:59:59Z"`                                                             // optional
}

type UpdateMemberRequest struct {
	Status    string     `json:"status" validate:"required,oneof=active expired block" enums:"active,expired,block" example:"block"` // required
	ExpiredAt *time.Time `json:"expiredAt,omitempty" example:"2026-12-31T23:59:59Z"`                                                 // optional
}

type TeamResponse struct {
	ID        uuid.UUID        `json:"id"`
	Name      string           `json:"name"`
	CreatedBy *uuid.UUID       `json:"createdBy,omitempty"`
	CreatedAt time.Time        `json:"createdAt"`
	Members   []MemberResponse `json:"members,omitempty"`
}

type MemberResponse struct {
	ID        uuid.UUID  `json:"id"`
	IDUser    uuid.UUID  `json:"idUser"`
	Status    string     `json:"status"`
	ExpiredAt *time.Time `json:"expiredAt,omitempty"`
	User      *UserBrief `json:"user,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

func NewMemberResponse(d *domain.TeamDetail) MemberResponse {
	return MemberResponse{
		ID:        d.ID,
		IDUser:    d.IDUser,
		Status:    string(d.Status),
		ExpiredAt: d.ExpiredAt,
		User:      NewUserBrief(d.User),
		CreatedAt: d.CreatedAt,
	}
}

func NewMemberResponses(members []domain.TeamDetail) []MemberResponse {
	out := make([]MemberResponse, 0, len(members))
	for i := range members {
		out = append(out, NewMemberResponse(&members[i]))
	}
	return out
}

func NewTeamResponse(t *domain.TeamMaster) TeamResponse {
	return TeamResponse{
		ID:        t.ID,
		Name:      t.Name,
		CreatedBy: t.CreatedBy,
		CreatedAt: t.CreatedAt,
		Members:   NewMemberResponses(t.Members),
	}
}

func NewTeamResponses(teams []domain.TeamMaster) []TeamResponse {
	out := make([]TeamResponse, 0, len(teams))
	for i := range teams {
		out = append(out, NewTeamResponse(&teams[i]))
	}
	return out
}

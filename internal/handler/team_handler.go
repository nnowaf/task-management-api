package handler

import (
	"github.com/gdcpay/task-api/internal/dto"
	"github.com/gdcpay/task-api/internal/pkg/response"
	"github.com/gdcpay/task-api/internal/service"
	"github.com/gofiber/fiber/v2"
)

type TeamHandler struct {
	svc *service.TeamService
}

func NewTeamHandler(svc *service.TeamService) *TeamHandler {
	return &TeamHandler{svc: svc}
}

// Create godoc
// @Summary      Create a team (creator becomes owner & first member)
// @Tags         teams
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.CreateTeamRequest true "Team payload"
// @Success      201 {object} response.Success{data=dto.TeamResponse}
// @Router       /teams [post]
func (h *TeamHandler) Create(c *fiber.Ctx) error {
	var req dto.CreateTeamRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	team, err := h.svc.Create(c.UserContext(), actorFrom(c), req)
	if err != nil {
		return err
	}
	return response.Created(c, dto.NewTeamResponse(team))
}

// List godoc
// @Summary      List teams the current user owns or belongs to
// @Tags         teams
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} response.Success{data=[]dto.TeamResponse}
// @Router       /teams [get]
func (h *TeamHandler) List(c *fiber.Ctx) error {
	teams, err := h.svc.List(c.UserContext(), actorFrom(c))
	if err != nil {
		return err
	}
	return response.OK(c, dto.NewTeamResponses(teams))
}

// Get godoc
// @Summary      Get a team by id
// @Tags         teams
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Team id"
// @Success      200 {object} response.Success{data=dto.TeamResponse}
// @Router       /teams/{id} [get]
func (h *TeamHandler) Get(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	team, err := h.svc.Get(c.UserContext(), actorFrom(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, dto.NewTeamResponse(team))
}

// AddMember godoc
// @Summary      Add a member to a team (owner only)
// @Tags         teams
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Team id"
// @Param        request body dto.AddMemberRequest true "Member payload"
// @Success      201 {object} response.Success{data=dto.MemberResponse}
// @Router       /teams/{id}/members [post]
func (h *TeamHandler) AddMember(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	var req dto.AddMemberRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	member, err := h.svc.AddMember(c.UserContext(), actorFrom(c), id, req)
	if err != nil {
		return err
	}
	return response.Created(c, dto.NewMemberResponse(member))
}

// ListMembers godoc
// @Summary      List members of a team
// @Tags         teams
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Team id"
// @Success      200 {object} response.Success{data=[]dto.MemberResponse}
// @Router       /teams/{id}/members [get]
func (h *TeamHandler) ListMembers(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	members, err := h.svc.ListMembers(c.UserContext(), actorFrom(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, dto.NewMemberResponses(members))
}

// UpdateMember godoc
// @Summary      Update a team member's status
// @Tags         teams
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Team id"
// @Param        userId path string true "User id"
// @Param        request body dto.UpdateMemberRequest true "Status payload"
// @Success      200 {object} response.Success{data=dto.MemberResponse}
// @Router       /teams/{id}/members/{userId} [put]
func (h *TeamHandler) UpdateMember(c *fiber.Ctx) error {
	teamID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	userID, err := parseUUIDParam(c, "userId")
	if err != nil {
		return err
	}
	var req dto.UpdateMemberRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	member, err := h.svc.UpdateMember(c.UserContext(), actorFrom(c), teamID, userID, req)
	if err != nil {
		return err
	}
	return response.OK(c, dto.NewMemberResponse(member))
}

// RemoveMember godoc
// @Summary      Remove a member from a team
// @Tags         teams
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Team id"
// @Param        userId path string true "User id"
// @Success      200 {object} response.Success
// @Router       /teams/{id}/members/{userId} [delete]
func (h *TeamHandler) RemoveMember(c *fiber.Ctx) error {
	teamID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	userID, err := parseUUIDParam(c, "userId")
	if err != nil {
		return err
	}
	if err := h.svc.RemoveMember(c.UserContext(), actorFrom(c), teamID, userID); err != nil {
		return err
	}
	return response.OK(c, fiber.Map{"teamId": teamID, "userId": userID, "removed": true})
}

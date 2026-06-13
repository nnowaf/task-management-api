package handler

import (
	"github.com/gdcpay/task-api/internal/dto"
	"github.com/gdcpay/task-api/internal/pkg/response"
	"github.com/gdcpay/task-api/internal/service"
	"github.com/gofiber/fiber/v2"
)

type CommentHandler struct {
	svc *service.CommentService
}

func NewCommentHandler(svc *service.CommentService) *CommentHandler {
	return &CommentHandler{svc: svc}
}

// Create godoc
// @Summary      Add a comment to a task
// @Tags         comments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Task id"
// @Param        request body dto.CreateCommentRequest true "Comment payload"
// @Success      201 {object} response.Success{data=dto.CommentResponse}
// @Router       /tasks/{id}/comments [post]
func (h *CommentHandler) Create(c *fiber.Ctx) error {
	taskID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	var req dto.CreateCommentRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	comment, err := h.svc.Create(c.UserContext(), actorFrom(c), taskID, req)
	if err != nil {
		return err
	}
	return response.Created(c, dto.NewCommentResponse(comment))
}

// List godoc
// @Summary      List comments on a task
// @Tags         comments
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Task id"
// @Success      200 {object} response.Success{data=[]dto.CommentResponse}
// @Router       /tasks/{id}/comments [get]
func (h *CommentHandler) List(c *fiber.Ctx) error {
	taskID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	comments, err := h.svc.ListByTask(c.UserContext(), actorFrom(c), taskID)
	if err != nil {
		return err
	}
	return response.OK(c, dto.NewCommentResponses(comments))
}

// Update godoc
// @Summary      Edit a comment (author only)
// @Tags         comments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Comment id"
// @Param        request body dto.UpdateCommentRequest true "Fields to update"
// @Success      200 {object} response.Success{data=dto.CommentResponse}
// @Router       /comments/{id} [put]
func (h *CommentHandler) Update(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	var req dto.UpdateCommentRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	comment, err := h.svc.Update(c.UserContext(), actorFrom(c), id, req)
	if err != nil {
		return err
	}
	return response.OK(c, dto.NewCommentResponse(comment))
}

// Delete godoc
// @Summary      Delete a comment (author or task owner)
// @Tags         comments
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Comment id"
// @Success      200 {object} response.Success
// @Router       /comments/{id} [delete]
func (h *CommentHandler) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	if err := h.svc.Delete(c.UserContext(), actorFrom(c), id); err != nil {
		return err
	}
	return response.OK(c, fiber.Map{"id": id, "deleted": true})
}

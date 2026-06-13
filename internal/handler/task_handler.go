package handler

import (
	"github.com/gdcpay/task-api/internal/dto"
	"github.com/gdcpay/task-api/internal/pkg/response"
	"github.com/gdcpay/task-api/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type TaskHandler struct {
	svc *service.TaskService
}

func NewTaskHandler(svc *service.TaskService) *TaskHandler {
	return &TaskHandler{svc: svc}
}

// Create godoc
// @Summary      Create a task (idempotent)
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Idempotency-Key header string true "Idempotency key (UUID)"
// @Param        request body dto.CreateTaskRequest true "Task payload (only title required)"
// @Success      201 {object} response.Success{data=dto.TaskResponse}
// @Failure      400 {object} response.ErrorBody
// @Failure      403 {object} response.ErrorBody
// @Failure      409 {object} response.ErrorBody
// @Failure      422 {object} response.ErrorBody
// @Router       /tasks [post]
func (h *TaskHandler) Create(c *fiber.Ctx) error {
	key, err := idempotencyKey(c)
	if err != nil {
		return err
	}
	var req dto.CreateTaskRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	result, err := h.svc.CreateTaskIdempotent(c.UserContext(), actorFrom(c), key, req)
	if err != nil {
		return err
	}
	c.Set("Idempotent-Replayed", boolStr(result.Replayed))
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	return c.Status(result.StatusCode).Send(result.Body)
}

// List godoc
// @Summary      List tasks
// @Tags         tasks
// @Produce      json
// @Security     BearerAuth
// @Param        status query string false "Filter by status (TODO, IN_PROGRESS, DONE, CANCELED, IN_REVIEW)"
// @Param        priority query string false "Filter by priority (LOW, MEDIUM, HIGH, URGENT)"
// @Param        q query string false "Search in title"
// @Param        assignedTo query string false "Filter by assignee user id"
// @Param        page query int false "Page number (default 1)"
// @Param        limit query int false "Page size (default 10, max 100)"
// @Param        sort query string false "Sort, e.g. created_at desc"
// @Success      200 {object} response.Success{data=[]dto.TaskResponse,meta=response.Meta}
// @Router       /tasks [get]
func (h *TaskHandler) List(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	if page < 1 {
		page = 1
	}
	limit := c.QueryInt("limit", 10)
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	var assignedTo *uuid.UUID
	if v := c.Query("assignedTo"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return parseUUIDQueryErr("assignedTo")
		}
		assignedTo = &id
	}

	params := service.ListTasksParams{
		Status:     c.Query("status"),
		Priority:   c.Query("priority"),
		Search:     c.Query("q"),
		AssignedTo: assignedTo,
		Page:       page,
		Limit:      limit,
		Sort:       c.Query("sort"),
	}
	tasks, total, err := h.svc.List(c.UserContext(), actorFrom(c), params)
	if err != nil {
		return err
	}
	return response.List(c, dto.NewTaskResponses(tasks), response.NewMeta(total, page, limit))
}

// Get godoc
// @Summary      Get a task by id
// @Tags         tasks
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Task id"
// @Success      200 {object} response.Success{data=dto.TaskResponse}
// @Failure      404 {object} response.ErrorBody
// @Router       /tasks/{id} [get]
func (h *TaskHandler) Get(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	task, err := h.svc.Get(c.UserContext(), actorFrom(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, dto.NewTaskResponse(task))
}

// Update godoc
// @Summary      Update a task
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Task id"
// @Param        request body dto.UpdateTaskRequest true "Fields to update (all optional)"
// @Success      200 {object} response.Success{data=dto.TaskResponse}
// @Failure      403 {object} response.ErrorBody
// @Failure      404 {object} response.ErrorBody
// @Failure      422 {object} response.ErrorBody
// @Router       /tasks/{id} [put]
func (h *TaskHandler) Update(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	var req dto.UpdateTaskRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	task, err := h.svc.Update(c.UserContext(), actorFrom(c), id, req)
	if err != nil {
		return err
	}
	return response.OK(c, dto.NewTaskResponse(task))
}

// Delete godoc
// @Summary      Delete a task (soft delete + cascade comments)
// @Tags         tasks
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Task id"
// @Success      200 {object} response.Success
// @Router       /tasks/{id} [delete]
func (h *TaskHandler) Delete(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	if err := h.svc.Delete(c.UserContext(), actorFrom(c), id); err != nil {
		return err
	}
	return response.OK(c, fiber.Map{"id": id, "deleted": true})
}

// Assign godoc
// @Summary      Assign a task to a team member
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Task id"
// @Param        request body dto.AssignTaskRequest true "Target user"
// @Success      200 {object} response.Success{data=dto.TaskResponse}
// @Router       /tasks/{id}/assign [post]
func (h *TaskHandler) Assign(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	var req dto.AssignTaskRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	task, err := h.svc.Assign(c.UserContext(), actorFrom(c), id, req.AssignedTo)
	if err != nil {
		return err
	}
	return response.OK(c, dto.NewTaskResponse(task))
}

// Logs godoc
// @Summary      Get the audit log of a task
// @Tags         tasks
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Task id"
// @Success      200 {object} response.Success{data=[]dto.TaskLogResponse}
// @Router       /tasks/{id}/logs [get]
func (h *TaskHandler) Logs(c *fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	logs, err := h.svc.Logs(c.UserContext(), actorFrom(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, dto.NewTaskLogResponses(logs))
}

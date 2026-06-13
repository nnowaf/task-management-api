package handler

import (
	"github.com/gdcpay/task-api/internal/dto"
	"github.com/gdcpay/task-api/internal/pkg/response"
	"github.com/gdcpay/task-api/internal/service"
	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Register godoc
// @Summary      Register a new user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.RegisterRequest true "Registration payload"
// @Success      201 {object} response.Success{data=dto.UserResponse}
// @Failure      400 {object} response.ErrorBody
// @Failure      409 {object} response.ErrorBody
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req dto.RegisterRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	user, err := h.svc.Register(c.UserContext(), req)
	if err != nil {
		return err
	}
	return response.Created(c, dto.NewUserResponse(user))
}

// Login godoc
// @Summary      Log in and obtain an access token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.LoginRequest true "Login payload"
// @Success      200 {object} response.Success{data=dto.AuthResponse}
// @Failure      400 {object} response.ErrorBody
// @Failure      401 {object} response.ErrorBody
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	token, exp, user, err := h.svc.Login(c.UserContext(), req)
	if err != nil {
		return err
	}
	return response.OK(c, dto.AuthResponse{
		Token:     token,
		ExpiresAt: exp,
		User:      dto.NewUserResponse(user),
	})
}

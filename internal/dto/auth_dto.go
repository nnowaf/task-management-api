package dto

import (
	"time"

	"github.com/gdcpay/task-api/internal/domain"
	"github.com/google/uuid"
)

type RegisterRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=150" example:"Nowaf Dev"`
	Username string `json:"username" validate:"required,min=3,max=100" example:"nowaf"`
	Email    string `json:"email" validate:"required,email" example:"nowaf@example.com"`
	Password string `json:"password" validate:"required,min=6,max=72" example:"secret123"` // bcrypt input is capped at 72 bytes
}

type LoginRequest struct {
	Login    string `json:"login" validate:"required" example:"nowaf"` // username or email
	Password string `json:"password" validate:"required" example:"secret123"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

type UserBrief struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Username string    `json:"username"`
}

type AuthResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expiresAt"`
	User      UserResponse `json:"user"`
}

func NewUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Name:      u.Name,
		Username:  u.Username,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}

func NewUserBrief(u *domain.User) *UserBrief {
	if u == nil {
		return nil
	}
	return &UserBrief{ID: u.ID, Name: u.Name, Username: u.Username}
}

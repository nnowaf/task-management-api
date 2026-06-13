package service

import (
	"context"
	"time"

	"github.com/gdcpay/task-api/internal/domain"
	"github.com/gdcpay/task-api/internal/dto"
	"github.com/gdcpay/task-api/internal/pkg/hash"
	"github.com/gdcpay/task-api/internal/pkg/jwt"
)

type AuthService struct {
	store domain.Store
	jwt   *jwt.Manager
}

func NewAuthService(store domain.Store, jwtMgr *jwt.Manager) *AuthService {
	return &AuthService{store: store, jwt: jwtMgr}
}

// Register creates a new user, rejecting duplicate username/email.
func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (*domain.User, error) {
	exists, err := s.store.Users().ExistsByUsernameOrEmail(ctx, req.Username, req.Email)
	if err != nil {
		return nil, domain.NewInternal(err)
	}
	if exists {
		return nil, domain.NewConflict(domain.CodeConflict, "username or email already in use")
	}

	hashed, err := hash.Password(req.Password)
	if err != nil {
		return nil, domain.NewInternal(err)
	}

	user := &domain.User{
		Name:     req.Name,
		Username: req.Username,
		Email:    req.Email,
		Password: hashed,
	}
	if err := s.store.Users().Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// Login verifies credentials and issues an access token.
func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (string, time.Time, *domain.User, error) {
	user, err := s.store.Users().GetByLogin(ctx, req.Login)
	if err != nil {
		return "", time.Time{}, nil, domain.NewInternal(err)
	}
	// Same response for unknown user and wrong password to avoid user enumeration.
	if user == nil || !hash.Check(user.Password, req.Password) {
		return "", time.Time{}, nil, domain.NewUnauthorized("invalid credentials")
	}

	token, exp, err := s.jwt.Generate(user.ID, user.Username)
	if err != nil {
		return "", time.Time{}, nil, domain.NewInternal(err)
	}
	return token, exp, user, nil
}

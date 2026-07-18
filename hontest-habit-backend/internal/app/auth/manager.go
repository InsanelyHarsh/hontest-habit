package auth

import (
	"context"

	"github.com/insanelyharsh/hontest-habit/internal/app/auth/models"
	"github.com/insanelyharsh/hontest-habit/internal/app/auth/repository"
	"github.com/insanelyharsh/hontest-habit/internal/common/errors"
)

type AuthManager struct {
	repository repository.AuthRepository
	jwtConfig  JWTConfig
}

func NewAuthManager(repo repository.AuthRepository, jwtCfg JWTConfig) *AuthManager {
	return &AuthManager{repository: repo, jwtConfig: jwtCfg}
}

func (m *AuthManager) Signup(ctx context.Context, req *models.SignupRequest) (*models.AuthResponse, error) {
	email, err := validateEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if err := validatePassword(req.Password); err != nil {
		return nil, err
	}

	exists, err := m.repository.EmailExists(ctx, email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.Conflict("email already registered", nil)
	}

	userID, err := m.repository.CreateUser(ctx, email, req.Password)
	if err != nil {
		return nil, err
	}

	token, err := generateJwtToken(ctx, m.jwtConfig, userID, email)
	if err != nil {
		return nil, errors.Internal("failed to issue token", err)
	}
	return &models.AuthResponse{UserID: userID, Token: token}, nil
}

func (m *AuthManager) Login(ctx context.Context, req *models.LoginRequest) (*models.AuthResponse, error) {
	email, err := validateEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if req.Password == "" {
		return nil, errors.BadRequest("password is required", nil)
	}

	userID, err := m.repository.ValidateCredentials(ctx, email, req.Password)
	if err != nil {
		return nil, err
	}

	token, err := generateJwtToken(ctx, m.jwtConfig, userID, email)
	if err != nil {
		return nil, errors.Internal("failed to issue token", err)
	}
	return &models.AuthResponse{UserID: userID, Token: token}, nil
}

func (m *AuthManager) ResetPassword(ctx context.Context) error {
	return nil
}

func (m *AuthManager) ForgotPassword(ctx context.Context) error {
	return nil
}

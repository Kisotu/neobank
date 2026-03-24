package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/Kisotu/neobank/internal/auth"
	"github.com/Kisotu/neobank/internal/domain"
	"github.com/Kisotu/neobank/internal/repository"
)

type userService struct {
	userRepo repository.UserRepository
	jwt      *auth.JWTManager
	logger   *slog.Logger
}

func NewUserService(userRepo repository.UserRepository, jwtManager *auth.JWTManager, logger *slog.Logger) UserService {
	if logger == nil {
		logger = slog.Default()
	}

	return &userService{
		userRepo: userRepo,
		jwt:      jwtManager,
		logger:   logger,
	}
}

func (s *userService) Register(ctx context.Context, req *RegisterRequest) (*UserResponse, error) {
	if req == nil {
		return nil, domain.ErrInvalidUserName
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" {
		return nil, domain.ErrInvalidUserEmail
	}

	if _, err := s.userRepo.GetByEmail(ctx, email); err == nil {
		return nil, domain.ErrDuplicateUser
	}

	password := strings.TrimSpace(req.Password)
	if len(password) < 8 {
		return nil, domain.ErrInvalidCredentials
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		Email:        email,
		PasswordHash: string(hash),
		FullName:     strings.TrimSpace(req.FullName),
		Status:       domain.UserStatusActive,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return mapUserResponse(user), nil
}

func (s *userService) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	if req == nil {
		return nil, domain.ErrInvalidCredentials
	}

	user, err := s.userRepo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		return nil, domain.ErrInvalidCredentials
	}

	if !user.IsActive() {
		return nil, domain.ErrUnauthorized
	}

	accessToken, refreshToken, expiresIn, err := s.jwt.GenerateTokenPair(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generate auth token: %w", err)
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}

func (s *userService) GetProfile(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return mapUserResponse(user), nil
}

func (s *userService) UpdateProfile(ctx context.Context, userID uuid.UUID, req *UpdateProfileRequest) error {
	if req == nil {
		return errors.New("update request is required")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if trimmedEmail := strings.ToLower(strings.TrimSpace(req.Email)); trimmedEmail != "" && trimmedEmail != user.Email {
		if _, err := s.userRepo.GetByEmail(ctx, trimmedEmail); err == nil {
			return domain.ErrDuplicateUser
		}
		user.Email = trimmedEmail
	}

	if trimmedName := strings.TrimSpace(req.FullName); trimmedName != "" {
		user.FullName = trimmedName
	}

	return s.userRepo.Update(ctx, user)
}

func mapUserResponse(user *domain.User) *UserResponse {
	if user == nil {
		return nil
	}

	return &UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		FullName:  user.FullName,
		Status:    string(user.Status),
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

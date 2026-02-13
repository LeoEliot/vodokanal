package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vodokanal/auth/internal/jwt"
	"github.com/vodokanal/auth/internal/model"
	"github.com/vodokanal/auth/internal/repository"
	"go.uber.org/zap"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserExists        = errors.New("user with this email already exists")
	ErrUserInactive      = errors.New("user account is inactive")
	ErrInvalidToken      = errors.New("invalid token")
	ErrTokenExpired      = errors.New("token has expired")
)

type Service struct {
	repo      *repository.Repository
	jwtMgr    *jwt.Manager
	logger    *zap.Logger
}

func New(repo *repository.Repository, jwtMgr *jwt.Manager, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		jwtMgr: jwtMgr,
		logger: logger,
	}
}

// Register registers a new user
func (s *Service) Register(ctx context.Context, email, password string, subscriberID *int) (*model.RegisterResponse, error) {
	// Check if user already exists
	_, err := s.repo.GetUserByEmail(ctx, email)
	if err == nil {
		return nil, ErrUserExists
	}

	// Validate password
	validator := model.DefaultPasswordValidator()
	if err := validator.Validate(password); err != nil {
		return nil, fmt.Errorf("password validation failed: %w", err)
	}

	// Check for common passwords
	if model.IsCommonPassword(password) {
		return nil, errors.New("password is too common")
	}

	// Hash password
	hash, err := model.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user, err := s.repo.CreateUser(ctx, email, hash, subscriberID, "subscriber")
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate tokens
	accessToken, _, err := s.jwtMgr.GenerateTokens(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &model.RegisterResponse{
		UserID:      user.ID,
		Email:       user.Email,
		Role:        user.Role,
		AccessToken: accessToken,
	}, nil
}

// Login authenticates a user and returns tokens
func (s *Service) Login(ctx context.Context, email, password string) (*model.LoginResponse, error) {
	// Get user
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Check if user is active
	if !user.IsActive {
		return nil, ErrUserInactive
	}

	// Check password
	if !model.CheckPassword(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// Update last login
	_ = s.repo.UpdateLastLogin(ctx, user.ID)

	// Generate tokens
	accessToken, refreshToken, err := s.jwtMgr.GenerateTokens(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Store refresh token
	err = s.repo.StoreRefreshToken(ctx, user.ID, refreshToken,
		time.Now().Add(168*time.Hour)) // 7 days
	if err != nil {
		s.logger.Error("Failed to store refresh token", zap.Error(err))
	}

	return &model.LoginResponse{
		UserID:       user.ID,
		Email:        user.Email,
		Role:         user.Role,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// RefreshToken refreshes an access token using a refresh token
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*model.RefreshResponse, error) {
	// Validate refresh token
	claims, err := s.jwtMgr.ValidateRefreshToken(refreshToken)
	if err != nil {
		if errors.Is(err, jwt.ErrExpiredToken) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	// Check if refresh token exists in database
	storedToken, err := s.repo.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Verify user ID matches
	if storedToken.UserID != claims.UserID {
		return nil, ErrInvalidToken
	}

	// Get user
	user, err := s.repo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Check if user is active
	if !user.IsActive {
		return nil, ErrUserInactive
	}

	// Generate new tokens
	newAccessToken, newRefreshToken, err := s.jwtMgr.GenerateTokens(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Delete old refresh token and store new one
	_ = s.repo.DeleteRefreshToken(ctx, refreshToken)
	err = s.repo.StoreRefreshToken(ctx, user.ID, newRefreshToken,
		time.Now().Add(168*time.Hour)) // 7 days
	if err != nil {
		s.logger.Error("Failed to store refresh token", zap.Error(err))
	}

	return &model.RefreshResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

// Logout logs out a user by deleting their refresh token
func (s *Service) Logout(ctx context.Context, userID int, refreshToken string) error {
	// Delete refresh token
	err := s.repo.DeleteRefreshToken(ctx, refreshToken)
	if err != nil {
		s.logger.Warn("Refresh token not found during logout", zap.Error(err))
	}
	return nil
}

// LogoutAll logs out a user from all devices
func (s *Service) LogoutAll(ctx context.Context, userID int) error {
	return s.repo.DeleteAllRefreshTokensForUser(ctx, userID)
}

// GetUser retrieves a user by ID
func (s *Service) GetUser(ctx context.Context, userID int) (*model.User, error) {
	return s.repo.GetUserByID(ctx, userID)
}

// ValidateToken validates a token and returns the user claims
func (s *Service) ValidateToken(token string) (*jwt.Claims, error) {
	return s.jwtMgr.ValidateAccessToken(token)
}

// ChangePassword changes a user's password
func (s *Service) ChangePassword(ctx context.Context, userID int, oldPassword, newPassword string) error {
	// Get user
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Check old password
	if !model.CheckPassword(oldPassword, user.PasswordHash) {
		return errors.New("current password is incorrect")
	}

	// Validate new password
	validator := model.DefaultPasswordValidator()
	if err := validator.Validate(newPassword); err != nil {
		return fmt.Errorf("password validation failed: %w", err)
	}

	// Check for common passwords
	if model.IsCommonPassword(newPassword) {
		return errors.New("password is too common")
	}

	// Hash new password
	hash, err := model.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	return s.repo.UpdatePassword(ctx, userID, hash)
}

package user

import (
	"context"
	"fmt"

	"github.com/yourusername/im-system/pkg/auth"
	"github.com/yourusername/im-system/pkg/logger"
	"go.uber.org/zap"
)

type Service struct {
	repo       *Repository
	jwtManager *auth.JWTManager
}

func NewService(repo *Repository, jwtManager *auth.JWTManager) *Service {
	return &Service{
		repo:       repo,
		jwtManager: jwtManager,
	}
}

// Register registers a new user
func (s *Service) Register(ctx context.Context, username, password, email, nickname string) (int64, error) {
	// Check if user already exists
	existingUser, _ := s.repo.GetUserByUsername(ctx, username)
	if existingUser != nil {
		return 0, fmt.Errorf("username already exists")
	}

	user, err := s.repo.CreateUser(ctx, username, password, email, nickname)
	if err != nil {
		logger.Log.Error("Failed to create user",
			zap.String("username", username),
			zap.Error(err),
		)
		return 0, err
	}

	logger.Log.Info("User registered successfully",
		zap.Int64("user_id", user.ID),
		zap.String("username", username),
	)

	return user.ID, nil
}

// Login authenticates a user and generates a token
func (s *Service) Login(ctx context.Context, username, password, deviceID string) (int64, string, int64, *User, error) {
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return 0, "", 0, nil, fmt.Errorf("invalid credentials")
	}

	if err := s.repo.VerifyPassword(user.PasswordHash, password); err != nil {
		return 0, "", 0, nil, fmt.Errorf("invalid credentials")
	}

	token, err := s.jwtManager.Generate(user.ID, deviceID)
	if err != nil {
		logger.Log.Error("Failed to generate token",
			zap.Int64("user_id", user.ID),
			zap.Error(err),
		)
		return 0, "", 0, nil, fmt.Errorf("failed to generate token: %w", err)
	}

	claims, _ := s.jwtManager.Validate(token)
	expiresAt := claims.ExpiresAt.Unix()

	logger.Log.Info("User logged in successfully",
		zap.Int64("user_id", user.ID),
		zap.String("username", username),
		zap.String("device_id", deviceID),
	)

	return user.ID, token, expiresAt, user, nil
}

// GetUserInfo retrieves user information
func (s *Service) GetUserInfo(ctx context.Context, userID int64) (*User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateUserInfo updates user information
func (s *Service) UpdateUserInfo(ctx context.Context, userID int64, nickname, avatar, bio *string) error {
	return s.repo.UpdateUser(ctx, userID, nickname, avatar, bio)
}

// ValidateToken validates a JWT token
func (s *Service) ValidateToken(ctx context.Context, token string) (int64, string, error) {
	claims, err := s.jwtManager.Validate(token)
	if err != nil {
		return 0, "", err
	}

	return claims.UserID, claims.DeviceID, nil
}

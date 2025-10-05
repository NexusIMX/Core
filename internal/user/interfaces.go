package user

import (
	"context"
)

// UserRepository defines the interface for user data persistence
type UserRepository interface {
	// CreateUser creates a new user with hashed password
	CreateUser(ctx context.Context, username, password, email, nickname string) (*User, error)

	// GetUserByUsername retrieves a user by username (includes password hash)
	GetUserByUsername(ctx context.Context, username string) (*User, error)

	// GetUserByID retrieves a user by ID (excludes password hash)
	GetUserByID(ctx context.Context, userID int64) (*User, error)

	// UpdateUser updates user information (nickname, avatar, bio)
	UpdateUser(ctx context.Context, userID int64, nickname, avatar, bio *string) error

	// VerifyPassword verifies if the provided password matches the hashed password
	VerifyPassword(hashedPassword, password string) error
}

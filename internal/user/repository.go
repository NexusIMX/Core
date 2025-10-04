package user

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Email        string
	Nickname     string
	Avatar       string
	Bio          string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateUser creates a new user
func (r *Repository) CreateUser(ctx context.Context, username, password, email, nickname string) (*User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &User{}
	query := `
		INSERT INTO users (username, password_hash, email, nickname, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, username, email, nickname, avatar, bio, created_at, updated_at
	`

	err = r.db.QueryRowContext(ctx, query, username, string(hashedPassword), email, nickname).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Nickname,
		&user.Avatar,
		&user.Bio,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// GetUserByUsername retrieves a user by username
func (r *Repository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	user := &User{}
	query := `
		SELECT id, username, password_hash, email, nickname, avatar, bio, created_at, updated_at
		FROM users
		WHERE username = $1
	`

	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Email,
		&user.Nickname,
		&user.Avatar,
		&user.Bio,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (r *Repository) GetUserByID(ctx context.Context, userID int64) (*User, error) {
	user := &User{}
	query := `
		SELECT id, username, email, nickname, avatar, bio, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Nickname,
		&user.Avatar,
		&user.Bio,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// UpdateUser updates user information
func (r *Repository) UpdateUser(ctx context.Context, userID int64, nickname, avatar, bio *string) error {
	query := `
		UPDATE users
		SET nickname = COALESCE($1, nickname),
		    avatar = COALESCE($2, avatar),
		    bio = COALESCE($3, bio),
		    updated_at = NOW()
		WHERE id = $4
	`

	_, err := r.db.ExecContext(ctx, query, nickname, avatar, bio, userID)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// VerifyPassword verifies a user's password
func (r *Repository) VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

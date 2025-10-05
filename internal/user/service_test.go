package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dollarkillerx/im-system/pkg/auth"
	"github.com/dollarkillerx/im-system/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Initialize logger for tests
	_ = logger.Init("error", "console", []string{"stdout"})
}

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	users             map[string]*User
	getUserByUsername func(ctx context.Context, username string) (*User, error)
	getUserByID       func(ctx context.Context, userID int64) (*User, error)
	createUser        func(ctx context.Context, username, password, email, nickname string) (*User, error)
	updateUser        func(ctx context.Context, userID int64, nickname, avatar, bio *string) error
	verifyPassword    func(hashedPassword, password string) error
}

func newMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: make(map[string]*User),
	}
}

func (m *MockUserRepository) CreateUser(ctx context.Context, username, password, email, nickname string) (*User, error) {
	if m.createUser != nil {
		return m.createUser(ctx, username, password, email, nickname)
	}

	user := &User{
		ID:           int64(len(m.users) + 1),
		Username:     username,
		PasswordHash: "hashed_" + password,
		Email:        email,
		Nickname:     nickname,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	m.users[username] = user
	return user, nil
}

func (m *MockUserRepository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	if m.getUserByUsername != nil {
		return m.getUserByUsername(ctx, username)
	}

	user, ok := m.users[username]
	if !ok {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (m *MockUserRepository) GetUserByID(ctx context.Context, userID int64) (*User, error) {
	if m.getUserByID != nil {
		return m.getUserByID(ctx, userID)
	}

	for _, user := range m.users {
		if user.ID == userID {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *MockUserRepository) UpdateUser(ctx context.Context, userID int64, nickname, avatar, bio *string) error {
	if m.updateUser != nil {
		return m.updateUser(ctx, userID, nickname, avatar, bio)
	}

	for _, user := range m.users {
		if user.ID == userID {
			if nickname != nil {
				user.Nickname = *nickname
			}
			if avatar != nil {
				user.Avatar = *avatar
			}
			if bio != nil {
				user.Bio = *bio
			}
			user.UpdatedAt = time.Now()
			return nil
		}
	}
	return errors.New("user not found")
}

func (m *MockUserRepository) VerifyPassword(hashedPassword, password string) error {
	if m.verifyPassword != nil {
		return m.verifyPassword(hashedPassword, password)
	}

	if hashedPassword == "hashed_"+password {
		return nil
	}
	return errors.New("invalid password")
}

func TestService_Register(t *testing.T) {
	tests := []struct {
		name      string
		username  string
		password  string
		email     string
		nickname  string
		setupMock func(*MockUserRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name:     "successful registration",
			username: "testuser",
			password: "password123",
			email:    "test@example.com",
			nickname: "Test User",
			wantErr:  false,
		},
		{
			name:     "duplicate username",
			username: "existing",
			password: "password123",
			email:    "test@example.com",
			nickname: "Test User",
			setupMock: func(m *MockUserRepository) {
				m.users["existing"] = &User{
					ID:       1,
					Username: "existing",
				}
			},
			wantErr: true,
			errMsg:  "username already exists",
		},
		{
			name:     "database error on create",
			username: "newuser",
			password: "password123",
			email:    "test@example.com",
			nickname: "Test User",
			setupMock: func(m *MockUserRepository) {
				m.createUser = func(ctx context.Context, username, password, email, nickname string) (*User, error) {
					return nil, errors.New("database error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockUserRepository()
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}
			jwtManager := auth.NewJWTManager("test-secret", 1*time.Hour)
			service := NewService(repo, jwtManager)

			userID, err := service.Register(context.Background(), tt.username, tt.password, tt.email, tt.nickname)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Equal(t, int64(0), userID)
			} else {
				require.NoError(t, err)
				assert.Greater(t, userID, int64(0))
			}
		})
	}
}

func TestService_Login(t *testing.T) {
	tests := []struct {
		name      string
		username  string
		password  string
		deviceID  string
		setupMock func(*MockUserRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name:     "successful login",
			username: "testuser",
			password: "password123",
			deviceID: "device-001",
			setupMock: func(m *MockUserRepository) {
				m.users["testuser"] = &User{
					ID:           100,
					Username:     "testuser",
					PasswordHash: "hashed_password123",
					Email:        "test@example.com",
					Nickname:     "Test User",
				}
			},
			wantErr: false,
		},
		{
			name:     "user not found",
			username: "nonexistent",
			password: "password123",
			deviceID: "device-001",
			wantErr:  true,
			errMsg:   "invalid credentials",
		},
		{
			name:     "wrong password",
			username: "testuser",
			password: "wrongpassword",
			deviceID: "device-001",
			setupMock: func(m *MockUserRepository) {
				m.users["testuser"] = &User{
					ID:           100,
					Username:     "testuser",
					PasswordHash: "hashed_password123",
				}
			},
			wantErr: true,
			errMsg:  "invalid credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockUserRepository()
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}
			jwtManager := auth.NewJWTManager("test-secret", 1*time.Hour)
			service := NewService(repo, jwtManager)

			userID, token, expiresAt, user, err := service.Login(context.Background(), tt.username, tt.password, tt.deviceID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Equal(t, int64(0), userID)
				assert.Empty(t, token)
				assert.Nil(t, user)
			} else {
				require.NoError(t, err)
				assert.Greater(t, userID, int64(0))
				assert.NotEmpty(t, token)
				assert.Greater(t, expiresAt, time.Now().Unix())
				assert.NotNil(t, user)
				assert.Equal(t, tt.username, user.Username)
			}
		})
	}
}

func TestService_GetUserInfo(t *testing.T) {
	repo := newMockUserRepository()
	repo.users["test"] = &User{
		ID:       100,
		Username: "testuser",
		Email:    "test@example.com",
		Nickname: "Test User",
	}

	jwtManager := auth.NewJWTManager("test-secret", 1*time.Hour)
	service := NewService(repo, jwtManager)

	tests := []struct {
		name    string
		userID  int64
		wantErr bool
	}{
		{
			name:    "user found",
			userID:  100,
			wantErr: false,
		},
		{
			name:    "user not found",
			userID:  999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := service.GetUserInfo(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.userID, user.ID)
			}
		})
	}
}

func TestService_UpdateUserInfo(t *testing.T) {
	newNickname := "Updated Nickname"
	newAvatar := "https://example.com/avatar.jpg"
	newBio := "Updated bio"

	tests := []struct {
		name      string
		userID    int64
		nickname  *string
		avatar    *string
		bio       *string
		setupMock func(*MockUserRepository)
		wantErr   bool
	}{
		{
			name:     "successful update",
			userID:   100,
			nickname: &newNickname,
			avatar:   &newAvatar,
			bio:      &newBio,
			setupMock: func(m *MockUserRepository) {
				m.users["test"] = &User{
					ID:       100,
					Username: "testuser",
				}
			},
			wantErr: false,
		},
		{
			name:    "user not found",
			userID:  999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockUserRepository()
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}
			jwtManager := auth.NewJWTManager("test-secret", 1*time.Hour)
			service := NewService(repo, jwtManager)

			err := service.UpdateUserInfo(context.Background(), tt.userID, tt.nickname, tt.avatar, tt.bio)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_ValidateToken(t *testing.T) {
	jwtManager := auth.NewJWTManager("test-secret", 1*time.Hour)
	repo := newMockUserRepository()
	service := NewService(repo, jwtManager)

	tests := []struct {
		name       string
		setupToken func() string
		wantUserID int64
		wantDevice string
		wantErr    bool
	}{
		{
			name: "valid token",
			setupToken: func() string {
				token, _ := jwtManager.Generate(100, "device-001")
				return token
			},
			wantUserID: 100,
			wantDevice: "device-001",
			wantErr:    false,
		},
		{
			name: "invalid token",
			setupToken: func() string {
				return "invalid-token"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := tt.setupToken()
			userID, deviceID, err := service.ValidateToken(context.Background(), token)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantUserID, userID)
				assert.Equal(t, tt.wantDevice, deviceID)
			}
		})
	}
}

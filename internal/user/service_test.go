package user

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/dollarkillerx/im-system/pkg/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRepository is a mock implementation of Repository for testing
type MockRepository struct {
	users              map[string]*User
	usersById          map[int64]*User
	createUserFunc     func(ctx context.Context, username, password, email, nickname string) (*User, error)
	getUserByUsername  func(ctx context.Context, username string) (*User, error)
	getUserByID        func(ctx context.Context, userID int64) (*User, error)
	updateUserFunc     func(ctx context.Context, userID int64, nickname, avatar, bio *string) error
	verifyPasswordFunc func(hashedPassword, password string) error
}

func (m *MockRepository) CreateUser(ctx context.Context, username, password, email, nickname string) (*User, error) {
	if m.createUserFunc != nil {
		return m.createUserFunc(ctx, username, password, email, nickname)
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
	m.usersById[user.ID] = user
	return user, nil
}

func (m *MockRepository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	if m.getUserByUsername != nil {
		return m.getUserByUsername(ctx, username)
	}
	user, ok := m.users[username]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return user, nil
}

func (m *MockRepository) GetUserByID(ctx context.Context, userID int64) (*User, error) {
	if m.getUserByID != nil {
		return m.getUserByID(ctx, userID)
	}
	user, ok := m.usersById[userID]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return user, nil
}

func (m *MockRepository) UpdateUser(ctx context.Context, userID int64, nickname, avatar, bio *string) error {
	if m.updateUserFunc != nil {
		return m.updateUserFunc(ctx, userID, nickname, avatar, bio)
	}
	user, ok := m.usersById[userID]
	if !ok {
		return errors.New("user not found")
	}
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

func (m *MockRepository) VerifyPassword(hashedPassword, password string) error {
	if m.verifyPasswordFunc != nil {
		return m.verifyPasswordFunc(hashedPassword, password)
	}
	// Simple mock verification
	if hashedPassword == "hashed_"+password {
		return nil
	}
	return errors.New("invalid password")
}

func newMockRepository() *MockRepository {
	return &MockRepository{
		users:     make(map[string]*User),
		usersById: make(map[int64]*User),
	}
}

func TestNewService(t *testing.T) {
	repo := newMockRepository()
	jwtManager := auth.NewJWTManager("test-secret", 1*time.Hour)

	service := NewService(repo, jwtManager)

	assert.NotNil(t, service)
	assert.NotNil(t, service.repo)
	assert.NotNil(t, service.jwtManager)
}

func TestService_Register(t *testing.T) {
	tests := []struct {
		name      string
		username  string
		password  string
		email     string
		nickname  string
		setupMock func(*MockRepository)
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
			setupMock: func(m *MockRepository) {
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
			setupMock: func(m *MockRepository) {
				m.createUserFunc = func(ctx context.Context, username, password, email, nickname string) (*User, error) {
					return nil, errors.New("database error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
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
				assert.NoError(t, err)
				assert.NotEqual(t, int64(0), userID)
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
		setupMock func(*MockRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name:     "successful login",
			username: "testuser",
			password: "password123",
			deviceID: "device-001",
			setupMock: func(m *MockRepository) {
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
			setupMock: func(m *MockRepository) {
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
			repo := newMockRepository()
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
				assert.NoError(t, err)
				assert.NotEqual(t, int64(0), userID)
				assert.NotEmpty(t, token)
				assert.Greater(t, expiresAt, time.Now().Unix())
				assert.NotNil(t, user)
				assert.Equal(t, tt.username, user.Username)
			}
		})
	}
}

func TestService_GetUserInfo(t *testing.T) {
	tests := []struct {
		name      string
		userID    int64
		setupMock func(*MockRepository)
		wantErr   bool
	}{
		{
			name:   "user found",
			userID: 100,
			setupMock: func(m *MockRepository) {
				m.usersById[100] = &User{
					ID:       100,
					Username: "testuser",
					Email:    "test@example.com",
					Nickname: "Test User",
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
			repo := newMockRepository()
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}
			jwtManager := auth.NewJWTManager("test-secret", 1*time.Hour)
			service := NewService(repo, jwtManager)

			user, err := service.GetUserInfo(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
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
		setupMock func(*MockRepository)
		wantErr   bool
	}{
		{
			name:     "successful update all fields",
			userID:   100,
			nickname: &newNickname,
			avatar:   &newAvatar,
			bio:      &newBio,
			setupMock: func(m *MockRepository) {
				m.usersById[100] = &User{
					ID:       100,
					Username: "testuser",
				}
			},
			wantErr: false,
		},
		{
			name:     "update partial fields",
			userID:   100,
			nickname: &newNickname,
			setupMock: func(m *MockRepository) {
				m.usersById[100] = &User{
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
			repo := newMockRepository()
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
	repo := newMockRepository()
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
		{
			name: "expired token",
			setupToken: func() string {
				expiredManager := auth.NewJWTManager("test-secret", -1*time.Hour)
				token, _ := expiredManager.Generate(100, "device-001")
				return token
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

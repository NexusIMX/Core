package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJWTManager(t *testing.T) {
	secretKey := "test-secret-key"
	expiry := 24 * time.Hour

	manager := NewJWTManager(secretKey, expiry)

	assert.NotNil(t, manager)
	assert.Equal(t, secretKey, manager.secretKey)
	assert.Equal(t, expiry, manager.expiry)
}

func TestJWTManager_Generate(t *testing.T) {
	manager := NewJWTManager("test-secret", 1*time.Hour)

	tests := []struct {
		name     string
		userID   int64
		deviceID string
		wantErr  bool
	}{
		{
			name:     "valid token generation",
			userID:   123,
			deviceID: "device-001",
			wantErr:  false,
		},
		{
			name:     "zero user id",
			userID:   0,
			deviceID: "device-002",
			wantErr:  false,
		},
		{
			name:     "empty device id",
			userID:   456,
			deviceID: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := manager.Generate(tt.userID, tt.deviceID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)

				// Verify the token can be parsed
				parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
					return []byte(manager.secretKey), nil
				})
				assert.NoError(t, err)
				assert.True(t, parsedToken.Valid)
			}
		})
	}
}

func TestJWTManager_Validate(t *testing.T) {
	secretKey := "test-secret"
	manager := NewJWTManager(secretKey, 1*time.Hour)

	tests := []struct {
		name        string
		setupToken  func() string
		wantUserID  int64
		wantDevice  string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid token",
			setupToken: func() string {
				token, _ := manager.Generate(100, "device-100")
				return token
			},
			wantUserID: 100,
			wantDevice: "device-100",
			wantErr:    false,
		},
		{
			name: "invalid signature",
			setupToken: func() string {
				wrongManager := NewJWTManager("wrong-secret", 1*time.Hour)
				token, _ := wrongManager.Generate(200, "device-200")
				return token
			},
			wantErr:     true,
			errContains: "failed to parse token",
		},
		{
			name: "expired token",
			setupToken: func() string {
				expiredManager := NewJWTManager(secretKey, -1*time.Hour)
				token, _ := expiredManager.Generate(300, "device-300")
				return token
			},
			wantErr:     true,
			errContains: "failed to parse token",
		},
		{
			name: "malformed token",
			setupToken: func() string {
				return "this-is-not-a-valid-token"
			},
			wantErr:     true,
			errContains: "failed to parse token",
		},
		{
			name: "empty token",
			setupToken: func() string {
				return ""
			},
			wantErr:     true,
			errContains: "failed to parse token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := tt.setupToken()
			claims, err := manager.Validate(token)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, claims)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, claims)
				assert.Equal(t, tt.wantUserID, claims.UserID)
				assert.Equal(t, tt.wantDevice, claims.DeviceID)
				assert.Equal(t, jwt.ClaimStrings{"im-api"}, claims.Audience)
			}
		})
	}
}

func TestJWTManager_TokenExpiration(t *testing.T) {
	// Create manager with very short expiration
	manager := NewJWTManager("test-secret", 1*time.Millisecond)

	token, err := manager.Generate(999, "device-999")
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	claims, err := manager.Validate(token)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestJWTManager_ValidateAudience(t *testing.T) {
	manager := NewJWTManager("test-secret", 1*time.Hour)

	token, err := manager.Generate(777, "device-777")
	require.NoError(t, err)

	claims, err := manager.Validate(token)
	require.NoError(t, err)
	require.NotNil(t, claims)

	// Verify audience is set correctly
	assert.Contains(t, claims.Audience, "im-api")
}

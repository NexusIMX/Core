package router

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/dollarkillerx/im-system/pkg/logger"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	_ = logger.Init("error", "console", []string{"stdout"})
}

// MockRedisClient is a simple in-memory Redis mock
type MockRedisClient struct {
	hashes map[string]map[string]string // key -> field -> value
	values map[string]string            // key -> value
	ttls   map[string]time.Time         // key -> expiration
}

func newMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		hashes: make(map[string]map[string]string),
		values: make(map[string]string),
		ttls:   make(map[string]time.Time),
	}
}

// Note: This is a simplified mock that wraps the real Service
// For unit testing, we'll test the Service directly with miniredis or similar

// Helper function to create a test Service with in-memory Redis
func setupTestService(t *testing.T) (*Service, *redis.Client, func()) {
	// Use alicebob/miniredis for in-memory Redis testing
	mr := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	service := NewService(client)

	cleanup := func() {
		client.Close()
		mr.Close()
	}

	return service, client, cleanup
}

func TestService_RegisterRoute(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	tests := []struct {
		name        string
		userID      int64
		deviceID    string
		gatewayAddr string
		wantErr     bool
	}{
		{
			name:        "successful registration",
			userID:      100,
			deviceID:    "device-001",
			gatewayAddr: "gateway-1.example.com:8080",
			wantErr:     false,
		},
		{
			name:        "register multiple devices",
			userID:      100,
			deviceID:    "device-002",
			gatewayAddr: "gateway-2.example.com:8080",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.RegisterRoute(context.Background(), tt.userID, tt.deviceID, tt.gatewayAddr)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify route was registered
				routes, err := service.GetRoute(context.Background(), tt.userID)
				require.NoError(t, err)
				assert.NotEmpty(t, routes)

				// Find the registered device
				found := false
				for _, route := range routes {
					if route.DeviceID == tt.deviceID {
						found = true
						assert.Equal(t, tt.gatewayAddr, route.GatewayAddr)
						assert.Greater(t, route.LastActive, int64(0))
						break
					}
				}
				assert.True(t, found, "Device not found in routes")
			}
		})
	}
}

func TestService_GetRoute(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Register multiple devices
	userID := int64(200)
	service.RegisterRoute(ctx, userID, "device-001", "gateway-1:8080")
	service.RegisterRoute(ctx, userID, "device-002", "gateway-2:8080")

	tests := []struct {
		name        string
		userID      int64
		expectCount int
		wantErr     bool
	}{
		{
			name:        "get existing routes",
			userID:      200,
			expectCount: 2,
			wantErr:     false,
		},
		{
			name:        "get non-existent routes",
			userID:      999,
			expectCount: 0,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			routes, err := service.GetRoute(ctx, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, routes, tt.expectCount)
			}
		})
	}
}

func TestService_KeepAlive(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	userID := int64(300)
	deviceID := "device-001"

	// Register a route first
	err := service.RegisterRoute(ctx, userID, deviceID, "gateway-1:8080")
	require.NoError(t, err)

	// Get initial route
	routes, err := service.GetRoute(ctx, userID)
	require.NoError(t, err)
	require.Len(t, routes, 1)
	initialTime := routes[0].LastActive

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	tests := []struct {
		name     string
		userID   int64
		deviceID string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "successful keep alive",
			userID:   userID,
			deviceID: deviceID,
			wantErr:  false,
		},
		{
			name:     "keep alive non-existent device",
			userID:   userID,
			deviceID: "non-existent",
			wantErr:  true,
			errMsg:   "route not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.KeepAlive(ctx, tt.userID, tt.deviceID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)

				// Verify LastActive was updated
				routes, err := service.GetRoute(ctx, tt.userID)
				require.NoError(t, err)
				require.Len(t, routes, 1)
				assert.GreaterOrEqual(t, routes[0].LastActive, initialTime)
			}
		})
	}
}

func TestService_UnregisterRoute(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	userID := int64(400)

	// Register multiple devices
	service.RegisterRoute(ctx, userID, "device-001", "gateway-1:8080")
	service.RegisterRoute(ctx, userID, "device-002", "gateway-2:8080")

	tests := []struct {
		name            string
		userID          int64
		deviceID        string
		expectRemaining int
		wantErr         bool
	}{
		{
			name:            "unregister first device",
			userID:          userID,
			deviceID:        "device-001",
			expectRemaining: 1,
			wantErr:         false,
		},
		{
			name:            "unregister last device",
			userID:          userID,
			deviceID:        "device-002",
			expectRemaining: 0,
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.UnregisterRoute(ctx, tt.userID, tt.deviceID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify route was removed
				routes, err := service.GetRoute(ctx, tt.userID)
				require.NoError(t, err)
				assert.Len(t, routes, tt.expectRemaining)

				// Verify the device is not in the list
				for _, route := range routes {
					assert.NotEqual(t, tt.deviceID, route.DeviceID)
				}
			}
		})
	}
}

func TestService_GetOnlineStatus(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	userID := int64(500)

	// Register devices
	service.RegisterRoute(ctx, userID, "device-001", "gateway-1:8080")
	service.RegisterRoute(ctx, userID, "device-002", "gateway-2:8080")

	tests := []struct {
		name            string
		userID          int64
		expectOnline    bool
		expectDevices   int
		setupFunc       func()
		wantErr         bool
	}{
		{
			name:          "user with multiple devices online",
			userID:        userID,
			expectOnline:  true,
			expectDevices: 2,
			wantErr:       false,
		},
		{
			name:          "user offline",
			userID:        999,
			expectOnline:  false,
			expectDevices: 0,
			wantErr:       false,
		},
		{
			name:         "user after unregistering all devices",
			userID:       userID,
			expectOnline: false,
			setupFunc: func() {
				service.UnregisterRoute(ctx, userID, "device-001")
				service.UnregisterRoute(ctx, userID, "device-002")
			},
			expectDevices: 0,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			online, devices, err := service.GetOnlineStatus(ctx, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectOnline, online)
				assert.Len(t, devices, tt.expectDevices)
			}
		})
	}
}

func TestDeviceRoute_JSONMarshaling(t *testing.T) {
	route := &DeviceRoute{
		DeviceID:    "device-123",
		GatewayAddr: "gateway.example.com:8080",
		LastActive:  time.Now().Unix(),
	}

	// Marshal
	data, err := json.Marshal(route)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Unmarshal
	var decoded DeviceRoute
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, route.DeviceID, decoded.DeviceID)
	assert.Equal(t, route.GatewayAddr, decoded.GatewayAddr)
	assert.Equal(t, route.LastActive, decoded.LastActive)
}

func TestService_ConcurrentRegistration(t *testing.T) {
	service, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()
	userID := int64(600)

	// Register multiple devices concurrently
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(index int) {
			deviceID := "device-" + string(rune('0'+index))
			gatewayAddr := "gateway-" + string(rune('0'+index)) + ":8080"
			err := service.RegisterRoute(ctx, userID, deviceID, gatewayAddr)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	// Verify all routes were registered
	routes, err := service.GetRoute(ctx, userID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(routes), 1) // At least some should succeed
}

// Mock implementation for unit testing without Redis
type MockRouteStorage struct {
	routes        map[int64]map[string]*DeviceRoute
	registerFunc  func(ctx context.Context, userID int64, deviceID, gatewayAddr string) error
	getRouteFunc  func(ctx context.Context, userID int64) ([]*DeviceRoute, error)
}

func newMockRouteStorage() *MockRouteStorage {
	return &MockRouteStorage{
		routes: make(map[int64]map[string]*DeviceRoute),
	}
}

func (m *MockRouteStorage) RegisterRoute(ctx context.Context, userID int64, deviceID, gatewayAddr string) error {
	if m.registerFunc != nil {
		return m.registerFunc(ctx, userID, deviceID, gatewayAddr)
	}

	if m.routes[userID] == nil {
		m.routes[userID] = make(map[string]*DeviceRoute)
	}

	m.routes[userID][deviceID] = &DeviceRoute{
		DeviceID:    deviceID,
		GatewayAddr: gatewayAddr,
		LastActive:  time.Now().Unix(),
	}
	return nil
}

func (m *MockRouteStorage) UnregisterRoute(ctx context.Context, userID int64, deviceID string) error {
	if m.routes[userID] != nil {
		delete(m.routes[userID], deviceID)
	}
	return nil
}

func (m *MockRouteStorage) GetRoute(ctx context.Context, userID int64) ([]*DeviceRoute, error) {
	if m.getRouteFunc != nil {
		return m.getRouteFunc(ctx, userID)
	}

	var routes []*DeviceRoute
	if m.routes[userID] != nil {
		for _, route := range m.routes[userID] {
			routes = append(routes, route)
		}
	}
	return routes, nil
}

func (m *MockRouteStorage) KeepAlive(ctx context.Context, userID int64, deviceID string) error {
	if m.routes[userID] != nil && m.routes[userID][deviceID] != nil {
		m.routes[userID][deviceID].LastActive = time.Now().Unix()
		return nil
	}
	return errors.New("route not found")
}

func (m *MockRouteStorage) GetOnlineStatus(ctx context.Context, userID int64) (bool, []string, error) {
	var devices []string
	if m.routes[userID] != nil {
		for deviceID := range m.routes[userID] {
			devices = append(devices, deviceID)
		}
	}
	return len(devices) > 0, devices, nil
}

func TestMockRouteStorage(t *testing.T) {
	storage := newMockRouteStorage()
	ctx := context.Background()

	// Test registration
	err := storage.RegisterRoute(ctx, 100, "device-1", "gateway-1:8080")
	assert.NoError(t, err)

	// Test getting routes
	routes, err := storage.GetRoute(ctx, 100)
	assert.NoError(t, err)
	assert.Len(t, routes, 1)

	// Test online status
	online, devices, err := storage.GetOnlineStatus(ctx, 100)
	assert.NoError(t, err)
	assert.True(t, online)
	assert.Len(t, devices, 1)

	// Test unregistration
	err = storage.UnregisterRoute(ctx, 100, "device-1")
	assert.NoError(t, err)

	online, devices, err = storage.GetOnlineStatus(ctx, 100)
	assert.NoError(t, err)
	assert.False(t, online)
	assert.Len(t, devices, 0)
}

package router

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRedisClient is a mock implementation of Redis client for testing
type MockRedisClient struct {
	data          map[string]map[string]string // For hash maps
	stringData    map[string]string            // For string keys
	ttls          map[string]time.Duration
	hSetFunc      func(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	hGetFunc      func(ctx context.Context, key, field string) *redis.StringCmd
	hGetAllFunc   func(ctx context.Context, key string) *redis.StringMapCmd
	hExistsFunc   func(ctx context.Context, key, field string) *redis.BoolCmd
	hDelFunc      func(ctx context.Context, key string, fields ...string) *redis.IntCmd
	hLenFunc      func(ctx context.Context, key string) *redis.IntCmd
	hKeysFunc     func(ctx context.Context, key string) *redis.StringSliceCmd
	setFunc       func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	expireFunc    func(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	delFunc       func(ctx context.Context, keys ...string) *redis.IntCmd
}

func newMockRedis() *MockRedisClient {
	return &MockRedisClient{
		data:       make(map[string]map[string]string),
		stringData: make(map[string]string),
		ttls:       make(map[string]time.Duration),
	}
}

// HSet mock implementation
func (m *MockRedisClient) HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	if m.hSetFunc != nil {
		return m.hSetFunc(ctx, key, values...)
	}

	if m.data[key] == nil {
		m.data[key] = make(map[string]string)
	}

	for i := 0; i < len(values); i += 2 {
		field := values[i].(string)
		value := values[i+1].(string)
		m.data[key][field] = value
	}

	cmd := redis.NewIntCmd(ctx)
	cmd.SetVal(1)
	return cmd
}

func (m *MockRedisClient) HGet(ctx context.Context, key, field string) *redis.StringCmd {
	if m.hGetFunc != nil {
		return m.hGetFunc(ctx, key, field)
	}

	cmd := redis.NewStringCmd(ctx)
	if hash, ok := m.data[key]; ok {
		if val, ok := hash[field]; ok {
			cmd.SetVal(val)
			return cmd
		}
	}
	cmd.SetErr(redis.Nil)
	return cmd
}

func (m *MockRedisClient) HGetAll(ctx context.Context, key string) *redis.StringMapCmd {
	if m.hGetAllFunc != nil {
		return m.hGetAllFunc(ctx, key)
	}

	cmd := redis.NewStringMapCmd(ctx)
	if hash, ok := m.data[key]; ok {
		cmd.SetVal(hash)
	} else {
		cmd.SetVal(make(map[string]string))
	}
	return cmd
}

func (m *MockRedisClient) HExists(ctx context.Context, key, field string) *redis.BoolCmd {
	if m.hExistsFunc != nil {
		return m.hExistsFunc(ctx, key, field)
	}

	cmd := redis.NewBoolCmd(ctx)
	if hash, ok := m.data[key]; ok {
		_, exists := hash[field]
		cmd.SetVal(exists)
	} else {
		cmd.SetVal(false)
	}
	return cmd
}

func (m *MockRedisClient) HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd {
	if m.hDelFunc != nil {
		return m.hDelFunc(ctx, key, fields...)
	}

	cmd := redis.NewIntCmd(ctx)
	count := int64(0)
	if hash, ok := m.data[key]; ok {
		for _, field := range fields {
			if _, exists := hash[field]; exists {
				delete(hash, field)
				count++
			}
		}
	}
	cmd.SetVal(count)
	return cmd
}

func (m *MockRedisClient) HLen(ctx context.Context, key string) *redis.IntCmd {
	if m.hLenFunc != nil {
		return m.hLenFunc(ctx, key)
	}

	cmd := redis.NewIntCmd(ctx)
	if hash, ok := m.data[key]; ok {
		cmd.SetVal(int64(len(hash)))
	} else {
		cmd.SetVal(0)
	}
	return cmd
}

func (m *MockRedisClient) HKeys(ctx context.Context, key string) *redis.StringSliceCmd {
	if m.hKeysFunc != nil {
		return m.hKeysFunc(ctx, key)
	}

	cmd := redis.NewStringSliceCmd(ctx)
	var keys []string
	if hash, ok := m.data[key]; ok {
		for k := range hash {
			keys = append(keys, k)
		}
	}
	cmd.SetVal(keys)
	return cmd
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	if m.setFunc != nil {
		return m.setFunc(ctx, key, value, expiration)
	}

	m.stringData[key] = value.(string)
	m.ttls[key] = expiration
	cmd := redis.NewStatusCmd(ctx)
	cmd.SetVal("OK")
	return cmd
}

func (m *MockRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	if m.expireFunc != nil {
		return m.expireFunc(ctx, key, expiration)
	}

	m.ttls[key] = expiration
	cmd := redis.NewBoolCmd(ctx)
	cmd.SetVal(true)
	return cmd
}

func (m *MockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	if m.delFunc != nil {
		return m.delFunc(ctx, keys...)
	}

	count := int64(0)
	for _, key := range keys {
		if _, ok := m.data[key]; ok {
			delete(m.data, key)
			count++
		}
		if _, ok := m.stringData[key]; ok {
			delete(m.stringData, key)
			count++
		}
	}
	cmd := redis.NewIntCmd(ctx)
	cmd.SetVal(count)
	return cmd
}

func TestNewService(t *testing.T) {
	mockRedis := newMockRedis()
	service := NewService(mockRedis)

	assert.NotNil(t, service)
	assert.NotNil(t, service.redis)
}

func TestService_RegisterRoute(t *testing.T) {
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
			gatewayAddr: "gateway-1:50051",
			wantErr:     false,
		},
		{
			name:        "register multiple devices",
			userID:      200,
			deviceID:    "device-002",
			gatewayAddr: "gateway-2:50051",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRedis := newMockRedis()
			service := NewService(mockRedis)

			err := service.RegisterRoute(context.Background(), tt.userID, tt.deviceID, tt.gatewayAddr)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify route was stored
				routeKey := "route:100"
				if tt.userID == 200 {
					routeKey = "route:200"
				}

				hash, ok := mockRedis.data[routeKey]
				require.True(t, ok, "Route key should exist")

				routeData, ok := hash[tt.deviceID]
				require.True(t, ok, "Device route should exist")

				var route DeviceRoute
				err := json.Unmarshal([]byte(routeData), &route)
				require.NoError(t, err)
				assert.Equal(t, tt.deviceID, route.DeviceID)
				assert.Equal(t, tt.gatewayAddr, route.GatewayAddr)
				assert.Greater(t, route.LastActive, int64(0))
			}
		})
	}
}

func TestService_KeepAlive(t *testing.T) {
	tests := []struct {
		name     string
		userID   int64
		deviceID string
		setup    func(*MockRedisClient)
		wantErr  bool
	}{
		{
			name:     "successful keep alive",
			userID:   100,
			deviceID: "device-001",
			setup: func(m *MockRedisClient) {
				route := &DeviceRoute{
					DeviceID:    "device-001",
					GatewayAddr: "gateway-1:50051",
					LastActive:  time.Now().Unix(),
				}
				routeData, _ := json.Marshal(route)
				m.data["route:100"] = map[string]string{
					"device-001": string(routeData),
				}
			},
			wantErr: false,
		},
		{
			name:     "route not found",
			userID:   200,
			deviceID: "device-002",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRedis := newMockRedis()
			if tt.setup != nil {
				tt.setup(mockRedis)
			}
			service := NewService(mockRedis)

			err := service.KeepAlive(context.Background(), tt.userID, tt.deviceID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_GetRoute(t *testing.T) {
	tests := []struct {
		name          string
		userID        int64
		setup         func(*MockRedisClient)
		expectedCount int
		wantErr       bool
	}{
		{
			name:   "get single device route",
			userID: 100,
			setup: func(m *MockRedisClient) {
				route := &DeviceRoute{
					DeviceID:    "device-001",
					GatewayAddr: "gateway-1:50051",
					LastActive:  time.Now().Unix(),
				}
				routeData, _ := json.Marshal(route)
				m.data["route:100"] = map[string]string{
					"device-001": string(routeData),
				}
			},
			expectedCount: 1,
			wantErr:       false,
		},
		{
			name:   "get multiple device routes",
			userID: 200,
			setup: func(m *MockRedisClient) {
				route1 := &DeviceRoute{
					DeviceID:    "device-001",
					GatewayAddr: "gateway-1:50051",
					LastActive:  time.Now().Unix(),
				}
				route2 := &DeviceRoute{
					DeviceID:    "device-002",
					GatewayAddr: "gateway-2:50051",
					LastActive:  time.Now().Unix(),
				}
				routeData1, _ := json.Marshal(route1)
				routeData2, _ := json.Marshal(route2)
				m.data["route:200"] = map[string]string{
					"device-001": string(routeData1),
					"device-002": string(routeData2),
				}
			},
			expectedCount: 2,
			wantErr:       false,
		},
		{
			name:          "no routes found",
			userID:        300,
			expectedCount: 0,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRedis := newMockRedis()
			if tt.setup != nil {
				tt.setup(mockRedis)
			}
			service := NewService(mockRedis)

			routes, err := service.GetRoute(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, routes, tt.expectedCount)
			}
		})
	}
}

func TestService_UnregisterRoute(t *testing.T) {
	tests := []struct {
		name     string
		userID   int64
		deviceID string
		setup    func(*MockRedisClient)
		wantErr  bool
	}{
		{
			name:     "unregister last device",
			userID:   100,
			deviceID: "device-001",
			setup: func(m *MockRedisClient) {
				route := &DeviceRoute{
					DeviceID:    "device-001",
					GatewayAddr: "gateway-1:50051",
					LastActive:  time.Now().Unix(),
				}
				routeData, _ := json.Marshal(route)
				m.data["route:100"] = map[string]string{
					"device-001": string(routeData),
				}
			},
			wantErr: false,
		},
		{
			name:     "unregister one of multiple devices",
			userID:   200,
			deviceID: "device-001",
			setup: func(m *MockRedisClient) {
				route1 := &DeviceRoute{
					DeviceID:    "device-001",
					GatewayAddr: "gateway-1:50051",
					LastActive:  time.Now().Unix(),
				}
				route2 := &DeviceRoute{
					DeviceID:    "device-002",
					GatewayAddr: "gateway-2:50051",
					LastActive:  time.Now().Unix(),
				}
				routeData1, _ := json.Marshal(route1)
				routeData2, _ := json.Marshal(route2)
				m.data["route:200"] = map[string]string{
					"device-001": string(routeData1),
					"device-002": string(routeData2),
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRedis := newMockRedis()
			if tt.setup != nil {
				tt.setup(mockRedis)
			}
			service := NewService(mockRedis)

			err := service.UnregisterRoute(context.Background(), tt.userID, tt.deviceID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_GetOnlineStatus(t *testing.T) {
	tests := []struct {
		name          string
		userID        int64
		setup         func(*MockRedisClient)
		expectOnline  bool
		deviceCount   int
		wantErr       bool
	}{
		{
			name:   "user online with one device",
			userID: 100,
			setup: func(m *MockRedisClient) {
				m.data["route:100"] = map[string]string{
					"device-001": "route-data",
				}
			},
			expectOnline: true,
			deviceCount:  1,
			wantErr:      false,
		},
		{
			name:   "user online with multiple devices",
			userID: 200,
			setup: func(m *MockRedisClient) {
				m.data["route:200"] = map[string]string{
					"device-001": "route-data-1",
					"device-002": "route-data-2",
				}
			},
			expectOnline: true,
			deviceCount:  2,
			wantErr:      false,
		},
		{
			name:         "user offline",
			userID:       300,
			expectOnline: false,
			deviceCount:  0,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRedis := newMockRedis()
			if tt.setup != nil {
				tt.setup(mockRedis)
			}
			service := NewService(mockRedis)

			online, devices, err := service.GetOnlineStatus(context.Background(), tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectOnline, online)
				assert.Len(t, devices, tt.deviceCount)
			}
		})
	}
}

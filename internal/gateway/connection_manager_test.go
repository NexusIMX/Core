package gateway

import (
	"context"
	"testing"
	"time"

	gatewaypb "github.com/dollarkillerx/im-system/api/proto/gateway"
	"github.com/dollarkillerx/im-system/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func init() {
	_ = logger.Init("error", "console", []string{"stdout"})
}

// Since gRPC streaming interfaces are complex to mock,
// we'll test the ConnectionManager logic directly
// by creating connections with nil streams for unit testing

func TestNewConnectionManager(t *testing.T) {
	mgr := NewConnectionManager()

	assert.NotNil(t, mgr)
	assert.NotNil(t, mgr.connections)
	assert.Equal(t, 0, mgr.GetTotalConnections())
}

func TestConnectionManager_AddConnection(t *testing.T) {
	mgr := NewConnectionManager()

	conn := &Connection{
		UserID:     100,
		DeviceID:   "device-001",
		Stream:     nil, // nil for unit testing
		SendChan:   make(chan *gatewaypb.GatewayMessage, 100),
		CloseChan:  make(chan struct{}),
		LastActive: time.Now(),
	}

	mgr.AddConnection(conn)

	assert.Equal(t, 1, mgr.GetTotalConnections())

	// Verify connection was added
	retrieved, exists := mgr.GetConnection(100, "device-001")
	assert.True(t, exists)
	assert.Equal(t, conn.UserID, retrieved.UserID)
	assert.Equal(t, conn.DeviceID, retrieved.DeviceID)
}

func TestConnectionManager_RemoveConnection(t *testing.T) {
	mgr := NewConnectionManager()

	conn := &Connection{
		UserID:     100,
		DeviceID:   "device-001",
		Stream:     nil,
		SendChan:   make(chan *gatewaypb.GatewayMessage, 100),
		CloseChan:  make(chan struct{}),
		LastActive: time.Now(),
	}

	mgr.AddConnection(conn)
	assert.Equal(t, 1, mgr.GetTotalConnections())

	mgr.RemoveConnection(100, "device-001")
	assert.Equal(t, 0, mgr.GetTotalConnections())

	// Verify connection was removed
	_, exists := mgr.GetConnection(100, "device-001")
	assert.False(t, exists)
}

func TestConnectionManager_GetConnection(t *testing.T) {
	mgr := NewConnectionManager()

	// Test non-existent connection
	_, exists := mgr.GetConnection(100, "device-001")
	assert.False(t, exists)

	// Add connection
	conn := &Connection{
		UserID:     100,
		DeviceID:   "device-001",
		Stream:     nil,
		SendChan:   make(chan *gatewaypb.GatewayMessage, 100),
		CloseChan:  make(chan struct{}),
		LastActive: time.Now(),
	}
	mgr.AddConnection(conn)

	// Test existing connection
	retrieved, exists := mgr.GetConnection(100, "device-001")
	assert.True(t, exists)
	assert.Equal(t, int64(100), retrieved.UserID)
	assert.Equal(t, "device-001", retrieved.DeviceID)
}

func TestConnectionManager_GetUserConnections(t *testing.T) {
	mgr := NewConnectionManager()

	// Add multiple devices for same user
	conn1 := &Connection{
		UserID:     100,
		DeviceID:   "device-001",
		Stream:     nil,
		SendChan:   make(chan *gatewaypb.GatewayMessage, 100),
		CloseChan:  make(chan struct{}),
		LastActive: time.Now(),
	}
	conn2 := &Connection{
		UserID:     100,
		DeviceID:   "device-002",
		Stream:     nil,
		SendChan:   make(chan *gatewaypb.GatewayMessage, 100),
		CloseChan:  make(chan struct{}),
		LastActive: time.Now(),
	}
	conn3 := &Connection{
		UserID:     200,
		DeviceID:   "device-003",
		Stream:     nil,
		SendChan:   make(chan *gatewaypb.GatewayMessage, 100),
		CloseChan:  make(chan struct{}),
		LastActive: time.Now(),
	}

	mgr.AddConnection(conn1)
	mgr.AddConnection(conn2)
	mgr.AddConnection(conn3)

	// Get connections for user 100
	conns := mgr.GetUserConnections(100)
	assert.Len(t, conns, 2)
	for _, conn := range conns {
		assert.Equal(t, int64(100), conn.UserID)
	}

	// Get connections for user 200
	conns = mgr.GetUserConnections(200)
	assert.Len(t, conns, 1)
	assert.Equal(t, int64(200), conns[0].UserID)

	// Get connections for non-existent user
	conns = mgr.GetUserConnections(999)
	assert.Len(t, conns, 0)
}

func TestConnectionManager_GetTotalConnections(t *testing.T) {
	mgr := NewConnectionManager()

	assert.Equal(t, 0, mgr.GetTotalConnections())

	// Add connections
	for i := 0; i < 5; i++ {
		conn := &Connection{
			UserID:     int64(i),
			DeviceID:   "device-001",
			Stream:     nil,
			SendChan:   make(chan *gatewaypb.GatewayMessage, 100),
			CloseChan:  make(chan struct{}),
			LastActive: time.Now(),
		}
		mgr.AddConnection(conn)
	}

	assert.Equal(t, 5, mgr.GetTotalConnections())
}

func TestConnectionManager_ReplaceConnection(t *testing.T) {
	mgr := NewConnectionManager()

	// Add first connection
	conn1 := &Connection{
		UserID:     100,
		DeviceID:   "device-001",
		Stream:     nil,
		SendChan:   make(chan *gatewaypb.GatewayMessage, 100),
		CloseChan:  make(chan struct{}),
		LastActive: time.Now(),
	}
	mgr.AddConnection(conn1)

	// Add second connection with same user/device
	conn2 := &Connection{
		UserID:     100,
		DeviceID:   "device-001",
		Stream:     nil,
		SendChan:   make(chan *gatewaypb.GatewayMessage, 100),
		CloseChan:  make(chan struct{}),
		LastActive: time.Now().Add(time.Second),
	}
	mgr.AddConnection(conn2)

	// Should only have one connection
	assert.Equal(t, 1, mgr.GetTotalConnections())

	// Should be the new connection
	retrieved, exists := mgr.GetConnection(100, "device-001")
	assert.True(t, exists)
	assert.Equal(t, conn2.LastActive, retrieved.LastActive)
}

func TestConnectionManager_ConcurrentAccess(t *testing.T) {
	mgr := NewConnectionManager()
	done := make(chan bool)

	// Concurrent adds
	for i := 0; i < 10; i++ {
		go func(index int) {
			conn := &Connection{
				UserID:     int64(index),
				DeviceID:   "device-001",
				Stream:     nil,
				SendChan:   make(chan *gatewaypb.GatewayMessage, 100),
				CloseChan:  make(chan struct{}),
				LastActive: time.Now(),
			}
			mgr.AddConnection(conn)
			done <- true
		}(i)
	}

	// Wait for all adds
	for i := 0; i < 10; i++ {
		<-done
	}

	assert.Equal(t, 10, mgr.GetTotalConnections())

	// Concurrent removes
	for i := 0; i < 10; i++ {
		go func(index int) {
			mgr.RemoveConnection(int64(index), "device-001")
			done <- true
		}(i)
	}

	// Wait for all removes
	for i := 0; i < 10; i++ {
		<-done
	}

	assert.Equal(t, 0, mgr.GetTotalConnections())
}

func TestConnectionManager_cleanupOnce(t *testing.T) {
	mgr := NewConnectionManager()

	// Add active connection
	activeConn := &Connection{
		UserID:     100,
		DeviceID:   "active-device",
		Stream:     nil,
		SendChan:   make(chan *gatewaypb.GatewayMessage, 100),
		CloseChan:  make(chan struct{}),
		LastActive: time.Now(),
	}
	mgr.AddConnection(activeConn)

	// Add inactive connection
	inactiveConn := &Connection{
		UserID:     200,
		DeviceID:   "inactive-device",
		Stream:     nil,
		SendChan:   make(chan *gatewaypb.GatewayMessage, 100),
		CloseChan:  make(chan struct{}),
		LastActive: time.Now().Add(-10 * time.Minute),
	}
	mgr.AddConnection(inactiveConn)

	assert.Equal(t, 2, mgr.GetTotalConnections())

	// Cleanup with 5 minute timeout
	mgr.cleanupOnce(5 * time.Minute)

	// Should only have active connection
	assert.Equal(t, 1, mgr.GetTotalConnections())

	_, exists := mgr.GetConnection(100, "active-device")
	assert.True(t, exists)

	_, exists = mgr.GetConnection(200, "inactive-device")
	assert.False(t, exists)
}

func TestConnectionManager_CleanupInactive_Cancellation(t *testing.T) {
	mgr := NewConnectionManager()
	ctx, cancel := context.WithCancel(context.Background())

	// Start cleanup in background
	done := make(chan bool)
	go func() {
		mgr.CleanupInactive(ctx, 5*time.Minute)
		done <- true
	}()

	// Cancel immediately
	cancel()

	// Should exit quickly
	select {
	case <-done:
		// Expected: cleanup exited
	case <-time.After(100 * time.Millisecond):
		t.Fatal("CleanupInactive did not exit after context cancellation")
	}
}

func TestConnectionManager_getKey(t *testing.T) {
	mgr := NewConnectionManager()

	key := mgr.getKey(100, "device-001")
	assert.Equal(t, "100:device-001", key)

	key = mgr.getKey(999, "test-device")
	assert.Equal(t, "999:test-device", key)
}

func TestConnection_UpdateActivity(t *testing.T) {
	conn := &Connection{
		UserID:     100,
		DeviceID:   "device-001",
		Stream:     nil,
		SendChan:   make(chan *gatewaypb.GatewayMessage, 100),
		CloseChan:  make(chan struct{}),
		LastActive: time.Now(),
	}

	initialTime := conn.LastActive
	time.Sleep(10 * time.Millisecond)

	conn.UpdateActivity()

	assert.True(t, conn.LastActive.After(initialTime))
}

func TestConnection_Close(t *testing.T) {
	conn := &Connection{
		UserID:     100,
		DeviceID:   "device-001",
		Stream:     nil,
		SendChan:   make(chan *gatewaypb.GatewayMessage, 100),
		CloseChan:  make(chan struct{}),
		LastActive: time.Now(),
	}

	// Close once
	conn.Close()

	// Verify channels are closed
	select {
	case <-conn.CloseChan:
		// Expected
	default:
		t.Fatal("CloseChan should be closed")
	}

	// Close again (should not panic)
	conn.Close()
}

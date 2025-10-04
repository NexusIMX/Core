package gateway

import (
	"context"
	"fmt"
	"sync"
	"time"

	gatewaypb "github.com/dollarkillerx/im-system/api/proto/gateway"
	"github.com/dollarkillerx/im-system/pkg/logger"
	"go.uber.org/zap"
)

// Connection 表示一个客户端连接
type Connection struct {
	UserID     int64
	DeviceID   string
	Stream     gatewaypb.GatewayService_ConnectServer
	SendChan   chan *gatewaypb.GatewayMessage
	CloseChan  chan struct{}
	LastActive time.Time
	mu         sync.RWMutex
}

// NewConnection 创建新连接
func NewConnection(userID int64, deviceID string, stream gatewaypb.GatewayService_ConnectServer) *Connection {
	return &Connection{
		UserID:     userID,
		DeviceID:   deviceID,
		Stream:     stream,
		SendChan:   make(chan *gatewaypb.GatewayMessage, 100),
		CloseChan:  make(chan struct{}),
		LastActive: time.Now(),
	}
}

// Send 发送消息到客户端
func (c *Connection) Send(msg *gatewaypb.GatewayMessage) {
	select {
	case c.SendChan <- msg:
	case <-c.CloseChan:
		logger.Log.Warn("Connection closed, cannot send message",
			zap.Int64("user_id", c.UserID),
			zap.String("device_id", c.DeviceID),
		)
	default:
		logger.Log.Warn("Send channel full, dropping message",
			zap.Int64("user_id", c.UserID),
			zap.String("device_id", c.DeviceID),
		)
	}
}

// Close 关闭连接
func (c *Connection) Close() {
	select {
	case <-c.CloseChan:
		// Already closed
	default:
		close(c.CloseChan)
		close(c.SendChan)
	}
}

// UpdateActivity 更新活跃时间
func (c *Connection) UpdateActivity() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.LastActive = time.Now()
}

// ConnectionManager 连接管理器
type ConnectionManager struct {
	connections map[string]*Connection // key: "userID:deviceID"
	mu          sync.RWMutex
}

// NewConnectionManager 创建连接管理器
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]*Connection),
	}
}

// AddConnection 添加连接
func (cm *ConnectionManager) AddConnection(conn *Connection) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	key := cm.getKey(conn.UserID, conn.DeviceID)

	// 如果已存在，先关闭旧连接
	if oldConn, exists := cm.connections[key]; exists {
		oldConn.Close()
		logger.Log.Info("Replacing existing connection",
			zap.Int64("user_id", conn.UserID),
			zap.String("device_id", conn.DeviceID),
		)
	}

	cm.connections[key] = conn

	logger.Log.Info("Connection added",
		zap.Int64("user_id", conn.UserID),
		zap.String("device_id", conn.DeviceID),
		zap.Int("total_connections", len(cm.connections)),
	)
}

// RemoveConnection 移除连接
func (cm *ConnectionManager) RemoveConnection(userID int64, deviceID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	key := cm.getKey(userID, deviceID)
	if conn, exists := cm.connections[key]; exists {
		conn.Close()
		delete(cm.connections, key)

		logger.Log.Info("Connection removed",
			zap.Int64("user_id", userID),
			zap.String("device_id", deviceID),
			zap.Int("total_connections", len(cm.connections)),
		)
	}
}

// GetConnection 获取连接
func (cm *ConnectionManager) GetConnection(userID int64, deviceID string) (*Connection, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	key := cm.getKey(userID, deviceID)
	conn, exists := cm.connections[key]
	return conn, exists
}

// GetUserConnections 获取用户的所有连接
func (cm *ConnectionManager) GetUserConnections(userID int64) []*Connection {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var conns []*Connection
	for _, conn := range cm.connections {
		if conn.UserID == userID {
			conns = append(conns, conn)
		}
	}
	return conns
}

// BroadcastToUser 向用户的所有设备广播消息
func (cm *ConnectionManager) BroadcastToUser(userID int64, msg *gatewaypb.GatewayMessage) int {
	conns := cm.GetUserConnections(userID)
	count := 0
	for _, conn := range conns {
		conn.Send(msg)
		count++
	}
	return count
}

// GetTotalConnections 获取总连接数
func (cm *ConnectionManager) GetTotalConnections() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.connections)
}

// CleanupInactive 清理不活跃的连接
func (cm *ConnectionManager) CleanupInactive(ctx context.Context, timeout time.Duration) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cm.cleanupOnce(timeout)
		}
	}
}

func (cm *ConnectionManager) cleanupOnce(timeout time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	var toRemove []string

	for key, conn := range cm.connections {
		conn.mu.RLock()
		inactive := now.Sub(conn.LastActive) > timeout
		conn.mu.RUnlock()

		if inactive {
			toRemove = append(toRemove, key)
		}
	}

	for _, key := range toRemove {
		if conn, exists := cm.connections[key]; exists {
			conn.Close()
			delete(cm.connections, key)
			logger.Log.Info("Cleaned up inactive connection",
				zap.Int64("user_id", conn.UserID),
				zap.String("device_id", conn.DeviceID),
			)
		}
	}

	if len(toRemove) > 0 {
		logger.Log.Info("Cleanup completed",
			zap.Int("removed", len(toRemove)),
			zap.Int("remaining", len(cm.connections)),
		)
	}
}

func (cm *ConnectionManager) getKey(userID int64, deviceID string) string {
	return fmt.Sprintf("%d:%s", userID, deviceID)
}

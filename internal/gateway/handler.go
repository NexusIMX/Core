package gateway

import (
	"context"
	"encoding/json"
	"time"

	gatewaypb "github.com/dollarkillerx/im-system/api/proto/gateway"
	messagepb "github.com/dollarkillerx/im-system/api/proto/message"
	"github.com/dollarkillerx/im-system/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

// Handler 消息处理器
type Handler struct {
	connMgr *ConnectionManager
	clients *ServiceClients
}

// NewHandler 创建消息处理器
func NewHandler(connMgr *ConnectionManager, clients *ServiceClients) *Handler {
	return &Handler{
		connMgr: connMgr,
		clients: clients,
	}
}

// HandleClientMessage 处理客户端消息
func (h *Handler) HandleClientMessage(ctx context.Context, conn *Connection, msg *gatewaypb.GatewayMessage) {
	switch msg.Type {
	case gatewaypb.MessageType_PING:
		h.handlePing(conn, msg)
	case gatewaypb.MessageType_CHAT:
		h.handleChat(ctx, conn, msg)
	case gatewaypb.MessageType_ACK:
		h.handleAck(conn, msg)
	case gatewaypb.MessageType_TYPING:
		h.handleTyping(conn, msg)
	case gatewaypb.MessageType_READ_RECEIPT:
		h.handleReadReceipt(ctx, conn, msg)
	default:
		logger.Log.Warn("Unknown message type",
			zap.String("type", msg.Type.String()),
			zap.Int64("user_id", conn.UserID),
		)
	}
}

// handlePing 处理心跳
func (h *Handler) handlePing(conn *Connection, msg *gatewaypb.GatewayMessage) {
	conn.UpdateActivity()

	// 发送 PONG 响应
	pong := &gatewaypb.GatewayMessage{
		Type:      gatewaypb.MessageType_PONG,
		Timestamp: time.Now().Unix(),
	}

	conn.Send(pong)

	logger.Log.Debug("Handled PING",
		zap.Int64("user_id", conn.UserID),
		zap.String("device_id", conn.DeviceID),
	)
}

// handleChat 处理聊天消息
func (h *Handler) handleChat(ctx context.Context, conn *Connection, msg *gatewaypb.GatewayMessage) {
	conn.UpdateActivity()

	payload := msg.Payload.AsMap()

	convID, ok := payload["conv_id"].(float64)
	if !ok {
		h.sendError(conn, "invalid conv_id", msg.MsgId)
		return
	}

	convTypeStr, ok := payload["conv_type"].(string)
	if !ok {
		h.sendError(conn, "invalid conv_type", msg.MsgId)
		return
	}

	body, ok := payload["body"].(map[string]interface{})
	if !ok {
		h.sendError(conn, "invalid body", msg.MsgId)
		return
	}

	// 转换 conv_type
	var convType messagepb.ConversationType
	switch convTypeStr {
	case "direct":
		convType = messagepb.ConversationType_DIRECT
	case "group":
		convType = messagepb.ConversationType_GROUP
	case "channel":
		convType = messagepb.ConversationType_CHANNEL
	default:
		h.sendError(conn, "invalid conv_type value", msg.MsgId)
		return
	}

	// 提取可选字段
	var replyTo *string
	if rt, ok := payload["reply_to"].(string); ok {
		replyTo = &rt
	}

	var mentions []int64
	if m, ok := payload["mentions"].([]interface{}); ok {
		for _, v := range m {
			if id, ok := v.(float64); ok {
				mentions = append(mentions, int64(id))
			}
		}
	}

	// 调用 Message 服务发送消息
	resp, err := h.clients.SendMessage(ctx, int64(convID), conn.UserID, convType, body, replyTo, mentions)
	if err != nil {
		logger.Log.Error("Failed to send message",
			zap.Int64("user_id", conn.UserID),
			zap.Error(err),
		)
		h.sendError(conn, err.Error(), msg.MsgId)
		return
	}

	// 发送 ACK 响应
	ackPayload, _ := structpb.NewStruct(map[string]interface{}{
		"msg_id":     resp.MsgId,
		"seq":        resp.Seq,
		"created_at": resp.CreatedAt,
	})

	ack := &gatewaypb.GatewayMessage{
		Type:      gatewaypb.MessageType_ACK,
		Payload:   ackPayload,
		Timestamp: time.Now().Unix(),
		MsgId:     &resp.MsgId,
	}

	conn.Send(ack)

	logger.Log.Info("Message sent",
		zap.Int64("user_id", conn.UserID),
		zap.String("msg_id", resp.MsgId),
		zap.Int64("seq", resp.Seq),
	)
}

// handleAck 处理 ACK 确认
func (h *Handler) handleAck(conn *Connection, msg *gatewaypb.GatewayMessage) {
	conn.UpdateActivity()

	logger.Log.Debug("Received ACK",
		zap.Int64("user_id", conn.UserID),
		zap.Any("msg_id", msg.MsgId),
	)
}

// handleTyping 处理输入状态
func (h *Handler) handleTyping(conn *Connection, msg *gatewaypb.GatewayMessage) {
	conn.UpdateActivity()

	payload := msg.Payload.AsMap()
	convID, ok := payload["conv_id"].(float64)
	if !ok {
		return
	}

	// 广播输入状态给会话中的其他成员（简化实现）
	logger.Log.Debug("User typing",
		zap.Int64("user_id", conn.UserID),
		zap.Int64("conv_id", int64(convID)),
	)
}

// handleReadReceipt 处理已读回执
func (h *Handler) handleReadReceipt(ctx context.Context, conn *Connection, msg *gatewaypb.GatewayMessage) {
	conn.UpdateActivity()

	payload := msg.Payload.AsMap()
	convID, ok := payload["conv_id"].(float64)
	if !ok {
		return
	}

	seq, ok := payload["seq"].(float64)
	if !ok {
		return
	}

	// 这里可以调用 Message 服务更新已读位置
	logger.Log.Debug("Read receipt",
		zap.Int64("user_id", conn.UserID),
		zap.Int64("conv_id", int64(convID)),
		zap.Int64("seq", int64(seq)),
	)
}

// sendError 发送错误消息
func (h *Handler) sendError(conn *Connection, errMsg string, msgID *string) {
	payload, _ := structpb.NewStruct(map[string]interface{}{
		"error": errMsg,
	})

	errorMsg := &gatewaypb.GatewayMessage{
		Type:      gatewaypb.MessageType_ERROR,
		Payload:   payload,
		Timestamp: time.Now().Unix(),
		MsgId:     msgID,
		ErrorMsg:  &errMsg,
	}

	conn.Send(errorMsg)
}

// PushNotification 推送通知给用户
func (h *Handler) PushNotification(userID int64, notification map[string]interface{}) int {
	payload, err := structpb.NewStruct(notification)
	if err != nil {
		logger.Log.Error("Failed to create notification payload", zap.Error(err))
		return 0
	}

	msg := &gatewaypb.GatewayMessage{
		Type:      gatewaypb.MessageType_NOTIFICATION,
		Payload:   payload,
		Timestamp: time.Now().Unix(),
	}

	count := h.connMgr.BroadcastToUser(userID, msg)

	logger.Log.Debug("Pushed notification",
		zap.Int64("user_id", userID),
		zap.Int("device_count", count),
	)

	return count
}

// MarshalJSON 辅助函数
func marshalJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

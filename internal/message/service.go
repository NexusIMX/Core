package message

import (
	"context"
	"fmt"
	"time"

	"github.com/dollarkillerx/im-system/pkg/logger"
	"github.com/dollarkillerx/im-system/pkg/types"
	"go.uber.org/zap"
)

type Service struct {
	repo         MessageRepository
	routerClient RouterClient
}

func NewService(repo MessageRepository, routerClient RouterClient) *Service {
	return &Service{
		repo:         repo,
		routerClient: routerClient,
	}
}

// SendMessage 发送消息
func (s *Service) SendMessage(ctx context.Context, convID int64, senderID int64, convType types.ConversationType, body map[string]interface{}, replyTo *string, mentions []int64) (string, int64, int64, error) {
	// 生成消息 ID
	msgID := GenerateMessageID()

	// 获取下一个序列号
	seq, err := s.repo.GetNextSeq(ctx, convID)
	if err != nil {
		logger.Log.Error("Failed to get next seq",
			zap.Int64("conv_id", convID),
			zap.Error(err),
		)
		return "", 0, 0, fmt.Errorf("failed to get sequence: %w", err)
	}

	// 创建消息
	msg := &Message{
		MsgID:      msgID,
		ConvID:     convID,
		Seq:        seq,
		SenderID:   senderID,
		ConvType:   convType,
		Body:       body,
		ReplyTo:    replyTo,
		Mentions:   mentions,
		Visibility: "normal",
		CreatedAt:  time.Now(),
	}

	// 保存消息
	if err := s.repo.SaveMessage(ctx, msg); err != nil {
		logger.Log.Error("Failed to save message",
			zap.String("msg_id", msgID),
			zap.Int64("conv_id", convID),
			zap.Error(err),
		)
		return "", 0, 0, fmt.Errorf("failed to save message: %w", err)
	}

	logger.Log.Info("Message sent successfully",
		zap.String("msg_id", msgID),
		zap.Int64("conv_id", convID),
		zap.Int64("seq", seq),
		zap.Int64("sender_id", senderID),
	)

	// 异步通知 Router 推送消息
	go s.notifyNewMessage(convID, msgID, seq, senderID)

	return msgID, seq, msg.CreatedAt.Unix(), nil
}

// PullMessages 拉取消息
func (s *Service) PullMessages(ctx context.Context, convID int64, sinceSeq int64, limit int32) ([]*Message, bool, error) {
	messages, hasMore, err := s.repo.PullMessages(ctx, convID, sinceSeq, limit)
	if err != nil {
		logger.Log.Error("Failed to pull messages",
			zap.Int64("conv_id", convID),
			zap.Int64("since_seq", sinceSeq),
			zap.Error(err),
		)
		return nil, false, err
	}

	logger.Log.Debug("Pulled messages",
		zap.Int64("conv_id", convID),
		zap.Int64("since_seq", sinceSeq),
		zap.Int("count", len(messages)),
		zap.Bool("has_more", hasMore),
	)

	return messages, hasMore, nil
}

// CreateConversation 创建会话
func (s *Service) CreateConversation(ctx context.Context, convType types.ConversationType, title string, ownerID int64, memberIDs []int64) (int64, error) {
	// 验证会话类型
	if !convType.IsValid() {
		return 0, fmt.Errorf("invalid conversation type: %s", convType)
	}

	// 确保 owner 在成员列表中
	hasOwner := false
	for _, id := range memberIDs {
		if id == ownerID {
			hasOwner = true
			break
		}
	}
	if !hasOwner {
		memberIDs = append(memberIDs, ownerID)
	}

	convID, err := s.repo.CreateConversation(ctx, convType, title, ownerID, memberIDs)
	if err != nil {
		logger.Log.Error("Failed to create conversation",
			zap.String("type", convType.String()),
			zap.Int64("owner_id", ownerID),
			zap.Error(err),
		)
		return 0, err
	}

	logger.Log.Info("Conversation created",
		zap.Int64("conv_id", convID),
		zap.String("type", convType.String()),
		zap.Int64("owner_id", ownerID),
		zap.Int("member_count", len(memberIDs)),
	)

	return convID, nil
}

// GetConversation 获取会话信息
func (s *Service) GetConversation(ctx context.Context, convID int64) (*Conversation, []*ConversationMember, error) {
	conv, members, err := s.repo.GetConversation(ctx, convID)
	if err != nil {
		logger.Log.Error("Failed to get conversation",
			zap.Int64("conv_id", convID),
			zap.Error(err),
		)
		return nil, nil, err
	}

	return conv, members, nil
}

// UpdateReadSeq 更新已读位置
func (s *Service) UpdateReadSeq(ctx context.Context, convID int64, userID int64, seq int64) error {
	err := s.repo.UpdateReadSeq(ctx, convID, userID, seq)
	if err != nil {
		logger.Log.Error("Failed to update read seq",
			zap.Int64("conv_id", convID),
			zap.Int64("user_id", userID),
			zap.Int64("seq", seq),
			zap.Error(err),
		)
		return err
	}

	logger.Log.Debug("Read seq updated",
		zap.Int64("conv_id", convID),
		zap.Int64("user_id", userID),
		zap.Int64("seq", seq),
	)

	return nil
}

// notifyNewMessage 通知 Router 有新消息
func (s *Service) notifyNewMessage(convID int64, msgID string, seq int64, senderID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 获取会话成员
	memberIDs, err := s.repo.GetConversationMembers(ctx, convID)
	if err != nil {
		logger.Log.Error("Failed to get conversation members for notification",
			zap.Int64("conv_id", convID),
			zap.Error(err),
		)
		return
	}

	// 过滤掉发送者
	var recipientIDs []int64
	for _, id := range memberIDs {
		if id != senderID {
			recipientIDs = append(recipientIDs, id)
		}
	}

	if len(recipientIDs) == 0 {
		return
	}

	// 通知 Router
	notifiedCount, err := s.routerClient.NotifyNewMessage(ctx, convID, msgID, seq, senderID, recipientIDs)
	if err != nil {
		logger.Log.Error("Failed to notify router",
			zap.Int64("conv_id", convID),
			zap.String("msg_id", msgID),
			zap.Error(err),
		)
		return
	}

	logger.Log.Debug("Notified router",
		zap.Int64("conv_id", convID),
		zap.String("msg_id", msgID),
		zap.Int32("notified_count", notifiedCount),
	)
}

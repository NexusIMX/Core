package message

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dollarkillerx/im-system/pkg/types"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Message struct {
	MsgID      string
	ConvID     int64
	Seq        int64
	SenderID   int64
	ConvType   types.ConversationType
	Body       map[string]interface{}
	ReplyTo    *string
	Mentions   []int64
	Visibility string
	CreatedAt  time.Time
}

type Conversation struct {
	ID        int64
	Type      types.ConversationType
	Title     string
	OwnerID   int64
	CreatedAt time.Time
}

type ConversationMember struct {
	ConvID      int64
	UserID      int64
	Role        types.ConversationRole
	Muted       bool
	LastReadSeq int64
	JoinedAt    time.Time
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateConversation 创建会话
func (r *Repository) CreateConversation(ctx context.Context, convType types.ConversationType, title string, ownerID int64, memberIDs []int64) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 创建会话
	var convID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO conversations (type, title, owner_id, created_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id
	`, convType.String(), title, ownerID).Scan(&convID)

	if err != nil {
		return 0, fmt.Errorf("failed to create conversation: %w", err)
	}

	// 初始化会话序列
	_, err = tx.ExecContext(ctx, `
		INSERT INTO conversation_seq (conv_id, current_seq)
		VALUES ($1, 0)
	`, convID)

	if err != nil {
		return 0, fmt.Errorf("failed to initialize conversation seq: %w", err)
	}

	// 添加会话成员
	for _, userID := range memberIDs {
		role := types.ConversationRoleMember
		if userID == ownerID {
			role = types.ConversationRoleOwner
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO conversation_members (conv_id, user_id, role, muted, last_read_seq, joined_at)
			VALUES ($1, $2, $3, false, 0, NOW())
		`, convID, userID, role.String())

		if err != nil {
			return 0, fmt.Errorf("failed to add member: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return convID, nil
}

// GetConversation 获取会话信息
func (r *Repository) GetConversation(ctx context.Context, convID int64) (*Conversation, []*ConversationMember, error) {
	// 获取会话信息
	conv := &Conversation{}
	var convType string
	err := r.db.QueryRowContext(ctx, `
		SELECT id, type, title, owner_id, created_at
		FROM conversations
		WHERE id = $1
	`, convID).Scan(&conv.ID, &convType, &conv.Title, &conv.OwnerID, &conv.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil, fmt.Errorf("conversation not found")
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	conv.Type = types.ConversationType(convType)

	// 获取成员列表
	rows, err := r.db.QueryContext(ctx, `
		SELECT conv_id, user_id, role, muted, last_read_seq, joined_at
		FROM conversation_members
		WHERE conv_id = $1
		ORDER BY joined_at ASC
	`, convID)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to get members: %w", err)
	}
	defer rows.Close()

	var members []*ConversationMember
	for rows.Next() {
		member := &ConversationMember{}
		var role string
		err := rows.Scan(&member.ConvID, &member.UserID, &role, &member.Muted, &member.LastReadSeq, &member.JoinedAt)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan member: %w", err)
		}
		member.Role = types.ConversationRole(role)
		members = append(members, member)
	}

	return conv, members, nil
}

// SaveMessage 保存消息
func (r *Repository) SaveMessage(ctx context.Context, msg *Message) error {
	bodyJSON, err := json.Marshal(msg.Body)
	if err != nil {
		return fmt.Errorf("failed to marshal body: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO messages (conv_id, seq, msg_id, sender_id, conv_type, body, reply_to, mentions, visibility, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
	`, msg.ConvID, msg.Seq, msg.MsgID, msg.SenderID, msg.ConvType.String(), bodyJSON, msg.ReplyTo, pq.Array(msg.Mentions), msg.Visibility)

	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	return nil
}

// GetNextSeq 获取下一个序列号
func (r *Repository) GetNextSeq(ctx context.Context, convID int64) (int64, error) {
	var seq int64
	err := r.db.QueryRowContext(ctx, `SELECT next_conv_seq($1)`, convID).Scan(&seq)
	if err != nil {
		return 0, fmt.Errorf("failed to get next seq: %w", err)
	}
	return seq, nil
}

// PullMessages 拉取消息
func (r *Repository) PullMessages(ctx context.Context, convID int64, sinceSeq int64, limit int32) ([]*Message, bool, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT msg_id, conv_id, seq, sender_id, conv_type, body, reply_to, mentions, visibility, created_at
		FROM messages
		WHERE conv_id = $1 AND seq > $2
		ORDER BY seq ASC
		LIMIT $3
	`, convID, sinceSeq, limit+1) // 多查一条判断是否还有更多

	if err != nil {
		return nil, false, fmt.Errorf("failed to pull messages: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		msg := &Message{}
		var bodyJSON []byte
		var convType string
		var mentions pq.Int64Array

		err := rows.Scan(
			&msg.MsgID,
			&msg.ConvID,
			&msg.Seq,
			&msg.SenderID,
			&convType,
			&bodyJSON,
			&msg.ReplyTo,
			&mentions,
			&msg.Visibility,
			&msg.CreatedAt,
		)

		if err != nil {
			return nil, false, fmt.Errorf("failed to scan message: %w", err)
		}

		msg.ConvType = types.ConversationType(convType)
		msg.Mentions = []int64(mentions)

		// 解析 body
		if err := json.Unmarshal(bodyJSON, &msg.Body); err != nil {
			return nil, false, fmt.Errorf("failed to unmarshal body: %w", err)
		}

		messages = append(messages, msg)
	}

	// 判断是否还有更多消息
	hasMore := false
	if len(messages) > int(limit) {
		hasMore = true
		messages = messages[:limit]
	}

	return messages, hasMore, nil
}

// UpdateReadSeq 更新已读位置
func (r *Repository) UpdateReadSeq(ctx context.Context, convID int64, userID int64, seq int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE conversation_members
		SET last_read_seq = $1
		WHERE conv_id = $2 AND user_id = $3 AND last_read_seq < $1
	`, seq, convID, userID)

	if err != nil {
		return fmt.Errorf("failed to update read seq: %w", err)
	}

	return nil
}

// GetConversationMembers 获取会话成员 ID 列表
func (r *Repository) GetConversationMembers(ctx context.Context, convID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT user_id FROM conversation_members WHERE conv_id = $1
	`, convID)

	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}
	defer rows.Close()

	var memberIDs []int64
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		memberIDs = append(memberIDs, userID)
	}

	return memberIDs, nil
}

// GenerateMessageID 生成消息 ID
func GenerateMessageID() string {
	return uuid.New().String()
}

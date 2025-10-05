package message

import (
	"context"

	"github.com/dollarkillerx/im-system/pkg/types"
)

// MessageRepository defines the interface for message data persistence
type MessageRepository interface {
	// SaveMessage saves a message to the database
	SaveMessage(ctx context.Context, msg *Message) error

	// PullMessages retrieves messages from a conversation since a given sequence number
	PullMessages(ctx context.Context, convID int64, sinceSeq int64, limit int32) ([]*Message, bool, error)

	// GetNextSeq generates the next sequence number for a conversation
	GetNextSeq(ctx context.Context, convID int64) (int64, error)

	// CreateConversation creates a new conversation with members
	CreateConversation(ctx context.Context, convType types.ConversationType, title string, ownerID int64, memberIDs []int64) (int64, error)

	// GetConversation retrieves conversation details and its members
	GetConversation(ctx context.Context, convID int64) (*Conversation, []*ConversationMember, error)

	// UpdateReadSeq updates the last read sequence number for a user in a conversation
	UpdateReadSeq(ctx context.Context, convID int64, userID int64, seq int64) error

	// GetConversationMembers retrieves all member IDs of a conversation
	GetConversationMembers(ctx context.Context, convID int64) ([]int64, error)
}

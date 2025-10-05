package message

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dollarkillerx/im-system/pkg/logger"
	"github.com/dollarkillerx/im-system/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	_ = logger.Init("error", "console", []string{"stdout"})
}

// MockMessageRepository is a mock implementation of MessageRepository
type MockMessageRepository struct {
	messages        map[int64][]*Message
	conversations   map[int64]*Conversation
	members         map[int64][]*ConversationMember
	seqCounters     map[int64]int64
	getNextSeqFunc  func(ctx context.Context, convID int64) (int64, error)
	saveMessageFunc func(ctx context.Context, msg *Message) error
}

func newMockMessageRepository() *MockMessageRepository {
	return &MockMessageRepository{
		messages:      make(map[int64][]*Message),
		conversations: make(map[int64]*Conversation),
		members:       make(map[int64][]*ConversationMember),
		seqCounters:   make(map[int64]int64),
	}
}

func (m *MockMessageRepository) GetNextSeq(ctx context.Context, convID int64) (int64, error) {
	if m.getNextSeqFunc != nil {
		return m.getNextSeqFunc(ctx, convID)
	}
	m.seqCounters[convID]++
	return m.seqCounters[convID], nil
}

func (m *MockMessageRepository) SaveMessage(ctx context.Context, msg *Message) error {
	if m.saveMessageFunc != nil {
		return m.saveMessageFunc(ctx, msg)
	}
	m.messages[msg.ConvID] = append(m.messages[msg.ConvID], msg)
	return nil
}

func (m *MockMessageRepository) PullMessages(ctx context.Context, convID int64, sinceSeq int64, limit int32) ([]*Message, bool, error) {
	msgs := m.messages[convID]
	var result []*Message
	for _, msg := range msgs {
		if msg.Seq > sinceSeq {
			result = append(result, msg)
		}
	}

	hasMore := int32(len(result)) > limit
	if hasMore {
		result = result[:limit]
	}

	return result, hasMore, nil
}

func (m *MockMessageRepository) CreateConversation(ctx context.Context, convType types.ConversationType, title string, ownerID int64, memberIDs []int64) (int64, error) {
	convID := int64(len(m.conversations) + 1)
	m.conversations[convID] = &Conversation{
		ID:        convID,
		Type:      convType,
		Title:     title,
		OwnerID:   ownerID,
		CreatedAt: time.Now(),
	}

	var members []*ConversationMember
	for _, userID := range memberIDs {
		role := types.ConversationRoleMember
		if userID == ownerID {
			role = types.ConversationRoleOwner
		}
		members = append(members, &ConversationMember{
			ConvID:      convID,
			UserID:      userID,
			Role:        role,
			LastReadSeq: 0,
		})
	}
	m.members[convID] = members

	return convID, nil
}

func (m *MockMessageRepository) GetConversation(ctx context.Context, convID int64) (*Conversation, []*ConversationMember, error) {
	conv, ok := m.conversations[convID]
	if !ok {
		return nil, nil, errors.New("conversation not found")
	}
	return conv, m.members[convID], nil
}

func (m *MockMessageRepository) UpdateReadSeq(ctx context.Context, convID int64, userID int64, seq int64) error {
	members := m.members[convID]
	for _, member := range members {
		if member.UserID == userID {
			member.LastReadSeq = seq
			return nil
		}
	}
	return errors.New("member not found")
}

func (m *MockMessageRepository) GetConversationMembers(ctx context.Context, convID int64) ([]int64, error) {
	members := m.members[convID]
	var userIDs []int64
	for _, member := range members {
		userIDs = append(userIDs, member.UserID)
	}
	return userIDs, nil
}

// Use the existing MockRouterClient from router_client.go

func TestService_SendMessage(t *testing.T) {
	tests := []struct {
		name      string
		convID    int64
		senderID  int64
		convType  types.ConversationType
		body      map[string]interface{}
		setupMock func(*MockMessageRepository)
		wantErr   bool
	}{
		{
			name:     "send text message",
			convID:   1,
			senderID: 100,
			convType: types.ConversationTypeDirect,
			body: map[string]interface{}{
				"type":    "text",
				"content": "Hello, World!",
			},
			wantErr: false,
		},
		{
			name:     "send image message",
			convID:   1,
			senderID: 100,
			convType: types.ConversationTypeGroup,
			body: map[string]interface{}{
				"type":    "image",
				"file_id": "file-123",
				"width":   1920,
				"height":  1080,
			},
			wantErr: false,
		},
		{
			name:     "failed to get next seq",
			convID:   1,
			senderID: 100,
			convType: types.ConversationTypeDirect,
			body:     map[string]interface{}{"type": "text"},
			setupMock: func(m *MockMessageRepository) {
				m.getNextSeqFunc = func(ctx context.Context, convID int64) (int64, error) {
					return 0, errors.New("database error")
				}
			},
			wantErr: true,
		},
		{
			name:     "failed to save message",
			convID:   1,
			senderID: 100,
			convType: types.ConversationTypeDirect,
			body:     map[string]interface{}{"type": "text"},
			setupMock: func(m *MockMessageRepository) {
				m.saveMessageFunc = func(ctx context.Context, msg *Message) error {
					return errors.New("save error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockMessageRepository()
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}
			routerClient := &MockRouterClient{}
			service := NewService(repo, routerClient)

			msgID, seq, timestamp, err := service.SendMessage(
				context.Background(),
				tt.convID,
				tt.senderID,
				tt.convType,
				tt.body,
				nil,
				nil,
			)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, msgID)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, msgID)
				assert.Greater(t, seq, int64(0))
				assert.Greater(t, timestamp, int64(0))
			}
		})
	}
}

func TestService_CreateConversation(t *testing.T) {
	tests := []struct {
		name      string
		convType  types.ConversationType
		title     string
		ownerID   int64
		memberIDs []int64
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "create direct conversation",
			convType:  types.ConversationTypeDirect,
			title:     "",
			ownerID:   100,
			memberIDs: []int64{100, 200},
			wantErr:   false,
		},
		{
			name:      "create group conversation",
			convType:  types.ConversationTypeGroup,
			title:     "Test Group",
			ownerID:   100,
			memberIDs: []int64{100, 200, 300},
			wantErr:   false,
		},
		{
			name:      "create channel",
			convType:  types.ConversationTypeChannel,
			title:     "Test Channel",
			ownerID:   100,
			memberIDs: []int64{100, 200},
			wantErr:   false,
		},
		{
			name:      "invalid conversation type",
			convType:  types.ConversationType("invalid"),
			title:     "Test",
			ownerID:   100,
			memberIDs: []int64{100},
			wantErr:   true,
			errMsg:    "invalid conversation type",
		},
		{
			name:      "owner not in members - should be added",
			convType:  types.ConversationTypeGroup,
			title:     "Test Group",
			ownerID:   100,
			memberIDs: []int64{200, 300},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockMessageRepository()
			routerClient := &MockRouterClient{}
			service := NewService(repo, routerClient)

			convID, err := service.CreateConversation(
				context.Background(),
				tt.convType,
				tt.title,
				tt.ownerID,
				tt.memberIDs,
			)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Equal(t, int64(0), convID)
			} else {
				require.NoError(t, err)
				assert.Greater(t, convID, int64(0))

				// Verify owner was added to members
				conv, members, _ := repo.GetConversation(context.Background(), convID)
				assert.NotNil(t, conv)
				assert.NotEmpty(t, members)

				ownerFound := false
				for _, member := range members {
					if member.UserID == tt.ownerID {
						ownerFound = true
						assert.Equal(t, types.ConversationRoleOwner, member.Role)
						break
					}
				}
				assert.True(t, ownerFound, "Owner should be in members list")
			}
		})
	}
}

func TestService_PullMessages(t *testing.T) {
	repo := newMockMessageRepository()
	routerClient := &MockRouterClient{}
	service := NewService(repo, routerClient)

	// Setup: Create some messages
	convID := int64(1)
	for i := 1; i <= 10; i++ {
		_, _, _, _ = service.SendMessage(
			context.Background(),
			convID,
			100,
			types.ConversationTypeDirect,
			map[string]interface{}{"type": "text", "content": "test"},
			nil,
			nil,
		)
	}

	tests := []struct {
		name          string
		convID        int64
		sinceSeq      int64
		limit         int32
		expectCount   int
		expectHasMore bool
		wantErr       bool
	}{
		{
			name:          "pull first batch",
			convID:        1,
			sinceSeq:      0,
			limit:         5,
			expectCount:   5,
			expectHasMore: true,
			wantErr:       false,
		},
		{
			name:          "pull all messages",
			convID:        1,
			sinceSeq:      0,
			limit:         20,
			expectCount:   10,
			expectHasMore: false,
			wantErr:       false,
		},
		{
			name:          "pull with offset",
			convID:        1,
			sinceSeq:      5,
			limit:         10,
			expectCount:   5,
			expectHasMore: false,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages, hasMore, err := service.PullMessages(
				context.Background(),
				tt.convID,
				tt.sinceSeq,
				tt.limit,
			)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, messages, tt.expectCount)
				assert.Equal(t, tt.expectHasMore, hasMore)
			}
		})
	}
}

func TestService_GetConversation(t *testing.T) {
	repo := newMockMessageRepository()
	routerClient := &MockRouterClient{}
	service := NewService(repo, routerClient)

	// Create a conversation
	convID, err := service.CreateConversation(
		context.Background(),
		types.ConversationTypeGroup,
		"Test Group",
		100,
		[]int64{100, 200, 300},
	)
	require.NoError(t, err)

	tests := []struct {
		name    string
		convID  int64
		wantErr bool
	}{
		{
			name:    "conversation found",
			convID:  convID,
			wantErr: false,
		},
		{
			name:    "conversation not found",
			convID:  999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, members, err := service.GetConversation(context.Background(), tt.convID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, conv)
				assert.Equal(t, convID, conv.ID)
				assert.Equal(t, types.ConversationTypeGroup, conv.Type)
				assert.Equal(t, "Test Group", conv.Title)
				assert.Len(t, members, 3)
			}
		})
	}
}

func TestService_UpdateReadSeq(t *testing.T) {
	repo := newMockMessageRepository()
	routerClient := &MockRouterClient{}
	service := NewService(repo, routerClient)

	// Create conversation
	convID, _ := service.CreateConversation(
		context.Background(),
		types.ConversationTypeDirect,
		"",
		100,
		[]int64{100, 200},
	)

	tests := []struct {
		name    string
		convID  int64
		userID  int64
		seq     int64
		wantErr bool
	}{
		{
			name:    "update read seq successfully",
			convID:  convID,
			userID:  100,
			seq:     50,
			wantErr: false,
		},
		{
			name:    "update for non-member",
			convID:  convID,
			userID:  999,
			seq:     60,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.UpdateReadSeq(
				context.Background(),
				tt.convID,
				tt.userID,
				tt.seq,
			)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

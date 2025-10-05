package message

import (
	"context"
	"testing"
	"time"

	"github.com/dollarkillerx/im-system/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRepository is a mock implementation of Repository
type MockRepository struct {
	getNextSeqFunc              func(ctx context.Context, convID int64) (int64, error)
	saveMessageFunc             func(ctx context.Context, msg *Message) error
	pullMessagesFunc            func(ctx context.Context, convID int64, sinceSeq int64, limit int32) ([]*Message, bool, error)
	createConversationFunc      func(ctx context.Context, convType types.ConversationType, title string, ownerID int64, memberIDs []int64) (int64, error)
	getConversationFunc         func(ctx context.Context, convID int64) (*Conversation, []*ConversationMember, error)
	updateReadSeqFunc           func(ctx context.Context, convID int64, userID int64, seq int64) error
	getConversationMembersFunc  func(ctx context.Context, convID int64) ([]int64, error)

	// In-memory storage
	conversations map[int64]*Conversation
	members       map[int64][]*ConversationMember
	messages      map[int64][]*Message
	seqCounters   map[int64]int64
}

func newMockRepository() *MockRepository {
	return &MockRepository{
		conversations: make(map[int64]*Conversation),
		members:       make(map[int64][]*ConversationMember),
		messages:      make(map[int64][]*Message),
		seqCounters:   make(map[int64]int64),
	}
}

func (m *MockRepository) GetNextSeq(ctx context.Context, convID int64) (int64, error) {
	if m.getNextSeqFunc != nil {
		return m.getNextSeqFunc(ctx, convID)
	}
	m.seqCounters[convID]++
	return m.seqCounters[convID], nil
}

func (m *MockRepository) SaveMessage(ctx context.Context, msg *Message) error {
	if m.saveMessageFunc != nil {
		return m.saveMessageFunc(ctx, msg)
	}
	m.messages[msg.ConvID] = append(m.messages[msg.ConvID], msg)
	return nil
}

func (m *MockRepository) PullMessages(ctx context.Context, convID int64, sinceSeq int64, limit int32) ([]*Message, bool, error) {
	if m.pullMessagesFunc != nil {
		return m.pullMessagesFunc(ctx, convID, sinceSeq, limit)
	}

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

func (m *MockRepository) CreateConversation(ctx context.Context, convType types.ConversationType, title string, ownerID int64, memberIDs []int64) (int64, error) {
	if m.createConversationFunc != nil {
		return m.createConversationFunc(ctx, convType, title, ownerID, memberIDs)
	}

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

func (m *MockRepository) GetConversation(ctx context.Context, convID int64) (*Conversation, []*ConversationMember, error) {
	if m.getConversationFunc != nil {
		return m.getConversationFunc(ctx, convID)
	}

	conv, ok := m.conversations[convID]
	if !ok {
		return nil, nil, assert.AnError
	}

	return conv, m.members[convID], nil
}

func (m *MockRepository) UpdateReadSeq(ctx context.Context, convID int64, userID int64, seq int64) error {
	if m.updateReadSeqFunc != nil {
		return m.updateReadSeqFunc(ctx, convID, userID, seq)
	}

	members := m.members[convID]
	for _, member := range members {
		if member.UserID == userID {
			member.LastReadSeq = seq
			return nil
		}
	}

	return assert.AnError
}

func (m *MockRepository) GetConversationMembers(ctx context.Context, convID int64) ([]int64, error) {
	if m.getConversationMembersFunc != nil {
		return m.getConversationMembersFunc(ctx, convID)
	}

	members := m.members[convID]
	var userIDs []int64
	for _, member := range members {
		userIDs = append(userIDs, member.UserID)
	}

	return userIDs, nil
}

// MockRouterClient is a mock implementation of RouterClient
type MockRouterClient struct {
	notifyNewMessageFunc func(ctx context.Context, convID int64, msgID string, seq int64, senderID int64, recipientIDs []int64) (int32, error)
}

func (m *MockRouterClient) NotifyNewMessage(ctx context.Context, convID int64, msgID string, seq int64, senderID int64, recipientIDs []int64) (int32, error) {
	if m.notifyNewMessageFunc != nil {
		return m.notifyNewMessageFunc(ctx, convID, msgID, seq, senderID, recipientIDs)
	}
	return int32(len(recipientIDs)), nil
}

func TestNewService(t *testing.T) {
	repo := newMockRepository()
	routerClient := &MockRouterClient{}

	service := NewService(repo, routerClient)

	assert.NotNil(t, service)
	assert.NotNil(t, service.repo)
	assert.NotNil(t, service.routerClient)
}

func TestService_SendMessage(t *testing.T) {
	tests := []struct {
		name     string
		convID   int64
		senderID int64
		convType types.ConversationType
		body     map[string]interface{}
		wantErr  bool
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
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
			repo := newMockRepository()
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
			}
		})
	}
}

func TestService_PullMessages(t *testing.T) {
	repo := newMockRepository()
	routerClient := &MockRouterClient{}
	service := NewService(repo, routerClient)

	// Setup: Create some messages
	convID := int64(1)
	for i := 0; i < 10; i++ {
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

	// Give some time for async notifications
	time.Sleep(10 * time.Millisecond)

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
	repo := newMockRepository()
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

	conv, members, err := service.GetConversation(context.Background(), convID)

	require.NoError(t, err)
	assert.NotNil(t, conv)
	assert.Equal(t, convID, conv.ID)
	assert.Equal(t, types.ConversationTypeGroup, conv.Type)
	assert.Equal(t, "Test Group", conv.Title)
	assert.Len(t, members, 3)
}

func TestService_UpdateReadSeq(t *testing.T) {
	repo := newMockRepository()
	routerClient := &MockRouterClient{}
	service := NewService(repo, routerClient)

	// Create conversation and member
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
			name:    "update read seq for another user",
			convID:  convID,
			userID:  200,
			seq:     60,
			wantErr: false,
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

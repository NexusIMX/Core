package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConversationType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		convType ConversationType
		want     bool
	}{
		{
			name:     "direct conversation",
			convType: ConversationTypeDirect,
			want:     true,
		},
		{
			name:     "group conversation",
			convType: ConversationTypeGroup,
			want:     true,
		},
		{
			name:     "channel conversation",
			convType: ConversationTypeChannel,
			want:     true,
		},
		{
			name:     "invalid conversation type",
			convType: ConversationType("invalid"),
			want:     false,
		},
		{
			name:     "empty conversation type",
			convType: ConversationType(""),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.convType.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConversationType_String(t *testing.T) {
	tests := []struct {
		name     string
		convType ConversationType
		want     string
	}{
		{
			name:     "direct to string",
			convType: ConversationTypeDirect,
			want:     "direct",
		},
		{
			name:     "group to string",
			convType: ConversationTypeGroup,
			want:     "group",
		},
		{
			name:     "channel to string",
			convType: ConversationTypeChannel,
			want:     "channel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.convType.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConversationRole_IsValid(t *testing.T) {
	tests := []struct {
		name string
		role ConversationRole
		want bool
	}{
		{
			name: "owner role",
			role: ConversationRoleOwner,
			want: true,
		},
		{
			name: "admin role",
			role: ConversationRoleAdmin,
			want: true,
		},
		{
			name: "publisher role",
			role: ConversationRolePublisher,
			want: true,
		},
		{
			name: "member role",
			role: ConversationRoleMember,
			want: true,
		},
		{
			name: "viewer role",
			role: ConversationRoleViewer,
			want: true,
		},
		{
			name: "invalid role",
			role: ConversationRole("invalid"),
			want: false,
		},
		{
			name: "empty role",
			role: ConversationRole(""),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.role.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConversationRole_String(t *testing.T) {
	tests := []struct {
		name string
		role ConversationRole
		want string
	}{
		{
			name: "owner to string",
			role: ConversationRoleOwner,
			want: "owner",
		},
		{
			name: "admin to string",
			role: ConversationRoleAdmin,
			want: "admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.role.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConversationRole_CanSendMessage(t *testing.T) {
	tests := []struct {
		name     string
		role     ConversationRole
		convType ConversationType
		want     bool
	}{
		// Direct conversation tests
		{
			name:     "owner can send in direct",
			role:     ConversationRoleOwner,
			convType: ConversationTypeDirect,
			want:     true,
		},
		{
			name:     "member can send in direct",
			role:     ConversationRoleMember,
			convType: ConversationTypeDirect,
			want:     true,
		},
		{
			name:     "viewer cannot send in direct",
			role:     ConversationRoleViewer,
			convType: ConversationTypeDirect,
			want:     false,
		},
		// Group conversation tests
		{
			name:     "owner can send in group",
			role:     ConversationRoleOwner,
			convType: ConversationTypeGroup,
			want:     true,
		},
		{
			name:     "admin can send in group",
			role:     ConversationRoleAdmin,
			convType: ConversationTypeGroup,
			want:     true,
		},
		{
			name:     "member can send in group",
			role:     ConversationRoleMember,
			convType: ConversationTypeGroup,
			want:     true,
		},
		{
			name:     "viewer cannot send in group",
			role:     ConversationRoleViewer,
			convType: ConversationTypeGroup,
			want:     false,
		},
		// Channel conversation tests
		{
			name:     "owner can send in channel",
			role:     ConversationRoleOwner,
			convType: ConversationTypeChannel,
			want:     true,
		},
		{
			name:     "admin can send in channel",
			role:     ConversationRoleAdmin,
			convType: ConversationTypeChannel,
			want:     true,
		},
		{
			name:     "publisher can send in channel",
			role:     ConversationRolePublisher,
			convType: ConversationTypeChannel,
			want:     true,
		},
		{
			name:     "member cannot send in channel",
			role:     ConversationRoleMember,
			convType: ConversationTypeChannel,
			want:     false,
		},
		{
			name:     "viewer cannot send in channel",
			role:     ConversationRoleViewer,
			convType: ConversationTypeChannel,
			want:     false,
		},
		// Invalid conversation type
		{
			name:     "invalid conversation type",
			role:     ConversationRoleOwner,
			convType: ConversationType("invalid"),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.role.CanSendMessage(tt.convType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConversationRole_CanManageMembers(t *testing.T) {
	tests := []struct {
		name string
		role ConversationRole
		want bool
	}{
		{
			name: "owner can manage members",
			role: ConversationRoleOwner,
			want: true,
		},
		{
			name: "admin can manage members",
			role: ConversationRoleAdmin,
			want: true,
		},
		{
			name: "publisher cannot manage members",
			role: ConversationRolePublisher,
			want: false,
		},
		{
			name: "member cannot manage members",
			role: ConversationRoleMember,
			want: false,
		},
		{
			name: "viewer cannot manage members",
			role: ConversationRoleViewer,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.role.CanManageMembers()
			assert.Equal(t, tt.want, got)
		})
	}
}

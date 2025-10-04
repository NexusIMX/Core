package types

// ConversationType represents the type of conversation
type ConversationType string

const (
	ConversationTypeDirect  ConversationType = "direct"
	ConversationTypeGroup   ConversationType = "group"
	ConversationTypeChannel ConversationType = "channel"
)

// IsValid checks if the conversation type is valid
func (ct ConversationType) IsValid() bool {
	switch ct {
	case ConversationTypeDirect, ConversationTypeGroup, ConversationTypeChannel:
		return true
	}
	return false
}

// String returns the string representation
func (ct ConversationType) String() string {
	return string(ct)
}

// ConversationRole represents the role of a user in a conversation
type ConversationRole string

const (
	ConversationRoleOwner     ConversationRole = "owner"
	ConversationRoleAdmin     ConversationRole = "admin"
	ConversationRolePublisher ConversationRole = "publisher"
	ConversationRoleMember    ConversationRole = "member"
	ConversationRoleViewer    ConversationRole = "viewer"
)

// IsValid checks if the conversation role is valid
func (cr ConversationRole) IsValid() bool {
	switch cr {
	case ConversationRoleOwner, ConversationRoleAdmin, ConversationRolePublisher,
		ConversationRoleMember, ConversationRoleViewer:
		return true
	}
	return false
}

// String returns the string representation
func (cr ConversationRole) String() string {
	return string(cr)
}

// CanSendMessage checks if the role can send messages
func (cr ConversationRole) CanSendMessage(convType ConversationType) bool {
	switch convType {
	case ConversationTypeDirect, ConversationTypeGroup:
		return cr != ConversationRoleViewer
	case ConversationTypeChannel:
		return cr == ConversationRoleOwner || cr == ConversationRoleAdmin || cr == ConversationRolePublisher
	default:
		return false
	}
}

// CanManageMembers checks if the role can manage members
func (cr ConversationRole) CanManageMembers() bool {
	return cr == ConversationRoleOwner || cr == ConversationRoleAdmin
}

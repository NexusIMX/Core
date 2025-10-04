package message

import (
	"context"

	messagepb "github.com/dollarkillerx/im-system/api/proto/message"
	"github.com/dollarkillerx/im-system/pkg/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type GRPCServer struct {
	messagepb.UnimplementedMessageServiceServer
	service *Service
}

func NewGRPCServer(service *Service) *GRPCServer {
	return &GRPCServer{service: service}
}

func (s *GRPCServer) SendMessage(ctx context.Context, req *messagepb.SendMessageRequest) (*messagepb.SendMessageResponse, error) {
	// 转换 conv_type
	convType := types.ConversationType("")
	switch req.ConvType {
	case messagepb.ConversationType_DIRECT:
		convType = types.ConversationTypeDirect
	case messagepb.ConversationType_GROUP:
		convType = types.ConversationTypeGroup
	case messagepb.ConversationType_CHANNEL:
		convType = types.ConversationTypeChannel
	default:
		return nil, status.Errorf(codes.InvalidArgument, "invalid conversation type")
	}

	// 转换 body
	body := req.Body.AsMap()

	msgID, seq, createdAt, err := s.service.SendMessage(
		ctx,
		req.ConvId,
		req.SenderId,
		convType,
		body,
		req.ReplyTo,
		req.Mentions,
	)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send message: %v", err)
	}

	return &messagepb.SendMessageResponse{
		MsgId:     msgID,
		Seq:       seq,
		CreatedAt: createdAt,
	}, nil
}

func (s *GRPCServer) PullMessages(ctx context.Context, req *messagepb.PullMessagesRequest) (*messagepb.PullMessagesResponse, error) {
	messages, hasMore, err := s.service.PullMessages(ctx, req.ConvId, req.SinceSeq, req.Limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to pull messages: %v", err)
	}

	var pbMessages []*messagepb.Message
	for _, msg := range messages {
		// 转换 conv_type
		var pbConvType messagepb.ConversationType
		switch msg.ConvType {
		case types.ConversationTypeDirect:
			pbConvType = messagepb.ConversationType_DIRECT
		case types.ConversationTypeGroup:
			pbConvType = messagepb.ConversationType_GROUP
		case types.ConversationTypeChannel:
			pbConvType = messagepb.ConversationType_CHANNEL
		}

		// 转换 body
		bodyStruct, err := structpb.NewStruct(msg.Body)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert body: %v", err)
		}

		pbMsg := &messagepb.Message{
			MsgId:      msg.MsgID,
			ConvId:     msg.ConvID,
			Seq:        msg.Seq,
			SenderId:   msg.SenderID,
			ConvType:   pbConvType,
			Body:       bodyStruct,
			ReplyTo:    msg.ReplyTo,
			Mentions:   msg.Mentions,
			Visibility: msg.Visibility,
			CreatedAt:  msg.CreatedAt.Unix(),
		}

		pbMessages = append(pbMessages, pbMsg)
	}

	return &messagepb.PullMessagesResponse{
		Messages: pbMessages,
		HasMore:  hasMore,
	}, nil
}

func (s *GRPCServer) GetConversation(ctx context.Context, req *messagepb.GetConversationRequest) (*messagepb.GetConversationResponse, error) {
	conv, members, err := s.service.GetConversation(ctx, req.ConvId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "conversation not found: %v", err)
	}

	// 转换 conv_type
	var pbConvType messagepb.ConversationType
	switch conv.Type {
	case types.ConversationTypeDirect:
		pbConvType = messagepb.ConversationType_DIRECT
	case types.ConversationTypeGroup:
		pbConvType = messagepb.ConversationType_GROUP
	case types.ConversationTypeChannel:
		pbConvType = messagepb.ConversationType_CHANNEL
	}

	// 转换成员列表
	var pbMembers []*messagepb.ConversationMember
	for _, member := range members {
		// 转换 role
		var pbRole messagepb.ConversationRole
		switch member.Role {
		case types.ConversationRoleOwner:
			pbRole = messagepb.ConversationRole_OWNER
		case types.ConversationRoleAdmin:
			pbRole = messagepb.ConversationRole_ADMIN
		case types.ConversationRolePublisher:
			pbRole = messagepb.ConversationRole_PUBLISHER
		case types.ConversationRoleMember:
			pbRole = messagepb.ConversationRole_MEMBER
		case types.ConversationRoleViewer:
			pbRole = messagepb.ConversationRole_VIEWER
		}

		pbMembers = append(pbMembers, &messagepb.ConversationMember{
			UserId:      member.UserID,
			Role:        pbRole,
			Muted:       member.Muted,
			LastReadSeq: member.LastReadSeq,
			JoinedAt:    member.JoinedAt.Unix(),
		})
	}

	return &messagepb.GetConversationResponse{
		Conversation: &messagepb.Conversation{
			Id:        conv.ID,
			Type:      pbConvType,
			Title:     conv.Title,
			OwnerId:   conv.OwnerID,
			CreatedAt: conv.CreatedAt.Unix(),
			Members:   pbMembers,
		},
	}, nil
}

func (s *GRPCServer) CreateConversation(ctx context.Context, req *messagepb.CreateConversationRequest) (*messagepb.CreateConversationResponse, error) {
	// 转换 conv_type
	convType := types.ConversationType("")
	switch req.Type {
	case messagepb.ConversationType_DIRECT:
		convType = types.ConversationTypeDirect
	case messagepb.ConversationType_GROUP:
		convType = types.ConversationTypeGroup
	case messagepb.ConversationType_CHANNEL:
		convType = types.ConversationTypeChannel
	default:
		return nil, status.Errorf(codes.InvalidArgument, "invalid conversation type")
	}

	convID, err := s.service.CreateConversation(ctx, convType, req.Title, req.OwnerId, req.MemberIds)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create conversation: %v", err)
	}

	return &messagepb.CreateConversationResponse{
		ConvId:  convID,
		Message: "Conversation created successfully",
	}, nil
}

func (s *GRPCServer) UpdateReadSeq(ctx context.Context, req *messagepb.UpdateReadSeqRequest) (*messagepb.UpdateReadSeqResponse, error) {
	err := s.service.UpdateReadSeq(ctx, req.ConvId, req.UserId, req.Seq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update read seq: %v", err)
	}

	return &messagepb.UpdateReadSeqResponse{
		Success: true,
	}, nil
}

func (s *GRPCServer) NotifyNewMessage(ctx context.Context, req *messagepb.NotifyNewMessageRequest) (*messagepb.NotifyNewMessageResponse, error) {
	// 这个方法由 Router/Gateway 调用，暂时返回成功
	// 实际的通知逻辑在 service 的 SendMessage 中处理
	return &messagepb.NotifyNewMessageResponse{
		Success:       true,
		NotifiedCount: int32(len(req.RecipientIds)),
	}, nil
}

package user

import (
	"context"

	userpb "github.com/dollarkillerx/im-system/api/proto/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCServer struct {
	userpb.UnimplementedUserServiceServer
	service *Service
}

func NewGRPCServer(service *Service) *GRPCServer {
	return &GRPCServer{service: service}
}

func (s *GRPCServer) Register(ctx context.Context, req *userpb.RegisterRequest) (*userpb.RegisterResponse, error) {
	userID, err := s.service.Register(ctx, req.Username, req.Password, req.Email, req.Nickname)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to register: %v", err)
	}

	return &userpb.RegisterResponse{
		UserId:  userID,
		Message: "User registered successfully",
	}, nil
}

func (s *GRPCServer) Login(ctx context.Context, req *userpb.LoginRequest) (*userpb.LoginResponse, error) {
	userID, token, expiresAt, user, err := s.service.Login(ctx, req.Username, req.Password, req.DeviceId)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "login failed: %v", err)
	}

	return &userpb.LoginResponse{
		UserId:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
		UserInfo: &userpb.UserInfo{
			UserId:    user.ID,
			Username:  user.Username,
			Nickname:  user.Nickname,
			Email:     user.Email,
			Avatar:    user.Avatar,
			Bio:       user.Bio,
			CreatedAt: user.CreatedAt.Unix(),
		},
	}, nil
}

func (s *GRPCServer) GetUserInfo(ctx context.Context, req *userpb.GetUserInfoRequest) (*userpb.GetUserInfoResponse, error) {
	user, err := s.service.GetUserInfo(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	return &userpb.GetUserInfoResponse{
		UserInfo: &userpb.UserInfo{
			UserId:    user.ID,
			Username:  user.Username,
			Nickname:  user.Nickname,
			Email:     user.Email,
			Avatar:    user.Avatar,
			Bio:       user.Bio,
			CreatedAt: user.CreatedAt.Unix(),
		},
	}, nil
}

func (s *GRPCServer) UpdateUserInfo(ctx context.Context, req *userpb.UpdateUserInfoRequest) (*userpb.UpdateUserInfoResponse, error) {
	var nickname, avatar, bio *string

	if req.Nickname != nil {
		nickname = req.Nickname
	}
	if req.Avatar != nil {
		avatar = req.Avatar
	}
	if req.Bio != nil {
		bio = req.Bio
	}

	err := s.service.UpdateUserInfo(ctx, req.UserId, nickname, avatar, bio)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update user: %v", err)
	}

	return &userpb.UpdateUserInfoResponse{
		Success: true,
		Message: "User info updated successfully",
	}, nil
}

func (s *GRPCServer) ValidateToken(ctx context.Context, req *userpb.ValidateTokenRequest) (*userpb.ValidateTokenResponse, error) {
	userID, deviceID, err := s.service.ValidateToken(ctx, req.Token)
	if err != nil {
		return &userpb.ValidateTokenResponse{
			Valid: false,
		}, nil
	}

	return &userpb.ValidateTokenResponse{
		Valid:    true,
		UserId:   userID,
		DeviceId: deviceID,
	}, nil
}

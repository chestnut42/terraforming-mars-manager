package app

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/chestnut42/terraforming-mars-manager/internal/auth"
	"github.com/chestnut42/terraforming-mars-manager/internal/framework/httpx"
	"github.com/chestnut42/terraforming-mars-manager/internal/framework/logx"
	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
	"github.com/chestnut42/terraforming-mars-manager/pkg/api"
)

func (s *Service) Login(ctx context.Context, _ *api.Login_Request) (*api.Login_Response, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not found")
	}
	newNickName := fmt.Sprintf("Player %X", rand.Int())
	curIP, ok := httpx.RemoteAddrFromContext(ctx)
	if !ok {
		curIP = ""
		logx.Logger(ctx).Warn("no IP in context")
	}

	if err := s.storage.UpsertUser(ctx, &storage.User{
		UserId:   user.Id,
		Nickname: newNickName,
		LastIp:   curIP,
	}); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	storageUser, err := s.storage.GetUserById(ctx, user.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &api.Login_Response{
		User: userToAPI(storageUser),
	}, nil
}

func (s *Service) GetMe(ctx context.Context, _ *api.GetMe_Request) (*api.GetMe_Response, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not found")
	}

	storageUser, err := s.storage.GetUserById(ctx, user.Id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &api.GetMe_Response{
		User: userToAPI(storageUser),
	}, nil
}

func (s *Service) UpdateMe(ctx context.Context, req *api.UpdateMe_Request) (*api.UpdateMe_Response, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not found")
	}

	col, err := fromAPIColor(req.GetColor())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	storageUser, err := s.storage.UpdateUser(ctx, &storage.User{
		UserId:   user.Id,
		Nickname: req.GetNickname(),
		Color:    col,
	})
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		if errors.Is(err, storage.ErrAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &api.UpdateMe_Response{
		User: userToAPI(storageUser),
	}, nil
}

func (s *Service) UpdateDeviceToken(ctx context.Context, req *api.UpdateDeviceToken_Request) (*api.UpdateDeviceToken_Response, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not found")
	}

	if err := s.storage.UpdateDeviceToken(ctx, user.Id, req.GetDeviceToken(), storage.DeviceTokenTypeProduction); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &api.UpdateDeviceToken_Response{}, nil
}

func (s *Service) SearchUser(ctx context.Context, req *api.SearchUser_Request) (*api.SearchUser_Response, error) {
	thisUser, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not found")
	}

	users, err := s.storage.SearchUsers(ctx, req.GetSearch(), 5, thisUser.Id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return &api.SearchUser_Response{Users: make([]*api.User, 0)}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	respUsers := make([]*api.User, len(users))
	for i, user := range users {
		respUsers[i] = userToAPI(user)
	}
	return &api.SearchUser_Response{
		Users: respUsers,
	}, nil
}

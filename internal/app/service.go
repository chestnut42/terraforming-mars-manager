package app

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/chestnut42/terraforming-mars-manager/internal/auth"
	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
	"github.com/chestnut42/terraforming-mars-manager/pkg/api"
)

type Storage interface {
	GetUserById(ctx context.Context, userId string) (*storage.User, error)
	UpdateUser(ctx context.Context, user *storage.User) (*storage.User, error)
	UpsertUser(ctx context.Context, userId string) error
}

type Service struct {
	storage Storage

	api.UnsafeUsersServer
}

func NewService(storage Storage) *Service {
	return &Service{
		storage: storage,
	}
}

func (s *Service) Login(ctx context.Context, req *api.Login_Request) (*api.Login_Response, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not found")
	}

	if err := s.storage.UpsertUser(ctx, user.Id); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	storageUser, err := s.storage.GetUserById(ctx, user.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &api.Login_Response{
		UserId:    storageUser.UserId,
		Nickname:  storageUser.Nickname,
		CreatedAt: timestamppb.New(storageUser.CreatedAt),
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
		UserId:    storageUser.UserId,
		Nickname:  storageUser.Nickname,
		CreatedAt: timestamppb.New(storageUser.CreatedAt),
	}, nil
}

func (s *Service) UpdateMe(ctx context.Context, req *api.UpdateMe_Request) (*api.UpdateMe_Response, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not found")
	}

	storageUser, err := s.storage.UpdateUser(ctx, &storage.User{
		UserId:   user.Id,
		Nickname: req.GetNickname(),
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
		UserId:    storageUser.UserId,
		Nickname:  storageUser.Nickname,
		CreatedAt: timestamppb.New(storageUser.CreatedAt),
	}, nil
}

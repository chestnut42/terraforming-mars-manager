package app

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/chestnut42/terraforming-mars-manager/internal/auth"
	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
	"github.com/chestnut42/terraforming-mars-manager/pkg/api"
)

type Storage interface {
	GetUserById(ctx context.Context, userId string) (*storage.User, error)
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

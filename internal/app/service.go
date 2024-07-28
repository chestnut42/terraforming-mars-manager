package app

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/chestnut42/terraforming-mars-manager/pkg/api"
)

type Service struct {
	api.UnsafeUsersServer
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Login(ctx context.Context, req *api.Login_Request) (*api.Login_Response, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

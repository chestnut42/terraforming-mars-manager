package app

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/chestnut42/terraforming-mars-manager/internal/auth"
	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
	"github.com/chestnut42/terraforming-mars-manager/pkg/api"
)

func (s *Service) CreateGame(ctx context.Context, req *api.CreateGame_Request) (*api.CreateGame_Response, error) {
	thisUser, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not found")
	}

	// Check if all users are unique
	if !isUnique(req.GetPlayers()) {
		return nil, status.Error(codes.InvalidArgument, "players are not unique")
	}

	users := make([]*storage.User, len(req.GetPlayers()))
	for i, player := range req.GetPlayers() {
		u, err := s.storage.GetUserByNickname(ctx, player)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return nil, status.Error(codes.NotFound, "user not found")
			}
			return nil, status.Error(codes.Internal, err.Error())
		}
		users[i] = u
	}

	hasThisUser := false
	for _, u := range users {
		if u.UserId == thisUser.Id {
			hasThisUser = true
		}
	}
	if !hasThisUser {
		return nil, status.Error(codes.InvalidArgument, "you can't create a game for somebody else")
	}

	if err := s.game.CreateGame(ctx, users); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &api.CreateGame_Response{}, nil
}

func isUnique(str []string) bool {
	m := make(map[string]struct{})
	for _, v := range str {
		m[v] = struct{}{}
	}
	return len(m) == len(str)
}

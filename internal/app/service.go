package app

import (
	"context"

	"github.com/chestnut42/terraforming-mars-manager/internal/client/mars"
	"github.com/chestnut42/terraforming-mars-manager/internal/service/game"
	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
	"github.com/chestnut42/terraforming-mars-manager/pkg/api"
)

type Storage interface {
	GetLeaderboard(ctx context.Context, ut storage.UserType, limit int64) ([]*storage.User, error)
	GetUserById(ctx context.Context, userId string) (*storage.User, error)
	GetUserByNickname(ctx context.Context, nickname string) (*storage.User, error)
	SearchUsers(ctx context.Context, req storage.SearchUsers) ([]*storage.User, error)
	UpdateDeviceToken(ctx context.Context, userId string, deviceToken []byte, tokenType storage.DeviceTokenType) error
	UpdateUser(ctx context.Context, req storage.UpdateUser) (*storage.User, error)
	UpsertUser(ctx context.Context, req storage.UpsertUser) error
}

type GameService interface {
	CreateGame(ctx context.Context, players []*storage.User, settings mars.GameSettings) error
	GetUserGames(ctx context.Context, userId string) ([]*game.UserGame, error)
}

type Service struct {
	storage Storage
	game    GameService

	api.UnsafeUsersServer
	api.UnsafeGamesServer
}

func NewService(storage Storage, game GameService) *Service {
	return &Service{
		storage: storage,
		game:    game,
	}
}

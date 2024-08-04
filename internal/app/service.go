package app

import (
	"context"

	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
	"github.com/chestnut42/terraforming-mars-manager/pkg/api"
)

type Storage interface {
	GetUserById(ctx context.Context, userId string) (*storage.User, error)
	GetUserByNickname(ctx context.Context, nickname string) (*storage.User, error)
	SearchUsers(ctx context.Context, search string, limit int, excludeUser string) ([]*storage.User, error)
	UpdateDeviceToken(ctx context.Context, userId string, deviceToken []byte) error
	UpdateUser(ctx context.Context, user *storage.User) (*storage.User, error)
	UpsertUser(ctx context.Context, user *storage.User) error
}

type GameService interface {
	CreateGame(ctx context.Context, players []*storage.User) error
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

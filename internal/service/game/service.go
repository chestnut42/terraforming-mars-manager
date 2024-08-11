package game

import (
	"context"

	"github.com/chestnut42/terraforming-mars-manager/internal/client/mars"
	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
)

type Storage interface {
	CreateGame(ctx context.Context, game *storage.Game) error
	GetGamesByUserId(ctx context.Context, userId string) ([]*storage.Game, error)
}

type MarsClient interface {
	CreateGame(ctx context.Context, game mars.CreateGameRequest) (mars.CreateGameResponse, error)
	GetGame(ctx context.Context, req mars.GetGameRequest) (mars.GetGameResponse, error)
	GetPlayerUrl(playerId string) string
	WaitingFor(ctx context.Context, req mars.WaitingForRequest) (mars.WaitingForResponse, error)
}

type Service struct {
	storage Storage
	mars    MarsClient
}

func NewService(storage Storage, mars MarsClient) *Service {
	return &Service{
		storage: storage,
		mars:    mars,
	}
}

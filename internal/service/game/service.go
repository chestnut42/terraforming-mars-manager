package game

import (
	"context"
	"time"

	"github.com/chestnut42/terraforming-mars-manager/internal/client/mars"
	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
)

type Config struct {
	ScanInterval time.Duration
}

type Storage interface {
	CreateGame(ctx context.Context, game *storage.Game) error
	GetActiveGames(ctx context.Context) ([]*storage.Game, error)
	GetGamesByUserId(ctx context.Context, userId string) ([]*storage.Game, error)
	UpdateElo(ctx context.Context, updater storage.EloUpdater) error
	UpdateGameResults(ctx context.Context, gameId string, results *storage.GameResults) error
}

type MarsClient interface {
	CreateGame(ctx context.Context, game mars.CreateGameRequest) (mars.CreateGameResponse, error)
	GetGame(ctx context.Context, req mars.GetGameRequest) (mars.GetGameResponse, error)
	GetPlayerUrl(playerId string) string
	WaitingFor(ctx context.Context, req mars.WaitingForRequest) (mars.WaitingForResponse, error)
}

type Service struct {
	cfg     Config
	storage Storage
	mars    MarsClient

	finishedGames chan string
}

func NewService(cfg Config, storage Storage, mars MarsClient) *Service {
	return &Service{
		cfg:     cfg,
		storage: storage,
		mars:    mars,

		finishedGames: make(chan string),
	}
}

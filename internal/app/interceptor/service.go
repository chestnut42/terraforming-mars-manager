package interceptor

import (
	"context"
	"net/http"

	"github.com/chestnut42/terraforming-mars-manager/internal/client/mars"
	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
)

type Storage interface {
	GetGameByPlayerId(ctx context.Context, playerId string) (*storage.Game, error)
}

type MarsClient interface {
	GetGame(ctx context.Context, req mars.GetGameRequest) (mars.GetGameResponse, error)
	WaitingFor(ctx context.Context, req mars.WaitingForRequest) (mars.WaitingForResponse, error)
}

type Notifier interface {
	NotifyUser(ctx context.Context, userId string) error
}

type GameNotifier interface {
	NotifyGameFinished(ctx context.Context, gameId string) error
}

type Service struct {
	origin       http.Handler
	storage      Storage
	mars         MarsClient
	notifier     Notifier
	gameNotifier GameNotifier
}

func NewService(origin http.Handler, storage Storage, mars MarsClient, notifier Notifier, gameNotifier GameNotifier) *Service {
	return &Service{
		origin:       origin,
		storage:      storage,
		mars:         mars,
		notifier:     notifier,
		gameNotifier: gameNotifier,
	}
}

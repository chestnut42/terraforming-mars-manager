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
	WaitingFor(ctx context.Context, req mars.WaitingForRequest) (mars.WaitingForResponse, error)
}

type Notifier interface {
	NotifyUser(ctx context.Context, userId string) error
}

type Service struct {
	origin   http.Handler
	storage  Storage
	mars     MarsClient
	notifier Notifier
}

func NewService(origin http.Handler, storage Storage, mars MarsClient, notifier Notifier) *Service {
	return &Service{
		origin:   origin,
		storage:  storage,
		mars:     mars,
		notifier: notifier,
	}
}

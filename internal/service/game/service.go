package game

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/chestnut42/terraforming-mars-manager/internal/client/mars"
	"github.com/chestnut42/terraforming-mars-manager/internal/framework/logx"
	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
)

type Storage interface {
	CreateGame(ctx context.Context, game *storage.Game) error
}

type MarsClient interface {
	CreateGame(ctx context.Context, game mars.CreateGame) (mars.CreateGameResponse, error)
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

func (s *Service) CreateGame(ctx context.Context, users []*storage.User) error {
	reqPlayers := make([]mars.NewPlayer, len(users))
	for i, p := range users {
		reqPlayers[i] = mars.NewPlayer{
			Name:  p.Nickname,
			Color: p.Color,
		}
	}
	resp, err := s.mars.CreateGame(ctx, mars.CreateGame{Players: reqPlayers})
	if err != nil {
		return fmt.Errorf("failed to create mars client game: %w", err)
	}
	logx.Logger(ctx).Info("create game", slog.Any("users", users), slog.Any("response", resp))

	gamePlayers := make([]*storage.Player, len(users))
	for i, u := range users {
		for _, p := range resp.Players {
			if u.Nickname == p.Name {
				gamePlayers[i] = &storage.Player{
					UserId:   u.UserId,
					PlayerId: p.Id,
					Color:    p.Color,
				}
			}
		}
		if gamePlayers[i] == nil {
			return fmt.Errorf("player not found: %s", u.Nickname)
		}
	}

	if err := s.storage.CreateGame(ctx, &storage.Game{
		GameId:      resp.Id,
		SpectatorId: resp.SpectatorId,
		ExpiresAt:   resp.PurgeDate,
		Players:     gamePlayers,
	}); err != nil {
		return fmt.Errorf("failed to store the game: %w", err)
	}
	return nil
}

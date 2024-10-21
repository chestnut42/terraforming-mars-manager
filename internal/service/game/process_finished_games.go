package game

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/chestnut42/terraforming-mars-manager/internal/client/mars"
	"github.com/chestnut42/terraforming-mars-manager/internal/framework/logx"
	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
)

func (s *Service) ProcessFinishedGames(ctx context.Context) error {
	for {
		games := s.getGamesToProcess(ctx)
		for _, g := range games {
			if err := s.processGame(ctx, g); err != nil {
				logx.Logger(ctx).Error("failed to process game",
					slog.String("id", g.GameId),
					slog.Any("error", err))
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.cfg.ScanInterval):
		}
	}
}

func (s *Service) getGamesToProcess(ctx context.Context) []*storage.Game {
	games, err := s.storage.GetActiveGames(ctx)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			logx.Logger(ctx).Error("failed to get games from storage", slog.Any("error", err))
		}
		return nil
	}
	return games
}

func (s *Service) processGame(ctx context.Context, game *storage.Game) error {
	r, err := s.mars.GetGame(ctx, mars.GetGameRequest{SpectatorId: game.SpectatorId})
	if err != nil {
		return fmt.Errorf("failed to get game details: %s: %w", game.GameId, err)
	}

	if r.Game.HasFinished {
		if err := s.storage.UpdateGameResults(ctx, game.GameId, &storage.GameResults{Raw: r.Raw}); err != nil {
			return fmt.Errorf("failed to update game results: %s: %w", game.GameId, err)
		}
		logx.Logger(ctx).Info("game finished", slog.String("id", game.GameId))
	}
	return nil
}

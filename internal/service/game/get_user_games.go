package game

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/chestnut42/terraforming-mars-manager/internal/client/mars"
)

type UserGame struct {
	PlayURL      string
	CreatedAt    time.Time
	ExpiresAt    time.Time
	PlayersCount int
	AwaitsInput  bool
}

func (s *Service) GetUserGames(ctx context.Context, userId string) ([]*UserGame, error) {
	games, err := s.storage.GetGamesByUserId(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("get games from storage: %w", err)
	}

	awaitInputs := make([]bool, len(games))
	eg, ctx := errgroup.WithContext(ctx)
	for idx, game := range games {
		idx := idx
		game := game

		if len(game.Players) == 0 || game.Players[0].UserId != userId {
			return nil, fmt.Errorf("unexpected players in the game")
		}
		thisPlayer := game.Players[0]

		eg.Go(func() error {
			wait, err := s.mars.WaitingFor(ctx, mars.WaitingForRequest{SpectatorId: game.SpectatorId})
			if err != nil {
				return fmt.Errorf("waiting for spectator (%s): %w", game.SpectatorId, err)
			}

			userIsWaited := false
			for _, c := range wait.Colors {
				if c == thisPlayer.Color {
					userIsWaited = true
				}
			}
			awaitInputs[idx] = userIsWaited
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	result := make([]*UserGame, len(games))
	for idx, g := range games {
		if len(g.Players) == 0 || g.Players[0].UserId != userId {
			return nil, fmt.Errorf("unexpected players in the game")
		}
		thisPlayer := g.Players[0]

		result[idx] = &UserGame{
			PlayURL:      s.mars.GetPlayerUrl(thisPlayer.PlayerId),
			CreatedAt:    g.CreatedAt,
			ExpiresAt:    g.ExpiresAt,
			PlayersCount: 0, // TODO: grab number of players somewhere
			AwaitsInput:  awaitInputs[idx],
		}
	}
	return result, nil
}

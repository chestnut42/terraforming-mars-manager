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

func (s *Service) GetUserGames(inctx context.Context, userId string) ([]*UserGame, error) {
	games, err := s.storage.GetGamesByUserId(inctx, userId)
	if err != nil {
		return nil, fmt.Errorf("get games from storage: %w", err)
	}

	awaitInputs := make([]bool, len(games))
	eg, ctx := errgroup.WithContext(inctx)
	for idx, game := range games {
		idx := idx
		game := game

		if len(game.Players) == 0 || game.Players[0].UserId != userId {
			return nil, fmt.Errorf("unexpected players in the game")
		}
		thisPlayer := game.Players[0]

		eg.Go(func() error {
			wait, err := s.mars.WaitingFor(ctx, mars.WaitingForRequest{PlayerId: thisPlayer.PlayerId})
			if err != nil {
				return fmt.Errorf("waiting for (%s): %w", thisPlayer.PlayerId, err)
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
	eg, ctx = errgroup.WithContext(inctx)
	for idx, g := range games {
		idx := idx
		g := g

		eg.Go(func() error {
			if len(g.Players) == 0 || g.Players[0].UserId != userId {
				return fmt.Errorf("unexpected players in the game")
			}
			thisPlayer := g.Players[0]

			game, err := s.mars.GetGame(ctx, mars.GetGameRequest{SpectatorId: g.SpectatorId})
			if err != nil {
				return fmt.Errorf("get game from mars: %w", err)
			}

			result[idx] = &UserGame{
				PlayURL:      s.mars.GetPlayerUrl(thisPlayer.PlayerId),
				CreatedAt:    g.CreatedAt,
				ExpiresAt:    g.ExpiresAt,
				PlayersCount: len(game.Game.Players),
				AwaitsInput:  awaitInputs[idx],
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return result, nil
}

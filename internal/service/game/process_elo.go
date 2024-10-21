package game

import (
	"cmp"
	"context"
	"fmt"
	"github.com/chestnut42/terraforming-mars-manager/internal/client/mars"
	"github.com/chestnut42/terraforming-mars-manager/internal/framework/logx"
	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
	"log/slog"
	"math"
	"time"
)

const (
	eloPowerDenominator = float64(480)
	kFactor             = float64(20)
)

func (s *Service) ProcessElo(ctx context.Context) error {
	for {
		if err := s.storage.UpdateElo(ctx, updateElo); err != nil {
			logx.Logger(ctx).Error("failed to update elo",
				slog.Any("error", err))
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.cfg.ScanInterval):
		}
	}
}

func updateElo(ctx context.Context, state storage.EloUpdateState) (storage.EloResults, error) {
	gameResponse, err := mars.GetGameResponseFromRaw(state.Game.GameResults.Raw)
	if err != nil {
		return storage.EloResults{}, fmt.Errorf("failed to get game response from raw: %w", err)
	}

	if len(gameResponse.Game.Players) < 2 {
		return storage.EloResults{}, fmt.Errorf("not enough players")
	}

	players := make([]storage.EloResultsPlayer, len(gameResponse.Game.Players))
	for i, player := range gameResponse.Game.Players {
		user, ok := findUser(state, player.Id)
		if !ok {
			return storage.EloResults{}, fmt.Errorf("player %s not found in game", player.Id)
		}

		players[i] = storage.EloResultsPlayer{
			PlayerId: player.Id,
			UserId:   user.UserId,
			OldElo:   user.Elo,
			NewElo:   user.Elo,
		}
	}

	var pairs []storage.EloResultsPair
	for leftIdx := 0; leftIdx < len(players); leftIdx++ {
		for rightIdx := leftIdx + 1; rightIdx < len(players); rightIdx++ {
			leftPlayer := players[leftIdx]
			rightPlayer := players[rightIdx]

			leftScore := getLeftScore(gameResponse.Game.Players[leftIdx], gameResponse.Game.Players[rightIdx])

			ratingPower := float64(rightPlayer.OldElo-leftPlayer.OldElo) / eloPowerDenominator
			expectedLeftScore := 1. / (1. + math.Pow(10., ratingPower))

			leftEloChange := int64(math.Ceil(kFactor * (leftScore - expectedLeftScore)))

			pairs = append(pairs, storage.EloResultsPair{
				LeftPlayerId:    leftPlayer.PlayerId,
				RightPlayerId:   rightPlayer.PlayerId,
				LeftPlayerElo:   leftPlayer.OldElo,
				RightPlayerElo:  rightPlayer.OldElo,
				LeftEloDelta:    leftEloChange,
				LeftPlayerScore: leftScore,
			})

			players[leftIdx].NewElo += leftEloChange
			players[rightIdx].NewElo -= leftEloChange
		}
	}

	return storage.EloResults{
		Pairs:   pairs,
		Players: players,
	}, nil
}

func findUser(state storage.EloUpdateState, playerId string) (storage.EloStateUser, bool) {
	for _, player := range state.Game.Players {
		if player.PlayerId == playerId {
			for _, user := range state.Users {
				if user.UserId == player.UserId {
					return user, true
				}
			}
		}
	}
	return storage.EloStateUser{}, false
}

func comparePlayers(a, b mars.GetGamePlayer) int {
	vpCmp := cmp.Compare(a.Score, b.Score)
	if vpCmp != 0 {
		return vpCmp
	}
	return cmp.Compare(a.MegaCredits, b.MegaCredits)
}

func getLeftScore(a, b mars.GetGamePlayer) float64 {
	c := comparePlayers(a, b)
	if c < 0 {
		return 0
	}
	if c == 0 {
		return 0.5
	}
	return 1
}

package interceptor

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/chestnut42/terraforming-mars-manager/internal/client/mars"
	"github.com/chestnut42/terraforming-mars-manager/internal/framework/logx"
	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
)

const backgroundTimeout = time.Minute

func (s *Service) PlayerInputHandler(w http.ResponseWriter, r *http.Request) {
	watcher, err := s.getStateWatcher(r)
	if err != nil {
		logx.Logger(r.Context()).Error("failed to get state watcher",
			slog.String("url", r.URL.String()),
			slog.Any("error", err))
	}
	defer func() {
		if watcher != nil {
			if err := watcher(); err != nil {
				logx.Logger(r.Context()).Error("failed to close watcher",
					slog.String("url", r.URL.String()),
					slog.Any("error", err))
			}
		}
	}()

	s.origin.ServeHTTP(w, r)
}

func (s *Service) getStateWatcher(r *http.Request) (func() error, error) {
	playerId := r.URL.Query().Get("id")
	if playerId == "" {
		return nil, fmt.Errorf("player is empty: %s", r.URL.String())
	}

	initialState, err := s.mars.WaitingFor(r.Context(), mars.WaitingForRequest{PlayerId: playerId})
	if err != nil {
		return nil, fmt.Errorf("failed get initial state %s: %w", playerId, err)
	}

	return func() error {
		// Request context might be cancelled
		ctx, cancel := context.WithTimeout(context.Background(), backgroundTimeout)
		defer cancel()

		newState, err := s.mars.WaitingFor(ctx, mars.WaitingForRequest{PlayerId: playerId})
		if err != nil {
			return fmt.Errorf("failed get new state %s: %w", playerId, err)
		}

		updatedColors := symmetricDifference(initialState.Colors, newState.Colors)
		if len(updatedColors) == 0 {
			return nil
		}

		game, err := s.storage.GetGameByPlayerId(ctx, playerId)
		if err != nil {
			return fmt.Errorf("failed get game %s: %w", playerId, err)
		}

		for _, pp := range game.Players {
			if _, ok := updatedColors[pp.Color]; ok {
				if err := s.notifier.NotifyUser(ctx, pp.UserId); err != nil {
					return fmt.Errorf("failed to notify user %s: %w", pp.UserId, err)
				}
			}
		}
		return nil
	}, nil
}

func symmetricDifference(s1 []storage.Color, s2 []storage.Color) map[storage.Color]struct{} {
	diff := make(map[storage.Color]struct{})

	// Add all elements from the first set
	for _, c1 := range s1 {
		diff[c1] = struct{}{}
	}

	// If element is present in the second set - remove it, otherwise add
	for _, c2 := range s2 {
		if _, ok := diff[c2]; ok {
			delete(diff, c2)
		} else {
			diff[c2] = struct{}{}
		}
	}
	return diff
}

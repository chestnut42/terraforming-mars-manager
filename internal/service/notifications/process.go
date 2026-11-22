package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/chestnut42/terraforming-mars-manager/internal/framework/logx"
	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
)

func (s *Service) processUser(ctx context.Context, userId string) error {
	user, err := s.deps.Storage.GetUserById(ctx, userId)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if len(user.DeviceToken) == 0 {
		logx.Logger(ctx).Debug("user has no device token", slog.String("uid", userId))
		return nil
	}

	games, err := s.deps.Game.GetUserGames(ctx, userId)
	if err != nil {
		return fmt.Errorf("get games: %w", err)
	}

	activeCount := 0
	for _, g := range games {
		if g.AwaitsInput {
			activeCount++
		}
	}

	if err := s.deps.Storage.UpdateSentNotification(ctx, userId,
		func(ctx context.Context, state storage.UserNotificationState) (storage.UserNotificationState, error) {
			if len(state.DeviceToken) == 0 {
				logx.Logger(ctx).Debug("user locked with no device token", slog.String("uid", userId))
				return state, nil
			}

			now := s.now()
			reminder := false
			if activeCount == state.SentNotification.ActiveGames {
				if state.SentNotification.UpdatedAt.IsZero() {
					// State saved before reminders existed: start the clock now.
					state.SentNotification.UpdatedAt = now
					return state, nil
				}
				if !s.reminderDue(state.SentNotification, now) {
					return state, nil
				}
				reminder = true
			}

			// Only count changes and reminders are logged, unchanged scans stay quiet.
			ctx = logx.AddArgs(ctx,
				slog.String("uid", userId),
				slog.String("token_type", string(state.DeviceTokenType)),
				slog.Int("prev_active", state.SentNotification.ActiveGames),
				slog.Int("active", activeCount),
				slog.Bool("reminder", reminder))

			alert := reminder || activeCount > state.SentNotification.ActiveGames
			next := storage.SentNotification{ActiveGames: activeCount, UpdatedAt: now}

			return s.getNotifier(state.DeviceTokenType).notify(ctx, state, next, alert)
		}); err != nil {
		return fmt.Errorf("update sent notification: %w", err)
	}
	return nil
}

// reminderDue reports whether the awaiting games count has stayed the same
// for a full reminder interval, so the alert should be sent again.
func (s *Service) reminderDue(sent storage.SentNotification, now time.Time) bool {
	return s.cfg.ReminderInterval > 0 &&
		sent.ActiveGames > 0 &&
		now.Sub(sent.UpdatedAt) >= s.cfg.ReminderInterval
}

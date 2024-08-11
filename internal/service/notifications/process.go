package notifications

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/chestnut42/terraforming-mars-manager/internal/client/apn"
	"github.com/chestnut42/terraforming-mars-manager/internal/framework/logx"
	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
)

func (s *Service) processUser(ctx context.Context, userId string) error {
	user, err := s.storage.GetUserById(ctx, userId)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if len(user.DeviceToken) == 0 {
		logx.Logger(ctx).Debug("user has no device token", slog.String("uid", userId))
		return nil
	}

	games, err := s.game.GetUserGames(ctx, userId)
	if err != nil {
		return fmt.Errorf("get games: %w", err)
	}

	activeCount := 0
	for _, g := range games {
		if g.AwaitsInput {
			activeCount++
		}
	}

	if err := s.storage.UpdateSentNotification(ctx, userId,
		func(ctx context.Context, sn storage.SentNotification) (storage.SentNotification, error) {
			if activeCount == sn.ActiveGames {
				return sn, nil
			}

			if activeCount > sn.ActiveGames {
				gameText := "game"
				if activeCount > 1 {
					gameText = "games"
				}
				if err := s.notifier.SendNotification(ctx, user.DeviceToken, apn.Notification{
					Alert: apn.Alert{
						Title:    "Mars awaits you!",
						Subtitle: "",
						Body:     fmt.Sprintf("%d %s are awaiting for your decision", activeCount, gameText),
					},
					Badge: activeCount,
					Sound: "default",
				}); err != nil {
					return storage.SentNotification{}, fmt.Errorf("send notification up: %w", err)
				}
			}
			if activeCount < sn.ActiveGames {
				if err := s.notifier.SendNotification(ctx, user.DeviceToken, apn.Notification{
					Badge: activeCount,
				}); err != nil {
					return storage.SentNotification{}, fmt.Errorf("send notification down: %w", err)
				}
			}
			return storage.SentNotification{ActiveGames: activeCount}, nil
		}); err != nil {
		return fmt.Errorf("update sent notification: %w", err)
	}
	return nil
}

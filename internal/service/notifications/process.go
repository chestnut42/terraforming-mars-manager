package notifications

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/chestnut42/terraforming-mars-manager/internal/client/apn"
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

			if activeCount == state.SentNotification.ActiveGames {
				return state, nil
			}

			// If active count decreased, just change the badge.
			notification := apn.Notification{
				Badge: activeCount,
			}
			if activeCount > state.SentNotification.ActiveGames {
				gameText := "game is"
				if activeCount > 1 {
					gameText = "games are"
				}
				notification = apn.Notification{
					Alert: apn.Alert{
						Title:    "Mars awaits you!",
						Subtitle: "",
						Body:     fmt.Sprintf("%d %s awaiting for your decision", activeCount, gameText),
					},
					Badge: activeCount,
					Sound: "default",
				}
			}

			notifier := s.getNotifier(state.DeviceTokenType)
			if err := notifier.SendNotification(ctx, state.DeviceToken, notification); err != nil {
				if errors.Is(err, apn.ErrBadDeviceToken) {
					state.DeviceTokenType = s.nextTokenType(state.DeviceTokenType)
					return state, nil
				}
				return storage.UserNotificationState{}, fmt.Errorf("send notification: %w", err)
			}

			state.SentNotification = storage.SentNotification{ActiveGames: activeCount}
			return state, nil
		}); err != nil {
		return fmt.Errorf("update sent notification: %w", err)
	}
	return nil
}

func (s *Service) getNotifier(t storage.DeviceTokenType) Notifier {
	if t == storage.DeviceTokenTypeSandbox {
		return s.deps.SandboxNotifier
	}
	return s.deps.ProdNotifier
}

func (s *Service) nextTokenType(t storage.DeviceTokenType) storage.DeviceTokenType {
	if t == storage.DeviceTokenTypeSandbox {
		return storage.DeviceTokenTypeProduction
	}
	return storage.DeviceTokenTypeSandbox
}

package notifications

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/chestnut42/terraforming-mars-manager/internal/client/apn"
	"github.com/chestnut42/terraforming-mars-manager/internal/client/fcm"
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
			sent := storage.SentNotification{ActiveGames: activeCount, UpdatedAt: now}

			if state.DeviceTokenType == storage.DeviceTokenTypeFCM {
				return s.notifyFCM(ctx, state, sent, alert)
			}

			// If active count decreased, just change the badge.
			kind := "badge"
			notification := apn.Notification{
				Badge: activeCount,
			}
			if alert {
				kind = "alert"
				notification = apn.Notification{
					Alert: apn.Alert{
						Title:    notificationTitle,
						Subtitle: "",
						Body:     notificationBody(activeCount),
					},
					Badge: activeCount,
					Sound: "default",
				}
			}

			notifier := s.getNotifier(state.DeviceTokenType)
			if err := notifier.SendNotification(ctx, state.DeviceToken, notification); err != nil {
				if errors.Is(err, apn.ErrBadDeviceToken) {
					next := s.nextTokenType(state.DeviceTokenType)
					logx.Logger(ctx).Info("apn bad device token, switching token type",
						slog.String("next_token_type", string(next)))
					state.DeviceTokenType = next
					return state, nil
				}
				return storage.UserNotificationState{}, fmt.Errorf("send notification: %w", err)
			}

			logx.Logger(ctx).Info("apn notification sent", slog.String("kind", kind))
			state.SentNotification = sent
			return state, nil
		}); err != nil {
		return fmt.Errorf("update sent notification: %w", err)
	}
	return nil
}

const (
	notificationTitle = "Mars awaits you!"
	// Android: the tag makes a new notification replace the previous one,
	// and the channel must match the one created by the app.
	fcmNotificationTag = "awaiting-games"
	fcmChannelId       = "turns"
)

func notificationBody(activeCount int) string {
	gameText := "game is"
	if activeCount > 1 {
		gameText = "games are"
	}
	return fmt.Sprintf("%d %s awaiting for your decision", activeCount, gameText)
}

// reminderDue reports whether the awaiting games count has stayed the same
// for a full reminder interval, so the alert should be sent again.
func (s *Service) reminderDue(sent storage.SentNotification, now time.Time) bool {
	return s.cfg.ReminderInterval > 0 &&
		sent.ActiveGames > 0 &&
		now.Sub(sent.UpdatedAt) >= s.cfg.ReminderInterval
}

// notifyFCM handles Android devices. Android has no badge counter, so a
// notification is only sent for an alert: a grown awaiting games count or a reminder.
// The context logger is expected to carry the user and count attributes.
func (s *Service) notifyFCM(
	ctx context.Context,
	state storage.UserNotificationState,
	sent storage.SentNotification,
	alert bool,
) (storage.UserNotificationState, error) {
	if s.deps.FCMNotifier == nil {
		logx.Logger(ctx).Debug("fcm is not configured")
		return state, nil
	}

	if !alert {
		logx.Logger(ctx).Info("fcm notification skipped: active count decreased")
	} else {
		err := s.deps.FCMNotifier.SendNotification(ctx, string(state.DeviceToken), fcm.Notification{
			Title:     notificationTitle,
			Body:      notificationBody(sent.ActiveGames),
			Tag:       fcmNotificationTag,
			ChannelId: fcmChannelId,
		})
		if err != nil {
			if errors.Is(err, fcm.ErrUnregistered) {
				logx.Logger(ctx).Info("fcm token unregistered")
				state.DeviceToken = nil
				return state, nil
			}
			return storage.UserNotificationState{}, fmt.Errorf("send fcm notification: %w", err)
		}
		logx.Logger(ctx).Info("fcm notification sent")
	}

	state.SentNotification = sent
	return state, nil
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

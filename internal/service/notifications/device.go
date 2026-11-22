package notifications

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/chestnut42/terraforming-mars-manager/internal/client/apn"
	"github.com/chestnut42/terraforming-mars-manager/internal/client/fcm"
	"github.com/chestnut42/terraforming-mars-manager/internal/framework/logx"
	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
)

const (
	notificationTitle = "Mars awaits you!"
	// Android: the tag makes a new notification replace the previous one,
	// and the channel must match the one created by the app.
	fcmNotificationTag = "awaiting-games"
	fcmChannelId       = "turns"
)

// deviceNotifier delivers the awaiting games count to one kind of device token.
// alert asks for a visible notification; otherwise the device may only be
// updated silently, or skipped. It returns the state to store: SentNotification
// is set to next once the device is up to date, and the token is adjusted when
// the push service rejects it.
// The context logger is expected to carry the user and count attributes.
type deviceNotifier interface {
	notify(
		ctx context.Context,
		state storage.UserNotificationState,
		next storage.SentNotification,
		alert bool,
	) (storage.UserNotificationState, error)
}

func (s *Service) getNotifier(t storage.DeviceTokenType) deviceNotifier {
	switch t {
	case storage.DeviceTokenTypeFCM:
		if s.deps.FCMNotifier == nil {
			return unconfiguredNotifier{}
		}
		return fcmNotifier{client: s.deps.FCMNotifier}
	case storage.DeviceTokenTypeSandbox:
		return apnNotifier{client: s.deps.SandboxNotifier, fallback: storage.DeviceTokenTypeProduction}
	default:
		return apnNotifier{client: s.deps.ProdNotifier, fallback: storage.DeviceTokenTypeSandbox}
	}
}

func notificationBody(activeCount int) string {
	gameText := "game is"
	if activeCount > 1 {
		gameText = "games are"
	}
	return fmt.Sprintf("%d %s awaiting for your decision", activeCount, gameText)
}

// apnNotifier handles iOS devices. Without an alert it still updates the badge.
type apnNotifier struct {
	client Notifier
	// fallback is the token type to try when APNs rejects the token:
	// the app may have been built for the other environment.
	fallback storage.DeviceTokenType
}

func (n apnNotifier) notify(
	ctx context.Context,
	state storage.UserNotificationState,
	next storage.SentNotification,
	alert bool,
) (storage.UserNotificationState, error) {
	kind := "badge"
	notification := apn.Notification{
		Badge: next.ActiveGames,
	}
	if alert {
		kind = "alert"
		notification = apn.Notification{
			Alert: apn.Alert{
				Title: notificationTitle,
				Body:  notificationBody(next.ActiveGames),
			},
			Badge: next.ActiveGames,
			Sound: "default",
		}
	}

	if err := n.client.SendNotification(ctx, state.DeviceToken, notification); err != nil {
		if errors.Is(err, apn.ErrBadDeviceToken) {
			logx.Logger(ctx).Info("apn bad device token, switching token type",
				slog.String("next_token_type", string(n.fallback)))
			state.DeviceTokenType = n.fallback
			return state, nil
		}
		return storage.UserNotificationState{}, fmt.Errorf("send apn notification: %w", err)
	}

	logx.Logger(ctx).Info("apn notification sent", slog.String("kind", kind))
	state.SentNotification = next
	return state, nil
}

// fcmNotifier handles Android devices. Android has no badge counter,
// so nothing is sent without an alert.
type fcmNotifier struct {
	client FCMNotifier
}

func (n fcmNotifier) notify(
	ctx context.Context,
	state storage.UserNotificationState,
	next storage.SentNotification,
	alert bool,
) (storage.UserNotificationState, error) {
	if !alert {
		logx.Logger(ctx).Info("fcm notification skipped: active count decreased")
		state.SentNotification = next
		return state, nil
	}

	err := n.client.SendNotification(ctx, string(state.DeviceToken), fcm.Notification{
		Title:     notificationTitle,
		Body:      notificationBody(next.ActiveGames),
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
	state.SentNotification = next
	return state, nil
}

// unconfiguredNotifier leaves the state untouched, so the notification is
// delivered once the push service is configured.
type unconfiguredNotifier struct{}

func (unconfiguredNotifier) notify(
	ctx context.Context,
	state storage.UserNotificationState,
	_ storage.SentNotification,
	_ bool,
) (storage.UserNotificationState, error) {
	logx.Logger(ctx).Debug("push service is not configured")
	return state, nil
}

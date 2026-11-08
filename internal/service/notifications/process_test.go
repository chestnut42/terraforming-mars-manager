package notifications

import (
	"context"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/chestnut42/terraforming-mars-manager/internal/client/apn"
	"github.com/chestnut42/terraforming-mars-manager/internal/client/fcm"
	"github.com/chestnut42/terraforming-mars-manager/internal/service/game"
	"github.com/chestnut42/terraforming-mars-manager/internal/storage"
)

type fakeStorage struct {
	state storage.UserNotificationState
}

func (f *fakeStorage) GetActiveUsers(context.Context, time.Duration) ([]string, error) {
	return nil, nil
}

func (f *fakeStorage) GetUserById(_ context.Context, userId string) (*storage.User, error) {
	return &storage.User{UserId: userId, DeviceToken: f.state.DeviceToken, DeviceTokenType: f.state.DeviceTokenType}, nil
}

func (f *fakeStorage) UpdateSentNotification(ctx context.Context, _ string, updater storage.SentNotificationUpdater) error {
	newState, err := updater(ctx, f.state)
	if err != nil {
		return err
	}
	f.state = newState
	return nil
}

type fakeGames struct {
	awaiting int
}

func (f *fakeGames) GetUserGames(context.Context, string) ([]*game.UserGame, error) {
	games := make([]*game.UserGame, f.awaiting)
	for i := range games {
		games[i] = &game.UserGame{AwaitsInput: true}
	}
	return games, nil
}

type fakeFCM struct {
	sent []fcm.Notification
	err  error
}

func (f *fakeFCM) SendNotification(_ context.Context, _ string, n fcm.Notification) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, n)
	return nil
}

type fakeAPN struct {
	sent int
}

func (f *fakeAPN) SendNotification(context.Context, []byte, apn.Notification) error {
	f.sent++
	return nil
}

func newFCMTestService(st *fakeStorage, games *fakeGames, fcmNotifier *fakeFCM, apnNotifier *fakeAPN) *Service {
	return NewService(Config{}, Dependencies{
		Storage:         st,
		Game:            games,
		SandboxNotifier: apnNotifier,
		ProdNotifier:    apnNotifier,
		FCMNotifier:     fcmNotifier,
	})
}

func TestProcessUserFCM(t *testing.T) {
	ctx := context.Background()
	st := &fakeStorage{state: storage.UserNotificationState{
		DeviceToken:     []byte("fcm-token"),
		DeviceTokenType: storage.DeviceTokenTypeFCM,
	}}
	games := &fakeGames{awaiting: 2}
	fcmNotifier := &fakeFCM{}
	apnNotifier := &fakeAPN{}
	s := newFCMTestService(st, games, fcmNotifier, apnNotifier)

	// More awaiting games: notify.
	assert.NilError(t, s.processUser(ctx, "u"))
	assert.Equal(t, len(fcmNotifier.sent), 1)
	assert.Equal(t, fcmNotifier.sent[0].Body, "2 games are awaiting for your decision")
	assert.Equal(t, fcmNotifier.sent[0].Tag, fcmNotificationTag)
	assert.Equal(t, fcmNotifier.sent[0].ChannelId, fcmChannelId)
	assert.Equal(t, st.state.SentNotification.ActiveGames, 2)

	// Fewer awaiting games: no notification, but the count is remembered.
	games.awaiting = 1
	assert.NilError(t, s.processUser(ctx, "u"))
	assert.Equal(t, len(fcmNotifier.sent), 1)
	assert.Equal(t, st.state.SentNotification.ActiveGames, 1)

	assert.Equal(t, apnNotifier.sent, 0)
	assert.Equal(t, st.state.DeviceTokenType, storage.DeviceTokenTypeFCM)
}

func TestProcessUserFCMUnregistered(t *testing.T) {
	st := &fakeStorage{state: storage.UserNotificationState{
		DeviceToken:     []byte("fcm-token"),
		DeviceTokenType: storage.DeviceTokenTypeFCM,
	}}
	s := newFCMTestService(st, &fakeGames{awaiting: 1}, &fakeFCM{err: fcm.ErrUnregistered}, &fakeAPN{})

	assert.NilError(t, s.processUser(context.Background(), "u"))
	assert.Equal(t, len(st.state.DeviceToken), 0)
	assert.Equal(t, st.state.DeviceTokenType, storage.DeviceTokenTypeFCM)
}

func TestProcessUserFCMDisabled(t *testing.T) {
	st := &fakeStorage{state: storage.UserNotificationState{
		DeviceToken:     []byte("fcm-token"),
		DeviceTokenType: storage.DeviceTokenTypeFCM,
	}}
	apnNotifier := &fakeAPN{}
	s := NewService(Config{}, Dependencies{
		Storage:         st,
		Game:            &fakeGames{awaiting: 1},
		SandboxNotifier: apnNotifier,
		ProdNotifier:    apnNotifier,
	})

	assert.NilError(t, s.processUser(context.Background(), "u"))
	assert.Equal(t, apnNotifier.sent, 0)
	// Not marked as sent, so it is delivered once FCM is configured.
	assert.Equal(t, st.state.SentNotification.ActiveGames, 0)
}

func TestProcessUserAPNUnchanged(t *testing.T) {
	st := &fakeStorage{state: storage.UserNotificationState{
		DeviceToken:     []byte{1, 2, 3},
		DeviceTokenType: storage.DeviceTokenTypeProduction,
	}}
	apnNotifier := &fakeAPN{}
	fcmNotifier := &fakeFCM{}
	s := newFCMTestService(st, &fakeGames{awaiting: 1}, fcmNotifier, apnNotifier)

	assert.NilError(t, s.processUser(context.Background(), "u"))
	assert.Equal(t, apnNotifier.sent, 1)
	assert.Equal(t, len(fcmNotifier.sent), 0)
	assert.Equal(t, st.state.SentNotification.ActiveGames, 1)
}

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

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time { return c.now }

func newReminderTestService(st *fakeStorage, games *fakeGames, fcmNotifier *fakeFCM, apnNotifier *fakeAPN) (*Service, *fakeClock) {
	clock := &fakeClock{now: time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)}
	s := NewService(Config{ReminderInterval: time.Hour}, Dependencies{
		Storage:         st,
		Game:            games,
		SandboxNotifier: apnNotifier,
		ProdNotifier:    apnNotifier,
		FCMNotifier:     fcmNotifier,
	})
	s.now = clock.Now
	return s, clock
}

func TestProcessUserReminderAPN(t *testing.T) {
	ctx := context.Background()
	st := &fakeStorage{state: storage.UserNotificationState{
		DeviceToken:     []byte{1, 2, 3},
		DeviceTokenType: storage.DeviceTokenTypeProduction,
	}}
	games := &fakeGames{awaiting: 1}
	apnNotifier := &fakeAPN{}
	s, clock := newReminderTestService(st, games, &fakeFCM{}, apnNotifier)

	assert.NilError(t, s.processUser(ctx, "u"))
	assert.Equal(t, apnNotifier.sent, 1)
	assert.Equal(t, st.state.SentNotification.UpdatedAt, clock.now)

	// Still stuck, but less than an hour passed.
	clock.now = clock.now.Add(59 * time.Minute)
	assert.NilError(t, s.processUser(ctx, "u"))
	assert.Equal(t, apnNotifier.sent, 1)

	// An hour without progress: remind.
	clock.now = clock.now.Add(time.Minute)
	assert.NilError(t, s.processUser(ctx, "u"))
	assert.Equal(t, apnNotifier.sent, 2)
	assert.Equal(t, st.state.SentNotification.UpdatedAt, clock.now)

	// And again an hour later.
	clock.now = clock.now.Add(time.Hour)
	assert.NilError(t, s.processUser(ctx, "u"))
	assert.Equal(t, apnNotifier.sent, 3)

	// Progress made: badge update resets the clock.
	clock.now = clock.now.Add(30 * time.Minute)
	games.awaiting = 0
	assert.NilError(t, s.processUser(ctx, "u"))
	assert.Equal(t, apnNotifier.sent, 4)

	// Nothing awaiting: no reminders.
	clock.now = clock.now.Add(2 * time.Hour)
	assert.NilError(t, s.processUser(ctx, "u"))
	assert.Equal(t, apnNotifier.sent, 4)
}

func TestProcessUserReminderFCM(t *testing.T) {
	ctx := context.Background()
	st := &fakeStorage{state: storage.UserNotificationState{
		DeviceToken:     []byte("fcm-token"),
		DeviceTokenType: storage.DeviceTokenTypeFCM,
	}}
	games := &fakeGames{awaiting: 2}
	fcmNotifier := &fakeFCM{}
	s, clock := newReminderTestService(st, games, fcmNotifier, &fakeAPN{})

	assert.NilError(t, s.processUser(ctx, "u"))
	assert.Equal(t, len(fcmNotifier.sent), 1)

	// One game played, one still waiting: no alert, clock reset.
	clock.now = clock.now.Add(50 * time.Minute)
	games.awaiting = 1
	assert.NilError(t, s.processUser(ctx, "u"))
	assert.Equal(t, len(fcmNotifier.sent), 1)

	// An hour since the first alert, but not since the progress.
	clock.now = clock.now.Add(10 * time.Minute)
	assert.NilError(t, s.processUser(ctx, "u"))
	assert.Equal(t, len(fcmNotifier.sent), 1)

	// An hour since the progress: remind.
	clock.now = clock.now.Add(50 * time.Minute)
	assert.NilError(t, s.processUser(ctx, "u"))
	assert.Equal(t, len(fcmNotifier.sent), 2)
	assert.Equal(t, fcmNotifier.sent[1].Body, "1 game is awaiting for your decision")
}

func TestProcessUserReminderLegacyState(t *testing.T) {
	ctx := context.Background()
	// Saved before reminders existed: count set, no timestamp.
	st := &fakeStorage{state: storage.UserNotificationState{
		DeviceToken:      []byte{1, 2, 3},
		DeviceTokenType:  storage.DeviceTokenTypeProduction,
		SentNotification: storage.SentNotification{ActiveGames: 1},
	}}
	apnNotifier := &fakeAPN{}
	s, clock := newReminderTestService(st, &fakeGames{awaiting: 1}, &fakeFCM{}, apnNotifier)

	// No immediate reminder, the clock starts now.
	assert.NilError(t, s.processUser(ctx, "u"))
	assert.Equal(t, apnNotifier.sent, 0)
	assert.Equal(t, st.state.SentNotification.UpdatedAt, clock.now)

	clock.now = clock.now.Add(time.Hour)
	assert.NilError(t, s.processUser(ctx, "u"))
	assert.Equal(t, apnNotifier.sent, 1)
}

func TestProcessUserReminderDisabled(t *testing.T) {
	ctx := context.Background()
	st := &fakeStorage{state: storage.UserNotificationState{
		DeviceToken:     []byte{1, 2, 3},
		DeviceTokenType: storage.DeviceTokenTypeProduction,
	}}
	apnNotifier := &fakeAPN{}
	s, clock := newReminderTestService(st, &fakeGames{awaiting: 1}, &fakeFCM{}, apnNotifier)
	s.cfg.ReminderInterval = 0

	assert.NilError(t, s.processUser(ctx, "u"))
	clock.now = clock.now.Add(5 * time.Hour)
	assert.NilError(t, s.processUser(ctx, "u"))
	assert.Equal(t, apnNotifier.sent, 1)
}

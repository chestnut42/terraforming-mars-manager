package storage

import (
	"context"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/chestnut42/terraforming-mars-manager/internal/database"
)

func TestStorage_Users(t *testing.T) {
	db, err := database.PrepareDB("postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	assert.NilError(t, err)
	defer db.Close()

	storage, err := New(db)
	assert.NilError(t, err)

	ctx := context.Background()

	t.Run("Create/Get users", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		storage.nowFunc = func() time.Time { return now }
		t.Run("UpsertUser - success", func(t *testing.T) {
			err := storage.UpsertUser(ctx, &User{
				UserId:   "test user id",
				Nickname: "test user nickname",
				Color:    ColorBronze,
			})
			assert.NilError(t, err)
		})

		t.Run("GetUserById - success", func(t *testing.T) {
			user, err := storage.GetUserById(ctx, "test user id")
			assert.NilError(t, err)
			assert.DeepEqual(t, user, &User{
				UserId:    "test user id",
				Nickname:  "test user nickname",
				Color:     ColorBronze,
				CreatedAt: now,
			})
		})

		now2 := now.Add(time.Second)
		storage.nowFunc = func() time.Time { return now2 }
		t.Run("UpsertUser - existing user", func(t *testing.T) {
			err := storage.UpsertUser(ctx, &User{
				UserId:   "test user id",
				Nickname: "test user nickname 2",
				Color:    ColorOrange,
			})
			assert.NilError(t, err)

			user, err := storage.GetUserById(ctx, "test user id")
			assert.NilError(t, err)
			assert.DeepEqual(t, user, &User{
				UserId:    "test user id",
				Nickname:  "test user nickname",
				Color:     ColorBronze,
				CreatedAt: now,
			})
		})

		t.Run("GetUserById - not found", func(t *testing.T) {
			_, err := storage.GetUserById(ctx, "test user not found")
			assert.ErrorIs(t, err, ErrNotFound)
		})

		t.Run("UpdateUser - success", func(t *testing.T) {
			expected := &User{
				UserId:    "test user id",
				Nickname:  "new test nickname",
				Color:     ColorGreen,
				CreatedAt: now,
			}
			updated, err := storage.UpdateUser(ctx, &User{
				UserId:   "test user id",
				Nickname: "new test nickname",
				Color:    ColorGreen,
			})
			assert.NilError(t, err)
			assert.DeepEqual(t, updated, expected)

			got, err := storage.GetUserById(ctx, "test user id")
			assert.NilError(t, err)
			assert.DeepEqual(t, got, expected)
		})

		t.Run("UpdateUser - not found", func(t *testing.T) {
			_, err := storage.UpdateUser(ctx, &User{
				UserId:   "not existing test user id",
				Nickname: "new test nickname",
			})
			assert.ErrorIs(t, err, ErrNotFound)
		})

		t.Run("UpdateUser - already exists", func(t *testing.T) {
			err := storage.UpsertUser(ctx, &User{
				UserId:   "second test user id",
				Nickname: "second test user nickname",
			})
			assert.NilError(t, err)

			err = storage.UpsertUser(ctx, &User{
				UserId:   "third test user id",
				Nickname: "third test user nickname",
			})
			assert.NilError(t, err)

			_, err = storage.UpdateUser(ctx, &User{
				UserId:   "second test user id",
				Nickname: "third test user nickname",
			})
			assert.ErrorIs(t, err, ErrAlreadyExists)
		})

		t.Run("GetUsersByNicknames - success", func(t *testing.T) {
			got, err := storage.GetUserByNickname(ctx, "second test user nickname")
			assert.NilError(t, err)
			assert.DeepEqual(t, got, &User{
				UserId:    "second test user id",
				Nickname:  "second test user nickname",
				CreatedAt: now2,
			})
		})

		t.Run("GetUsersByNicknames - not found", func(t *testing.T) {
			_, err := storage.GetUserByNickname(ctx, "not existing user")
			assert.ErrorIs(t, err, ErrNotFound)
		})

		t.Run("UpdateDeviceToken - success", func(t *testing.T) {
			err := storage.UpsertUser(ctx, &User{
				UserId:   "device token user",
				Nickname: "device token user nickname",
			})
			assert.NilError(t, err)

			err = storage.UpdateDeviceToken(ctx, "device token user", []byte("device token"))
			assert.NilError(t, err)

			got, err := storage.GetUserById(ctx, "device token user")
			assert.NilError(t, err)
			assert.DeepEqual(t, got.DeviceToken, []byte("device token"))
		})
	})

	t.Run("SearchUsers", func(t *testing.T) {
		searchNow := time.Now().Truncate(time.Second)
		storage.nowFunc = func() time.Time { return searchNow }
		for _, u := range []*User{
			{UserId: "search 1", Nickname: "prefix middle nickname suffix"},
			{UserId: "search 2", Nickname: "prefix middlenick surname"},
			{UserId: "search 3", Nickname: "prefix nsuffix"},
		} {
			err := storage.UpsertUser(ctx, u)
			assert.NilError(t, err)
		}

		t.Run("success - exact", func(t *testing.T) {
			got, err := storage.SearchUsers(ctx, "prefix middle nickname suffix", 5, "")
			assert.NilError(t, err)
			assert.DeepEqual(t, got, []*User{{UserId: "search 1", Nickname: "prefix middle nickname suffix", CreatedAt: searchNow}})
		})
		t.Run("success - prefix", func(t *testing.T) {
			got, err := storage.SearchUsers(ctx, "prefix ", 5, "")
			assert.NilError(t, err)
			assert.DeepEqual(t, got, []*User{
				{UserId: "search 1", Nickname: "prefix middle nickname suffix", CreatedAt: searchNow},
				{UserId: "search 2", Nickname: "prefix middlenick surname", CreatedAt: searchNow},
				{UserId: "search 3", Nickname: "prefix nsuffix", CreatedAt: searchNow},
			})
		})
		t.Run("success - suffix", func(t *testing.T) {
			got, err := storage.SearchUsers(ctx, "suffix", 5, "")
			assert.NilError(t, err)
			assert.DeepEqual(t, got, []*User{
				{UserId: "search 1", Nickname: "prefix middle nickname suffix", CreatedAt: searchNow},
				{UserId: "search 3", Nickname: "prefix nsuffix", CreatedAt: searchNow},
			})
		})
		t.Run("success - middle", func(t *testing.T) {
			got, err := storage.SearchUsers(ctx, "middle", 5, "")
			assert.NilError(t, err)
			assert.DeepEqual(t, got, []*User{
				{UserId: "search 1", Nickname: "prefix middle nickname suffix", CreatedAt: searchNow},
				{UserId: "search 2", Nickname: "prefix middlenick surname", CreatedAt: searchNow},
			})
		})
		t.Run("success - limit", func(t *testing.T) {
			got, err := storage.SearchUsers(ctx, "prefix ", 2, "")
			assert.NilError(t, err)
			assert.DeepEqual(t, got, []*User{
				{UserId: "search 1", Nickname: "prefix middle nickname suffix", CreatedAt: searchNow},
				{UserId: "search 2", Nickname: "prefix middlenick surname", CreatedAt: searchNow},
			})
		})
		t.Run("success - exclude", func(t *testing.T) {
			got, err := storage.SearchUsers(ctx, "prefix ", 5, "search 2")
			assert.NilError(t, err)
			assert.DeepEqual(t, got, []*User{
				{UserId: "search 1", Nickname: "prefix middle nickname suffix", CreatedAt: searchNow},
				{UserId: "search 3", Nickname: "prefix nsuffix", CreatedAt: searchNow},
			})
		})
		t.Run("error - not found", func(t *testing.T) {
			_, err := storage.SearchUsers(ctx, "some invalid term", 5, "")
			assert.ErrorIs(t, err, ErrNotFound)
		})
	})

	t.Run("CreateGame", func(t *testing.T) {
		gameNow := time.Now().Truncate(time.Second)
		storage.nowFunc = func() time.Time { return gameNow }
		for _, u := range []*User{
			{UserId: "game 1", Nickname: "game player 1"},
			{UserId: "game 2", Nickname: "game player 2"},
			{UserId: "game 3", Nickname: "game player 3"},
		} {
			err := storage.UpsertUser(ctx, u)
			assert.NilError(t, err)
		}

		tests := []struct {
			name    string
			game    Game
			wantErr bool
		}{
			{
				name: "success",
				game: Game{
					GameId:      "g1",
					SpectatorId: "s1",
					ExpiresAt:   gameNow.Add(time.Hour),
					Players: []*Player{
						{UserId: "game 1", PlayerId: "p1", Color: ColorBlue},
						{UserId: "game 2", PlayerId: "p2", Color: ColorRed},
						{UserId: "game 3", PlayerId: "p3", Color: ColorYellow},
					},
				},
			},
			{
				name: "error - non unique game id",
				game: Game{
					GameId:    "g1",
					ExpiresAt: gameNow.Add(time.Hour),
					Players: []*Player{
						{UserId: "game 1", PlayerId: "p4", Color: ColorBlue},
						{UserId: "game 2", PlayerId: "p5", Color: ColorRed},
						{UserId: "game 3", PlayerId: "p6", Color: ColorYellow},
					},
				},
				wantErr: true,
			},
			{
				name: "error - non unique player id",
				game: Game{
					GameId:    "g2",
					ExpiresAt: gameNow.Add(time.Hour),
					Players: []*Player{
						{UserId: "game 1", PlayerId: "p4", Color: ColorBlue},
						{UserId: "game 2", PlayerId: "p2", Color: ColorRed},
						{UserId: "game 3", PlayerId: "p6", Color: ColorYellow},
					},
				},
				wantErr: true,
			},
			{
				name: "error - non existing player",
				game: Game{
					GameId:    "g2",
					ExpiresAt: gameNow.Add(time.Hour),
					Players: []*Player{
						{UserId: "game 1", PlayerId: "p4", Color: ColorBlue},
						{UserId: "game 4", PlayerId: "p5", Color: ColorRed},
						{UserId: "game 3", PlayerId: "p6", Color: ColorYellow},
					},
				},
				wantErr: true,
			},
			{
				name: "error - non unique color",
				game: Game{
					GameId:    "g2",
					ExpiresAt: gameNow.Add(time.Hour),
					Players: []*Player{
						{UserId: "game 1", PlayerId: "p4", Color: ColorBlue},
						{UserId: "game 2", PlayerId: "p5", Color: ColorRed},
						{UserId: "game 3", PlayerId: "p6", Color: ColorBlue},
					},
				},
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := storage.CreateGame(ctx, &tt.game)
				if tt.wantErr {
					assert.Assert(t, err != nil)
				} else {
					assert.NilError(t, err)
				}
			})
		}
	})

	t.Run("GetGamesByUserId", func(t *testing.T) {
		gameNow := time.Now().Truncate(time.Second)
		storage.nowFunc = func() time.Time { return gameNow }
		for _, u := range []*User{
			{UserId: "game_by_user1", Nickname: "game by user 1"},
			{UserId: "game_by_user2", Nickname: "game by user 2"},
			{UserId: "game_by_user3", Nickname: "game by user 3"},
		} {
			err := storage.UpsertUser(ctx, u)
			assert.NilError(t, err)
		}

		for _, g := range []*Game{
			{
				GameId:      "gbu1",
				SpectatorId: "sbu1",
				ExpiresAt:   gameNow.Add(time.Hour),
				Players: []*Player{
					{UserId: "game_by_user1", PlayerId: "p1_1", Color: ColorBlue},
					{UserId: "game_by_user2", PlayerId: "p1_2", Color: ColorRed},
					{UserId: "game_by_user3", PlayerId: "p1_3", Color: ColorYellow},
				},
			},
			{
				GameId:      "gbu2",
				SpectatorId: "sbu2",
				ExpiresAt:   gameNow.Add(time.Hour),
				Players: []*Player{
					{UserId: "game_by_user1", PlayerId: "p2_1", Color: ColorBlue},
					{UserId: "game_by_user3", PlayerId: "p2_3", Color: ColorYellow},
				},
			},
			{ // Expired game
				GameId:      "gbu3",
				SpectatorId: "sbu3",
				ExpiresAt:   gameNow.Add(-time.Hour),
				Players: []*Player{
					{UserId: "game_by_user1", PlayerId: "p3_1", Color: ColorBlue},
					{UserId: "game_by_user3", PlayerId: "p3_3", Color: ColorYellow},
					{UserId: "game_by_user2", PlayerId: "p3_2", Color: ColorRed},
				},
			},
			{
				GameId:      "gbu4",
				SpectatorId: "sbu4",
				ExpiresAt:   gameNow.Add(time.Hour),
				Players: []*Player{
					{UserId: "game_by_user1", PlayerId: "p4_1", Color: ColorBlue},
					{UserId: "game_by_user3", PlayerId: "p4_3", Color: ColorYellow},
					{UserId: "game_by_user2", PlayerId: "p4_2", Color: ColorBronze},
				},
			},
		} {
			err := storage.CreateGame(ctx, g)
			assert.NilError(t, err)
		}

		got, err := storage.GetGamesByUserId(ctx, "game_by_user2")
		assert.NilError(t, err)
		assert.DeepEqual(t, got, []*Game{
			{
				GameId:      "gbu1",
				SpectatorId: "sbu1",
				CreatedAt:   gameNow,
				ExpiresAt:   gameNow.Add(time.Hour),
				Players: []*Player{
					{UserId: "game_by_user2", PlayerId: "p1_2", Color: ColorRed},
				},
			},
			{
				GameId:      "gbu4",
				SpectatorId: "sbu4",
				CreatedAt:   gameNow,
				ExpiresAt:   gameNow.Add(time.Hour),
				Players: []*Player{
					{UserId: "game_by_user2", PlayerId: "p4_2", Color: ColorBronze},
				},
			},
		})
	})

	t.Run("GetActiveUsers", func(t *testing.T) {
		gameNow := time.Now().Truncate(time.Second).Add(2 * time.Hour) // 2 Hours added to avoid querying other games
		storage.nowFunc = func() time.Time { return gameNow }
		for _, u := range []*User{
			{UserId: "active_user1", Nickname: "active user 1"},
			{UserId: "active_user2", Nickname: "active user 2"},
			{UserId: "active_user3", Nickname: "active user 3"},
			{UserId: "active_user4", Nickname: "active user 4"},
		} {
			err := storage.UpsertUser(ctx, u)
			assert.NilError(t, err)
		}

		for _, g := range []*Game{
			{
				GameId:      "au1",
				SpectatorId: "sau1",
				ExpiresAt:   gameNow.Add(time.Hour),
				Players: []*Player{
					{UserId: "active_user1", PlayerId: "aup1_1", Color: ColorBlue},
					{UserId: "active_user2", PlayerId: "aup1_2", Color: ColorRed},
				},
			},
			{ // Recently expired game
				GameId:      "au2",
				SpectatorId: "sau2",
				ExpiresAt:   gameNow.Add(-time.Minute),
				Players: []*Player{
					{UserId: "active_user2", PlayerId: "aup2_2", Color: ColorBlue},
					{UserId: "active_user3", PlayerId: "aup2_3", Color: ColorYellow},
				},
			},
			{ // Expired game
				GameId:      "au3",
				SpectatorId: "sau3",
				ExpiresAt:   gameNow.Add(-time.Hour),
				Players: []*Player{
					{UserId: "active_user3", PlayerId: "aup3_3", Color: ColorBlue},
					{UserId: "active_user4", PlayerId: "aup3_4", Color: ColorYellow},
				},
			},
		} {
			err := storage.CreateGame(ctx, g)
			assert.NilError(t, err)
		}

		got, err := storage.GetActiveUsers(ctx, 2*time.Minute)
		assert.NilError(t, err)
		assert.DeepEqual(t, got, []string{
			"active_user1", "active_user2", "active_user3",
		})
	})

	t.Run("UpdateSentNotification", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		storage.nowFunc = func() time.Time { return now }
		for _, u := range []*User{
			{UserId: "notification_user1", Nickname: "notification user 1"},
		} {
			err := storage.UpsertUser(ctx, u)
			assert.NilError(t, err)
		}

		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		wait := make(chan struct{})
		backgroundErr := make(chan error)
		go func() {
			select {
			case <-wait:
			case <-ctx.Done():
				backgroundErr <- ctx.Err()
				return
			}
			err := storage.UpdateSentNotification(ctx, "notification_user1",
				func(ctx context.Context, sn SentNotification) (SentNotification, error) {
					return SentNotification{ActiveGames: 3}, nil
				})
			backgroundErr <- err
		}()

		err := storage.UpdateSentNotification(ctx, "notification_user1",
			func(ctx context.Context, sn SentNotification) (SentNotification, error) {
				wait <- struct{}{}
				return SentNotification{ActiveGames: 2}, nil
			})
		assert.NilError(t, err)
		err = <-backgroundErr
		assert.NilError(t, err)

		err = storage.UpdateSentNotification(ctx, "notification_user1",
			func(ctx context.Context, sn SentNotification) (SentNotification, error) {
				assert.Equal(t, sn.ActiveGames, 3)
				return SentNotification{ActiveGames: 2}, nil
			})
		assert.NilError(t, err)
	})
}

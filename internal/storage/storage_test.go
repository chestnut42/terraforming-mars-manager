package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"

	"github.com/chestnut42/terraforming-mars-manager/internal/database"
)

const (
	defaultDatabase = "postgres"
)

func TestMigrations(t *testing.T) {
	t.Parallel()

	db, err := database.PrepareDB(getDSN(defaultDatabase))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = New(db)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStorage_GetGamesByUserId(t *testing.T) {
	t.Parallel()

	storage := prepareStorage(t)
	ctx := context.Background()

	gameNow := time.Now().Truncate(time.Second)
	storage.nowFunc = func() time.Time { return gameNow }
	for _, u := range []UpsertUser{
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
			Players: []Player{
				{UserId: "game_by_user1", PlayerId: "p1_1", Color: ColorBlue},
				{UserId: "game_by_user2", PlayerId: "p1_2", Color: ColorRed},
				{UserId: "game_by_user3", PlayerId: "p1_3", Color: ColorYellow},
			},
		},
		{
			GameId:      "gbu2",
			SpectatorId: "sbu2",
			ExpiresAt:   gameNow.Add(time.Hour),
			Players: []Player{
				{UserId: "game_by_user1", PlayerId: "p2_1", Color: ColorBlue},
				{UserId: "game_by_user3", PlayerId: "p2_3", Color: ColorYellow},
			},
		},
		{ // Expired game
			GameId:      "gbu3",
			SpectatorId: "sbu3",
			ExpiresAt:   gameNow.Add(-time.Hour),
			Players: []Player{
				{UserId: "game_by_user1", PlayerId: "p3_1", Color: ColorBlue},
				{UserId: "game_by_user3", PlayerId: "p3_3", Color: ColorYellow},
				{UserId: "game_by_user2", PlayerId: "p3_2", Color: ColorRed},
			},
		},
		{
			GameId:      "gbu4",
			SpectatorId: "sbu4",
			ExpiresAt:   gameNow.Add(time.Hour),
			Players: []Player{
				{UserId: "game_by_user1", PlayerId: "p4_1", Color: ColorBlue},
				{UserId: "game_by_user3", PlayerId: "p4_3", Color: ColorYellow},
				{UserId: "game_by_user2", PlayerId: "p4_2", Color: ColorBronze},
			},
		},
		{ // Finished game
			GameId:      "gbu5",
			SpectatorId: "sbu5",
			ExpiresAt:   gameNow.Add(time.Hour),
			Players: []Player{
				{UserId: "game_by_user1", PlayerId: "p5_1", Color: ColorBlue},
				{UserId: "game_by_user3", PlayerId: "p5_3", Color: ColorYellow},
				{UserId: "game_by_user2", PlayerId: "p5_2", Color: ColorBronze},
			},
		},
	} {
		err := storage.CreateGame(ctx, g)
		assert.NilError(t, err)
	}

	finishTime := gameNow.Add(-time.Hour)
	storage.nowFunc = func() time.Time { return finishTime }
	err := storage.UpdateGameResults(ctx, "gbu5", nil)
	if err != nil {
		t.Fatal(err)
	}
	storage.nowFunc = func() time.Time { return gameNow }

	t.Run("filter out finished game", func(t *testing.T) {
		got, err := storage.GetGamesByUserId(ctx, "game_by_user2", time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(got, []*Game{
			{
				GameId:      "gbu1",
				SpectatorId: "sbu1",
				CreatedAt:   gameNow,
				ExpiresAt:   gameNow.Add(time.Hour),
				Players: []Player{
					{UserId: "game_by_user2", PlayerId: "p1_2", Color: ColorRed},
				},
			},
			{
				GameId:      "gbu4",
				SpectatorId: "sbu4",
				CreatedAt:   gameNow,
				ExpiresAt:   gameNow.Add(time.Hour),
				Players: []Player{
					{UserId: "game_by_user2", PlayerId: "p4_2", Color: ColorBronze},
				},
			},
		}); diff != "" {
			t.Errorf("GetGamesByUserId (-want +got):\n%s", diff)
		}
	})

	t.Run("query finished game", func(t *testing.T) {
		got, err := storage.GetGamesByUserId(ctx, "game_by_user2", time.Hour*2)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(got, []*Game{
			{
				GameId:      "gbu1",
				SpectatorId: "sbu1",
				CreatedAt:   gameNow,
				ExpiresAt:   gameNow.Add(time.Hour),
				Players: []Player{
					{UserId: "game_by_user2", PlayerId: "p1_2", Color: ColorRed},
				},
			},
			{
				GameId:      "gbu4",
				SpectatorId: "sbu4",
				CreatedAt:   gameNow,
				ExpiresAt:   gameNow.Add(time.Hour),
				Players: []Player{
					{UserId: "game_by_user2", PlayerId: "p4_2", Color: ColorBronze},
				},
			},
			{
				GameId:      "gbu5",
				SpectatorId: "sbu5",
				CreatedAt:   gameNow,
				ExpiresAt:   gameNow.Add(time.Hour),
				Players: []Player{
					{UserId: "game_by_user2", PlayerId: "p5_2", Color: ColorBronze},
				},
				FinishedAt: &finishTime,
			},
		}); diff != "" {
			t.Errorf("GetGamesByUserId (-want +got):\n%s", diff)
		}
	})

	t.Run("GetGameByPlayerId", func(t *testing.T) {
		got, err := storage.GetGameByPlayerId(ctx, "p1_3")
		assert.NilError(t, err)
		assert.DeepEqual(t, got, &Game{
			GameId:      "gbu1",
			SpectatorId: "sbu1",
			CreatedAt:   gameNow,
			ExpiresAt:   gameNow.Add(time.Hour),
			Players: []Player{
				{UserId: "game_by_user1", PlayerId: "p1_1", Color: ColorBlue},
				{UserId: "game_by_user2", PlayerId: "p1_2", Color: ColorRed},
				{UserId: "game_by_user3", PlayerId: "p1_3", Color: ColorYellow},
			},
		})
	})
}

func TestStorage(t *testing.T) {
	t.Parallel()

	storage := prepareStorage(t)
	ctx := context.Background()

	t.Run("Create/Get users", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		storage.nowFunc = func() time.Time { return now }
		t.Run("UpsertUser - success", func(t *testing.T) {
			err := storage.UpsertUser(ctx, UpsertUser{
				UserId:   "test user id",
				Nickname: "test user nickname",
				Color:    ColorBronze,
				LastIp:   "last ip 1",
			})
			assert.NilError(t, err)
		})

		t.Run("GetUserById - success", func(t *testing.T) {
			user, err := storage.GetUserById(ctx, "test user id")
			assert.NilError(t, err)
			assert.DeepEqual(t, user, &User{
				UserId:          "test user id",
				Nickname:        "test user nickname",
				Color:           ColorBronze,
				CreatedAt:       now,
				DeviceTokenType: DeviceTokenTypeProduction,
				LastIp:          "last ip 1",
				Type:            UserTypeBlank, // Upsert creates blank user
				Elo:             1000,
			})
		})

		now2 := now.Add(time.Second)
		storage.nowFunc = func() time.Time { return now2 }
		t.Run("UpsertUser - existing user", func(t *testing.T) {
			err := storage.UpsertUser(ctx, UpsertUser{
				UserId:   "test user id",
				Nickname: "test user nickname 2",
				Color:    ColorOrange,
				LastIp:   "last ip 2",
			})
			assert.NilError(t, err)

			user, err := storage.GetUserById(ctx, "test user id")
			assert.NilError(t, err)
			assert.DeepEqual(t, user, &User{
				UserId:          "test user id",
				Nickname:        "test user nickname",
				Color:           ColorBronze,
				CreatedAt:       now,
				DeviceTokenType: DeviceTokenTypeProduction,
				LastIp:          "last ip 2",
				Type:            UserTypeBlank,
				Elo:             1000,
			})
		})

		t.Run("GetUserById - no IP", func(t *testing.T) {
			err := storage.UpsertUser(ctx, UpsertUser{
				UserId:   "test get user no ip id",
				Nickname: "test get user no ip nickname",
				Color:    ColorBronze,
			})
			assert.NilError(t, err)

			expected := &User{
				UserId:          "test get user no ip id",
				Nickname:        "test get user no ip nickname",
				Color:           ColorBronze,
				CreatedAt:       now2,
				DeviceTokenType: DeviceTokenTypeProduction,
				Type:            UserTypeBlank,
				Elo:             1000,
			}

			user, err := storage.GetUserById(ctx, "test get user no ip id")
			assert.NilError(t, err)
			assert.DeepEqual(t, expected, user)

			user, err = storage.GetUserByNickname(ctx, "test get user no ip nickname")
			assert.NilError(t, err)
			assert.DeepEqual(t, expected, user)
		})

		t.Run("GetUserById - not found", func(t *testing.T) {
			_, err := storage.GetUserById(ctx, "test user not found")
			assert.ErrorIs(t, err, ErrNotFound)
		})

		t.Run("UpdateUser - success", func(t *testing.T) {
			expectedAfterUpdate := &User{
				UserId:    "test user id",
				Nickname:  "new test nickname",
				Color:     ColorGreen,
				CreatedAt: now,
			}
			updated, err := storage.UpdateUser(ctx, UpdateUser{
				UserId:   "test user id",
				Nickname: "new test nickname",
				Color:    ColorGreen,
				Type:     UserTypeActive,
			})
			assert.NilError(t, err)
			assert.DeepEqual(t, expectedAfterUpdate, updated)

			expectedAfterGet := &User{
				UserId:          "test user id",
				Nickname:        "new test nickname",
				Color:           ColorGreen,
				CreatedAt:       now,
				DeviceTokenType: DeviceTokenTypeProduction,
				LastIp:          "last ip 2",
				Type:            UserTypeActive,
				Elo:             1000,
			}
			got, err := storage.GetUserById(ctx, "test user id")
			assert.NilError(t, err)
			assert.DeepEqual(t, expectedAfterGet, got)
		})

		t.Run("UpsertUser - does not affect active type", func(t *testing.T) {
			err := storage.UpsertUser(ctx, UpsertUser{
				UserId:   "test user id",
				Nickname: "test user nickname 3",
				Color:    ColorOrange,
				LastIp:   "last ip 3",
			})
			assert.NilError(t, err)

			user, err := storage.GetUserById(ctx, "test user id")
			assert.NilError(t, err)
			assert.DeepEqual(t, user, &User{
				UserId:          "test user id",
				Nickname:        "new test nickname",
				Color:           ColorGreen,
				CreatedAt:       now,
				DeviceTokenType: DeviceTokenTypeProduction,
				LastIp:          "last ip 3",
				Type:            UserTypeActive,
				Elo:             1000,
			})
		})

		t.Run("UpdateUser - not found", func(t *testing.T) {
			_, err := storage.UpdateUser(ctx, UpdateUser{
				UserId:   "not existing test user id",
				Nickname: "new test nickname",
			})
			assert.ErrorIs(t, err, ErrNotFound)
		})

		t.Run("UpdateUser - already exists", func(t *testing.T) {
			err := storage.UpsertUser(ctx, UpsertUser{
				UserId:   "second test user id",
				Nickname: "second test user nickname",
			})
			assert.NilError(t, err)

			err = storage.UpsertUser(ctx, UpsertUser{
				UserId:   "third test user id",
				Nickname: "third test user nickname",
			})
			assert.NilError(t, err)

			_, err = storage.UpdateUser(ctx, UpdateUser{
				UserId:   "second test user id",
				Nickname: "third test user nickname",
			})
			assert.ErrorIs(t, err, ErrAlreadyExists)
		})

		t.Run("GetUsersByNicknames - success", func(t *testing.T) {
			got, err := storage.GetUserByNickname(ctx, "second test user nickname")
			assert.NilError(t, err)
			assert.DeepEqual(t, got, &User{
				UserId:          "second test user id",
				Nickname:        "second test user nickname",
				CreatedAt:       now2,
				DeviceTokenType: DeviceTokenTypeProduction,
				Type:            UserTypeBlank,
				Elo:             1000,
			})
		})

		t.Run("GetUsersByNicknames - not found", func(t *testing.T) {
			_, err := storage.GetUserByNickname(ctx, "not existing user")
			assert.ErrorIs(t, err, ErrNotFound)
		})

		t.Run("UpdateDeviceToken - success", func(t *testing.T) {
			err := storage.UpsertUser(ctx, UpsertUser{
				UserId:   "device token user",
				Nickname: "device token user nickname",
			})
			assert.NilError(t, err)

			err = storage.UpdateDeviceToken(ctx, "device token user", []byte("device token"), DeviceTokenTypeSandbox)
			assert.NilError(t, err)

			got, err := storage.GetUserById(ctx, "device token user")
			assert.NilError(t, err)
			assert.DeepEqual(t, got.DeviceToken, []byte("device token"))
			assert.Equal(t, got.DeviceTokenType, DeviceTokenTypeSandbox)
		})
	})

	t.Run("SearchUsers", func(t *testing.T) {
		searchNow := time.Now().Truncate(time.Second)
		storage.nowFunc = func() time.Time { return searchNow }
		for _, u := range []UpsertUser{
			{UserId: "search 1", Nickname: "prefix middle nickname suffix"},
			{UserId: "search 2", Nickname: "prefix middlenick surname"},
			{UserId: "search 3", Nickname: "prefix nsuffix"},
			{UserId: "search 4", Nickname: "free middle nickname"},
		} {
			err := storage.UpsertUser(ctx, u)
			assert.NilError(t, err)
		}
		_, err := storage.UpdateUser(ctx, UpdateUser{
			UserId:   "search 4",
			Nickname: "free middle nickname",
			Type:     UserTypeActive,
		})
		if err != nil {
			t.Fatal(err)
		}

		tests := []struct {
			name    string
			search  SearchUsers
			want    []*User
			wantErr error
		}{
			{
				name:   "success - exact",
				search: SearchUsers{Search: "prefix middle nickname suffix", Limit: 5, Type: UserTypeBlank},
				want: []*User{
					{UserId: "search 1", Nickname: "prefix middle nickname suffix", CreatedAt: searchNow, Elo: 1000},
				},
			},
			{
				name:   "success - prefix",
				search: SearchUsers{Search: "prefix", Limit: 5, Type: UserTypeBlank},
				want: []*User{
					{UserId: "search 1", Nickname: "prefix middle nickname suffix", CreatedAt: searchNow, Elo: 1000},
					{UserId: "search 2", Nickname: "prefix middlenick surname", CreatedAt: searchNow, Elo: 1000},
					{UserId: "search 3", Nickname: "prefix nsuffix", CreatedAt: searchNow, Elo: 1000},
				},
			},
			{
				name:   "success - suffix",
				search: SearchUsers{Search: "suffix", Limit: 5, Type: UserTypeBlank},
				want: []*User{
					{UserId: "search 1", Nickname: "prefix middle nickname suffix", CreatedAt: searchNow, Elo: 1000},
					{UserId: "search 3", Nickname: "prefix nsuffix", CreatedAt: searchNow, Elo: 1000},
				},
			},
			{
				name:   "success - middle",
				search: SearchUsers{Search: "middle", Limit: 5, Type: UserTypeBlank},
				want: []*User{
					{UserId: "search 1", Nickname: "prefix middle nickname suffix", CreatedAt: searchNow, Elo: 1000},
					{UserId: "search 2", Nickname: "prefix middlenick surname", CreatedAt: searchNow, Elo: 1000},
				},
			},
			{
				name:   "success - middle and active type",
				search: SearchUsers{Search: "middle", Limit: 5, Type: UserTypeActive},
				want: []*User{
					{UserId: "search 4", Nickname: "free middle nickname", CreatedAt: searchNow, Elo: 1000},
				},
			},
			{
				name:   "success - limit",
				search: SearchUsers{Search: "prefix", Limit: 2, Type: UserTypeBlank},
				want: []*User{
					{UserId: "search 1", Nickname: "prefix middle nickname suffix", CreatedAt: searchNow, Elo: 1000},
					{UserId: "search 2", Nickname: "prefix middlenick surname", CreatedAt: searchNow, Elo: 1000},
				},
			},
			{
				name:   "success - exclude",
				search: SearchUsers{Search: "prefix", Limit: 5, ExcludedUserId: "search 2", Type: UserTypeBlank},
				want: []*User{
					{UserId: "search 1", Nickname: "prefix middle nickname suffix", CreatedAt: searchNow, Elo: 1000},
					{UserId: "search 3", Nickname: "prefix nsuffix", CreatedAt: searchNow, Elo: 1000},
				},
			},
			{
				name:    "error - term not found",
				search:  SearchUsers{Search: "some invalid term", Limit: 5, Type: UserTypeBlank},
				wantErr: ErrNotFound,
			},
			{
				name:    "error - type not found",
				search:  SearchUsers{Search: "middle", Limit: 5, Type: UserType("some invalid type")},
				wantErr: ErrNotFound,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := storage.SearchUsers(ctx, tt.search)
				if tt.wantErr != nil {
					assert.ErrorIs(t, err, tt.wantErr)
					return
				}
				assert.NilError(t, err)
				assert.DeepEqual(t, tt.want, got)
			})
		}
	})

	t.Run("CreateGame", func(t *testing.T) {
		gameNow := time.Now().Truncate(time.Second)
		storage.nowFunc = func() time.Time { return gameNow }
		for _, u := range []UpsertUser{
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
					Players: []Player{
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
					Players: []Player{
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
					Players: []Player{
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
					Players: []Player{
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
					Players: []Player{
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

	t.Run("GetActiveUsers", func(t *testing.T) {
		gameNow := time.Now().Truncate(time.Second).Add(2 * time.Hour) // 2 Hours added to avoid querying other games
		storage.nowFunc = func() time.Time { return gameNow }
		buffer := 2 * time.Minute

		_, err := storage.GetActiveUsers(ctx, buffer)
		assert.ErrorIs(t, err, ErrNotFound)

		for _, u := range []UpsertUser{
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
				Players: []Player{
					{UserId: "active_user1", PlayerId: "aup1_1", Color: ColorBlue},
					{UserId: "active_user2", PlayerId: "aup1_2", Color: ColorRed},
				},
			},
			{ // Recently expired game
				GameId:      "au2",
				SpectatorId: "sau2",
				ExpiresAt:   gameNow.Add(-time.Minute),
				Players: []Player{
					{UserId: "active_user2", PlayerId: "aup2_2", Color: ColorBlue},
					{UserId: "active_user3", PlayerId: "aup2_3", Color: ColorYellow},
				},
			},
			{ // Expired game
				GameId:      "au3",
				SpectatorId: "sau3",
				ExpiresAt:   gameNow.Add(-time.Hour),
				Players: []Player{
					{UserId: "active_user3", PlayerId: "aup3_3", Color: ColorBlue},
					{UserId: "active_user4", PlayerId: "aup3_4", Color: ColorYellow},
				},
			},
		} {
			err := storage.CreateGame(ctx, g)
			assert.NilError(t, err)
		}

		got, err := storage.GetActiveUsers(ctx, buffer)
		assert.NilError(t, err)
		assert.DeepEqual(t, got, []string{
			"active_user1", "active_user2", "active_user3",
		})
	})

	t.Run("UpdateSentNotification", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		storage.nowFunc = func() time.Time { return now }
		for _, u := range []UpsertUser{
			{UserId: "notification_user1", Nickname: "notification user 1"},
		} {
			err := storage.UpsertUser(ctx, u)
			assert.NilError(t, err)
		}

		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
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
				func(ctx context.Context, state UserNotificationState) (UserNotificationState, error) {
					assert.DeepEqual(t, UserNotificationState{
						DeviceToken:      []byte("some token"),
						DeviceTokenType:  DeviceTokenTypeProduction,
						SentNotification: SentNotification{ActiveGames: 2},
					}, state)
					return UserNotificationState{
						DeviceToken:      []byte("some token"),
						DeviceTokenType:  DeviceTokenTypeSandbox,
						SentNotification: SentNotification{ActiveGames: 3},
					}, nil
				})
			backgroundErr <- err
		}()

		err := storage.UpdateSentNotification(ctx, "notification_user1",
			func(ctx context.Context, state UserNotificationState) (UserNotificationState, error) {
				wait <- struct{}{}
				return UserNotificationState{
					DeviceToken:      []byte("some token"),
					DeviceTokenType:  DeviceTokenTypeProduction,
					SentNotification: SentNotification{ActiveGames: 2},
				}, nil
			})
		assert.NilError(t, err)
		err = <-backgroundErr
		assert.NilError(t, err)

		err = storage.UpdateSentNotification(ctx, "notification_user1",
			func(ctx context.Context, state UserNotificationState) (UserNotificationState, error) {
				assert.DeepEqual(t, UserNotificationState{
					DeviceToken:      []byte("some token"),
					DeviceTokenType:  DeviceTokenTypeSandbox,
					SentNotification: SentNotification{ActiveGames: 3},
				}, state)
				return state, nil
			})
		assert.NilError(t, err)
	})

	t.Run("GetActiveGames", func(t *testing.T) {
		now := time.Now().Truncate(time.Second).Add(4 * time.Hour) // 4 Hours added to avoid querying other games
		storage.nowFunc = func() time.Time { return now }

		_, err := storage.GetActiveGames(ctx)
		assert.ErrorIs(t, err, ErrNotFound)

		for _, u := range []UpsertUser{
			{UserId: "active_game_user1", Nickname: "active game user 1"},
			{UserId: "active_game_user2", Nickname: "active game user 2"},
			{UserId: "active_game_user3", Nickname: "active game user 3"},
			{UserId: "active_game_user4", Nickname: "active game user 4"},
		} {
			err := storage.UpsertUser(ctx, u)
			assert.NilError(t, err)
		}

		for _, g := range []*Game{
			{
				GameId:      "gag1",
				SpectatorId: "sgag1",
				ExpiresAt:   now.Add(time.Hour),
				Players: []Player{
					{UserId: "active_game_user1", PlayerId: "agagp1_1", Color: ColorBlue},
					{UserId: "active_game_user2", PlayerId: "agagp1_2", Color: ColorRed},
				},
			},
			{
				GameId:      "gag2",
				SpectatorId: "sgag2",
				ExpiresAt:   now.Add(time.Hour),
				Players: []Player{
					{UserId: "active_game_user3", PlayerId: "agagp2_3", Color: ColorBlue},
					{UserId: "active_game_user2", PlayerId: "agagp2_2", Color: ColorRed},
				},
			},
			{ // Expired game
				GameId:      "gag3",
				SpectatorId: "sgag3",
				ExpiresAt:   now.Add(-time.Minute),
				Players: []Player{
					{UserId: "active_game_user1", PlayerId: "agagp3_1", Color: ColorBlue},
					{UserId: "active_game_user4", PlayerId: "agagp3_4", Color: ColorRed},
				},
			},
		} {
			err := storage.CreateGame(ctx, g)
			assert.NilError(t, err)
		}

		got, err := storage.GetActiveGames(ctx)
		assert.NilError(t, err)
		assert.DeepEqual(t, got, []*Game{
			{
				GameId:      "gag1",
				SpectatorId: "sgag1",
				CreatedAt:   now,
				ExpiresAt:   now.Add(time.Hour),
			},
			{
				GameId:      "gag2",
				SpectatorId: "sgag2",
				CreatedAt:   now,
				ExpiresAt:   now.Add(time.Hour),
			},
		})

		t.Run("UpdateGameResults", func(t *testing.T) {
			updateNow := now.Add(30 * time.Minute)
			storage.nowFunc = func() time.Time { return updateNow }

			// 3 players still have active games
			u, err := storage.GetActiveUsers(ctx, 0)
			assert.NilError(t, err)
			assert.DeepEqual(t, u, []string{
				"active_game_user1", "active_game_user2", "active_game_user3",
			})

			err = storage.UpdateGameResults(ctx, "gag1", &GameResults{Raw: map[string]any{
				"some data": 42,
			}})
			assert.NilError(t, err)

			got, err := storage.GetActiveGames(ctx)
			assert.NilError(t, err)
			assert.DeepEqual(t, got, []*Game{
				{
					GameId:      "gag2",
					SpectatorId: "sgag2",
					CreatedAt:   now,
					ExpiresAt:   now.Add(time.Hour),
				},
			})

			// After 5 minutes 3 players have active games with the buffer of 10 minutes
			updateNow = updateNow.Add(5 * time.Minute)
			u, err = storage.GetActiveUsers(ctx, 10*time.Minute)
			assert.NilError(t, err)
			assert.DeepEqual(t, u, []string{
				"active_game_user1", "active_game_user2", "active_game_user3",
			})

			// Only 2 have games with a small buffer
			u, err = storage.GetActiveUsers(ctx, time.Minute)
			assert.NilError(t, err)
			assert.DeepEqual(t, u, []string{
				"active_game_user2", "active_game_user3",
			})
		})
	})

	t.Run("UpdateElo", func(t *testing.T) {
		now := time.Now().Truncate(time.Second).Add(-1 * time.Hour)
		storage.nowFunc = func() time.Time { return now }

		for _, u := range []UpsertUser{
			{UserId: "update elo 1", Nickname: "update elo player 1"},
			{UserId: "update elo 2", Nickname: "update elo player 2"},
			{UserId: "update elo 3", Nickname: "update elo player 3"},
			{UserId: "update elo 4", Nickname: "update elo player 4"},
		} {
			err := storage.UpsertUser(ctx, u)
			assert.NilError(t, err)
		}

		err := storage.CreateGame(ctx, &Game{
			GameId:      "update elo 1",
			SpectatorId: "spec update elo 1",
			ExpiresAt:   now.Add(time.Hour),
			Players: []Player{
				{UserId: "update elo 1", PlayerId: "update elo player 1 1", Color: ColorBlue},
				{UserId: "update elo 2", PlayerId: "update elo player 1 2", Color: ColorRed},
				{UserId: "update elo 3", PlayerId: "update elo player 1 3", Color: ColorBronze},
				{UserId: "update elo 4", PlayerId: "update elo player 1 4", Color: ColorPink},
			},
		})
		assert.NilError(t, err)

		err = storage.UpdateGameResults(ctx, "update elo 1", &GameResults{Raw: map[string]any{
			"some data": 42,
		}})
		assert.NilError(t, err)

		err = storage.UpdateElo(ctx, func(ctx context.Context, state EloUpdateState) (EloResults, error) {
			assert.DeepEqual(t, EloUpdateState{
				Game: Game{
					GameId:      "update elo 1",
					SpectatorId: "spec update elo 1",
					CreatedAt:   now,
					ExpiresAt:   now.Add(time.Hour),
					Players: []Player{
						{UserId: "update elo 1", PlayerId: "update elo player 1 1", Color: ColorBlue},
						{UserId: "update elo 2", PlayerId: "update elo player 1 2", Color: ColorRed},
						{UserId: "update elo 3", PlayerId: "update elo player 1 3", Color: ColorBronze},
						{UserId: "update elo 4", PlayerId: "update elo player 1 4", Color: ColorPink},
					},
					GameResults: &GameResults{Raw: map[string]any{
						"some data": float64(42),
					}},
				},
				Users: []EloStateUser{
					{UserId: "update elo 1", Elo: 1000},
					{UserId: "update elo 2", Elo: 1000},
					{UserId: "update elo 3", Elo: 1000},
					{UserId: "update elo 4", Elo: 1000},
				},
			}, state)
			return EloResults{
				Players: []EloResultsPlayer{
					{UserId: "update elo 1", PlayerId: "update elo player 1 1", OldElo: 1000, NewElo: 1010},
					{UserId: "update elo 2", PlayerId: "update elo player 1 2", OldElo: 1000, NewElo: 1000},
					{UserId: "update elo 3", PlayerId: "update elo player 1 3", OldElo: 1000, NewElo: 985},
					{UserId: "update elo 4", PlayerId: "update elo player 1 4", OldElo: 1000, NewElo: 1024},
				},
			}, nil
		})
		assert.NilError(t, err)

		for _, u := range []struct {
			userId      string
			expectedElo int64
		}{
			{userId: "update elo 1", expectedElo: 1010},
			{userId: "update elo 2", expectedElo: 1000},
			{userId: "update elo 3", expectedElo: 985},
			{userId: "update elo 4", expectedElo: 1024},
		} {
			got, err := storage.GetUserById(ctx, u.userId)
			assert.NilError(t, err)
			assert.Equal(t, u.expectedElo, got.Elo)
		}

		t.Run("leaderboard", func(t *testing.T) {
			got, err := storage.GetLeaderboard(ctx, UserTypeBlank, 2)
			assert.NilError(t, err)
			assert.DeepEqual(t, []*User{
				{UserId: "update elo 4", Nickname: "update elo player 4", CreatedAt: now, Elo: 1024},
				{UserId: "update elo 1", Nickname: "update elo player 1", CreatedAt: now, Elo: 1010},
			}, got)
		})
	})
}

func prepareStorage(t *testing.T) *Storage {
	const (
		prepareTimeout = time.Second * 10
	)

	mainDB, err := sql.Open("pgx", getDSN(defaultDatabase))
	if err != nil {
		t.Fatal(err)
	}
	defer mainDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), prepareTimeout)
	defer cancel()

	newDBName := strings.ToLower(t.Name())
	_, err = mainDB.ExecContext(ctx, "CREATE DATABASE "+newDBName)
	if err != nil {
		t.Fatal(err)
	}

	db, err := database.PrepareDB(getDSN(newDBName))
	if err != nil {
		t.Fatal(err)
	}

	s, err := New(db)
	if err != nil {
		t.Fatal(err)
	}

	return s
}

func getDSN(name string) string {
	return fmt.Sprintf("postgres://postgres:postgres@localhost:5432/%s?sslmode=disable", name)
}

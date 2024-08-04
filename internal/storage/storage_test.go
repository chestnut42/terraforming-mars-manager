package storage

import (
	"context"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/chestnut42/terraforming-mars-manager/internal/database"
	"github.com/chestnut42/terraforming-mars-manager/pkg/api"
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
				Color:    api.PlayerColor_BRONZE,
			})
			assert.NilError(t, err)
		})

		t.Run("GetUserById - success", func(t *testing.T) {
			user, err := storage.GetUserById(ctx, "test user id")
			assert.NilError(t, err)
			assert.Assert(t, user != nil)
			assert.Assert(t, user.UserId == "test user id")
			assert.Assert(t, user.Nickname == "test user nickname")
			assert.Assert(t, user.Color == api.PlayerColor_BRONZE)
			assert.Assert(t, user.CreatedAt == now)
		})

		now2 := now.Add(time.Second)
		storage.nowFunc = func() time.Time { return now2 }
		t.Run("UpsertUser - existing user", func(t *testing.T) {
			err := storage.UpsertUser(ctx, &User{
				UserId:   "test user id",
				Nickname: "test user nickname 2",
				Color:    api.PlayerColor_ORANGE,
			})
			assert.NilError(t, err)

			user, err := storage.GetUserById(ctx, "test user id")
			assert.NilError(t, err)
			assert.Assert(t, user != nil)
			assert.Assert(t, user.UserId == "test user id")
			assert.Assert(t, user.Nickname == "test user nickname")
			assert.Assert(t, user.Color == api.PlayerColor_BRONZE)
			assert.Assert(t, user.CreatedAt == now)
		})

		t.Run("GetUserById - not found", func(t *testing.T) {
			_, err := storage.GetUserById(ctx, "test user not found")
			assert.ErrorIs(t, err, ErrNotFound)
		})

		t.Run("UpdateUser - success", func(t *testing.T) {
			updated, err := storage.UpdateUser(ctx, &User{
				UserId:   "test user id",
				Nickname: "new test nickname",
				Color:    api.PlayerColor_GREEN,
			})
			assert.NilError(t, err)
			assert.Assert(t, updated.UserId == "test user id")
			assert.Assert(t, updated.Nickname == "new test nickname")
			assert.Assert(t, updated.Color == api.PlayerColor_GREEN)
			assert.Assert(t, updated.CreatedAt == now)

			refetched, err := storage.GetUserById(ctx, "test user id")
			assert.NilError(t, err)
			assert.Assert(t, refetched.UserId == "test user id")
			assert.Assert(t, refetched.Nickname == "new test nickname")
			assert.Assert(t, refetched.Color == api.PlayerColor_GREEN)
			assert.Assert(t, refetched.CreatedAt == now)
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
						{UserId: "game 1", PlayerId: "p1"},
						{UserId: "game 2", PlayerId: "p2"},
						{UserId: "game 3", PlayerId: "p3"},
					},
				},
			},
			{
				name: "error - non unique game id",
				game: Game{
					GameId:    "g1",
					ExpiresAt: gameNow.Add(time.Hour),
					Players: []*Player{
						{UserId: "game 1", PlayerId: "p4"},
						{UserId: "game 2", PlayerId: "p5"},
						{UserId: "game 3", PlayerId: "p6"},
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
						{UserId: "game 1", PlayerId: "p4"},
						{UserId: "game 2", PlayerId: "p2"},
						{UserId: "game 3", PlayerId: "p6"},
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
						{UserId: "game 1", PlayerId: "p4"},
						{UserId: "game 4", PlayerId: "p5"},
						{UserId: "game 3", PlayerId: "p6"},
					},
				},
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := storage.CreateGame(ctx, &tt.game)
				assert.Assert(t, (err != nil) == tt.wantErr)
			})
		}
	})
}

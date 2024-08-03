package storage

import (
	"bytes"
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
		assert.Assert(t, bytes.Equal(got.DeviceToken, []byte("device token")))
	})
}

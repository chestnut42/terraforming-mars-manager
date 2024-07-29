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

	now := time.Now().Truncate(time.Second)
	storage.nowFunc = func() time.Time { return now }
	t.Run("create new user", func(t *testing.T) {
		err := storage.UpsertUser(ctx, "test user id")
		assert.NilError(t, err)
	})

	t.Run("get user by id", func(t *testing.T) {
		user, err := storage.GetUserById(ctx, "test user id")
		assert.NilError(t, err)
		assert.Assert(t, user != nil)
		assert.Assert(t, user.UserId == "test user id")
		assert.Assert(t, user.Nickname == "")
		assert.Assert(t, user.CreatedAt == now)
	})

	now2 := now.Add(time.Second)
	storage.nowFunc = func() time.Time { return now2 }
	t.Run("create existing user", func(t *testing.T) {
		err := storage.UpsertUser(ctx, "test user id")
		assert.NilError(t, err)

		user, err := storage.GetUserById(ctx, "test user id")
		assert.NilError(t, err)
		assert.Assert(t, user != nil)
		assert.Assert(t, user.UserId == "test user id")
		assert.Assert(t, user.Nickname == "")
		assert.Assert(t, user.CreatedAt == now)
	})

	t.Run("user not found", func(t *testing.T) {
		_, err := storage.GetUserById(ctx, "test user not found")
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

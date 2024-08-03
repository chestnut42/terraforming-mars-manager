package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/chestnut42/terraforming-mars-manager/pkg/api"
)

type Storage struct {
	db *sql.DB

	getUserById       *sql.Stmt
	updateDeviceToken *sql.Stmt
	updateUser        *sql.Stmt
	upsertUser        *sql.Stmt

	nowFunc func() time.Time
}

func New(db *sql.DB) (*Storage, error) {
	getUserById, err := db.Prepare(`
		SELECT id, nickname, color, created_at, device_token FROM users WHERE id = $1
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare getUserById: %w", err)
	}

	updateDeviceToken, err := db.Prepare(`
		UPDATE users SET device_token = $1 WHERE id = $2
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare updateDeviceToken: %w", err)
	}

	updateUser, err := db.Prepare(`
		UPDATE users SET nickname = $1, color = $2 WHERE id = $3
		RETURNING id, nickname, color, created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare updateUser: %w", err)
	}

	upsertUser, err := db.Prepare(`
		INSERT INTO users(id, nickname, color, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT(id) DO NOTHING
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare upsertUser: %w", err)
	}

	return &Storage{
		db: db,

		getUserById:       getUserById,
		updateDeviceToken: updateDeviceToken,
		updateUser:        updateUser,
		upsertUser:        upsertUser,

		nowFunc: time.Now,
	}, nil
}

func (s *Storage) GetUserById(ctx context.Context, userId string) (*User, error) {
	user := User{}
	var colorStr string

	err := s.getUserById.QueryRowContext(ctx, userId).
		Scan(&user.UserId, &user.Nickname, &colorStr, &user.CreatedAt, &user.DeviceToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	user.Color = api.PlayerColor(api.PlayerColor_value[colorStr])
	return &user, nil
}

func (s *Storage) UpdateDeviceToken(ctx context.Context, userId string, deviceToken []byte) error {
	_, err := s.updateDeviceToken.ExecContext(ctx, deviceToken, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to update device token: %w", err)
	}
	return nil
}

func (s *Storage) UpdateUser(ctx context.Context, user *User) (*User, error) {
	inColor := api.PlayerColor_name[int32(user.Color)]

	updated := User{}
	var outColor string

	err := s.updateUser.QueryRowContext(ctx, user.Nickname, inColor, user.UserId).
		Scan(&updated.UserId, &updated.Nickname, &outColor, &updated.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		if errIsUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	updated.Color = api.PlayerColor(api.PlayerColor_value[outColor])
	return &updated, nil
}

func (s *Storage) UpsertUser(ctx context.Context, user *User) error {
	now := s.nowFunc()
	colorStr := api.PlayerColor_name[int32(user.Color)]

	if _, err := s.upsertUser.ExecContext(ctx, &user.UserId, &user.Nickname, &colorStr, &now); err != nil {
		return fmt.Errorf("failed to upsert user: %w", err)
	}
	return nil
}

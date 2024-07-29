package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Storage struct {
	db *sql.DB

	getUserById *sql.Stmt
	upsertUser  *sql.Stmt

	nowFunc func() time.Time
}

func New(db *sql.DB) (*Storage, error) {
	getUserById, err := db.Prepare(`
		SELECT id, nickname, created_at FROM users WHERE id = $1
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare getUserById: %w", err)
	}

	upsertUser, err := db.Prepare(`
		INSERT INTO users(id, nickname, created_at)
		VALUES ($1, '', $2)
		ON CONFLICT(id) DO NOTHING
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare upsertUser: %w", err)
	}

	return &Storage{
		db: db,

		getUserById: getUserById,
		upsertUser:  upsertUser,

		nowFunc: time.Now,
	}, nil
}

func (s *Storage) GetUserById(ctx context.Context, userId string) (*User, error) {
	user := User{}

	err := s.getUserById.QueryRowContext(ctx, userId).
		Scan(&user.UserId, &user.Nickname, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}
	return &user, nil
}

func (s *Storage) UpsertUser(ctx context.Context, userId string) error {
	now := s.nowFunc()

	if _, err := s.upsertUser.ExecContext(ctx, &userId, &now); err != nil {
		return fmt.Errorf("failed to upsert user: %w", err)
	}
	return nil
}

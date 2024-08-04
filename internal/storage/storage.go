package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/chestnut42/terraforming-mars-manager/internal/framework/logx"
	"github.com/chestnut42/terraforming-mars-manager/pkg/api"
)

type Storage struct {
	db *sql.DB

	getUserById       *sql.Stmt
	getUserByNickname *sql.Stmt
	insertGame        *sql.Stmt
	insertPlayer      *sql.Stmt
	searchUsers       *sql.Stmt
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

	getUserByNickname, err := db.Prepare(`
		SELECT id, nickname, color, created_at, device_token FROM users WHERE nickname = $1
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare getUsersByNicknames: %w", err)
	}

	insertGame, err := db.Prepare(`
		INSERT INTO games (id, spectator_id, created_at, expires_at) 
			VALUES ($1, $2, $3, $4) 
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare insertGame: %w", err)
	}

	insertPlayer, err := db.Prepare(`
		INSERT INTO game_players (game_id, user_id, player_id)
			VALUES ($1, $2, $3)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare insertPlayer: %w", err)
	}

	searchUsers, err := db.Prepare(`
		SELECT id, nickname, color, created_at FROM users
			WHERE nickname LIKE $1 AND id != $2 ORDER BY nickname LIMIT $3
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare searchUsers: %w", err)
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
		getUserByNickname: getUserByNickname,
		insertGame:        insertGame,
		insertPlayer:      insertPlayer,
		searchUsers:       searchUsers,
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

func (s *Storage) GetUserByNickname(ctx context.Context, nickname string) (*User, error) {
	user := User{}
	var colorStr string

	err := s.getUserByNickname.QueryRowContext(ctx, nickname).
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

func (s *Storage) SearchUsers(ctx context.Context, search string, limit int, excludeUser string) ([]*User, error) {
	rows, err := s.searchUsers.QueryContext(ctx, "%"+search+"%", excludeUser, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query searchUsers: %w", err)
	}
	defer rows.Close()

	users := make([]*User, 0, limit)
	for rows.Next() {
		user := User{}
		var colorStr string

		if err := rows.Scan(&user.UserId, &user.Nickname, &colorStr, &user.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to query searchUsers: %w", err)
		}
		user.Color = api.PlayerColor(api.PlayerColor_value[colorStr])
		users = append(users, &user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to query searchUsers: %w", err)
	}
	if len(users) == 0 {
		return nil, ErrNotFound
	}
	return users, nil
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

func (s *Storage) CreateGame(ctx context.Context, game *Game) error {
	now := s.nowFunc()

	if err := s.withTX(ctx, func(ctx context.Context, tx *sql.Tx) error {
		insertGame := tx.Stmt(s.insertGame)
		insertPlayer := tx.Stmt(s.insertPlayer)

		_, err := insertGame.ExecContext(ctx, &game.GameId, &game.SpectatorId, &now, &game.ExpiresAt)
		if err != nil {
			return fmt.Errorf("failed to insert game: %w", err)
		}
		for _, p := range game.Players {
			_, err := insertPlayer.ExecContext(ctx, &game.GameId, &p.UserId, &p.PlayerId)
			if err != nil {
				return fmt.Errorf("failed to insert player(%s): %w", p.UserId, err)
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to create game: %w", err)
	}
	return nil
}

func (s *Storage) withTX(ctx context.Context, f func(ctx context.Context, tx *sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	if err := f(ctx, tx); err != nil {
		if err := tx.Rollback(); err != nil {
			logx.Logger(ctx).Info("failed to rollback transaction", slog.Any("error", err))
		}
		return err
	}
	return tx.Commit()
}

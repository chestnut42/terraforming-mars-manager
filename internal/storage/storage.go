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

	getGamesByUserId  *sql.Stmt
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
	getGamesByUserId, err := db.Prepare(`
		SELECT games.id, games.spectator_id, games.created_at, games.expires_at,
		       game_players.user_id, game_players.player_id, game_players.color
			FROM game_players INNER JOIN games ON game_players.game_id = games.id
			WHERE game_players.user_id = $1 AND games.expires_at > $2
			ORDER BY games.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare getGamesByUserId: %w", err)
	}

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
		INSERT INTO game_players (game_id, user_id, player_id, color)
			VALUES ($1, $2, $3, $4)
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

		getGamesByUserId:  getGamesByUserId,
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

	err := s.getUserById.QueryRowContext(ctx, userId).
		Scan(&user.UserId, &user.Nickname, &user.Color, &user.CreatedAt, &user.DeviceToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}
	return &user, nil
}

func (s *Storage) GetUserByNickname(ctx context.Context, nickname string) (*User, error) {
	user := User{}

	err := s.getUserByNickname.QueryRowContext(ctx, nickname).
		Scan(&user.UserId, &user.Nickname, &user.Color, &user.CreatedAt, &user.DeviceToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}
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

		if err := rows.Scan(&user.UserId, &user.Nickname, &user.Color, &user.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to query searchUsers: %w", err)
		}
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
	updated := User{}

	err := s.updateUser.QueryRowContext(ctx, user.Nickname, user.Color, user.UserId).
		Scan(&updated.UserId, &updated.Nickname, &updated.Color, &updated.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		if errIsUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	return &updated, nil
}

func (s *Storage) UpsertUser(ctx context.Context, user *User) error {
	now := s.nowFunc()

	if _, err := s.upsertUser.ExecContext(ctx, &user.UserId, &user.Nickname, &user.Color, &now); err != nil {
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
			_, err := insertPlayer.ExecContext(ctx, &game.GameId, &p.UserId, &p.PlayerId, &p.Color)
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

func (s *Storage) GetGamesByUserId(ctx context.Context, userId string) ([]*Game, error) {
	now := s.nowFunc()
	rows, err := s.getGamesByUserId.QueryContext(ctx, userId, now)
	if err != nil {
		return nil, fmt.Errorf("failed to query getGamesByUserId: %w", err)
	}
	defer rows.Close()

	games := make([]*Game, 0)
	for rows.Next() {
		game := Game{}
		player := Player{}

		if err := rows.Scan(&game.GameId, &game.SpectatorId, &game.CreatedAt, &game.ExpiresAt,
			&player.UserId, &player.PlayerId, &player.Color); err != nil {
			return nil, fmt.Errorf("failed to query searchUsers: %w", err)
		}
		game.Players = []*Player{&player}
		games = append(games, &game)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to query searchUsers: %w", err)
	}
	if len(games) == 0 {
		return nil, ErrNotFound
	}
	return games, nil
}

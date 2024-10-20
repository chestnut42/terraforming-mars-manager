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

	getActiveGames        *sql.Stmt
	getActiveUsers        *sql.Stmt
	getGameByPlayerId     *sql.Stmt
	getGamePlayersAndElo  *sql.Stmt
	getGamesByUserId      *sql.Stmt
	getOldestFinishedGame *sql.Stmt
	getUserById           *sql.Stmt
	getUserByNickname     *sql.Stmt
	insertGame            *sql.Stmt
	insertPlayer          *sql.Stmt
	lockUser              *sql.Stmt
	searchUsers           *sql.Stmt
	updateDeviceToken     *sql.Stmt
	updateGameEloResults  *sql.Stmt
	updateGameResults     *sql.Stmt
	updateLockedUser      *sql.Stmt
	updateUser            *sql.Stmt
	updateUserElo         *sql.Stmt
	upsertUser            *sql.Stmt

	nowFunc func() time.Time
}

func New(db *sql.DB) (*Storage, error) {
	getActiveGames, err := db.Prepare(`
		SELECT id, spectator_id, created_at, expires_at
			FROM manager_games WHERE results is null and expires_at > $1
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare getActiveGames: %w", err)
	}

	getActiveUsers, err := db.Prepare(`
		SELECT distinct manager_game_players.user_id
			FROM manager_game_players INNER JOIN manager_games ON manager_game_players.game_id = manager_games.id
			WHERE manager_games.expires_at > $1 and coalesce(manager_games.finished_at, 'infinity') > $1
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare getActiveUsers: %w", err)
	}

	getGameByPlayerId, err := db.Prepare(`
		SELECT manager_games.id, manager_games.spectator_id, manager_games.created_at, manager_games.expires_at,
		       manager_game_players.user_id, manager_game_players.player_id, manager_game_players.color
			FROM manager_game_players INNER JOIN manager_games ON manager_game_players.game_id = manager_games.id
			WHERE manager_games.id = (SELECT game_id FROM manager_game_players WHERE player_id = $1)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare getGameByPlayerId: %w", err)
	}

	getGamePlayersAndElo, err := db.Prepare(`
		SELECT manager_game_players.user_id, manager_game_players.player_id, manager_game_players.color,
		       manager_users.elo
			FROM manager_game_players INNER JOIN manager_users ON manager_users.id = manager_game_players.user_id
			WHERE manager_game_players.game_id = $1
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare getGamePlayersAndElo: %w", err)
	}

	getGamesByUserId, err := db.Prepare(`
		SELECT manager_games.id, manager_games.spectator_id, manager_games.created_at, manager_games.expires_at,
		       manager_game_players.user_id, manager_game_players.player_id, manager_game_players.color
			FROM manager_game_players INNER JOIN manager_games ON manager_game_players.game_id = manager_games.id
			WHERE manager_game_players.user_id = $1 AND manager_games.expires_at > $2
			ORDER BY manager_games.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare getGamesByUserId: %w", err)
	}

	getOldestFinishedGame, err := db.Prepare(`
		SELECT id, spectator_id, created_at, expires_at, results
		    FROM manager_games
		    WHERE results is not null AND finished_at is not null AND elo_results is null
			ORDER BY finished_at LIMIT 1
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare getFinishedGameForUpdate: %w", err)
	}

	getUserById, err := db.Prepare(`
		SELECT id, nickname, color, created_at, device_token, device_token_type, last_ip, type, elo 
		FROM manager_users WHERE id = $1
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare getUserById: %w", err)
	}

	getUserByNickname, err := db.Prepare(`
		SELECT id, nickname, color, created_at, device_token, device_token_type, last_ip, type, elo
		FROM manager_users WHERE nickname = $1
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare getUsersByNicknames: %w", err)
	}

	insertGame, err := db.Prepare(`
		INSERT INTO manager_games (id, spectator_id, created_at, expires_at) 
			VALUES ($1, $2, $3, $4) 
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare insertGame: %w", err)
	}

	insertPlayer, err := db.Prepare(`
		INSERT INTO manager_game_players (game_id, user_id, player_id, color)
			VALUES ($1, $2, $3, $4)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare insertPlayer: %w", err)
	}

	lockUser, err := db.Prepare(`
		SELECT device_token, device_token_type, sent_notification FROM manager_users
			WHERE id = $1 FOR UPDATE
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare lockUser: %w", err)
	}

	searchUsers, err := db.Prepare(`
		SELECT id, nickname, color, created_at, elo FROM manager_users
			WHERE nickname LIKE $1 AND type = $2 AND id != $3 ORDER BY nickname LIMIT $4
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare searchUsers: %w", err)
	}

	updateDeviceToken, err := db.Prepare(`
		UPDATE manager_users SET device_token = $1, device_token_type = $2 WHERE id = $3
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare updateDeviceToken: %w", err)
	}

	updateGameEloResults, err := db.Prepare(`
		UPDATE manager_games SET elo_results = $1 WHERE id = $2 AND elo_results is null
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare updateGameEloResults: %w", err)
	}

	updateGameResults, err := db.Prepare(`
		UPDATE manager_games SET results = $1, finished_at = $2 WHERE id = $3
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare updateGameResults: %w", err)
	}

	updateLockedUser, err := db.Prepare(`
		UPDATE manager_users SET device_token = $1, device_token_type = $2, sent_notification = $3 
			WHERE id = $4
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare updateLockedUser: %w", err)
	}

	updateUser, err := db.Prepare(`
		UPDATE manager_users SET nickname = $1, color = $2, type = $3 WHERE id = $4
			RETURNING id, nickname, color, created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare updateUser: %w", err)
	}

	updateUserElo, err := db.Prepare(`
		UPDATE manager_users SET elo = $1 WHERE id = $2 and elo = $3
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare updateUserElo: %w", err)
	}

	upsertUser, err := db.Prepare(`
		INSERT INTO manager_users (id, nickname, color, created_at, last_ip)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT(id) DO UPDATE SET last_ip = $5
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare upsertUser: %w", err)
	}

	return &Storage{
		db: db,

		getActiveGames:        getActiveGames,
		getActiveUsers:        getActiveUsers,
		getGameByPlayerId:     getGameByPlayerId,
		getGamePlayersAndElo:  getGamePlayersAndElo,
		getGamesByUserId:      getGamesByUserId,
		getOldestFinishedGame: getOldestFinishedGame,
		getUserById:           getUserById,
		getUserByNickname:     getUserByNickname,
		insertGame:            insertGame,
		insertPlayer:          insertPlayer,
		lockUser:              lockUser,
		searchUsers:           searchUsers,
		updateDeviceToken:     updateDeviceToken,
		updateGameEloResults:  updateGameEloResults,
		updateGameResults:     updateGameResults,
		updateLockedUser:      updateLockedUser,
		updateUser:            updateUser,
		updateUserElo:         updateUserElo,
		upsertUser:            upsertUser,

		nowFunc: time.Now,
	}, nil
}

func (s *Storage) GetUserById(ctx context.Context, userId string) (*User, error) {
	user := User{}
	var lastIp sql.NullString

	err := s.getUserById.QueryRowContext(ctx, userId).
		Scan(&user.UserId, &user.Nickname, &user.Color, &user.CreatedAt,
			&user.DeviceToken, &user.DeviceTokenType, &lastIp, &user.Type, &user.Elo)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}
	user.LastIp = fromStrPtr(lastIp)
	return &user, nil
}

func (s *Storage) GetUserByNickname(ctx context.Context, nickname string) (*User, error) {
	user := User{}
	var lastIp sql.NullString

	err := s.getUserByNickname.QueryRowContext(ctx, nickname).
		Scan(&user.UserId, &user.Nickname, &user.Color, &user.CreatedAt,
			&user.DeviceToken, &user.DeviceTokenType, &lastIp, &user.Type, &user.Elo)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}
	user.LastIp = fromStrPtr(lastIp)
	return &user, nil
}

type SearchUsers struct {
	Search         string
	ExcludedUserId string
	Limit          int
	Type           UserType
}

func (s *Storage) SearchUsers(ctx context.Context, req SearchUsers) ([]*User, error) {
	rows, err := s.searchUsers.QueryContext(ctx,
		"%"+req.Search+"%", req.Type, req.ExcludedUserId, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query searchUsers: %w", err)
	}
	defer rows.Close()

	users := make([]*User, 0, req.Limit)
	for rows.Next() {
		user := User{}
		if err := rows.Scan(&user.UserId, &user.Nickname, &user.Color, &user.CreatedAt, &user.Elo); err != nil {
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

func (s *Storage) UpdateDeviceToken(ctx context.Context, userId string, deviceToken []byte, tokenType DeviceTokenType) error {
	_, err := s.updateDeviceToken.ExecContext(ctx, deviceToken, tokenType, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to update device token: %w", err)
	}
	return nil
}

type UpdateUser struct {
	UserId   string
	Nickname string
	Color    Color
	Type     UserType
}

func (s *Storage) UpdateUser(ctx context.Context, req UpdateUser) (*User, error) {
	updated := User{}

	err := s.updateUser.QueryRowContext(ctx, req.Nickname, req.Color, req.Type, req.UserId).
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

type UpsertUser struct {
	UserId   string
	Nickname string
	LastIp   string
	Color    Color
}

func (s *Storage) UpsertUser(ctx context.Context, req UpsertUser) error {
	now := s.nowFunc()
	lastIp := toStrPtr(req.LastIp)
	if _, err := s.upsertUser.ExecContext(ctx, req.UserId, req.Nickname, req.Color, now, lastIp); err != nil {
		return fmt.Errorf("failed to upsert user: %w", err)
	}
	return nil
}

func (s *Storage) CreateGame(ctx context.Context, game *Game) error {
	now := s.nowFunc()

	if err := s.withTX(ctx, func(ctx context.Context, tx *sql.Tx) error {
		insertGame := tx.StmtContext(ctx, s.insertGame)
		insertPlayer := tx.StmtContext(ctx, s.insertPlayer)

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
		game.Players = []Player{player}
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

func (s *Storage) GetGameByPlayerId(ctx context.Context, playerId string) (*Game, error) {
	rows, err := s.getGameByPlayerId.QueryContext(ctx, playerId)
	if err != nil {
		return nil, fmt.Errorf("failed to query getGamesByUserId: %w", err)
	}
	defer rows.Close()

	var game Game
	for rows.Next() {
		player := Player{}

		if err := rows.Scan(&game.GameId, &game.SpectatorId, &game.CreatedAt, &game.ExpiresAt,
			&player.UserId, &player.PlayerId, &player.Color); err != nil {
			return nil, fmt.Errorf("failed to query searchUsers: %w", err)
		}
		game.Players = append(game.Players, player)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to query searchUsers: %w", err)
	}
	if len(game.Players) == 0 {
		return nil, ErrNotFound
	}
	return &game, nil
}

func (s *Storage) GetActiveUsers(ctx context.Context, activityBuffer time.Duration) ([]string, error) {
	expiration := s.nowFunc().Add(-activityBuffer)
	rows, err := s.getActiveUsers.QueryContext(ctx, &expiration)
	if err != nil {
		return nil, fmt.Errorf("failed to query getActiveUsers: %w", err)
	}
	defer rows.Close()

	activeUsers := make([]string, 0)
	for rows.Next() {
		var userid string

		if err := rows.Scan(&userid); err != nil {
			return nil, fmt.Errorf("failed to query getActiveUsers: %w", err)
		}
		activeUsers = append(activeUsers, userid)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to query getActiveUsers: %w", err)
	}
	if len(activeUsers) == 0 {
		return nil, ErrNotFound
	}
	return activeUsers, nil
}

func (s *Storage) UpdateSentNotification(ctx context.Context, userId string, updater SentNotificationUpdater) error {
	if err := s.withTX(ctx, func(ctx context.Context, tx *sql.Tx) error {
		lockUser := tx.StmtContext(ctx, s.lockUser)
		updateUser := tx.StmtContext(ctx, s.updateLockedUser)

		var state UserNotificationState
		if err := lockUser.QueryRowContext(ctx, userId).
			Scan(&state.DeviceToken, &state.DeviceTokenType, &state.SentNotification); err != nil {
			return fmt.Errorf("failed to query lockUser: %w", err)
		}

		newState, err := updater(ctx, state)
		if err != nil {
			return fmt.Errorf("failed to call notification updater: %w", err)
		}

		if _, err := updateUser.ExecContext(ctx,
			newState.DeviceToken, newState.DeviceTokenType, newState.SentNotification,
			userId); err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to update sent notification: %w", err)
	}
	return nil
}

func (s *Storage) GetActiveGames(ctx context.Context) ([]*Game, error) {
	now := s.nowFunc()
	rows, err := s.getActiveGames.QueryContext(ctx, &now)
	if err != nil {
		return nil, fmt.Errorf("failed to query getActiveGames: %w", err)
	}
	defer rows.Close()

	games := make([]*Game, 0)
	for rows.Next() {
		game := Game{}
		if err := rows.Scan(&game.GameId, &game.SpectatorId, &game.CreatedAt, &game.ExpiresAt); err != nil {
			return nil, fmt.Errorf("failed to scan a row getActiveGames: %w", err)
		}
		games = append(games, &game)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate over rows getActiveGames: %w", err)
	}
	if len(games) == 0 {
		return nil, ErrNotFound
	}
	return games, nil
}

func (s *Storage) UpdateGameResults(ctx context.Context, gameId string, results *GameResults) error {
	now := s.nowFunc()
	if _, err := s.updateGameResults.ExecContext(ctx, results, now, gameId); err != nil {
		return fmt.Errorf("failed to update results: %w", err)
	}
	return nil
}

func (s *Storage) UpdateElo(ctx context.Context, updater EloUpdater) error {
	if err := s.withTX(ctx, func(ctx context.Context, tx *sql.Tx) error {
		getGamePlayersAndElo := tx.StmtContext(ctx, s.getGamePlayersAndElo)
		getOldestFinishedGame := tx.StmtContext(ctx, s.getOldestFinishedGame)
		updateGameEloResults := tx.StmtContext(ctx, s.updateGameEloResults)
		updateUserElo := tx.StmtContext(ctx, s.updateUserElo)

		var game Game
		err := getOldestFinishedGame.QueryRowContext(ctx).
			Scan(&game.GameId, &game.SpectatorId, &game.CreatedAt, &game.ExpiresAt, &game.GameResults)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("no finished game found: %w", ErrNotFound)
			}
			return fmt.Errorf("failed to query getOldestFinishedGame: %w", err)
		}

		rows, err := getGamePlayersAndElo.QueryContext(ctx, game.GameId)
		if err != nil {
			return fmt.Errorf("failed to query getGamePlayersAndElo: %w", err)
		}
		defer rows.Close()

		var players []Player
		var stateUsers []EloStateUser
		for rows.Next() {
			p := Player{}
			u := EloStateUser{}
			if err := rows.Scan(&p.UserId, &p.PlayerId, &p.Color, &u.Elo); err != nil {
				return fmt.Errorf("failed to scan a row getGamePlayersAndElo: %w", err)
			}
			u.UserId = p.UserId
			players = append(players, p)
			stateUsers = append(stateUsers, u)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("failed to iterate over rows getActiveGames: %w", err)
		}
		if len(players) == 0 || len(stateUsers) == 0 {
			return fmt.Errorf("no players found for the game: %s", game.GameId)
		}

		game.Players = players
		eloResults, err := updater(ctx, EloUpdateState{
			Game:  game,
			Users: stateUsers,
		})
		if err != nil {
			return fmt.Errorf("failed to update results: %w", err)
		}

		eloResultsUpdate, err := updateGameEloResults.ExecContext(ctx, eloResults, game.GameId)
		if err != nil {
			return fmt.Errorf("failed to exec updateGameEloResults: %w", err)
		}
		eloResultsAffected, err := eloResultsUpdate.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected updateGameEloResults: %w", err)
		}
		if eloResultsAffected != 1 {
			return fmt.Errorf("updateGameEloResults unexpected affected rows: %d", eloResultsAffected)
		}

		for _, u := range eloResults.Changes {
			r, err := updateUserElo.ExecContext(ctx, u.NewElo, u.UserId, u.OldElo)
			if err != nil {
				return fmt.Errorf("failed to updateUserElo: %w", err)
			}
			affected, err := r.RowsAffected()
			if err != nil {
				return fmt.Errorf("failed to get rows affected updateUserEloResults: %w", err)
			}
			if affected != 1 {
				return fmt.Errorf("updateUserElo unexpected affected rows: %d", affected)
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to update elo: %w", err)
	}
	return nil
}

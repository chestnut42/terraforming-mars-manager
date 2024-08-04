CREATE TABLE games (
    id              TEXT NOT NULL CHECK (id != ''),
    spectator_id    TEXT NOT NULL CHECK (id != ''),
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL,
    expires_at      TIMESTAMP WITH TIME ZONE NOT NULL,

    PRIMARY KEY(id)
);

CREATE TABLE game_players (
    game_id         TEXT NOT NULL,
    user_id         TEXT NOT NULL,
    player_id       TEXT NOT NULL CHECK (player_id != ''),
    color           TEXT NOT NULL CHECK (color != ''),

    CONSTRAINT fk_games_id FOREIGN KEY (game_id) REFERENCES games(id),
    CONSTRAINT fk_users_id FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT uniq_game_id_user_id UNIQUE (game_id, user_id),
    CONSTRAINT uniq_game_id_color UNIQUE (game_id, color)
);

CREATE UNIQUE INDEX idx_uniq_spectator_id ON games(spectator_id);
CREATE UNIQUE INDEX idx_uniq_player_id ON game_players(player_id);

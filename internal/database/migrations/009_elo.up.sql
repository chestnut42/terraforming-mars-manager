ALTER TABLE manager_users
    ADD COLUMN elo BIGINT NOT NULL default 1000;

ALTER TABLE manager_games
    ADD COLUMN elo_results JSONB;

ALTER TABLE manager_game_players
    ADD COLUMN elo_change BIGINT;

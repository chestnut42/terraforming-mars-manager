WITH target_users AS (
  SELECT id AS user_id, nickname
  FROM manager_users
  WHERE nickname IN ('Siarhei ', 'Andy', 'Max', 'Maximus ', 'Belka', 'iDrew', 'Truemakar')
),
filtered_games AS (
  SELECT *
  FROM manager_games
  WHERE results IS NOT NULL
),
games_players_named AS (
  SELECT gp.game_id, gp.player_id, tu.nickname
  FROM manager_game_players gp
  JOIN target_users tu ON gp.user_id = tu.user_id
  WHERE gp.game_id IN (SELECT id FROM filtered_games)
),
pivoted AS (
  SELECT
    game_id,
    MAX(CASE WHEN nickname = 'Siarhei ' THEN player_id END) AS siarhei,
    MAX(CASE WHEN nickname = 'Andy' THEN player_id END) AS andy,
    MAX(CASE WHEN nickname = 'Max' THEN player_id END) AS maxx,
    MAX(CASE WHEN nickname = 'Maximus ' THEN player_id END) AS maximus,
    MAX(CASE WHEN nickname = 'Belka' THEN player_id END) AS belka,
    MAX(CASE WHEN nickname = 'iDrew' THEN player_id END) AS idrew,
    MAX(CASE WHEN nickname = 'Truemakar' THEN player_id END) AS truemakar
  FROM games_players_named
  GROUP BY game_id
)
SELECT
  p.siarhei,
  p.andy,
  p.maxx,
  p.maximus,
  p.belka,
  p.idrew,
  p.truemakar,
  g.*
FROM filtered_games g
LEFT JOIN pivoted p ON g.id = p.game_id
ORDER BY g.created_at ASC

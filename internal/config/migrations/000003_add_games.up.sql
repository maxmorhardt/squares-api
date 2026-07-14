CREATE TABLE IF NOT EXISTS games (
    id          uuid PRIMARY KEY,
    espn_id     text NOT NULL,
    home_team   text NOT NULL DEFAULT '',
    away_team   text NOT NULL DEFAULT '',
    home_abbr   text NOT NULL DEFAULT '',
    away_abbr   text NOT NULL DEFAULT '',
    game_time   timestamptz NOT NULL,
    week        int NOT NULL DEFAULT 0,
    season      int NOT NULL DEFAULT 0,
    season_type int NOT NULL DEFAULT 2,
    status      text NOT NULL DEFAULT 'scheduled',
    period      int NOT NULL DEFAULT 0,
    home_score  int NOT NULL DEFAULT 0,
    away_score  int NOT NULL DEFAULT 0,
    created_at  timestamptz,
    updated_at  timestamptz
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_games_espn_id ON games (espn_id);
CREATE INDEX IF NOT EXISTS idx_games_status ON games (status);
CREATE INDEX IF NOT EXISTS idx_games_game_time ON games (game_time);

CREATE TABLE IF NOT EXISTS game_scores (
    id         uuid PRIMARY KEY,
    game_id    uuid NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    quarter    int NOT NULL,
    home_score int NOT NULL DEFAULT 0,
    away_score int NOT NULL DEFAULT 0,
    created_at timestamptz,
    updated_at timestamptz,
    UNIQUE (game_id, quarter)
);
CREATE INDEX IF NOT EXISTS idx_game_scores_game_id ON game_scores (game_id);

ALTER TABLE contests ADD COLUMN IF NOT EXISTS game_id uuid REFERENCES games (id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_contests_game_id ON contests (game_id);

-- collapse duplicates written before the constraint existed
DELETE FROM quarter_results
WHERE id IN (
    SELECT id
    FROM (
        SELECT id,
               ROW_NUMBER() OVER (
                   PARTITION BY contest_id, quarter
                   ORDER BY created_at NULLS LAST, id
               ) AS rn
        FROM quarter_results
    ) ranked
    WHERE ranked.rn > 1
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_quarter_results_contest_quarter
    ON quarter_results (contest_id, quarter);

-- game-linked contests only published their quarters, so recreate the rows the leaderboard reads
INSERT INTO quarter_results (
    id, contest_id, quarter, home_team_score, away_team_score,
    winner_row, winner_col, winner, winner_name,
    created_at, updated_at, created_by, updated_by
)
SELECT gen_random_uuid(), c.id, gs.quarter, gs.home_score, gs.away_score,
       y.ord - 1, x.ord - 1, COALESCE(s.owner, ''), COALESCE(s.owner_name, ''),
       now(), now(), 'system', 'system'
FROM contests c
JOIN game_scores gs ON gs.game_id = c.game_id
JOIN LATERAL (
    SELECT ord
    FROM jsonb_array_elements_text(c.x_labels) WITH ORDINALITY AS t (val, ord)
    WHERE t.val::int = gs.home_score % 10
    LIMIT 1
) x ON TRUE
JOIN LATERAL (
    SELECT ord
    FROM jsonb_array_elements_text(c.y_labels) WITH ORDINALITY AS t (val, ord)
    WHERE t.val::int = gs.away_score % 10
    LIMIT 1
) y ON TRUE
LEFT JOIN squares s ON s.contest_id = c.id AND s."row" = y.ord - 1 AND s.col = x.ord - 1
WHERE c.game_id IS NOT NULL
  AND c.status <> 'DELETED'
ON CONFLICT (contest_id, quarter) DO NOTHING;

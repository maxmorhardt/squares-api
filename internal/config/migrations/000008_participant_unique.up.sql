-- collapse joins duplicated before the constraint existed, keeping the earliest row per user
DELETE FROM contest_participants
WHERE id IN (
    SELECT id
    FROM (
        SELECT id,
               ROW_NUMBER() OVER (
                   PARTITION BY contest_id, user_id
                   ORDER BY joined_at NULLS LAST, created_at NULLS LAST, id
               ) AS rn
        FROM contest_participants
    ) ranked
    WHERE ranked.rn > 1
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_contest_participants_contest_user
    ON contest_participants (contest_id, user_id);

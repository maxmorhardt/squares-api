CREATE TABLE IF NOT EXISTS deleted_accounts (
    email      text PRIMARY KEY,
    deleted_at timestamptz NOT NULL
);

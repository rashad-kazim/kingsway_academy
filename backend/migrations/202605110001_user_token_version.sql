-- +goose Up
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS token_version integer NOT NULL DEFAULT 1;

UPDATE users
SET token_version = 1
WHERE token_version < 1;

-- +goose Down
ALTER TABLE users
  DROP COLUMN IF EXISTS token_version;

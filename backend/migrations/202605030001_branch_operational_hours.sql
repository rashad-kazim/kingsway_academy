-- +goose Up
ALTER TABLE branches
  ADD COLUMN IF NOT EXISTS opening_time text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS closing_time text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE branches
  DROP COLUMN IF EXISTS closing_time,
  DROP COLUMN IF EXISTS opening_time;

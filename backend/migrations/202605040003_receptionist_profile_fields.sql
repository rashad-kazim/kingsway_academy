-- +goose Up
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS last_login_at timestamptz;

ALTER TABLE staff_profiles
  ADD COLUMN IF NOT EXISTS gender text NOT NULL DEFAULT '' CHECK (gender IN ('', 'male', 'female', 'other')),
  ADD COLUMN IF NOT EXISTS address text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS hired_at date;

-- +goose Down
ALTER TABLE staff_profiles
  DROP COLUMN IF EXISTS hired_at,
  DROP COLUMN IF EXISTS address,
  DROP COLUMN IF EXISTS gender;

ALTER TABLE users
  DROP COLUMN IF EXISTS last_login_at;

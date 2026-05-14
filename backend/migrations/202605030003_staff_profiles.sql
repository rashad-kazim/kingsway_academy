-- +goose Up
CREATE TABLE IF NOT EXISTS staff_profiles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches(id),
    user_id uuid NOT NULL UNIQUE REFERENCES users(id),
    role text NOT NULL CHECK (role IN ('receptionist')),
    birth_date date,
    phone text NOT NULL DEFAULT '',
    salary_amount_azn integer NOT NULL DEFAULT 0 CHECK (salary_amount_azn >= 0),
    profile_photo_file_id uuid REFERENCES files(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS staff_profiles_branch_role_idx
    ON staff_profiles(branch_id, role);

ALTER TABLE files
  DROP CONSTRAINT IF EXISTS files_owner_type_check;

ALTER TABLE files
  ADD CONSTRAINT files_owner_type_check
  CHECK (owner_type IN ('student', 'teacher', 'class', 'exam', 'payment', 'event', 'branch', 'staff'));

-- +goose Down
ALTER TABLE files
  DROP CONSTRAINT IF EXISTS files_owner_type_check;

ALTER TABLE files
  ADD CONSTRAINT files_owner_type_check
  CHECK (owner_type IN ('student', 'teacher', 'class', 'exam', 'payment', 'event', 'branch'));

DROP TABLE IF EXISTS staff_profiles;

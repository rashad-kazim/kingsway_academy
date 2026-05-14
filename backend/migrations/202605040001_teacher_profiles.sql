-- +goose Up
ALTER TABLE teachers
    ADD COLUMN birth_date date,
    ADD COLUMN gender text CHECK (gender IS NULL OR gender IN ('male', 'female', 'other')),
    ADD COLUMN phone text NOT NULL DEFAULT '',
    ADD COLUMN address text NOT NULL DEFAULT '',
    ADD COLUMN profile_photo_file_id uuid REFERENCES files(id);

CREATE INDEX teachers_profile_photo_file_idx
    ON teachers(profile_photo_file_id)
    WHERE profile_photo_file_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS teachers_profile_photo_file_idx;

ALTER TABLE teachers
    DROP COLUMN IF EXISTS profile_photo_file_id,
    DROP COLUMN IF EXISTS address,
    DROP COLUMN IF EXISTS phone,
    DROP COLUMN IF EXISTS gender,
    DROP COLUMN IF EXISTS birth_date;

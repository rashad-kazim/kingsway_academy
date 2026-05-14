-- +goose Up
ALTER TABLE students
    ADD COLUMN birth_date date,
    ADD COLUMN gender text CHECK (gender IS NULL OR gender IN ('male', 'female', 'other')),
    ADD COLUMN phone text NOT NULL DEFAULT '',
    ADD COLUMN address text NOT NULL DEFAULT '',
    ADD COLUMN profile_photo_file_id uuid REFERENCES files(id);

CREATE INDEX students_profile_photo_file_idx
    ON students(profile_photo_file_id)
    WHERE profile_photo_file_id IS NOT NULL;

CREATE TABLE student_parent_contacts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    student_id uuid NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    relation text NOT NULL CHECK (relation IN ('father', 'mother', 'sister', 'brother', 'other')),
    name text NOT NULL,
    phones text[] NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX student_parent_contacts_student_idx
    ON student_parent_contacts(student_id);

CREATE TABLE student_course_registrations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    student_id uuid NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    course_id uuid NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    teacher_id uuid REFERENCES teachers(id) ON DELETE SET NULL,
    monthly_amount_cents bigint NOT NULL CHECK (monthly_amount_cents >= 0),
    start_date date NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (student_id, course_id)
);

CREATE INDEX student_course_registrations_lookup_idx
    ON student_course_registrations(branch_id, student_id, course_id);

-- +goose Down
DROP TABLE IF EXISTS student_course_registrations;
DROP TABLE IF EXISTS student_parent_contacts;

DROP INDEX IF EXISTS students_profile_photo_file_idx;

ALTER TABLE students
    DROP COLUMN IF EXISTS profile_photo_file_id,
    DROP COLUMN IF EXISTS address,
    DROP COLUMN IF EXISTS phone,
    DROP COLUMN IF EXISTS gender,
    DROP COLUMN IF EXISTS birth_date;

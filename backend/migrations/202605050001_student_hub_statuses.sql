-- +goose Up
ALTER TABLE students DROP CONSTRAINT IF EXISTS students_status_check;
ALTER TABLE students
    ADD CONSTRAINT students_status_check
    CHECK (status IN ('active', 'left', 'graduated'));

-- +goose Down
UPDATE students SET status = 'left' WHERE status = 'graduated';
ALTER TABLE students DROP CONSTRAINT IF EXISTS students_status_check;
ALTER TABLE students
    ADD CONSTRAINT students_status_check
    CHECK (status IN ('active', 'left'));

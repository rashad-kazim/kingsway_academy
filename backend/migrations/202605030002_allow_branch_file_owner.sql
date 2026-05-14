-- +goose Up
ALTER TABLE files
  DROP CONSTRAINT IF EXISTS files_owner_type_check;

ALTER TABLE files
  ADD CONSTRAINT files_owner_type_check
  CHECK (owner_type IN ('student', 'teacher', 'class', 'exam', 'payment', 'event', 'branch'));

-- +goose Down
ALTER TABLE files
  DROP CONSTRAINT IF EXISTS files_owner_type_check;

ALTER TABLE files
  ADD CONSTRAINT files_owner_type_check
  CHECK (owner_type IN ('student', 'teacher', 'class', 'exam', 'payment', 'event'));

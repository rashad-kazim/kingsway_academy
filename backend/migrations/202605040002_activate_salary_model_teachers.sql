-- +goose Up
UPDATE teachers
SET status = 'active',
    updated_at = now()
WHERE status = 'pending_owner_approval'
  AND EXISTS (
    SELECT 1
    FROM salary_models
    WHERE salary_models.teacher_id = teachers.id
  );

-- +goose Down
-- Existing active teachers are intentionally not downgraded.

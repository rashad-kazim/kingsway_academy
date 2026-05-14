-- +goose Up
ALTER TABLE audit_logs
    ADD COLUMN IF NOT EXISTS before_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS after_json jsonb NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE files
    ADD COLUMN IF NOT EXISTS policy text;

UPDATE files
SET policy = CASE
    WHEN category = 'special' THEN 'special'
    WHEN purpose = 'profile_photo' THEN 'standard-ui'
    ELSE 'standard-downloadable'
END
WHERE policy IS NULL OR policy = '';

ALTER TABLE files
    ALTER COLUMN policy SET NOT NULL,
    ALTER COLUMN policy SET DEFAULT 'standard-downloadable';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'files_policy_check'
    ) THEN
        ALTER TABLE files
            ADD CONSTRAINT files_policy_check
            CHECK (policy IN ('standard-ui', 'standard-downloadable', 'special'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS files_policy_idx
    ON files (policy);

-- +goose Down
DROP INDEX IF EXISTS files_policy_idx;

ALTER TABLE files
    DROP CONSTRAINT IF EXISTS files_policy_check,
    DROP COLUMN IF EXISTS policy;

ALTER TABLE audit_logs
    DROP COLUMN IF EXISTS after_json,
    DROP COLUMN IF EXISTS before_json;

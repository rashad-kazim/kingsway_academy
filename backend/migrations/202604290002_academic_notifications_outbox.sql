-- +goose Up
CREATE TABLE IF NOT EXISTS assignments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches(id),
    class_id uuid NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    title text NOT NULL,
    description text NOT NULL DEFAULT '',
    due_at timestamptz,
    material_file_id uuid REFERENCES files(id),
    created_by_user_id uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS assignments_branch_class_due_idx
    ON assignments(branch_id, class_id, due_at);

ALTER TABLE notifications
    ADD COLUMN IF NOT EXISTS dedupe_key text;

CREATE UNIQUE INDEX IF NOT EXISTS notifications_dedupe_key_idx
    ON notifications(dedupe_key)
    WHERE dedupe_key IS NOT NULL;

CREATE TABLE IF NOT EXISTS outbox_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    topic text NOT NULL,
    payload jsonb NOT NULL,
    status text NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'publishing', 'published', 'failed')),
    attempts integer NOT NULL DEFAULT 0,
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    locked_at timestamptz,
    last_error text,
    published_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS outbox_events_pending_idx
    ON outbox_events(status, next_attempt_at, created_at)
    WHERE status IN ('pending', 'publishing');

-- +goose Down
DROP TABLE IF EXISTS outbox_events;
DROP INDEX IF EXISTS notifications_dedupe_key_idx;
ALTER TABLE notifications DROP COLUMN IF EXISTS dedupe_key;
DROP TABLE IF EXISTS assignments;

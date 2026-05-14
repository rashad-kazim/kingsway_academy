-- +goose Up
CREATE TABLE IF NOT EXISTS audit_logs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    actor_role text NOT NULL CHECK (actor_role IN ('owner', 'receptionist', 'teacher', 'student')),
    actor_branch_id uuid REFERENCES branches(id) ON DELETE SET NULL,
    action text NOT NULL,
    entity_type text NOT NULL,
    entity_id text NOT NULL DEFAULT '',
    entity_branch_id uuid REFERENCES branches(id) ON DELETE SET NULL,
    request_id text NOT NULL DEFAULT '',
    idempotency_key text NOT NULL DEFAULT '',
    before_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    after_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS audit_logs_created_at_idx
    ON audit_logs (created_at DESC);

CREATE INDEX IF NOT EXISTS audit_logs_actor_created_at_idx
    ON audit_logs (actor_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS audit_logs_entity_created_at_idx
    ON audit_logs (entity_type, entity_id, created_at DESC);

CREATE INDEX IF NOT EXISTS audit_logs_branch_created_at_idx
    ON audit_logs (entity_branch_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS audit_logs;

-- +goose Up
CREATE TABLE IF NOT EXISTS idempotency_keys (
    actor_user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    method text NOT NULL,
    path text NOT NULL,
    key text NOT NULL,
    request_hash char(64) NOT NULL,
    status text NOT NULL CHECK (status IN ('pending', 'completed')),
    response_status integer,
    response_body jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL DEFAULT (now() + interval '24 hours'),
    PRIMARY KEY (actor_user_id, method, path, key)
);

CREATE INDEX IF NOT EXISTS idempotency_keys_expires_at_idx
    ON idempotency_keys (expires_at);

-- +goose Down
DROP TABLE IF EXISTS idempotency_keys;

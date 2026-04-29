package store

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/jackc/pgx/v5"

	"kingsway/backend/internal/domain"
)

func insertOutboxTx(ctx context.Context, tx pgx.Tx, topic string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events (topic, payload)
		VALUES ($1, $2::jsonb)
	`, topic, string(body))
	return err
}

func (p *Postgres) ListOutboxEvents(ctx context.Context, status domain.OutboxEventStatus, limit int, offset int) ([]domain.OutboxEvent, int, error) {
	var total int
	if err := p.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM outbox_events
		WHERE ($1::text = '' OR status = $1::text)
	`, string(status)).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	rows, err := p.pool.Query(ctx, `
		SELECT id::text, topic, payload::text, status, attempts, next_attempt_at,
			locked_at, coalesce(last_error, ''), published_at, created_at
		FROM outbox_events
		WHERE ($1::text = '' OR status = $1::text)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, string(status), limit, offset)
	if err != nil {
		return nil, 0, mapPostgresError(err)
	}
	defer rows.Close()

	events := make([]domain.OutboxEvent, 0)
	for rows.Next() {
		event, err := scanOutboxEvent(rows)
		if err != nil {
			return nil, 0, mapPostgresError(err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	return events, total, nil
}

func (p *Postgres) RetryOutboxEvent(ctx context.Context, id string) (domain.OutboxEvent, error) {
	row := p.pool.QueryRow(ctx, `
		UPDATE outbox_events
		SET status = 'pending',
			attempts = 0,
			next_attempt_at = now(),
			locked_at = null,
			last_error = null,
			published_at = null
		WHERE id = $1
			AND status = 'failed'
		RETURNING id::text, topic, payload::text, status, attempts, next_attempt_at,
			locked_at, coalesce(last_error, ''), published_at, created_at
	`, id)

	event, err := scanOutboxEvent(row)
	if err != nil {
		return domain.OutboxEvent{}, mapPostgresError(err)
	}

	return event, nil
}

func scanOutboxEvent(row scanner) (domain.OutboxEvent, error) {
	var event domain.OutboxEvent
	var payload string
	var lockedAt sql.NullTime
	var publishedAt sql.NullTime
	err := row.Scan(
		&event.ID,
		&event.Topic,
		&payload,
		&event.Status,
		&event.Attempts,
		&event.NextAttemptAt,
		&lockedAt,
		&event.LastError,
		&publishedAt,
		&event.CreatedAt,
	)
	if err != nil {
		return domain.OutboxEvent{}, err
	}
	if err := decodeOutboxPayload(payload, &event); err != nil {
		return domain.OutboxEvent{}, err
	}
	if lockedAt.Valid {
		event.LockedAt = &lockedAt.Time
	}
	if publishedAt.Valid {
		event.PublishedAt = &publishedAt.Time
	}

	return event, nil
}

func decodeOutboxPayload(payload string, event *domain.OutboxEvent) error {
	if payload != "" {
		if err := json.Unmarshal([]byte(payload), &event.Payload); err != nil {
			return err
		}
	}
	if event.Payload == nil {
		event.Payload = map[string]any{}
	}

	return nil
}

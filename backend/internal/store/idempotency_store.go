package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"kingsway/backend/internal/domain"
)

func (p *Postgres) RunIdempotent(ctx context.Context, record domain.IdempotencyRecord, execute func(context.Context) (int, []byte, error)) (domain.IdempotencyRunResult, error) {
	record, err := normalizeIdempotencyRecord(record)
	if err != nil {
		return domain.IdempotencyRunResult{}, err
	}
	if execute == nil {
		return domain.IdempotencyRunResult{}, domain.ErrInvalidInput
	}

	var result domain.IdempotencyRunResult
	err = p.withTx(ctx, func(ctx context.Context, _ pgx.Tx) error {
		var runErr error
		result, runErr = p.runIdempotentInTx(ctx, record, execute)
		return runErr
	})
	if err != nil {
		return domain.IdempotencyRunResult{}, err
	}

	return result, nil
}

func (p *Postgres) runIdempotentInTx(ctx context.Context, record domain.IdempotencyRecord, execute func(context.Context) (int, []byte, error)) (domain.IdempotencyRunResult, error) {
	for attempt := 0; attempt < 2; attempt++ {
		_, _ = p.exec(ctx, `
			DELETE FROM idempotency_keys
			WHERE actor_user_id = $1 AND method = $2 AND path = $3 AND key = $4 AND expires_at < now()
		`, record.ActorUserID, record.Method, record.Path, record.Key)

		inserted, err := scanIdempotencyRecord(p.queryRow(ctx, `
			INSERT INTO idempotency_keys (
				actor_user_id, method, path, key, request_hash, status, expires_at
			)
			VALUES ($1, $2, $3, $4, $5, 'pending', $6)
			ON CONFLICT DO NOTHING
			RETURNING actor_user_id::text, method, path, key, request_hash, status,
				coalesce(response_status, 0), coalesce(response_body::text, ''),
				created_at, updated_at, expires_at
		`, record.ActorUserID, record.Method, record.Path, record.Key, record.RequestHash, record.ExpiresAt))
		if err == nil {
			status, payload, err := execute(ctx)
			if err != nil {
				return domain.IdempotencyRunResult{}, err
			}
			if err := p.completeIdempotency(ctx, inserted.ActorUserID, inserted.Method, inserted.Path, inserted.Key, status, payload); err != nil {
				return domain.IdempotencyRunResult{}, err
			}
			return domain.IdempotencyRunResult{
				ResponseStatus: status,
				ResponseBody:   append([]byte(nil), payload...),
			}, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return domain.IdempotencyRunResult{}, mapPostgresError(err)
		}

		existing, err := scanIdempotencyRecord(p.queryRow(ctx, `
			SELECT actor_user_id::text, method, path, key, request_hash, status,
				coalesce(response_status, 0), coalesce(response_body::text, ''),
				created_at, updated_at, expires_at
			FROM idempotency_keys
			WHERE actor_user_id = $1 AND method = $2 AND path = $3 AND key = $4
			FOR UPDATE
		`, record.ActorUserID, record.Method, record.Path, record.Key))
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return domain.IdempotencyRunResult{}, mapPostgresError(err)
		}
		if existing.RequestHash != record.RequestHash {
			return domain.IdempotencyRunResult{}, domain.ErrConflict
		}
		if existing.Status == domain.IdempotencyStatusCompleted {
			return domain.IdempotencyRunResult{
				Replayed:       true,
				ResponseStatus: existing.ResponseStatus,
				ResponseBody:   append([]byte(nil), existing.ResponseBody...),
			}, nil
		}

		return domain.IdempotencyRunResult{}, domain.ErrConflict
	}

	return domain.IdempotencyRunResult{}, domain.ErrConflict
}

func (p *Postgres) completeIdempotency(ctx context.Context, actorUserID string, method string, path string, key string, responseStatus int, responseBody []byte) error {
	tag, err := p.exec(ctx, `
		UPDATE idempotency_keys
		SET status = 'completed',
			response_status = $5,
			response_body = $6::jsonb,
			updated_at = now()
		WHERE actor_user_id = $1 AND method = $2 AND path = $3 AND key = $4
	`, actorUserID, method, path, key, responseStatus, string(responseBody))
	if err != nil {
		return mapPostgresError(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func normalizeIdempotencyRecord(record domain.IdempotencyRecord) (domain.IdempotencyRecord, error) {
	if record.ActorUserID == "" || record.Method == "" || record.Path == "" || record.Key == "" || record.RequestHash == "" {
		return domain.IdempotencyRecord{}, domain.ErrInvalidInput
	}
	if record.ExpiresAt.IsZero() {
		record.ExpiresAt = time.Now().UTC().Add(24 * time.Hour)
	}

	return record, nil
}

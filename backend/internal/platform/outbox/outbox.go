package outbox

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Publisher struct {
	pool *pgxpool.Pool
}

func NewPublisher(pool *pgxpool.Pool) *Publisher {
	return &Publisher{pool: pool}
}

func (p *Publisher) Publish(ctx context.Context, topic string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = p.pool.Exec(ctx, `
		INSERT INTO outbox_events (topic, payload)
		VALUES ($1, $2::jsonb)
	`, topic, string(body))
	return err
}

type RawPublisher interface {
	PublishJSON(ctx context.Context, topic string, body []byte) error
}

type Dispatcher struct {
	pool      *pgxpool.Pool
	publisher RawPublisher
}

type DispatcherOptions struct {
	Interval    time.Duration
	BatchSize   int
	MaxAttempts int
	Logger      *zap.Logger
}

type event struct {
	ID       string
	Topic    string
	Payload  []byte
	Attempts int
}

func NewDispatcher(pool *pgxpool.Pool, publisher RawPublisher) *Dispatcher {
	return &Dispatcher{pool: pool, publisher: publisher}
}

func (d *Dispatcher) Start(ctx context.Context, options DispatcherOptions) <-chan struct{} {
	done := make(chan struct{})
	interval := options.Interval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	batchSize := options.BatchSize
	if batchSize <= 0 || batchSize > 500 {
		batchSize = 50
	}
	maxAttempts := options.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 10
	}
	log := options.Logger
	if log == nil {
		log = zap.NewNop()
	}

	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Info("outbox dispatcher stopped")
				return
			default:
			}

			dispatched, err := d.DispatchOnce(ctx, batchSize, maxAttempts)
			if err != nil {
				log.Error("outbox dispatch failed", zap.Error(err))
			}
			if dispatched == 0 {
				select {
				case <-ctx.Done():
					log.Info("outbox dispatcher stopped")
					return
				case <-ticker.C:
				}
			}
		}
	}()

	return done
}

func (d *Dispatcher) DispatchOnce(ctx context.Context, limit int, maxAttempts int) (int, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	if maxAttempts <= 0 {
		maxAttempts = 10
	}

	events, err := d.claim(ctx, limit)
	if err != nil {
		return 0, err
	}

	for _, evt := range events {
		if err := d.publisher.PublishJSON(ctx, evt.Topic, evt.Payload); err != nil {
			if markErr := d.markFailed(ctx, evt, err, maxAttempts); markErr != nil {
				return len(events), markErr
			}
			continue
		}
		if err := d.markPublished(ctx, evt.ID); err != nil {
			return len(events), err
		}
	}

	return len(events), nil
}

func (d *Dispatcher) claim(ctx context.Context, limit int) ([]event, error) {
	rows, err := d.pool.Query(ctx, `
		UPDATE outbox_events
		SET status = 'publishing',
			locked_at = now(),
			attempts = attempts + 1
		WHERE id IN (
			SELECT id
			FROM outbox_events
			WHERE (
				status = 'pending'
				AND next_attempt_at <= now()
			) OR (
				status = 'publishing'
				AND locked_at < now() - interval '5 minutes'
			)
			ORDER BY created_at
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id::text, topic, payload::text, attempts
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]event, 0)
	for rows.Next() {
		var evt event
		if err := rows.Scan(&evt.ID, &evt.Topic, &evt.Payload, &evt.Attempts); err != nil {
			return nil, err
		}
		events = append(events, evt)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func (d *Dispatcher) markPublished(ctx context.Context, id string) error {
	_, err := d.pool.Exec(ctx, `
		UPDATE outbox_events
		SET status = 'published',
			published_at = now(),
			last_error = null
		WHERE id = $1
	`, id)
	return err
}

func (d *Dispatcher) markFailed(ctx context.Context, evt event, publishErr error, maxAttempts int) error {
	status := "pending"
	nextAttempt := time.Now().UTC().Add(backoff(evt.Attempts))
	if evt.Attempts >= maxAttempts {
		status = "failed"
	}

	_, err := d.pool.Exec(ctx, `
		UPDATE outbox_events
		SET status = $2,
			next_attempt_at = $3,
			last_error = $4
		WHERE id = $1
	`, evt.ID, status, nextAttempt, publishErr.Error())
	return err
}

func backoff(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	seconds := 1 << min(attempts, 8)
	return time.Duration(seconds) * time.Second
}

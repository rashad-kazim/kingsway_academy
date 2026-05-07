package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"kingsway/backend/internal/domain"
)

func (p *Postgres) CreateNotification(ctx context.Context, notification domain.Notification) (domain.Notification, error) {
	if notification.RecipientUserID == "" || strings.TrimSpace(notification.Type) == "" {
		return domain.Notification{}, domain.ErrInvalidInput
	}
	body, err := json.Marshal(notification.Payload)
	if err != nil {
		return domain.Notification{}, err
	}

	row := p.pool.QueryRow(ctx, `
		INSERT INTO notifications (branch_id, recipient_user_id, type, payload, dedupe_key)
		VALUES (nullif($1, '')::uuid, $2, $3, $4::jsonb, nullif($5, ''))
		ON CONFLICT (dedupe_key) WHERE dedupe_key IS NOT NULL DO UPDATE SET dedupe_key = notifications.dedupe_key
		RETURNING id::text, coalesce(branch_id::text, ''), recipient_user_id::text, type, payload::text,
			coalesce(dedupe_key, ''), read_at, created_at
	`, notification.BranchID, notification.RecipientUserID, strings.TrimSpace(notification.Type), string(body), notification.DedupeKey)

	created, err := scanNotification(row)
	if err != nil {
		return domain.Notification{}, mapPostgresError(err)
	}

	return created, nil
}

func (p *Postgres) ListNotifications(ctx context.Context, recipientUserID string, unreadOnly bool) ([]domain.Notification, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id::text, coalesce(branch_id::text, ''), recipient_user_id::text, type, payload::text,
			coalesce(dedupe_key, ''), read_at, created_at
		FROM notifications
		WHERE recipient_user_id = $1
			AND ($2 = false OR read_at IS NULL)
		ORDER BY created_at DESC
		LIMIT 200
	`, recipientUserID, unreadOnly)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	notifications := make([]domain.Notification, 0)
	for rows.Next() {
		notification, err := scanNotification(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		notifications = append(notifications, notification)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return notifications, nil
}

func (p *Postgres) MarkNotificationRead(ctx context.Context, id string, recipientUserID string, readAt time.Time) (domain.Notification, error) {
	row := p.pool.QueryRow(ctx, `
		UPDATE notifications
		SET read_at = coalesce(read_at, $3)
		WHERE id = $1 AND recipient_user_id = $2
		RETURNING id::text, coalesce(branch_id::text, ''), recipient_user_id::text, type, payload::text,
			coalesce(dedupe_key, ''), read_at, created_at
	`, id, recipientUserID, readAt)

	notification, err := scanNotification(row)
	if err != nil {
		return domain.Notification{}, mapPostgresError(err)
	}

	return notification, nil
}

func (p *Postgres) ListUsersByBranchAndRoles(ctx context.Context, branchID string, roles []domain.Role) ([]domain.User, error) {
	roleValues := make([]string, 0, len(roles))
	for _, role := range roles {
		if role.IsValid() {
			roleValues = append(roleValues, string(role))
		}
	}
	if len(roleValues) == 0 {
		return nil, domain.ErrInvalidInput
	}

	rows, err := p.pool.Query(ctx, `
		SELECT id::text, coalesce(branch_id::text, ''), role, email, password_hash, first_name, last_name, is_active, coalesce(last_login_at::text, ''), created_at, updated_at
		FROM users
		WHERE is_active = true
			AND role = ANY($2::text[])
			AND (
				role = 'owner'
				OR branch_id = $1::uuid
			)
		ORDER BY role, first_name, last_name
	`, branchID, roleValues)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	users := make([]domain.User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return users, nil
}

func (p *Postgres) ListPendingPaymentReminders(ctx context.Context, now time.Time, horizon time.Time, limit int) ([]domain.Payment, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id::text, branch_id::text, student_id::text, amount_cents, currency, due_date,
			paid_at, status, coalesce(receipt_file_id::text, ''), created_by_user_id::text, created_at, updated_at
		FROM payments
		WHERE status IN ('pending', 'overdue')
			AND paid_at IS NULL
			AND due_date >= ($1::timestamptz - interval '30 days')::date
			AND due_date <= $2
		ORDER BY due_date
		LIMIT $3
	`, now, horizon, limit)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	payments := make([]domain.Payment, 0)
	for rows.Next() {
		payment, err := scanPayment(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		payments = append(payments, payment)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return payments, nil
}

func (p *Postgres) ListFilesNearRetention(ctx context.Context, now time.Time, horizon time.Time, limit int) ([]domain.FileObject, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id::text, branch_id::text, uploader_user_id::text, owner_type, owner_id::text,
			category, purpose, original_filename, mime_type, original_size_bytes, stored_size_bytes,
			original_sha256, storage_bucket, storage_key, retention_until, deleted_at, created_at
		FROM files
		WHERE deleted_at IS NULL
			AND retention_until IS NOT NULL
			AND retention_until > $1
			AND retention_until <= $2
		ORDER BY retention_until
		LIMIT $3
	`, now, horizon, limit)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	files := make([]domain.FileObject, 0)
	for rows.Next() {
		file, err := scanFileObject(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return files, nil
}

func scanNotification(row scanner) (domain.Notification, error) {
	var notification domain.Notification
	var payload string
	var readAt sql.NullTime
	err := row.Scan(
		&notification.ID,
		&notification.BranchID,
		&notification.RecipientUserID,
		&notification.Type,
		&payload,
		&notification.DedupeKey,
		&readAt,
		&notification.CreatedAt,
	)
	if err != nil {
		return domain.Notification{}, err
	}
	if payload != "" {
		if err := json.Unmarshal([]byte(payload), &notification.Payload); err != nil {
			return domain.Notification{}, err
		}
	}
	if notification.Payload == nil {
		notification.Payload = map[string]any{}
	}
	if readAt.Valid {
		notification.ReadAt = &readAt.Time
	}

	return notification, nil
}

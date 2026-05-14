package store

import (
	"context"
	"encoding/json"
	"strings"

	"kingsway/backend/internal/domain"
)

func (p *Postgres) CreateAuditLog(ctx context.Context, log domain.AuditLog) (domain.AuditLog, error) {
	if log.ActorUserID == "" || !log.ActorRole.IsValid() || log.Action == "" || log.EntityType == "" {
		return domain.AuditLog{}, domain.ErrInvalidInput
	}
	metadata, err := marshalAuditMap(log.Metadata)
	if err != nil {
		return domain.AuditLog{}, domain.ErrInvalidInput
	}
	beforeJSON, err := marshalAuditMap(log.BeforeJSON)
	if err != nil {
		return domain.AuditLog{}, domain.ErrInvalidInput
	}
	afterJSON, err := marshalAuditMap(log.AfterJSON)
	if err != nil {
		return domain.AuditLog{}, domain.ErrInvalidInput
	}

	row := p.queryRow(ctx, `
		INSERT INTO audit_logs (
			actor_user_id, actor_role, actor_branch_id, action, entity_type, entity_id,
			entity_branch_id, request_id, idempotency_key, before_json, after_json, metadata
		)
		VALUES (
			$1, $2, nullif($3, '')::uuid, $4, $5, $6,
			nullif($7, '')::uuid, $8, $9, $10::jsonb, $11::jsonb, $12::jsonb
		)
		RETURNING id::text, coalesce(actor_user_id::text, ''), actor_role,
			coalesce(actor_branch_id::text, ''), action, entity_type, entity_id,
			coalesce(entity_branch_id::text, ''), request_id, idempotency_key,
			before_json, after_json, metadata, created_at
	`, log.ActorUserID, log.ActorRole, strings.TrimSpace(log.ActorBranchID), log.Action, log.EntityType, strings.TrimSpace(log.EntityID), strings.TrimSpace(log.EntityBranchID), strings.TrimSpace(log.RequestID), strings.TrimSpace(log.IdempotencyKey), string(beforeJSON), string(afterJSON), string(metadata))

	created, err := scanAuditLog(row)
	if err != nil {
		return domain.AuditLog{}, mapPostgresError(err)
	}

	return created, nil
}

func marshalAuditMap(value map[string]any) ([]byte, error) {
	if value == nil {
		value = map[string]any{}
	}
	return json.Marshal(value)
}

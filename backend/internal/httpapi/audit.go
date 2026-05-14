package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"kingsway/backend/internal/domain"
)

type AuditStore interface {
	CreateAuditLog(ctx context.Context, log domain.AuditLog) (domain.AuditLog, error)
}

type auditDetailsKey struct{}

type auditDetails struct {
	Action         string
	EntityType     string
	EntityID       string
	EntityBranchID string
	BeforeJSON     map[string]any
	AfterJSON      map[string]any
}

func (s *Server) recordAudit(r *http.Request, principal domain.Principal, status int, responseBody []byte) {
	if s.audit == nil || !shouldAudit(r.Method, r.URL.Path, status) {
		return
	}

	entityType, entityID, entityBranchID := auditEntity(r.Method, r.URL.Path, principal)
	details := auditDetailsFromContext(r.Context())
	if details.EntityType != "" {
		entityType = details.EntityType
	}
	if details.EntityID != "" {
		entityID = details.EntityID
	}
	if details.EntityBranchID != "" {
		entityBranchID = details.EntityBranchID
	}
	if entityType == "" {
		return
	}
	action := auditAction(r.Method, r.URL.Path)
	if details.Action != "" {
		action = details.Action
	}
	afterJSON := details.AfterJSON
	if afterJSON == nil {
		afterJSON = auditMapFromBody(responseBody)
	}

	_, err := s.audit.CreateAuditLog(r.Context(), domain.AuditLog{
		ActorUserID:    principal.UserID,
		ActorRole:      principal.Role,
		ActorBranchID:  principal.BranchID,
		Action:         action,
		EntityType:     entityType,
		EntityID:       entityID,
		EntityBranchID: entityBranchID,
		RequestID:      requestIDFromContext(r.Context()),
		IdempotencyKey: safeHeaderFingerprint(r.Header.Get(idempotencyKeyHeader)),
		BeforeJSON:     details.BeforeJSON,
		AfterJSON:      afterJSON,
		Metadata: map[string]any{
			"method": r.Method,
			"path":   safeLogPath(r.URL.Path),
			"status": status,
		},
	})
	if err != nil && s.logger != nil {
		s.logger.Warn("audit log write failed",
			zap.String("request_id", requestIDFromContext(r.Context())),
			zap.String("method", r.Method),
			zap.String("path", safeLogPath(r.URL.Path)),
			zap.Error(err),
		)
	}
}

func setAuditBefore(ctx context.Context, value any) {
	details := auditDetailsFromContext(ctx)
	details.BeforeJSON = auditMapFromValue(value)
}

func setAuditEntity(ctx context.Context, entityType string, entityID string, branchID string) {
	details := auditDetailsFromContext(ctx)
	details.EntityType = strings.TrimSpace(entityType)
	details.EntityID = strings.TrimSpace(entityID)
	details.EntityBranchID = strings.TrimSpace(branchID)
}

func auditDetailsFromContext(ctx context.Context) *auditDetails {
	details, _ := ctx.Value(auditDetailsKey{}).(*auditDetails)
	if details == nil {
		return &auditDetails{}
	}
	return details
}

func auditMapFromBody(body []byte) map[string]any {
	if len(body) == 0 {
		return nil
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return nil
	}
	return auditMapFromValue(value)
}

func auditMapFromValue(value any) map[string]any {
	if value == nil {
		return nil
	}
	if mapped, ok := value.(map[string]any); ok {
		return sanitizeAuditMap(mapped)
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var mapped map[string]any
	if err := json.Unmarshal(raw, &mapped); err != nil {
		return map[string]any{"value": string(raw)}
	}
	return sanitizeAuditMap(mapped)
}

func sanitizeAuditMap(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	sanitized := make(map[string]any, len(value))
	for key, item := range value {
		if isSensitiveLogKey(key) {
			sanitized[key] = "[REDACTED]"
			continue
		}
		sanitized[key] = sanitizeAuditValue(item)
	}
	return sanitized
}

func sanitizeAuditValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return sanitizeAuditMap(typed)
	case []any:
		next := make([]any, 0, len(typed))
		for _, item := range typed {
			next = append(next, sanitizeAuditValue(item))
		}
		return next
	default:
		return value
	}
}

func isSensitiveLogKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(key), "-", "_"))
	switch normalized {
	case "password",
		"password_hash",
		"token",
		"access_token",
		"refresh_token",
		"jwt",
		"authorization",
		"cookie",
		"secret",
		"client_secret",
		"api_key",
		"idempotency_key",
		"email",
		"phone",
		"phones",
		"parent_phone",
		"parent_phones",
		"address",
		"birth_date",
		"date_of_birth",
		"fin",
		"fin_code",
		"first_name",
		"last_name",
		"parent_name":
		return true
	default:
		return false
	}
}

func safeHeaderFingerprint(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])[:16]
}

func safeLogPath(path string) string {
	parts := splitPath(path)
	if len(parts) == 0 {
		return "/"
	}
	for i, part := range parts {
		normalized := strings.ToLower(strings.TrimSpace(part))
		if normalized == "by-fin" && i+1 < len(parts) {
			parts[i+1] = ":fin"
			continue
		}
		if isSensitivePathSegment(normalized) && i+1 < len(parts) {
			parts[i+1] = ":redacted"
		}
	}
	return "/" + strings.Join(parts, "/")
}

func isSensitivePathSegment(segment string) bool {
	switch segment {
	case "token", "password", "secret", "authorization":
		return true
	default:
		return false
	}
}

func shouldAudit(method string, path string, status int) bool {
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return false
	}
	if !strings.HasPrefix(path, "/v1/") {
		return false
	}
	switch method {
	case http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete:
		return true
	default:
		return false
	}
}

func auditEntity(method string, path string, principal domain.Principal) (string, string, string) {
	parts := splitPath(strings.TrimPrefix(path, "/v1/"))
	if len(parts) == 0 || parts[0] == "" {
		return "", "", ""
	}

	entityType := parts[0]
	entityID := ""
	if len(parts) > 1 {
		entityID = parts[1]
	}

	entityBranchID := principal.BranchID
	if entityType == "branches" && entityID != "" && method != http.MethodDelete {
		entityBranchID = entityID
	}

	return entityType, entityID, entityBranchID
}

func auditAction(method string, path string) string {
	parts := splitPath(strings.TrimPrefix(path, "/v1/"))
	labels := []string{strings.ToLower(method)}
	for i, part := range parts {
		if part == "" {
			continue
		}
		if i > 0 && looksLikePathID(part) {
			continue
		}
		labels = append(labels, part)
	}

	return strings.Join(labels, ".")
}

func looksLikePathID(value string) bool {
	if len(value) < 8 {
		return false
	}
	for _, ch := range value {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' {
			continue
		}
		return false
	}

	return true
}

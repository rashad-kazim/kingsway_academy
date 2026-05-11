package httpapi

import (
	"context"
	"net/http"
	"strings"

	"kingsway/backend/internal/admin"
	"kingsway/backend/internal/domain"
)

func (s *Server) listOutboxEvents(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	page, err := parsePage(r)
	if err != nil {
		writeError(w, err)
		return
	}

	events, total, err := s.admin.ListOutboxEvents(r.Context(), principal, admin.ListOutboxEventsInput{
		Status: domain.OutboxEventStatus(strings.TrimSpace(r.URL.Query().Get("status"))),
		Limit:  page.Limit,
		Offset: page.Offset,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writePageHeaders(w, page, total)
	writeJSON(w, http.StatusOK, events)
}

func (s *Server) outboxAction(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/admin/outbox/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[1] != "retry" {
		writeError(w, domain.ErrNotFound)
		return
	}

	s.writeIdempotentNoBodyJSON(w, r, principal, func(ctx context.Context) (int, any, error) {
		event, err := s.admin.RetryOutboxEvent(ctx, principal, parts[0])
		return http.StatusOK, event, err
	})
}

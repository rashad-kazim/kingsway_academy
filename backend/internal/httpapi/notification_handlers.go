package httpapi

import (
	"context"
	"net/http"
	"strings"

	"kingsway/backend/internal/domain"
)

func (s *Server) listNotifications(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	unreadOnly := strings.EqualFold(r.URL.Query().Get("unread_only"), "true")
	page, err := parsePage(r)
	if err != nil {
		writeError(w, err)
		return
	}
	notifications, total, err := s.notify.ListNotificationsPage(r.Context(), principal, unreadOnly, page.domainPage())
	if err != nil {
		writeError(w, err)
		return
	}
	writePageJSON(w, page, total, notifications)
}

func (s *Server) notificationAction(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/notifications/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[1] != "read" {
		writeError(w, domain.ErrNotFound)
		return
	}

	s.writeIdempotentNoBodyJSON(w, r, principal, func(ctx context.Context) (int, any, error) {
		notification, err := s.notify.MarkNotificationRead(ctx, principal, parts[0])
		return http.StatusOK, notification, err
	})
}

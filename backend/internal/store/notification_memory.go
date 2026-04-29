package store

import (
	"context"
	"sort"
	"time"

	"kingsway/backend/internal/domain"
)

func (m *Memory) CreateNotification(_ context.Context, notification domain.Notification) (domain.Notification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if notification.RecipientUserID == "" || notification.Type == "" {
		return domain.Notification{}, domain.ErrInvalidInput
	}
	if _, ok := m.users[notification.RecipientUserID]; !ok {
		return domain.Notification{}, domain.ErrNotFound
	}
	if notification.DedupeKey != "" {
		for _, existing := range m.notifications {
			if existing.DedupeKey == notification.DedupeKey {
				return existing, nil
			}
		}
	}
	if notification.Payload == nil {
		notification.Payload = map[string]any{}
	}
	notification.ID = newID()
	notification.CreatedAt = now()
	m.notifications[notification.ID] = notification

	return notification, nil
}

func (m *Memory) ListNotifications(_ context.Context, recipientUserID string, unreadOnly bool) ([]domain.Notification, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	notifications := make([]domain.Notification, 0)
	for _, notification := range m.notifications {
		if notification.RecipientUserID != recipientUserID {
			continue
		}
		if unreadOnly && notification.ReadAt != nil {
			continue
		}
		notifications = append(notifications, notification)
	}
	sort.Slice(notifications, func(i, j int) bool {
		return notifications[i].CreatedAt.After(notifications[j].CreatedAt)
	})

	return notifications, nil
}

func (m *Memory) MarkNotificationRead(_ context.Context, id string, recipientUserID string, readAt time.Time) (domain.Notification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	notification, ok := m.notifications[id]
	if !ok || notification.RecipientUserID != recipientUserID {
		return domain.Notification{}, domain.ErrNotFound
	}
	if notification.ReadAt == nil {
		notification.ReadAt = &readAt
	}
	m.notifications[id] = notification

	return notification, nil
}

func (m *Memory) ListUsersByBranchAndRoles(_ context.Context, branchID string, roles []domain.Role) ([]domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	wanted := make(map[domain.Role]bool, len(roles))
	for _, role := range roles {
		wanted[role] = true
	}

	users := make([]domain.User, 0)
	for _, user := range m.users {
		if !user.IsActive || !wanted[user.Role] {
			continue
		}
		if user.Role == domain.RoleOwner || user.BranchID == branchID {
			users = append(users, user)
		}
	}

	return users, nil
}

func (m *Memory) ListPendingPaymentReminders(_ context.Context, nowValue time.Time, horizon time.Time, limit int) ([]domain.Payment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	payments := make([]domain.Payment, 0)
	oldest := nowValue.AddDate(0, 0, -30)
	for _, payment := range m.payments {
		if payment.PaidAt != nil || (payment.Status != domain.PaymentStatusPending && payment.Status != domain.PaymentStatusOverdue) {
			continue
		}
		if payment.DueDate.Before(oldest) || payment.DueDate.After(horizon) {
			continue
		}
		payments = append(payments, payment)
	}
	sort.Slice(payments, func(i, j int) bool { return payments[i].DueDate.Before(payments[j].DueDate) })
	if limit > 0 && len(payments) > limit {
		payments = payments[:limit]
	}

	return payments, nil
}

func (m *Memory) ListFilesNearRetention(_ context.Context, nowValue time.Time, horizon time.Time, limit int) ([]domain.FileObject, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	files := make([]domain.FileObject, 0)
	for _, file := range m.files {
		if file.DeletedAt != nil || file.RetentionUntil == nil {
			continue
		}
		if file.RetentionUntil.After(nowValue) && !file.RetentionUntil.After(horizon) {
			files = append(files, file)
		}
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].RetentionUntil.Before(*files[j].RetentionUntil)
	})
	if limit > 0 && len(files) > limit {
		files = files[:limit]
	}

	return files, nil
}

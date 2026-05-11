package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"kingsway/backend/internal/auth"
	"kingsway/backend/internal/domain"
)

type Store interface {
	CreateNotification(ctx context.Context, notification domain.Notification) (domain.Notification, error)
	ListNotifications(ctx context.Context, recipientUserID string, unreadOnly bool) ([]domain.Notification, error)
	ListNotificationsPage(ctx context.Context, recipientUserID string, unreadOnly bool, page domain.PageRequest) ([]domain.Notification, int, error)
	MarkNotificationRead(ctx context.Context, id string, recipientUserID string, readAt time.Time) (domain.Notification, error)
	ListUsersByBranchAndRoles(ctx context.Context, branchID string, roles []domain.Role) ([]domain.User, error)
	GetTeacher(ctx context.Context, id string) (domain.Teacher, error)
	ListPendingPaymentReminders(ctx context.Context, now time.Time, horizon time.Time, limit int) ([]domain.Payment, error)
	ListFilesNearRetention(ctx context.Context, now time.Time, horizon time.Time, limit int) ([]domain.FileObject, error)
}

type Service struct {
	store Store
}

type WorkerOptions struct {
	Interval time.Duration
	Horizon  time.Duration
	Limit    int
	Logger   *zap.Logger
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) ListNotifications(ctx context.Context, actor domain.Principal, unreadOnly bool) ([]domain.Notification, error) {
	if actor.UserID == "" {
		return nil, domain.ErrUnauthorized
	}

	return s.store.ListNotifications(ctx, actor.UserID, unreadOnly)
}

func (s *Service) ListNotificationsPage(ctx context.Context, actor domain.Principal, unreadOnly bool, page domain.PageRequest) ([]domain.Notification, int, error) {
	if actor.UserID == "" {
		return nil, 0, domain.ErrUnauthorized
	}

	return s.store.ListNotificationsPage(ctx, actor.UserID, unreadOnly, page)
}

func (s *Service) MarkNotificationRead(ctx context.Context, actor domain.Principal, id string) (domain.Notification, error) {
	if actor.UserID == "" {
		return domain.Notification{}, domain.ErrUnauthorized
	}

	return s.store.MarkNotificationRead(ctx, strings.TrimSpace(id), actor.UserID, time.Now().UTC())
}

func (s *Service) HandleEvent(ctx context.Context, topic string, body []byte) error {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return err
	}

	switch topic {
	case "finance.payment.created":
		var payment domain.Payment
		if err := json.Unmarshal(body, &payment); err != nil {
			return err
		}
		if _, err := s.notifyBranchRoles(ctx, payment.BranchID, "payment.created", fmt.Sprintf("event:%s:%s", topic, payment.ID), payload, domain.RoleOwner, domain.RoleReceptionist); err != nil {
			return err
		}
		_, err := s.notifyBranchRoles(ctx, payment.BranchID, "salary.recalculation.requested", fmt.Sprintf("salary-recalc:payment:%s", payment.ID), payload, domain.RoleOwner)
		return err
	case "finance.salary_model.created":
		var model domain.SalaryModel
		if err := json.Unmarshal(body, &model); err != nil {
			return err
		}
		if _, err := s.notifyBranchRoles(ctx, model.BranchID, "salary.recalculation.requested", fmt.Sprintf("salary-recalc:model:%s", model.ID), payload, domain.RoleOwner); err != nil {
			return err
		}
		teacher, err := s.store.GetTeacher(ctx, model.TeacherID)
		if err != nil {
			return err
		}
		_, err = s.store.CreateNotification(ctx, domain.Notification{
			BranchID:        model.BranchID,
			RecipientUserID: teacher.UserID,
			Type:            "salary.model.created",
			Payload:         payload,
			DedupeKey:       fmt.Sprintf("event:%s:%s:%s", topic, model.ID, teacher.UserID),
		})
		return err
	case "files.retention.scheduled":
		branchID, _ := payload["branch_id"].(string)
		fileID, _ := payload["file_id"].(string)
		_, err := s.notifyBranchRoles(ctx, branchID, "file.retention.scheduled", fmt.Sprintf("event:%s:%s", topic, fileID), payload, domain.RoleOwner, domain.RoleReceptionist)
		return err
	case "files.retention.deleted":
		var file domain.FileObject
		if err := json.Unmarshal(body, &file); err != nil {
			return err
		}
		_, err := s.notifyBranchRoles(ctx, file.BranchID, "file.retention.deleted", fmt.Sprintf("event:%s:%s", topic, file.ID), payload, domain.RoleOwner, domain.RoleReceptionist)
		return err
	default:
		return nil
	}
}

func (s *Service) StartPaymentReminderWorker(ctx context.Context, options WorkerOptions) <-chan struct{} {
	done := make(chan struct{})
	interval, horizon, limit, log := normalizeWorkerOptions(options, 6*time.Hour, 7*24*time.Hour, 100)
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			count, err := s.CreatePaymentReminders(ctx, horizon, limit)
			if err != nil {
				log.Error("payment reminder worker failed", zap.Error(err))
			} else if count > 0 {
				log.Info("payment reminder worker created notifications", zap.Int("count", count))
			}

			select {
			case <-ctx.Done():
				log.Info("payment reminder worker stopped")
				return
			case <-ticker.C:
			}
		}
	}()

	return done
}

func (s *Service) StartFileRetentionNoticeWorker(ctx context.Context, options WorkerOptions) <-chan struct{} {
	done := make(chan struct{})
	interval, horizon, limit, log := normalizeWorkerOptions(options, 6*time.Hour, 7*24*time.Hour, 100)
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			count, err := s.CreateFileRetentionNotices(ctx, horizon, limit)
			if err != nil {
				log.Error("file retention notice worker failed", zap.Error(err))
			} else if count > 0 {
				log.Info("file retention notice worker created notifications", zap.Int("count", count))
			}

			select {
			case <-ctx.Done():
				log.Info("file retention notice worker stopped")
				return
			case <-ticker.C:
			}
		}
	}()

	return done
}

func (s *Service) CreatePaymentReminders(ctx context.Context, horizon time.Duration, limit int) (int, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	now := time.Now().UTC()
	payments, err := s.store.ListPendingPaymentReminders(ctx, now, now.Add(horizon), limit)
	if err != nil {
		return 0, err
	}

	created := 0
	for _, payment := range payments {
		payload := map[string]any{
			"payment_id":   payment.ID,
			"branch_id":    payment.BranchID,
			"student_id":   payment.StudentID,
			"amount_cents": payment.AmountCents,
			"currency":     payment.Currency,
			"due_date":     payment.DueDate,
			"status":       payment.Status,
		}
		count, err := s.notifyBranchRoles(ctx, payment.BranchID, "payment.reminder", fmt.Sprintf("payment-reminder:%s:%s", payment.ID, payment.DueDate.Format("2006-01-02")), payload, domain.RoleOwner, domain.RoleReceptionist)
		if err != nil {
			return created, err
		}
		created += count
	}

	return created, nil
}

func (s *Service) CreateFileRetentionNotices(ctx context.Context, horizon time.Duration, limit int) (int, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	now := time.Now().UTC()
	files, err := s.store.ListFilesNearRetention(ctx, now, now.Add(horizon), limit)
	if err != nil {
		return 0, err
	}

	created := 0
	for _, file := range files {
		payload := map[string]any{
			"file_id":         file.ID,
			"branch_id":       file.BranchID,
			"owner_type":      file.OwnerType,
			"owner_id":        file.OwnerID,
			"purpose":         file.Purpose,
			"retention_until": file.RetentionUntil,
		}
		dedupeKey := fmt.Sprintf("file-retention-notice:%s", file.ID)
		count, err := s.notifyBranchRoles(ctx, file.BranchID, "file.retention.due", dedupeKey, payload, domain.RoleOwner, domain.RoleReceptionist)
		if err != nil {
			return created, err
		}
		created += count
	}

	return created, nil
}

func (s *Service) notifyBranchRoles(ctx context.Context, branchID string, notificationType string, dedupeBase string, payload map[string]any, roles ...domain.Role) (int, error) {
	if strings.TrimSpace(branchID) == "" {
		return 0, domain.ErrInvalidInput
	}

	users, err := s.store.ListUsersByBranchAndRoles(ctx, branchID, roles)
	if err != nil {
		return 0, err
	}

	created := 0
	for _, user := range users {
		if !user.IsActive {
			continue
		}
		_, err := s.store.CreateNotification(ctx, domain.Notification{
			BranchID:        branchID,
			RecipientUserID: user.ID,
			Type:            notificationType,
			Payload:         payload,
			DedupeKey:       fmt.Sprintf("%s:%s", dedupeBase, user.ID),
		})
		if err != nil {
			return created, err
		}
		created++
	}

	return created, nil
}

func normalizeWorkerOptions(options WorkerOptions, defaultInterval time.Duration, defaultHorizon time.Duration, defaultLimit int) (time.Duration, time.Duration, int, *zap.Logger) {
	interval := options.Interval
	if interval <= 0 {
		interval = defaultInterval
	}
	horizon := options.Horizon
	if horizon <= 0 {
		horizon = defaultHorizon
	}
	limit := options.Limit
	if limit <= 0 || limit > 500 {
		limit = defaultLimit
	}
	log := options.Logger
	if log == nil {
		log = zap.NewNop()
	}

	return interval, horizon, limit, log
}

func RequireNotificationsAccess(actor domain.Principal) error {
	return auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist, domain.RoleTeacher, domain.RoleStudent)
}

package admin

import (
	"context"
	"strings"

	"kingsway/backend/internal/auth"
	"kingsway/backend/internal/domain"
)

type Store interface {
	ListOutboxEvents(ctx context.Context, status domain.OutboxEventStatus, limit int, offset int) ([]domain.OutboxEvent, int, error)
	RetryOutboxEvent(ctx context.Context, id string) (domain.OutboxEvent, error)
}

type Service struct {
	store Store
}

type ListOutboxEventsInput struct {
	Status domain.OutboxEventStatus
	Limit  int
	Offset int
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) ListOutboxEvents(ctx context.Context, actor domain.Principal, input ListOutboxEventsInput) ([]domain.OutboxEvent, int, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner); err != nil {
		return nil, 0, err
	}
	if input.Status != "" && !input.Status.IsValid() {
		return nil, 0, domain.ErrInvalidInput
	}
	if input.Limit <= 0 || input.Limit > 500 {
		input.Limit = 100
	}
	if input.Offset < 0 {
		input.Offset = 0
	}

	return s.store.ListOutboxEvents(ctx, input.Status, input.Limit, input.Offset)
}

func (s *Service) RetryOutboxEvent(ctx context.Context, actor domain.Principal, id string) (domain.OutboxEvent, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner); err != nil {
		return domain.OutboxEvent{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.OutboxEvent{}, domain.ErrInvalidInput
	}

	return s.store.RetryOutboxEvent(ctx, id)
}

package admin

import (
	"context"
	"errors"
	"testing"

	"kingsway/backend/internal/domain"
)

type fakeOutboxStore struct {
	listedStatus domain.OutboxEventStatus
	retriedID    string
}

func (s *fakeOutboxStore) ListOutboxEvents(_ context.Context, status domain.OutboxEventStatus, _ int, _ int) ([]domain.OutboxEvent, int, error) {
	s.listedStatus = status
	return []domain.OutboxEvent{{ID: "event-1", Status: domain.OutboxEventFailed}}, 1, nil
}

func (s *fakeOutboxStore) RetryOutboxEvent(_ context.Context, id string) (domain.OutboxEvent, error) {
	s.retriedID = id
	return domain.OutboxEvent{ID: id, Status: domain.OutboxEventPending}, nil
}

func TestOutboxAdminRequiresOwner(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeOutboxStore{})
	_, _, err := service.ListOutboxEvents(context.Background(), domain.Principal{Role: domain.RoleReceptionist}, ListOutboxEventsInput{})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden for non-owner list, got %v", err)
	}

	_, err = service.RetryOutboxEvent(context.Background(), domain.Principal{Role: domain.RoleTeacher}, "event-1")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden for non-owner retry, got %v", err)
	}
}

func TestOutboxAdminValidatesStatusAndRetries(t *testing.T) {
	t.Parallel()

	store := &fakeOutboxStore{}
	service := NewService(store)
	owner := domain.Principal{Role: domain.RoleOwner}

	_, _, err := service.ListOutboxEvents(context.Background(), owner, ListOutboxEventsInput{Status: "unknown"})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected invalid status error, got %v", err)
	}

	events, total, err := service.ListOutboxEvents(context.Background(), owner, ListOutboxEventsInput{Status: domain.OutboxEventFailed})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(events) != 1 || store.listedStatus != domain.OutboxEventFailed {
		t.Fatalf("unexpected list result: events=%#v total=%d status=%s", events, total, store.listedStatus)
	}

	event, err := service.RetryOutboxEvent(context.Background(), owner, "event-1")
	if err != nil {
		t.Fatal(err)
	}
	if event.Status != domain.OutboxEventPending || store.retriedID != "event-1" {
		t.Fatalf("unexpected retry result: event=%#v retried=%s", event, store.retriedID)
	}
}

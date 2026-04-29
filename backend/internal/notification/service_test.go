package notification_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"kingsway/backend/internal/domain"
	"kingsway/backend/internal/notification"
	"kingsway/backend/internal/store"
)

func TestNotificationsAreRecipientScoped(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := store.NewMemory()
	service := notification.NewService(repo)

	branch, err := repo.CreateBranch(ctx, domain.Branch{Name: "Main", Slug: "notification-main"})
	if err != nil {
		t.Fatal(err)
	}
	studentUser, err := repo.CreateUser(ctx, domain.User{BranchID: branch.ID, Role: domain.RoleStudent, Email: "notification-student@test.local", PasswordHash: "hash", FirstName: "Student", LastName: "User"})
	if err != nil {
		t.Fatal(err)
	}
	otherUser, err := repo.CreateUser(ctx, domain.User{BranchID: branch.ID, Role: domain.RoleStudent, Email: "notification-other@test.local", PasswordHash: "hash", FirstName: "Other", LastName: "User"})
	if err != nil {
		t.Fatal(err)
	}
	created, err := repo.CreateNotification(ctx, domain.Notification{
		BranchID:        branch.ID,
		RecipientUserID: studentUser.ID,
		Type:            "payment.reminder",
		Payload:         map[string]any{"payment_id": "payment-1"},
		DedupeKey:       "test:payment-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	visible, err := service.ListNotifications(ctx, domain.Principal{UserID: studentUser.ID, BranchID: branch.ID, Role: domain.RoleStudent}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(visible) != 1 || visible[0].ID != created.ID {
		t.Fatalf("expected recipient to see created notification, got %#v", visible)
	}

	hidden, err := service.ListNotifications(ctx, domain.Principal{UserID: otherUser.ID, BranchID: branch.ID, Role: domain.RoleStudent}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(hidden) != 0 {
		t.Fatalf("other user should not see recipient notification, got %#v", hidden)
	}

	_, err = service.MarkNotificationRead(ctx, domain.Principal{UserID: otherUser.ID, BranchID: branch.ID, Role: domain.RoleStudent}, created.ID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected not found when another user marks notification read, got %v", err)
	}

	read, err := service.MarkNotificationRead(ctx, domain.Principal{UserID: studentUser.ID, BranchID: branch.ID, Role: domain.RoleStudent}, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if read.ReadAt == nil || read.ReadAt.After(time.Now().UTC().Add(time.Second)) {
		t.Fatalf("expected read timestamp to be set, got %#v", read.ReadAt)
	}
}

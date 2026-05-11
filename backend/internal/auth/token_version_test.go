package auth

import (
	"context"
	"testing"

	"kingsway/backend/internal/domain"
	"kingsway/backend/internal/store"
)

func TestLogoutRevokesExistingToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	service := NewService(store.NewMemory(), "test-secret-that-is-long-enough-for-tests")
	created, err := service.CreateUser(ctx, domain.Principal{Role: domain.RoleOwner}, CreateUserInput{
		Role:      domain.RoleOwner,
		Email:     "owner@test.local",
		Password:  "Kingsway123!",
		FirstName: "Owner",
		LastName:  "User",
	})
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}

	login, err := service.Login(ctx, LoginInput{Email: created.Email, Password: "Kingsway123!"})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	principal, err := service.AuthenticateToken(ctx, login.Token)
	if err != nil {
		t.Fatalf("authenticate before logout: %v", err)
	}

	if err := service.Logout(ctx, principal); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := service.AuthenticateToken(ctx, login.Token); err == nil {
		t.Fatal("expected old token to be revoked after logout")
	}
}

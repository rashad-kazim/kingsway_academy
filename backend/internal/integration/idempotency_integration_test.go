package integration_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"kingsway/backend/internal/domain"
)

func TestRunIdempotentIntegrationSuite(t *testing.T) {
	fx := newIntegrationFixture(t)
	actor := integrationUser(t, fx, "", domain.RoleOwner, "IdempotencyActor")
	record := domain.IdempotencyRecord{
		ActorUserID: actor.ID,
		Method:      "POST",
		Path:        "/v1/integration",
		Key:         "idem-key",
		RequestHash: "same-request",
		ExpiresAt:   time.Now().UTC().Add(time.Hour),
	}

	result, err := fx.repo.RunIdempotent(fx.ctx, record, func(context.Context) (int, []byte, error) {
		return 201, []byte(`{"ok":true}`), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Replayed || result.ResponseStatus != 201 || string(result.ResponseBody) != `{"ok":true}` {
		t.Fatalf("unexpected first result: %+v", result)
	}

	called := false
	replay, err := fx.repo.RunIdempotent(fx.ctx, record, func(context.Context) (int, []byte, error) {
		called = true
		return 200, []byte(`{"wrong":true}`), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("replay must not execute business callback")
	}
	if !replay.Replayed || replay.ResponseStatus != 201 || string(replay.ResponseBody) != `{"ok": true}` {
		t.Fatalf("unexpected replay result: %+v body=%s", replay, string(replay.ResponseBody))
	}
}

func TestRunIdempotentRollsBackBusinessWriteOnError(t *testing.T) {
	fx := newIntegrationFixture(t)
	actor := integrationUser(t, fx, "", domain.RoleOwner, "IdempotencyRollbackActor")
	name := fmt.Sprintf("Rollback Branch %d", fx.stamp)
	record := domain.IdempotencyRecord{
		ActorUserID: actor.ID,
		Method:      "POST",
		Path:        "/v1/branches",
		Key:         fmt.Sprintf("rollback-%d", fx.stamp),
		RequestHash: "same-request-rollback",
		ExpiresAt:   time.Now().UTC().Add(time.Hour),
	}

	_, err := fx.repo.RunIdempotent(fx.ctx, record, func(ctx context.Context) (int, []byte, error) {
		_, err := fx.repo.CreateBranch(ctx, domain.Branch{
			Name: name,
			Slug: fmt.Sprintf("rollback-branch-%d", fx.stamp),
		})
		if err != nil {
			return 0, nil, err
		}

		return 0, nil, domain.ErrInvalidInput
	})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected business error, got %v", err)
	}

	branches, err := fx.repo.ListBranches(fx.ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, branch := range branches {
		if branch.Name == name {
			t.Fatalf("branch %q should have rolled back", name)
		}
	}

	result, err := fx.repo.RunIdempotent(fx.ctx, record, func(context.Context) (int, []byte, error) {
		return 202, []byte(`{"recovered":true}`), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Replayed || result.ResponseStatus != 202 || string(result.ResponseBody) != `{"recovered":true}` {
		t.Fatalf("idempotency row should have rolled back and allowed retry: %+v", result)
	}
}

func TestRunIdempotentReplaysCommittedResponse(t *testing.T) {
	fx := newIntegrationFixture(t)
	actor := integrationUser(t, fx, "", domain.RoleOwner, "IdempotencyReplayActor")
	record := domain.IdempotencyRecord{
		ActorUserID: actor.ID,
		Method:      "POST",
		Path:        "/v1/replay",
		Key:         fmt.Sprintf("replay-%d", fx.stamp),
		RequestHash: "same-request-replay",
		ExpiresAt:   time.Now().UTC().Add(time.Hour),
	}

	result, err := fx.repo.RunIdempotent(fx.ctx, record, func(context.Context) (int, []byte, error) {
		return 201, []byte(`{"ok":true}`), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Replayed || result.ResponseStatus != 201 || string(result.ResponseBody) != `{"ok":true}` {
		t.Fatalf("unexpected first result: %+v", result)
	}

	called := false
	replay, err := fx.repo.RunIdempotent(fx.ctx, record, func(context.Context) (int, []byte, error) {
		called = true
		return 200, []byte(`{"wrong":true}`), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("replay must not execute business callback")
	}
	if !replay.Replayed || replay.ResponseStatus != 201 || string(replay.ResponseBody) != `{"ok": true}` {
		t.Fatalf("unexpected replay result: %+v body=%s", replay, string(replay.ResponseBody))
	}
}

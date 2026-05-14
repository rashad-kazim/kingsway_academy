package integration_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"kingsway/backend/internal/domain"
	"kingsway/backend/internal/platform/config"
	"kingsway/backend/internal/platform/database"
	"kingsway/backend/internal/store"
)

type integrationFixture struct {
	ctx    context.Context
	cancel context.CancelFunc
	pool   *pgxpool.Pool
	repo   *store.Postgres
	stamp  int64
}

func newIntegrationFixture(t *testing.T) *integrationFixture {
	t.Helper()
	if os.Getenv("KINGSWAY_INTEGRATION") != "1" {
		t.Skip("set KINGSWAY_INTEGRATION=1 to run PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	cfg, err := config.Load()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	pool, err := database.Connect(ctx, database.DSN(cfg.Postgres()))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		cancel()
		t.Fatal(err)
	}
	if err := database.ApplyMigrations(ctx, pool, filepath.Join("..", "..", "migrations")); err != nil {
		pool.Close()
		cancel()
		t.Fatal(err)
	}

	fixture := &integrationFixture{
		ctx:    ctx,
		cancel: cancel,
		pool:   pool,
		repo:   store.NewPostgres(pool),
		stamp:  time.Now().UTC().UnixNano(),
	}
	t.Cleanup(func() {
		pool.Close()
		cancel()
	})

	return fixture
}

func integrationBranch(t *testing.T, fx *integrationFixture, label string) domain.Branch {
	t.Helper()

	branch, err := fx.repo.CreateBranch(fx.ctx, domain.Branch{
		Name: fmt.Sprintf("%s Branch %d", label, fx.stamp),
		Slug: fmt.Sprintf("%s-branch-%d", label, fx.stamp),
	})
	if err != nil {
		t.Fatal(err)
	}

	return branch
}

func integrationUser(t *testing.T, fx *integrationFixture, branchID string, role domain.Role, label string) domain.User {
	t.Helper()

	user, err := fx.repo.CreateUser(fx.ctx, domain.User{
		BranchID:     branchID,
		Role:         role,
		Email:        fmt.Sprintf("%s-%d@integration.local", label, fx.stamp),
		PasswordHash: "hash",
		FirstName:    "Integration",
		LastName:     label,
	})
	if err != nil {
		t.Fatal(err)
	}

	return user
}

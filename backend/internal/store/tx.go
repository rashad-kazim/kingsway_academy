package store

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type activeTransactionKey struct{}

func (p *Postgres) withTx(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
	if tx, ok := txFromContext(ctx); ok {
		return fn(ctx, tx)
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return mapPostgresError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	txCtx := context.WithValue(ctx, activeTransactionKey{}, tx)
	if err := fn(txCtx, tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return mapPostgresError(err)
	}

	return nil
}

func txFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(activeTransactionKey{}).(pgx.Tx)
	return tx, ok && tx != nil
}

func (p *Postgres) queryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if tx, ok := txFromContext(ctx); ok {
		return tx.QueryRow(ctx, sql, args...)
	}
	return p.pool.QueryRow(ctx, sql, args...)
}

func (p *Postgres) query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if tx, ok := txFromContext(ctx); ok {
		return tx.Query(ctx, sql, args...)
	}
	return p.pool.Query(ctx, sql, args...)
}

func (p *Postgres) exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if tx, ok := txFromContext(ctx); ok {
		return tx.Exec(ctx, sql, args...)
	}
	return p.pool.Exec(ctx, sql, args...)
}

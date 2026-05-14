package store

import (
	"context"
	"strings"

	"kingsway/backend/internal/domain"
)

func (p *Postgres) CountOwners(ctx context.Context) (int, error) {
	var count int
	err := p.queryRow(ctx, `SELECT count(*) FROM users WHERE role = 'owner'`).Scan(&count)
	if err != nil {
		return 0, mapPostgresError(err)
	}

	return count, nil
}

func (p *Postgres) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	user.Email = normalizeEmail(user.Email)
	if user.Email == "" || user.PasswordHash == "" || !user.Role.IsValid() {
		return domain.User{}, domain.ErrInvalidInput
	}
	if user.Role.RequiresBranchScope() && user.BranchID == "" {
		return domain.User{}, domain.ErrInvalidInput
	}

	row := p.queryRow(ctx, `
		INSERT INTO users (branch_id, role, email, password_hash, first_name, last_name, is_active)
		VALUES (nullif($1, '')::uuid, $2, $3, $4, $5, $6, true)
		RETURNING id::text, coalesce(branch_id::text, ''), role, email, password_hash, first_name, last_name, is_active, token_version, coalesce(last_login_at::text, ''), created_at, updated_at
	`, user.BranchID, user.Role, user.Email, user.PasswordHash, strings.TrimSpace(user.FirstName), strings.TrimSpace(user.LastName))

	created, err := scanUser(row)
	if err != nil {
		return domain.User{}, mapPostgresError(err)
	}

	return created, nil
}

func (p *Postgres) GetUser(ctx context.Context, id string) (domain.User, error) {
	row := p.queryRow(ctx, `
		SELECT id::text, coalesce(branch_id::text, ''), role, email, password_hash, first_name, last_name, is_active, token_version, coalesce(last_login_at::text, ''), created_at, updated_at
		FROM users
		WHERE id = $1
	`, id)

	user, err := scanUser(row)
	if err != nil {
		return domain.User{}, mapPostgresError(err)
	}

	return user, nil
}

func (p *Postgres) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	row := p.queryRow(ctx, `
		SELECT id::text, coalesce(branch_id::text, ''), role, email, password_hash, first_name, last_name, is_active, token_version, coalesce(last_login_at::text, ''), created_at, updated_at
		FROM users
		WHERE email = $1
	`, normalizeEmail(email))

	user, err := scanUser(row)
	if err != nil {
		return domain.User{}, mapPostgresError(err)
	}

	return user, nil
}

func (p *Postgres) RecordUserLogin(ctx context.Context, id string) (domain.User, error) {
	row := p.queryRow(ctx, `
		UPDATE users
		SET last_login_at = now(), updated_at = now()
		WHERE id = $1
		RETURNING id::text, coalesce(branch_id::text, ''), role, email, password_hash, first_name, last_name, is_active, token_version, coalesce(last_login_at::text, ''), created_at, updated_at
	`, id)

	user, err := scanUser(row)
	if err != nil {
		return domain.User{}, mapPostgresError(err)
	}

	return user, nil
}

func (p *Postgres) RevokeUserTokens(ctx context.Context, id string) (domain.User, error) {
	row := p.queryRow(ctx, `
		UPDATE users
		SET token_version = token_version + 1,
			updated_at = now()
		WHERE id = $1
		RETURNING id::text, coalesce(branch_id::text, ''), role, email, password_hash, first_name, last_name, is_active, token_version, coalesce(last_login_at::text, ''), created_at, updated_at
	`, id)

	user, err := scanUser(row)
	if err != nil {
		return domain.User{}, mapPostgresError(err)
	}

	return user, nil
}

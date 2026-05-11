package store

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"

	"kingsway/backend/internal/domain"
)

func (p *Postgres) CreateBranch(ctx context.Context, branch domain.Branch) (domain.Branch, error) {
	branch.Slug = normalizeSlug(branch.Slug)
	if strings.TrimSpace(branch.Name) == "" || branch.Slug == "" {
		return domain.Branch{}, domain.ErrInvalidInput
	}

	row := p.queryRow(ctx, `
		INSERT INTO branches (name, slug, address, opening_time, closing_time)
		VALUES ($1, $2, nullif($3, ''), $4, $5)
		RETURNING id::text, name, slug, coalesce(address, ''), opening_time, closing_time, created_at, updated_at
	`, strings.TrimSpace(branch.Name), branch.Slug, strings.TrimSpace(branch.Address), strings.TrimSpace(branch.OpeningTime), strings.TrimSpace(branch.ClosingTime))

	created, err := scanBranch(row)
	if err != nil {
		return domain.Branch{}, mapPostgresError(err)
	}

	return created, nil
}

func (p *Postgres) GetBranch(ctx context.Context, id string) (domain.Branch, error) {
	row := p.queryRow(ctx, `
		SELECT id::text, name, slug, coalesce(address, ''), opening_time, closing_time, created_at, updated_at
		FROM branches
		WHERE id = $1
	`, id)

	branch, err := scanBranch(row)
	if err != nil {
		return domain.Branch{}, mapPostgresError(err)
	}

	return branch, nil
}

func (p *Postgres) UpdateBranch(ctx context.Context, branch domain.Branch) (domain.Branch, error) {
	branch.Slug = normalizeSlug(branch.Slug)
	if strings.TrimSpace(branch.Name) == "" || branch.Slug == "" {
		return domain.Branch{}, domain.ErrInvalidInput
	}

	row := p.queryRow(ctx, `
		UPDATE branches
		SET name = $2, slug = $3, address = nullif($4, ''), opening_time = $5, closing_time = $6, updated_at = now()
		WHERE id = $1
		RETURNING id::text, name, slug, coalesce(address, ''), opening_time, closing_time, created_at, updated_at
	`, branch.ID, strings.TrimSpace(branch.Name), branch.Slug, strings.TrimSpace(branch.Address), strings.TrimSpace(branch.OpeningTime), strings.TrimSpace(branch.ClosingTime))

	updated, err := scanBranch(row)
	if err != nil {
		return domain.Branch{}, mapPostgresError(err)
	}

	return updated, nil
}

func (p *Postgres) ListBranches(ctx context.Context) ([]domain.Branch, error) {
	rows, err := p.query(ctx, `
		SELECT id::text, name, slug, coalesce(address, ''), opening_time, closing_time, created_at, updated_at
		FROM branches
		ORDER BY name
	`)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	branches := make([]domain.Branch, 0)
	for rows.Next() {
		branch, err := scanBranch(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		branches = append(branches, branch)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return branches, nil
}

func (p *Postgres) ListBranchesPage(ctx context.Context, page domain.PageRequest) ([]domain.Branch, int, error) {
	page = normalizePage(page)

	var total int
	if err := p.queryRow(ctx, `SELECT count(*) FROM branches`).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	rows, err := p.query(ctx, `
		SELECT id::text, name, slug, coalesce(address, ''), opening_time, closing_time, created_at, updated_at
		FROM branches
		ORDER BY name
		LIMIT $1 OFFSET $2
	`, page.Limit, page.Offset)
	if err != nil {
		return nil, 0, mapPostgresError(err)
	}
	defer rows.Close()

	branches := make([]domain.Branch, 0, page.Limit)
	for rows.Next() {
		branch, err := scanBranch(rows)
		if err != nil {
			return nil, 0, mapPostgresError(err)
		}
		branches = append(branches, branch)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	return branches, total, nil
}

func (p *Postgres) DeleteBranch(ctx context.Context, id string) (domain.Branch, error) {
	var branch domain.Branch
	err := p.withTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		found, err := scanBranch(tx.QueryRow(ctx, `
			SELECT id::text, name, slug, coalesce(address, ''), opening_time, closing_time, created_at, updated_at
			FROM branches
			WHERE id = $1
		`, id))
		if err != nil {
			return mapPostgresError(err)
		}
		branch = found

		statements := []string{
			`UPDATE branches SET logo_file_id = NULL WHERE id = $1`,
			`DELETE FROM notifications WHERE branch_id = $1 OR recipient_user_id IN (SELECT id FROM users WHERE branch_id = $1)`,
			`DELETE FROM exam_results WHERE branch_id = $1`,
			`DELETE FROM exam_participants WHERE branch_id = $1`,
			`DELETE FROM exams WHERE branch_id = $1`,
			`DELETE FROM assignments WHERE branch_id = $1`,
			`DELETE FROM schedule_items WHERE branch_id = $1`,
			`DELETE FROM class_students WHERE branch_id = $1`,
			`DELETE FROM student_teacher_assignments WHERE branch_id = $1`,
			`DELETE FROM payments WHERE branch_id = $1`,
			`DELETE FROM salary_models WHERE branch_id = $1`,
			`DELETE FROM staff_profiles WHERE branch_id = $1`,
			`DELETE FROM classes WHERE branch_id = $1`,
			`DELETE FROM course_score_categories WHERE branch_id = $1`,
			`DELETE FROM teacher_course_specializations WHERE branch_id = $1`,
			`DELETE FROM courses WHERE branch_id = $1`,
			`DELETE FROM rooms WHERE branch_id = $1`,
			`DELETE FROM students WHERE branch_id = $1`,
			`DELETE FROM teachers WHERE branch_id = $1`,
			`DELETE FROM files WHERE branch_id = $1`,
			`DELETE FROM users WHERE branch_id = $1`,
			`DELETE FROM outbox_events WHERE payload->>'branch_id' = $1 OR payload->'file'->>'branch_id' = $1`,
			`DELETE FROM branches WHERE id = $1`,
		}
		for _, statement := range statements {
			if _, err := tx.Exec(ctx, statement, id); err != nil {
				return mapPostgresError(err)
			}
		}

		return nil
	})
	if err != nil {
		return domain.Branch{}, err
	}
	return branch, nil
}

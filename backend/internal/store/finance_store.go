package store

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"

	"kingsway/backend/internal/domain"
)

func (p *Postgres) CreatePayment(ctx context.Context, payment domain.Payment) (domain.Payment, error) {
	student, err := p.GetStudent(ctx, payment.StudentID)
	if err != nil {
		return domain.Payment{}, err
	}
	if student.BranchID != payment.BranchID || payment.AmountCents <= 0 {
		return domain.Payment{}, domain.ErrInvalidInput
	}
	if payment.Currency == "" {
		payment.Currency = "AZN"
	}
	if payment.Status == "" {
		payment.Status = domain.PaymentStatusPending
	}

	var created domain.Payment
	err = p.withTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			INSERT INTO payments (
				branch_id, student_id, amount_cents, currency, due_date, paid_at,
				status, receipt_file_id, created_by_user_id
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, nullif($8, '')::uuid, $9)
			RETURNING id::text, branch_id::text, student_id::text, amount_cents, currency, due_date,
				paid_at, status, coalesce(receipt_file_id::text, ''), created_by_user_id::text, created_at, updated_at
		`, payment.BranchID, payment.StudentID, payment.AmountCents, strings.ToUpper(payment.Currency), payment.DueDate, payment.PaidAt, payment.Status, payment.ReceiptFileID, payment.CreatedByUserID)

		next, err := scanPayment(row)
		if err != nil {
			return mapPostgresError(err)
		}
		created = next
		if err := insertOutboxTx(ctx, tx, "finance.payment.created", created); err != nil {
			return mapPostgresError(err)
		}

		return nil
	})
	if err != nil {
		return domain.Payment{}, err
	}

	return created, nil
}

func (p *Postgres) ListPayments(ctx context.Context, branchID string) ([]domain.Payment, error) {
	rows, err := p.query(ctx, `
		SELECT id::text, branch_id::text, student_id::text, amount_cents, currency, due_date,
			paid_at, status, coalesce(receipt_file_id::text, ''), created_by_user_id::text, created_at, updated_at
		FROM payments
		WHERE ($1 = '' OR branch_id = $1::uuid)
		ORDER BY due_date
	`, branchID)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	payments := make([]domain.Payment, 0)
	for rows.Next() {
		payment, err := scanPayment(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		payments = append(payments, payment)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return payments, nil
}

func (p *Postgres) ListPaymentsPage(ctx context.Context, branchID string, status domain.PaymentStatus, page domain.PageRequest) ([]domain.Payment, int, error) {
	page = normalizePage(page)

	var total int
	if err := p.queryRow(ctx, `
		SELECT count(*)
		FROM payments
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR status = $2)
	`, branchID, string(status)).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	rows, err := p.query(ctx, `
		SELECT id::text, branch_id::text, student_id::text, amount_cents, currency, due_date,
			paid_at, status, coalesce(receipt_file_id::text, ''), created_by_user_id::text, created_at, updated_at
		FROM payments
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR status = $2)
		ORDER BY due_date
		LIMIT $3 OFFSET $4
	`, branchID, string(status), page.Limit, page.Offset)
	if err != nil {
		return nil, 0, mapPostgresError(err)
	}
	defer rows.Close()

	payments := make([]domain.Payment, 0, page.Limit)
	for rows.Next() {
		payment, err := scanPayment(rows)
		if err != nil {
			return nil, 0, mapPostgresError(err)
		}
		payments = append(payments, payment)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	return payments, total, nil
}

func (p *Postgres) GetPayment(ctx context.Context, id string) (domain.Payment, error) {
	row := p.queryRow(ctx, `
		SELECT id::text, branch_id::text, student_id::text, amount_cents, currency, due_date,
			paid_at, status, coalesce(receipt_file_id::text, ''), created_by_user_id::text, created_at, updated_at
		FROM payments
		WHERE id = $1
	`, id)

	payment, err := scanPayment(row)
	if err != nil {
		return domain.Payment{}, mapPostgresError(err)
	}

	return payment, nil
}

func (p *Postgres) CreateSalaryModel(ctx context.Context, model domain.SalaryModel) (domain.SalaryModel, error) {
	teacher, err := p.GetTeacher(ctx, model.TeacherID)
	if err != nil {
		return domain.SalaryModel{}, err
	}
	if teacher.BranchID != model.BranchID || !model.ModelType.IsValid() || model.ApprovedByOwnerUserID == "" {
		return domain.SalaryModel{}, domain.ErrInvalidInput
	}

	var created domain.SalaryModel
	err = p.withTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			INSERT INTO salary_models (
				branch_id, teacher_id, model_type, fixed_monthly_amount_cents,
				student_percent_basis_points, active_from, active_to, approved_by_owner_user_id
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id::text, branch_id::text, teacher_id::text, model_type,
				coalesce(fixed_monthly_amount_cents, 0), coalesce(student_percent_basis_points, 0),
				active_from, active_to, approved_by_owner_user_id::text, created_at
		`, model.BranchID, model.TeacherID, model.ModelType, fixedSalaryValue(model), percentSalaryValue(model), model.ActiveFrom, model.ActiveTo, model.ApprovedByOwnerUserID)

		next, err := scanSalaryModel(row)
		if err != nil {
			return mapPostgresError(err)
		}
		created = next
		if err := insertOutboxTx(ctx, tx, "finance.salary_model.created", created); err != nil {
			return mapPostgresError(err)
		}

		return nil
	})
	if err != nil {
		return domain.SalaryModel{}, err
	}

	return created, nil
}

func (p *Postgres) ListSalaryModels(ctx context.Context, branchID string) ([]domain.SalaryModel, error) {
	rows, err := p.query(ctx, `
		SELECT id::text, branch_id::text, teacher_id::text, model_type,
			coalesce(fixed_monthly_amount_cents, 0), coalesce(student_percent_basis_points, 0),
			active_from, active_to, approved_by_owner_user_id::text, created_at
		FROM salary_models
		WHERE ($1 = '' OR branch_id = $1::uuid)
		ORDER BY created_at DESC
	`, branchID)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	models := make([]domain.SalaryModel, 0)
	for rows.Next() {
		model, err := scanSalaryModel(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		models = append(models, model)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return models, nil
}

func (p *Postgres) ListSalaryModelsPage(ctx context.Context, branchID string, teacherID string, page domain.PageRequest) ([]domain.SalaryModel, int, error) {
	page = normalizePage(page)

	var total int
	if err := p.queryRow(ctx, `
		SELECT count(*)
		FROM salary_models
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR teacher_id = $2::uuid)
	`, branchID, teacherID).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	rows, err := p.query(ctx, `
		SELECT id::text, branch_id::text, teacher_id::text, model_type,
			coalesce(fixed_monthly_amount_cents, 0), coalesce(student_percent_basis_points, 0),
			active_from, active_to, approved_by_owner_user_id::text, created_at
		FROM salary_models
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR teacher_id = $2::uuid)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`, branchID, teacherID, page.Limit, page.Offset)
	if err != nil {
		return nil, 0, mapPostgresError(err)
	}
	defer rows.Close()

	models := make([]domain.SalaryModel, 0, page.Limit)
	for rows.Next() {
		model, err := scanSalaryModel(rows)
		if err != nil {
			return nil, 0, mapPostgresError(err)
		}
		models = append(models, model)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	return models, total, nil
}

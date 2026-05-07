package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"kingsway/backend/internal/domain"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

type scanner interface {
	Scan(dest ...any) error
}

func (p *Postgres) BeginIdempotency(ctx context.Context, record domain.IdempotencyRecord) (domain.IdempotencyBeginResult, error) {
	if record.ActorUserID == "" || record.Method == "" || record.Path == "" || record.Key == "" || record.RequestHash == "" {
		return domain.IdempotencyBeginResult{}, domain.ErrInvalidInput
	}
	if record.ExpiresAt.IsZero() {
		record.ExpiresAt = time.Now().UTC().Add(24 * time.Hour)
	}

	_, _ = p.pool.Exec(ctx, `
		DELETE FROM idempotency_keys
		WHERE actor_user_id = $1 AND method = $2 AND path = $3 AND key = $4 AND expires_at < now()
	`, record.ActorUserID, record.Method, record.Path, record.Key)

	row := p.pool.QueryRow(ctx, `
		INSERT INTO idempotency_keys (
			actor_user_id, method, path, key, request_hash, status, expires_at
		)
		VALUES ($1, $2, $3, $4, $5, 'pending', $6)
		ON CONFLICT DO NOTHING
		RETURNING actor_user_id::text, method, path, key, request_hash, status,
			coalesce(response_status, 0), coalesce(response_body::text, ''),
			created_at, updated_at, expires_at
	`, record.ActorUserID, record.Method, record.Path, record.Key, record.RequestHash, record.ExpiresAt)

	inserted, err := scanIdempotencyRecord(row)
	if err == nil {
		return domain.IdempotencyBeginResult{Started: true, Record: inserted}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.IdempotencyBeginResult{}, mapPostgresError(err)
	}

	existing, err := scanIdempotencyRecord(p.pool.QueryRow(ctx, `
		SELECT actor_user_id::text, method, path, key, request_hash, status,
			coalesce(response_status, 0), coalesce(response_body::text, ''),
			created_at, updated_at, expires_at
		FROM idempotency_keys
		WHERE actor_user_id = $1 AND method = $2 AND path = $3 AND key = $4
	`, record.ActorUserID, record.Method, record.Path, record.Key))
	if err != nil {
		return domain.IdempotencyBeginResult{}, mapPostgresError(err)
	}
	if existing.RequestHash != record.RequestHash {
		return domain.IdempotencyBeginResult{}, domain.ErrConflict
	}

	return domain.IdempotencyBeginResult{Record: existing}, nil
}

func (p *Postgres) CompleteIdempotency(ctx context.Context, actorUserID string, method string, path string, key string, responseStatus int, responseBody []byte) error {
	tag, err := p.pool.Exec(ctx, `
		UPDATE idempotency_keys
		SET status = 'completed',
			response_status = $5,
			response_body = $6::jsonb,
			updated_at = now()
		WHERE actor_user_id = $1 AND method = $2 AND path = $3 AND key = $4
	`, actorUserID, method, path, key, responseStatus, string(responseBody))
	if err != nil {
		return mapPostgresError(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (p *Postgres) ClearIdempotency(ctx context.Context, actorUserID string, method string, path string, key string) error {
	_, err := p.pool.Exec(ctx, `
		DELETE FROM idempotency_keys
		WHERE actor_user_id = $1 AND method = $2 AND path = $3 AND key = $4
	`, actorUserID, method, path, key)
	if err != nil {
		return mapPostgresError(err)
	}

	return nil
}

func (p *Postgres) CountOwners(ctx context.Context) (int, error) {
	var count int
	err := p.pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE role = 'owner'`).Scan(&count)
	if err != nil {
		return 0, mapPostgresError(err)
	}

	return count, nil
}

func (p *Postgres) CreateBranch(ctx context.Context, branch domain.Branch) (domain.Branch, error) {
	branch.Slug = normalizeSlug(branch.Slug)
	if strings.TrimSpace(branch.Name) == "" || branch.Slug == "" {
		return domain.Branch{}, domain.ErrInvalidInput
	}

	row := p.pool.QueryRow(ctx, `
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
	row := p.pool.QueryRow(ctx, `
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

	row := p.pool.QueryRow(ctx, `
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
	rows, err := p.pool.Query(ctx, `
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

func (p *Postgres) DeleteBranch(ctx context.Context, id string) (domain.Branch, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return domain.Branch{}, mapPostgresError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	branch, err := scanBranch(tx.QueryRow(ctx, `
		SELECT id::text, name, slug, coalesce(address, ''), opening_time, closing_time, created_at, updated_at
		FROM branches
		WHERE id = $1
	`, id))
	if err != nil {
		return domain.Branch{}, mapPostgresError(err)
	}

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
			return domain.Branch{}, mapPostgresError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Branch{}, mapPostgresError(err)
	}

	return branch, nil
}

func (p *Postgres) CreateStaffMember(ctx context.Context, user domain.User, staff domain.StaffMember) (domain.StaffMember, error) {
	user.Email = normalizeEmail(user.Email)
	if user.BranchID == "" || user.Email == "" || user.PasswordHash == "" || user.FirstName == "" || user.LastName == "" {
		return domain.StaffMember{}, domain.ErrInvalidInput
	}
	if user.Role != domain.RoleReceptionist {
		return domain.StaffMember{}, domain.ErrInvalidInput
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return domain.StaffMember{}, mapPostgresError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	createdUser, err := scanUser(tx.QueryRow(ctx, `
		INSERT INTO users (branch_id, role, email, password_hash, first_name, last_name, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, true)
		RETURNING id::text, coalesce(branch_id::text, ''), role, email, password_hash, first_name, last_name, is_active, coalesce(last_login_at::text, ''), created_at, updated_at
	`, user.BranchID, user.Role, user.Email, user.PasswordHash, strings.TrimSpace(user.FirstName), strings.TrimSpace(user.LastName)))
	if err != nil {
		return domain.StaffMember{}, mapPostgresError(err)
	}

	var staffID string
	err = tx.QueryRow(ctx, `
		INSERT INTO staff_profiles (branch_id, user_id, role, birth_date, gender, phone, address, hired_at, salary_amount_azn, profile_photo_file_id)
		VALUES ($1, $2, $3, CASE WHEN $4 = '' THEN NULL ELSE to_date($4, 'DD/MM/YYYY') END, $5, $6, $7, CASE WHEN $8 = '' THEN NULL ELSE to_date($8, 'DD/MM/YYYY') END, $9, nullif($10, '')::uuid)
		RETURNING id::text
	`, createdUser.BranchID, createdUser.ID, domain.RoleReceptionist, strings.TrimSpace(staff.BirthDate), strings.TrimSpace(staff.Gender), strings.TrimSpace(staff.Phone), strings.TrimSpace(staff.Address), strings.TrimSpace(staff.HiredAt), staff.SalaryAmountAZN, strings.TrimSpace(staff.ProfilePhotoFileID)).Scan(&staffID)
	if err != nil {
		return domain.StaffMember{}, mapPostgresError(err)
	}

	created, err := p.staffMemberByIDTx(ctx, tx, staffID)
	if err != nil {
		return domain.StaffMember{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.StaffMember{}, mapPostgresError(err)
	}

	return created, nil
}

func (p *Postgres) UpdateStaffMember(ctx context.Context, staff domain.StaffMember, passwordHash string) (domain.StaffMember, error) {
	if staff.ID == "" || staff.FirstName == "" || staff.LastName == "" || staff.Email == "" || staff.SalaryAmountAZN < 0 {
		return domain.StaffMember{}, domain.ErrInvalidInput
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return domain.StaffMember{}, mapPostgresError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	current, err := p.staffMemberByIDTx(ctx, tx, staff.ID)
	if err != nil {
		return domain.StaffMember{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE users
		SET branch_id = $2,
			email = $3,
			first_name = $4,
			last_name = $5,
			password_hash = CASE WHEN $6 = '' THEN password_hash ELSE $6 END,
			is_active = $7,
			updated_at = now()
		WHERE id = $1
	`, current.UserID, staff.BranchID, normalizeEmail(staff.Email), strings.TrimSpace(staff.FirstName), strings.TrimSpace(staff.LastName), strings.TrimSpace(passwordHash), staff.IsActive)
	if err != nil {
		return domain.StaffMember{}, mapPostgresError(err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE staff_profiles
		SET branch_id = $2,
			birth_date = CASE WHEN $3 = '' THEN NULL ELSE to_date($3, 'DD/MM/YYYY') END,
			gender = $4,
			phone = $5,
			address = $6,
			hired_at = CASE WHEN $7 = '' THEN NULL ELSE to_date($7, 'DD/MM/YYYY') END,
			salary_amount_azn = $8,
			profile_photo_file_id = nullif($9, '')::uuid,
			updated_at = now()
		WHERE id = $1
	`, staff.ID, staff.BranchID, strings.TrimSpace(staff.BirthDate), strings.TrimSpace(staff.Gender), strings.TrimSpace(staff.Phone), strings.TrimSpace(staff.Address), strings.TrimSpace(staff.HiredAt), staff.SalaryAmountAZN, strings.TrimSpace(staff.ProfilePhotoFileID))
	if err != nil {
		return domain.StaffMember{}, mapPostgresError(err)
	}

	updated, err := p.staffMemberByIDTx(ctx, tx, staff.ID)
	if err != nil {
		return domain.StaffMember{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.StaffMember{}, mapPostgresError(err)
	}

	return updated, nil
}

func (p *Postgres) DeleteStaffMember(ctx context.Context, id string) (domain.StaffMember, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return domain.StaffMember{}, mapPostgresError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	staff, err := p.staffMemberByIDTx(ctx, tx, id)
	if err != nil {
		return domain.StaffMember{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE staff_profiles SET profile_photo_file_id = NULL WHERE id = $1`, id); err != nil {
		return domain.StaffMember{}, mapPostgresError(err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM notifications WHERE recipient_user_id = $1`, staff.UserID); err != nil {
		return domain.StaffMember{}, mapPostgresError(err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM staff_profiles WHERE id = $1`, id); err != nil {
		return domain.StaffMember{}, mapPostgresError(err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, staff.UserID); err != nil {
		return domain.StaffMember{}, mapPostgresError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.StaffMember{}, mapPostgresError(err)
	}

	return staff, nil
}

func (p *Postgres) GetStaffMember(ctx context.Context, id string) (domain.StaffMember, error) {
	row := p.pool.QueryRow(ctx, staffMemberSelect()+` WHERE sp.id = $1`, id)
	staff, err := scanStaffMember(row)
	if err != nil {
		return domain.StaffMember{}, mapPostgresError(err)
	}

	return staff, nil
}

func (p *Postgres) ListStaffMembers(ctx context.Context, branchID string) ([]domain.StaffMember, error) {
	rows, err := p.pool.Query(ctx, staffMemberSelect()+`
		WHERE sp.branch_id = $1
		ORDER BY u.last_name, u.first_name
	`, branchID)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	staff := make([]domain.StaffMember, 0)
	for rows.Next() {
		member, err := scanStaffMember(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		staff = append(staff, member)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return staff, nil
}

func (p *Postgres) staffMemberByIDTx(ctx context.Context, tx pgx.Tx, id string) (domain.StaffMember, error) {
	staff, err := scanStaffMember(tx.QueryRow(ctx, staffMemberSelect()+` WHERE sp.id = $1`, id))
	if err != nil {
		return domain.StaffMember{}, mapPostgresError(err)
	}

	return staff, nil
}

func (p *Postgres) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	user.Email = normalizeEmail(user.Email)
	if user.Email == "" || user.PasswordHash == "" || !user.Role.IsValid() {
		return domain.User{}, domain.ErrInvalidInput
	}
	if user.Role.RequiresBranchScope() && user.BranchID == "" {
		return domain.User{}, domain.ErrInvalidInput
	}

	row := p.pool.QueryRow(ctx, `
		INSERT INTO users (branch_id, role, email, password_hash, first_name, last_name, is_active)
		VALUES (nullif($1, '')::uuid, $2, $3, $4, $5, $6, true)
		RETURNING id::text, coalesce(branch_id::text, ''), role, email, password_hash, first_name, last_name, is_active, coalesce(last_login_at::text, ''), created_at, updated_at
	`, user.BranchID, user.Role, user.Email, user.PasswordHash, strings.TrimSpace(user.FirstName), strings.TrimSpace(user.LastName))

	created, err := scanUser(row)
	if err != nil {
		return domain.User{}, mapPostgresError(err)
	}

	return created, nil
}

func (p *Postgres) GetUser(ctx context.Context, id string) (domain.User, error) {
	row := p.pool.QueryRow(ctx, `
		SELECT id::text, coalesce(branch_id::text, ''), role, email, password_hash, first_name, last_name, is_active, coalesce(last_login_at::text, ''), created_at, updated_at
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
	row := p.pool.QueryRow(ctx, `
		SELECT id::text, coalesce(branch_id::text, ''), role, email, password_hash, first_name, last_name, is_active, coalesce(last_login_at::text, ''), created_at, updated_at
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
	row := p.pool.QueryRow(ctx, `
		UPDATE users
		SET last_login_at = now(), updated_at = now()
		WHERE id = $1
		RETURNING id::text, coalesce(branch_id::text, ''), role, email, password_hash, first_name, last_name, is_active, coalesce(last_login_at::text, ''), created_at, updated_at
	`, id)

	user, err := scanUser(row)
	if err != nil {
		return domain.User{}, mapPostgresError(err)
	}

	return user, nil
}

func (p *Postgres) CreateStudent(ctx context.Context, student domain.Student) (domain.Student, error) {
	if student.FirstName == "" || student.LastName == "" || student.FIN == "" {
		return domain.Student{}, domain.ErrInvalidInput
	}
	if student.Status == "" {
		student.Status = domain.StudentStatusActive
	}
	if student.Status == domain.StudentStatusLeft && strings.TrimSpace(student.LeftReason) == "" {
		return domain.Student{}, domain.ErrInvalidInput
	}

	row := p.pool.QueryRow(ctx, `
		INSERT INTO students (
			branch_id, user_id, fin_code, first_name, last_name, birth_date,
			gender, phone, address, profile_photo_file_id, status, left_reason
		)
		VALUES (
			$1, nullif($2, '')::uuid, $3, $4, $5,
			CASE WHEN $6 = '' THEN NULL ELSE to_date($6, 'DD/MM/YYYY') END,
			nullif($7, ''), $8, $9, nullif($10, '')::uuid, $11, nullif($12, '')
		)
		RETURNING id::text, branch_id::text, coalesce(user_id::text, ''), fin_code,
			first_name, last_name, coalesce(to_char(birth_date, 'DD/MM/YYYY'), ''),
			coalesce(gender, ''), phone, address, coalesce(profile_photo_file_id::text, ''),
			status, coalesce(left_reason, ''), created_at, updated_at
	`, student.BranchID, student.UserID, student.FIN.String(), strings.TrimSpace(student.FirstName), strings.TrimSpace(student.LastName), strings.TrimSpace(student.BirthDate), strings.TrimSpace(student.Gender), strings.TrimSpace(student.Phone), strings.TrimSpace(student.Address), strings.TrimSpace(student.ProfilePhotoFileID), student.Status, strings.TrimSpace(student.LeftReason))

	created, err := scanStudent(row)
	if err != nil {
		return domain.Student{}, mapPostgresError(err)
	}

	return created, nil
}

func (p *Postgres) CreateStudentWithDetails(ctx context.Context, student domain.Student, contacts []domain.StudentParentContact, registrations []domain.StudentCourseRegistration) (domain.Student, error) {
	if student.FirstName == "" || student.LastName == "" || student.FIN == "" {
		return domain.Student{}, domain.ErrInvalidInput
	}
	if student.Status == "" {
		student.Status = domain.StudentStatusActive
	}
	if student.Status == domain.StudentStatusLeft && strings.TrimSpace(student.LeftReason) == "" {
		return domain.Student{}, domain.ErrInvalidInput
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return domain.Student{}, mapPostgresError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	created, err := scanStudent(tx.QueryRow(ctx, `
		INSERT INTO students (
			branch_id, user_id, fin_code, first_name, last_name, birth_date,
			gender, phone, address, profile_photo_file_id, status, left_reason
		)
		VALUES (
			$1, nullif($2, '')::uuid, $3, $4, $5,
			CASE WHEN $6 = '' THEN NULL ELSE to_date($6, 'DD/MM/YYYY') END,
			nullif($7, ''), $8, $9, nullif($10, '')::uuid, $11, nullif($12, '')
		)
		RETURNING id::text, branch_id::text, coalesce(user_id::text, ''), fin_code,
			first_name, last_name, coalesce(to_char(birth_date, 'DD/MM/YYYY'), ''),
			coalesce(gender, ''), phone, address, coalesce(profile_photo_file_id::text, ''),
			status, coalesce(left_reason, ''), created_at, updated_at
	`, student.BranchID, student.UserID, student.FIN.String(), strings.TrimSpace(student.FirstName), strings.TrimSpace(student.LastName), strings.TrimSpace(student.BirthDate), strings.TrimSpace(student.Gender), strings.TrimSpace(student.Phone), strings.TrimSpace(student.Address), strings.TrimSpace(student.ProfilePhotoFileID), student.Status, strings.TrimSpace(student.LeftReason)))
	if err != nil {
		return domain.Student{}, mapPostgresError(err)
	}

	for _, contact := range contacts {
		if contact.BranchID != created.BranchID || strings.TrimSpace(contact.Relation) == "" || strings.TrimSpace(contact.Name) == "" || len(contact.Phones) == 0 {
			return domain.Student{}, domain.ErrInvalidInput
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO student_parent_contacts (branch_id, student_id, relation, name, phones)
			VALUES ($1, $2, $3, $4, $5)
		`, contact.BranchID, created.ID, strings.TrimSpace(contact.Relation), strings.TrimSpace(contact.Name), contact.Phones); err != nil {
			return domain.Student{}, mapPostgresError(err)
		}
	}

	for _, registration := range registrations {
		if registration.BranchID != created.BranchID || registration.CourseID == "" || registration.StartDate == "" || registration.MonthlyAmountCents < 0 {
			return domain.Student{}, domain.ErrInvalidInput
		}
		var courseBranchID string
		var courseActive bool
		if err := tx.QueryRow(ctx, `
			SELECT branch_id::text, is_active
			FROM courses
			WHERE id = $1
		`, registration.CourseID).Scan(&courseBranchID, &courseActive); err != nil {
			return domain.Student{}, mapPostgresError(err)
		}
		if courseBranchID != created.BranchID || !courseActive {
			return domain.Student{}, domain.ErrInvalidInput
		}
		if registration.TeacherID != "" {
			var teacherBranchID string
			var teacherStatus domain.TeacherStatus
			if err := tx.QueryRow(ctx, `
				SELECT branch_id::text, status
				FROM teachers
				WHERE id = $1
			`, registration.TeacherID).Scan(&teacherBranchID, &teacherStatus); err != nil {
				return domain.Student{}, mapPostgresError(err)
			}
			if teacherBranchID != created.BranchID || teacherStatus != domain.TeacherStatusActive {
				return domain.Student{}, domain.ErrInvalidInput
			}
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO student_course_registrations (
				branch_id, student_id, course_id, teacher_id, monthly_amount_cents, start_date
			)
			VALUES ($1, $2, $3, nullif($4, '')::uuid, $5, to_date($6, 'DD/MM/YYYY'))
		`, registration.BranchID, created.ID, registration.CourseID, registration.TeacherID, registration.MonthlyAmountCents, registration.StartDate); err != nil {
			return domain.Student{}, mapPostgresError(err)
		}
		if registration.TeacherID != "" {
			if _, err := tx.Exec(ctx, `
				UPDATE student_teacher_assignments
				SET valid_to = to_date($3, 'DD/MM/YYYY')
				WHERE branch_id = $1 AND student_id = $2 AND valid_to IS NULL
			`, registration.BranchID, created.ID, registration.StartDate); err != nil {
				return domain.Student{}, mapPostgresError(err)
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO student_teacher_assignments (branch_id, student_id, teacher_id, valid_from, reason)
				VALUES ($1, $2, $3, to_date($4, 'DD/MM/YYYY'), 'initial')
			`, registration.BranchID, created.ID, registration.TeacherID, registration.StartDate); err != nil {
				return domain.Student{}, mapPostgresError(err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Student{}, mapPostgresError(err)
	}

	return created, nil
}

func (p *Postgres) CreateStudentParentContacts(ctx context.Context, studentID string, contacts []domain.StudentParentContact) ([]domain.StudentParentContact, error) {
	student, err := p.GetStudent(ctx, studentID)
	if err != nil {
		return nil, err
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	created := make([]domain.StudentParentContact, 0, len(contacts))
	for _, contact := range contacts {
		if contact.BranchID != student.BranchID || strings.TrimSpace(contact.Relation) == "" || strings.TrimSpace(contact.Name) == "" || len(contact.Phones) == 0 {
			return nil, domain.ErrInvalidInput
		}
		row := tx.QueryRow(ctx, `
			INSERT INTO student_parent_contacts (branch_id, student_id, relation, name, phones)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id::text, branch_id::text, student_id::text, relation, name, phones, created_at
		`, contact.BranchID, student.ID, strings.TrimSpace(contact.Relation), strings.TrimSpace(contact.Name), contact.Phones)
		var next domain.StudentParentContact
		if err := row.Scan(&next.ID, &next.BranchID, &next.StudentID, &next.Relation, &next.Name, &next.Phones, &next.CreatedAt); err != nil {
			return nil, mapPostgresError(err)
		}
		created = append(created, next)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, mapPostgresError(err)
	}

	return created, nil
}

func (p *Postgres) CreateStudentCourseRegistrations(ctx context.Context, studentID string, registrations []domain.StudentCourseRegistration) ([]domain.StudentCourseRegistration, error) {
	student, err := p.GetStudent(ctx, studentID)
	if err != nil {
		return nil, err
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	created := make([]domain.StudentCourseRegistration, 0, len(registrations))
	for _, registration := range registrations {
		if registration.BranchID != student.BranchID || registration.CourseID == "" || registration.StartDate == "" || registration.MonthlyAmountCents < 0 {
			return nil, domain.ErrInvalidInput
		}
		if err := p.ensureCourseBranch(ctx, registration.CourseID, student.BranchID); err != nil {
			return nil, err
		}
		if registration.TeacherID != "" {
			teacher, err := p.GetTeacher(ctx, registration.TeacherID)
			if err != nil {
				return nil, err
			}
			if teacher.BranchID != student.BranchID || teacher.Status != domain.TeacherStatusActive {
				return nil, domain.ErrInvalidInput
			}
		}
		row := tx.QueryRow(ctx, `
			INSERT INTO student_course_registrations (
				branch_id, student_id, course_id, teacher_id, monthly_amount_cents, start_date
			)
			VALUES ($1, $2, $3, nullif($4, '')::uuid, $5, to_date($6, 'DD/MM/YYYY'))
			RETURNING id::text, branch_id::text, student_id::text, course_id::text,
				coalesce(teacher_id::text, ''), monthly_amount_cents,
				to_char(start_date, 'DD/MM/YYYY'), created_at
		`, registration.BranchID, student.ID, registration.CourseID, registration.TeacherID, registration.MonthlyAmountCents, registration.StartDate)
		var next domain.StudentCourseRegistration
		if err := row.Scan(&next.ID, &next.BranchID, &next.StudentID, &next.CourseID, &next.TeacherID, &next.MonthlyAmountCents, &next.StartDate, &next.CreatedAt); err != nil {
			return nil, mapPostgresError(err)
		}
		created = append(created, next)
		if registration.TeacherID != "" {
			if _, err := tx.Exec(ctx, `
				UPDATE student_teacher_assignments
				SET valid_to = to_date($4, 'DD/MM/YYYY')
				WHERE branch_id = $1 AND student_id = $2 AND valid_to IS NULL;

				INSERT INTO student_teacher_assignments (branch_id, student_id, teacher_id, valid_from, reason)
				VALUES ($1, $2, $3, to_date($4, 'DD/MM/YYYY'), 'initial')
			`, registration.BranchID, student.ID, registration.TeacherID, registration.StartDate); err != nil {
				return nil, mapPostgresError(err)
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, mapPostgresError(err)
	}

	return created, nil
}

func (p *Postgres) GetStudent(ctx context.Context, id string) (domain.Student, error) {
	row := p.pool.QueryRow(ctx, `
		SELECT id::text, branch_id::text, coalesce(user_id::text, ''), fin_code,
			first_name, last_name, coalesce(to_char(birth_date, 'DD/MM/YYYY'), ''),
			coalesce(gender, ''), phone, address, coalesce(profile_photo_file_id::text, ''),
			status, coalesce(left_reason, ''), created_at, updated_at
		FROM students
		WHERE id = $1
	`, id)

	student, err := scanStudent(row)
	if err != nil {
		return domain.Student{}, mapPostgresError(err)
	}

	return student, nil
}

func (p *Postgres) GetStudentByUser(ctx context.Context, userID string) (domain.Student, error) {
	row := p.pool.QueryRow(ctx, `
		SELECT id::text, branch_id::text, coalesce(user_id::text, ''), fin_code,
			first_name, last_name, coalesce(to_char(birth_date, 'DD/MM/YYYY'), ''),
			coalesce(gender, ''), phone, address, coalesce(profile_photo_file_id::text, ''),
			status, coalesce(left_reason, ''), created_at, updated_at
		FROM students
		WHERE user_id = $1
	`, userID)

	student, err := scanStudent(row)
	if err != nil {
		return domain.Student{}, mapPostgresError(err)
	}

	return student, nil
}

func (p *Postgres) GetStudentByFIN(ctx context.Context, fin domain.FIN) (domain.Student, error) {
	row := p.pool.QueryRow(ctx, `
		SELECT id::text, branch_id::text, coalesce(user_id::text, ''), fin_code,
			first_name, last_name, coalesce(to_char(birth_date, 'DD/MM/YYYY'), ''),
			coalesce(gender, ''), phone, address, coalesce(profile_photo_file_id::text, ''),
			status, coalesce(left_reason, ''), created_at, updated_at
		FROM students
		WHERE fin_code = $1
	`, fin.String())

	student, err := scanStudent(row)
	if err != nil {
		return domain.Student{}, mapPostgresError(err)
	}

	return student, nil
}

func (p *Postgres) UpdateStudent(ctx context.Context, student domain.Student) (domain.Student, error) {
	if student.ID == "" || student.BranchID == "" || student.FIN == "" || student.FirstName == "" || student.LastName == "" {
		return domain.Student{}, domain.ErrInvalidInput
	}
	if student.UserID != "" {
		user, err := p.GetUser(ctx, student.UserID)
		if err != nil {
			return domain.Student{}, err
		}
		if user.BranchID != student.BranchID || user.Role != domain.RoleStudent {
			return domain.Student{}, domain.ErrInvalidInput
		}
	}

	row := p.pool.QueryRow(ctx, `
		UPDATE students
		SET user_id = nullif($2, '')::uuid,
			first_name = $3,
			last_name = $4,
			birth_date = CASE WHEN $5 = '' THEN NULL ELSE to_date($5, 'DD/MM/YYYY') END,
			gender = nullif($6, ''),
			phone = $7,
			address = $8,
			profile_photo_file_id = nullif($9, '')::uuid,
			status = $10,
			left_reason = nullif($11, ''),
			updated_at = now()
		WHERE id = $1 AND branch_id = $12
		RETURNING id::text, branch_id::text, coalesce(user_id::text, ''), fin_code,
			first_name, last_name, coalesce(to_char(birth_date, 'DD/MM/YYYY'), ''),
			coalesce(gender, ''), phone, address, coalesce(profile_photo_file_id::text, ''),
			status, coalesce(left_reason, ''), created_at, updated_at
	`, student.ID, student.UserID, strings.TrimSpace(student.FirstName), strings.TrimSpace(student.LastName), strings.TrimSpace(student.BirthDate), strings.TrimSpace(student.Gender), strings.TrimSpace(student.Phone), strings.TrimSpace(student.Address), strings.TrimSpace(student.ProfilePhotoFileID), student.Status, strings.TrimSpace(student.LeftReason), student.BranchID)

	updated, err := scanStudent(row)
	if err != nil {
		return domain.Student{}, mapPostgresError(err)
	}

	return updated, nil
}

func (p *Postgres) ListStudents(ctx context.Context, branchID string) ([]domain.Student, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id::text, branch_id::text, coalesce(user_id::text, ''), fin_code,
			first_name, last_name, coalesce(to_char(birth_date, 'DD/MM/YYYY'), ''),
			coalesce(gender, ''), phone, address, coalesce(profile_photo_file_id::text, ''),
			status, coalesce(left_reason, ''), created_at, updated_at
		FROM students
		WHERE ($1 = '' OR branch_id = $1::uuid)
		ORDER BY last_name, first_name
	`, branchID)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	students := make([]domain.Student, 0)
	for rows.Next() {
		student, err := scanStudent(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		students = append(students, student)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return students, nil
}

func (p *Postgres) ListStudentAssignmentHub(ctx context.Context, filter domain.StudentAssignmentHubFilter) (domain.StudentAssignmentHubPage, error) {
	rows, err := p.pool.Query(ctx, `
		WITH student_records AS (
			SELECT
				s.id::text,
				s.branch_id::text,
				b.name AS branch_name,
				coalesce(s.user_id::text, '') AS user_id,
				s.fin_code,
				s.first_name,
				s.last_name,
				coalesce(s.profile_photo_file_id::text, '') AS profile_photo_file_id,
				s.status,
				coalesce(direct_teacher.teacher_id::text, class_teacher.teacher_id::text, '') AS active_teacher_id,
				coalesce(direct_teacher.first_name, class_teacher.first_name, '') AS active_teacher_first_name,
				coalesce(direct_teacher.last_name, class_teacher.last_name, '') AS active_teacher_last_name,
				s.created_at AS registered_at,
				s.created_at,
				s.updated_at
			FROM students s
			JOIN branches b ON b.id = s.branch_id
			LEFT JOIN LATERAL (
				SELECT sta.teacher_id, u.first_name, u.last_name
				FROM student_teacher_assignments sta
				JOIN teachers t ON t.id = sta.teacher_id
				JOIN users u ON u.id = t.user_id
				WHERE sta.student_id = s.id
					AND sta.branch_id = s.branch_id
					AND sta.valid_from <= current_date
					AND (sta.valid_to IS NULL OR sta.valid_to >= current_date)
				ORDER BY sta.valid_from DESC, sta.created_at DESC
				LIMIT 1
			) direct_teacher ON true
			LEFT JOIN LATERAL (
				SELECT c.teacher_id, u.first_name, u.last_name
				FROM class_students cs
				JOIN classes c ON c.id = cs.class_id
				JOIN teachers t ON t.id = c.teacher_id
				JOIN users u ON u.id = t.user_id
				WHERE cs.student_id = s.id
					AND cs.branch_id = s.branch_id
					AND cs.left_at IS NULL
					AND c.is_active = true
				ORDER BY cs.joined_at DESC, cs.created_at DESC
				LIMIT 1
			) class_teacher ON true
			WHERE ($1 = '' OR s.branch_id = $1::uuid)
		)
		SELECT
			*,
			count(*) OVER()::integer AS total_count
		FROM student_records
		WHERE ($2 = '' OR status = $2)
			AND ($3 = '' OR active_teacher_id = $3)
			AND (
				$4 = ''
				OR lower(first_name) LIKE '%' || lower($4) || '%'
				OR lower(last_name) LIKE '%' || lower($4) || '%'
				OR lower(first_name || ' ' || last_name) LIKE '%' || lower($4) || '%'
				OR lower(fin_code) LIKE '%' || lower($4) || '%'
			)
		ORDER BY created_at DESC, last_name, first_name
		LIMIT $5 OFFSET $6
	`, filter.BranchID, string(filter.Status), filter.TeacherID, strings.TrimSpace(filter.Query), filter.Limit, filter.Offset)
	if err != nil {
		return domain.StudentAssignmentHubPage{}, mapPostgresError(err)
	}
	defer rows.Close()

	items := make([]domain.StudentAssignmentHubRecord, 0)
	total := 0
	for rows.Next() {
		record, rowTotal, err := scanStudentAssignmentHubRecord(rows)
		if err != nil {
			return domain.StudentAssignmentHubPage{}, mapPostgresError(err)
		}
		total = rowTotal
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return domain.StudentAssignmentHubPage{}, mapPostgresError(err)
	}

	return domain.StudentAssignmentHubPage{
		Items:  items,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}, nil
}

func (p *Postgres) CreateTeacher(ctx context.Context, teacher domain.Teacher) (domain.Teacher, error) {
	user, err := p.GetUser(ctx, teacher.UserID)
	if err != nil {
		return domain.Teacher{}, err
	}
	if user.Role != domain.RoleTeacher || user.BranchID != teacher.BranchID {
		return domain.Teacher{}, domain.ErrInvalidInput
	}
	if teacher.Status == "" {
		teacher.Status = domain.TeacherStatusPending
	}

	row := p.pool.QueryRow(ctx, `
		INSERT INTO teachers (branch_id, user_id, status, birth_date, gender, phone, address, profile_photo_file_id)
		VALUES ($1, $2, $3, CASE WHEN $4 = '' THEN NULL ELSE to_date($4, 'DD/MM/YYYY') END, nullif($5, ''), $6, $7, nullif($8, '')::uuid)
		RETURNING id::text, branch_id::text, user_id::text, status,
			coalesce(to_char(birth_date, 'DD/MM/YYYY'), ''),
			coalesce(gender, ''),
			coalesce(phone, ''),
			coalesce(address, ''),
			coalesce(profile_photo_file_id::text, ''),
			created_at, updated_at
	`, teacher.BranchID, teacher.UserID, teacher.Status, strings.TrimSpace(teacher.BirthDate), strings.TrimSpace(teacher.Gender), strings.TrimSpace(teacher.Phone), strings.TrimSpace(teacher.Address), strings.TrimSpace(teacher.ProfilePhotoFileID))

	created, err := scanTeacher(row)
	if err != nil {
		return domain.Teacher{}, mapPostgresError(err)
	}

	return created, nil
}

func (p *Postgres) GetTeacher(ctx context.Context, id string) (domain.Teacher, error) {
	row := p.pool.QueryRow(ctx, `
		SELECT id::text, branch_id::text, user_id::text, status,
			coalesce(to_char(birth_date, 'DD/MM/YYYY'), ''),
			coalesce(gender, ''),
			coalesce(phone, ''),
			coalesce(address, ''),
			coalesce(profile_photo_file_id::text, ''),
			created_at, updated_at
		FROM teachers
		WHERE id = $1
	`, id)

	teacher, err := scanTeacher(row)
	if err != nil {
		return domain.Teacher{}, mapPostgresError(err)
	}

	return teacher, nil
}

func (p *Postgres) GetTeacherByUser(ctx context.Context, userID string) (domain.Teacher, error) {
	row := p.pool.QueryRow(ctx, `
		SELECT id::text, branch_id::text, user_id::text, status,
			coalesce(to_char(birth_date, 'DD/MM/YYYY'), ''),
			coalesce(gender, ''),
			coalesce(phone, ''),
			coalesce(address, ''),
			coalesce(profile_photo_file_id::text, ''),
			created_at, updated_at
		FROM teachers
		WHERE user_id = $1
	`, userID)

	teacher, err := scanTeacher(row)
	if err != nil {
		return domain.Teacher{}, mapPostgresError(err)
	}

	return teacher, nil
}

func (p *Postgres) UpdateTeacher(ctx context.Context, teacher domain.Teacher) (domain.Teacher, error) {
	row := p.pool.QueryRow(ctx, `
		UPDATE teachers
		SET status = $2,
			birth_date = CASE WHEN $3 = '' THEN NULL ELSE to_date($3, 'DD/MM/YYYY') END,
			gender = nullif($4, ''),
			phone = $5,
			address = $6,
			profile_photo_file_id = nullif($7, '')::uuid,
			updated_at = now()
		WHERE id = $1
		RETURNING id::text, branch_id::text, user_id::text, status,
			coalesce(to_char(birth_date, 'DD/MM/YYYY'), ''),
			coalesce(gender, ''),
			coalesce(phone, ''),
			coalesce(address, ''),
			coalesce(profile_photo_file_id::text, ''),
			created_at, updated_at
	`, teacher.ID, teacher.Status, strings.TrimSpace(teacher.BirthDate), strings.TrimSpace(teacher.Gender), strings.TrimSpace(teacher.Phone), strings.TrimSpace(teacher.Address), strings.TrimSpace(teacher.ProfilePhotoFileID))

	updated, err := scanTeacher(row)
	if err != nil {
		return domain.Teacher{}, mapPostgresError(err)
	}

	return updated, nil
}

func (p *Postgres) UpdateTeacherAccount(ctx context.Context, teacher domain.Teacher, user domain.User, passwordHash string) (domain.Teacher, domain.User, error) {
	if teacher.ID == "" || teacher.BranchID == "" || user.Email == "" || user.FirstName == "" || user.LastName == "" {
		return domain.Teacher{}, domain.User{}, domain.ErrInvalidInput
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return domain.Teacher{}, domain.User{}, mapPostgresError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	current, err := scanTeacher(tx.QueryRow(ctx, `
		SELECT id::text, branch_id::text, user_id::text, status,
			coalesce(to_char(birth_date, 'DD/MM/YYYY'), ''),
			coalesce(gender, ''),
			coalesce(phone, ''),
			coalesce(address, ''),
			coalesce(profile_photo_file_id::text, ''),
			created_at, updated_at
		FROM teachers
		WHERE id = $1
	`, teacher.ID))
	if err != nil {
		return domain.Teacher{}, domain.User{}, mapPostgresError(err)
	}

	updatedUser, err := scanUser(tx.QueryRow(ctx, `
		UPDATE users
		SET branch_id = $2,
			email = $3,
			first_name = $4,
			last_name = $5,
			password_hash = CASE WHEN $6 = '' THEN password_hash ELSE $6 END,
			updated_at = now()
		WHERE id = $1
		RETURNING id::text, coalesce(branch_id::text, ''), role, email, password_hash, first_name, last_name, is_active, coalesce(last_login_at::text, ''), created_at, updated_at
	`, current.UserID, teacher.BranchID, normalizeEmail(user.Email), strings.TrimSpace(user.FirstName), strings.TrimSpace(user.LastName), strings.TrimSpace(passwordHash)))
	if err != nil {
		return domain.Teacher{}, domain.User{}, mapPostgresError(err)
	}

	updatedTeacher, err := scanTeacher(tx.QueryRow(ctx, `
		UPDATE teachers
		SET branch_id = $2,
			status = $3,
			birth_date = CASE WHEN $4 = '' THEN NULL ELSE to_date($4, 'DD/MM/YYYY') END,
			gender = nullif($5, ''),
			phone = $6,
			address = $7,
			profile_photo_file_id = nullif($8, '')::uuid,
			updated_at = now()
		WHERE id = $1
		RETURNING id::text, branch_id::text, user_id::text, status,
			coalesce(to_char(birth_date, 'DD/MM/YYYY'), ''),
			coalesce(gender, ''),
			coalesce(phone, ''),
			coalesce(address, ''),
			coalesce(profile_photo_file_id::text, ''),
			created_at, updated_at
	`, teacher.ID, teacher.BranchID, teacher.Status, strings.TrimSpace(teacher.BirthDate), strings.TrimSpace(teacher.Gender), strings.TrimSpace(teacher.Phone), strings.TrimSpace(teacher.Address), strings.TrimSpace(teacher.ProfilePhotoFileID)))
	if err != nil {
		return domain.Teacher{}, domain.User{}, mapPostgresError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Teacher{}, domain.User{}, mapPostgresError(err)
	}

	return updatedTeacher, updatedUser, nil
}

func (p *Postgres) DeleteTeacher(ctx context.Context, id string) (domain.Teacher, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return domain.Teacher{}, mapPostgresError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	teacher, err := scanTeacher(tx.QueryRow(ctx, `
		SELECT id::text, branch_id::text, user_id::text, status,
			coalesce(to_char(birth_date, 'DD/MM/YYYY'), ''),
			coalesce(gender, ''),
			coalesce(phone, ''),
			coalesce(address, ''),
			coalesce(profile_photo_file_id::text, ''),
			created_at, updated_at
		FROM teachers
		WHERE id = $1
	`, id))
	if err != nil {
		return domain.Teacher{}, mapPostgresError(err)
	}

	exec := func(statement string, args ...any) error {
		_, err := tx.Exec(ctx, statement, args...)
		if err != nil {
			return mapPostgresError(err)
		}
		return nil
	}

	userID := teacher.UserID
	teacherID := teacher.ID
	teacherFileFilter := `
		SELECT id
		FROM files
		WHERE uploader_user_id = $1
			OR (owner_type = 'teacher' AND owner_id = $2)
	`

	if err := exec(`DELETE FROM notifications WHERE recipient_user_id = $1`, userID); err != nil {
		return domain.Teacher{}, err
	}
	if err := exec(`
		DELETE FROM exam_results
		WHERE entered_by_teacher_id = $1
			OR exam_id IN (
				SELECT e.id
				FROM exams e
				WHERE e.created_by_user_id = $2
					OR e.class_id IN (SELECT c.id FROM classes c WHERE c.teacher_id = $1)
					OR e.schedule_item_id IN (
						SELECT si.id
						FROM schedule_items si
						WHERE si.teacher_id = $1 OR si.created_by_user_id = $2
					)
			)
	`, teacherID, userID); err != nil {
		return domain.Teacher{}, err
	}
	if err := exec(`
		DELETE FROM exam_participants
		WHERE exam_id IN (
			SELECT e.id
			FROM exams e
			WHERE e.created_by_user_id = $2
				OR e.class_id IN (SELECT c.id FROM classes c WHERE c.teacher_id = $1)
				OR e.schedule_item_id IN (
					SELECT si.id
					FROM schedule_items si
					WHERE si.teacher_id = $1 OR si.created_by_user_id = $2
				)
		)
	`, teacherID, userID); err != nil {
		return domain.Teacher{}, err
	}
	if err := exec(`
		DELETE FROM exams
		WHERE created_by_user_id = $2
			OR class_id IN (SELECT c.id FROM classes c WHERE c.teacher_id = $1)
			OR schedule_item_id IN (
				SELECT si.id
				FROM schedule_items si
				WHERE si.teacher_id = $1 OR si.created_by_user_id = $2
			)
	`, teacherID, userID); err != nil {
		return domain.Teacher{}, err
	}
	if err := exec(`
		DELETE FROM assignments
		WHERE created_by_user_id = $2
			OR class_id IN (SELECT c.id FROM classes c WHERE c.teacher_id = $1)
	`, teacherID, userID); err != nil {
		return domain.Teacher{}, err
	}
	if err := exec(`DELETE FROM payments WHERE created_by_user_id = $1`, userID); err != nil {
		return domain.Teacher{}, err
	}
	if err := exec(`
		DELETE FROM schedule_items
		WHERE teacher_id = $1
			OR created_by_user_id = $2
			OR class_id IN (SELECT c.id FROM classes c WHERE c.teacher_id = $1)
	`, teacherID, userID); err != nil {
		return domain.Teacher{}, err
	}
	if err := exec(`
		DELETE FROM class_students
		WHERE class_id IN (SELECT c.id FROM classes c WHERE c.teacher_id = $1)
	`, teacherID); err != nil {
		return domain.Teacher{}, err
	}
	if err := exec(`
		DELETE FROM student_teacher_assignments
		WHERE teacher_id = $1
			OR class_id IN (SELECT c.id FROM classes c WHERE c.teacher_id = $1)
	`, teacherID); err != nil {
		return domain.Teacher{}, err
	}
	if err := exec(`DELETE FROM salary_models WHERE teacher_id = $1`, teacherID); err != nil {
		return domain.Teacher{}, err
	}
	if err := exec(`DELETE FROM teacher_course_specializations WHERE teacher_id = $1`, teacherID); err != nil {
		return domain.Teacher{}, err
	}
	if err := exec(`DELETE FROM classes WHERE teacher_id = $1`, teacherID); err != nil {
		return domain.Teacher{}, err
	}
	if err := exec(`UPDATE exam_results SET document_file_id = NULL WHERE document_file_id IN (`+teacherFileFilter+`)`, userID, teacherID); err != nil {
		return domain.Teacher{}, err
	}
	if err := exec(`UPDATE assignments SET material_file_id = NULL WHERE material_file_id IN (`+teacherFileFilter+`)`, userID, teacherID); err != nil {
		return domain.Teacher{}, err
	}
	if err := exec(`UPDATE payments SET receipt_file_id = NULL WHERE receipt_file_id IN (`+teacherFileFilter+`)`, userID, teacherID); err != nil {
		return domain.Teacher{}, err
	}
	if err := exec(`UPDATE teachers SET profile_photo_file_id = NULL WHERE id = $1`, teacherID); err != nil {
		return domain.Teacher{}, err
	}
	if err := exec(`DELETE FROM files WHERE uploader_user_id = $1 OR (owner_type = 'teacher' AND owner_id = $2)`, userID, teacherID); err != nil {
		return domain.Teacher{}, err
	}
	if err := exec(`DELETE FROM teachers WHERE id = $1`, teacherID); err != nil {
		return domain.Teacher{}, err
	}
	if err := exec(`UPDATE students SET user_id = NULL WHERE user_id = $1`, userID); err != nil {
		return domain.Teacher{}, err
	}
	if err := exec(`DELETE FROM users WHERE id = $1`, userID); err != nil {
		return domain.Teacher{}, err
	}
	if err := exec(`
		DELETE FROM outbox_events
		WHERE payload->>'teacher_id' = $1
			OR payload->>'user_id' = $2
			OR payload->'teacher'->>'id' = $1
			OR payload->'teacher'->>'user_id' = $2
	`, teacherID, userID); err != nil {
		return domain.Teacher{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Teacher{}, mapPostgresError(err)
	}

	return teacher, nil
}

func (p *Postgres) ListTeachers(ctx context.Context, branchID string) ([]domain.Teacher, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id::text, branch_id::text, user_id::text, status,
			coalesce(to_char(birth_date, 'DD/MM/YYYY'), ''),
			coalesce(gender, ''),
			coalesce(phone, ''),
			coalesce(address, ''),
			coalesce(profile_photo_file_id::text, ''),
			created_at, updated_at
		FROM teachers
		WHERE ($1 = '' OR branch_id = $1::uuid)
		ORDER BY created_at, id
	`, branchID)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	teachers := make([]domain.Teacher, 0)
	for rows.Next() {
		teacher, err := scanTeacher(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		teachers = append(teachers, teacher)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return teachers, nil
}

func (p *Postgres) ReplaceTeacherCourseSpecializations(ctx context.Context, teacherID string, courseIDs []string) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return mapPostgresError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var branchID string
	if err := tx.QueryRow(ctx, `SELECT branch_id::text FROM teachers WHERE id = $1`, teacherID).Scan(&branchID); err != nil {
		return mapPostgresError(err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM teacher_course_specializations WHERE teacher_id = $1`, teacherID); err != nil {
		return mapPostgresError(err)
	}

	seen := make(map[string]struct{}, len(courseIDs))
	for _, courseID := range courseIDs {
		courseID = strings.TrimSpace(courseID)
		if courseID == "" {
			continue
		}
		if _, exists := seen[courseID]; exists {
			continue
		}
		seen[courseID] = struct{}{}

		var courseBranchID string
		if err := tx.QueryRow(ctx, `SELECT branch_id::text FROM courses WHERE id = $1`, courseID).Scan(&courseBranchID); err != nil {
			return mapPostgresError(err)
		}
		if courseBranchID != branchID {
			return domain.ErrInvalidInput
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO teacher_course_specializations (teacher_id, course_id, branch_id)
			VALUES ($1, $2, $3)
			ON CONFLICT (teacher_id, course_id) DO NOTHING
		`, teacherID, courseID, branchID); err != nil {
			return mapPostgresError(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return mapPostgresError(err)
	}

	return nil
}

func (p *Postgres) ListTeacherFinanceRecords(ctx context.Context, branchID string) ([]domain.TeacherFinanceRecord, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT
			t.id::text,
			t.branch_id::text,
			b.name,
			t.user_id::text,
			u.email,
			u.first_name,
			u.last_name,
			coalesce(t.profile_photo_file_id::text, ''),
			coalesce(string_agg(DISTINCT c.name, ', '), ''),
			t.status,
			coalesce(latest_salary.model_type, ''),
			CASE
				WHEN latest_salary.model_type IN ('fixed', 'hybrid') THEN coalesce(latest_salary.fixed_monthly_amount_cents, 0)
				ELSE 0
			END,
			coalesce(active_assignments.assigned_students, 0)::integer,
			t.created_at,
			t.updated_at
		FROM teachers t
		JOIN users u ON u.id = t.user_id
		JOIN branches b ON b.id = t.branch_id
		LEFT JOIN teacher_course_specializations tcs ON tcs.teacher_id = t.id
		LEFT JOIN courses c ON c.id = tcs.course_id
		LEFT JOIN LATERAL (
			SELECT model_type, fixed_monthly_amount_cents
			FROM salary_models sm
			WHERE sm.teacher_id = t.id
				AND (sm.active_to IS NULL OR sm.active_to >= current_date)
			ORDER BY sm.active_from DESC, sm.created_at DESC
			LIMIT 1
		) latest_salary ON true
		LEFT JOIN LATERAL (
			SELECT count(DISTINCT sta.student_id) AS assigned_students
			FROM student_teacher_assignments sta
			JOIN students s ON s.id = sta.student_id
			WHERE sta.teacher_id = t.id
				AND sta.branch_id = t.branch_id
				AND (sta.valid_to IS NULL OR sta.valid_to >= current_date)
				AND s.status = 'active'
		) active_assignments ON true
		WHERE ($1 = '' OR t.branch_id = $1::uuid)
		GROUP BY
			t.id,
			b.name,
			u.id,
			latest_salary.model_type,
			latest_salary.fixed_monthly_amount_cents,
			active_assignments.assigned_students
		ORDER BY u.last_name, u.first_name
	`, branchID)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	records := make([]domain.TeacherFinanceRecord, 0)
	for rows.Next() {
		record, err := scanTeacherFinanceRecord(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return records, nil
}

func (p *Postgres) CreateCourse(ctx context.Context, course domain.Course, categories []domain.ScoreCategory) (domain.Course, []domain.ScoreCategory, error) {
	if strings.TrimSpace(course.Name) == "" {
		return domain.Course{}, nil, domain.ErrInvalidInput
	}
	for _, category := range categories {
		if category.Name == "" || category.MinScore > category.MaxScore {
			return domain.Course{}, nil, domain.ErrInvalidInput
		}
		if category.RequiresDocument && category.DocumentPurpose == "" {
			return domain.Course{}, nil, domain.ErrInvalidInput
		}
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return domain.Course{}, nil, mapPostgresError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row := tx.QueryRow(ctx, `
		INSERT INTO courses (branch_id, name, is_active)
		VALUES ($1, $2, true)
		RETURNING id::text, branch_id::text, name, is_active, created_at, updated_at
	`, course.BranchID, strings.TrimSpace(course.Name))

	created, err := scanCourse(row)
	if err != nil {
		return domain.Course{}, nil, mapPostgresError(err)
	}

	createdCategories := make([]domain.ScoreCategory, 0, len(categories))
	for i, category := range categories {
		row := tx.QueryRow(ctx, `
			INSERT INTO course_score_categories (
				branch_id, course_id, name, min_score, max_score, requires_feedback,
				requires_document, document_purpose, sort_order
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, nullif($8, ''), $9)
			RETURNING id::text, branch_id::text, course_id::text, name, min_score, max_score,
				requires_feedback, requires_document, coalesce(document_purpose, ''), sort_order, created_at
		`, course.BranchID, created.ID, strings.TrimSpace(category.Name), category.MinScore, category.MaxScore, category.RequiresFeedback, category.RequiresDocument, strings.TrimSpace(category.DocumentPurpose), i)

		createdCategory, err := scanScoreCategory(row)
		if err != nil {
			return domain.Course{}, nil, mapPostgresError(err)
		}
		createdCategories = append(createdCategories, createdCategory)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Course{}, nil, mapPostgresError(err)
	}

	return created, createdCategories, nil
}

func (p *Postgres) GetCourse(ctx context.Context, id string) (domain.Course, error) {
	row := p.pool.QueryRow(ctx, `
		SELECT id::text, branch_id::text, name, is_active, created_at, updated_at
		FROM courses
		WHERE id = $1
	`, id)

	course, err := scanCourse(row)
	if err != nil {
		return domain.Course{}, mapPostgresError(err)
	}

	return course, nil
}

func (p *Postgres) ListCourses(ctx context.Context, branchID string) ([]domain.Course, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id::text, branch_id::text, name, is_active, created_at, updated_at
		FROM courses
		WHERE ($1 = '' OR branch_id = $1::uuid)
		ORDER BY name
	`, branchID)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	courses := make([]domain.Course, 0)
	for rows.Next() {
		course, err := scanCourse(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		courses = append(courses, course)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return courses, nil
}

func (p *Postgres) CreateRoom(ctx context.Context, room domain.Room) (domain.Room, error) {
	if strings.TrimSpace(room.Name) == "" {
		return domain.Room{}, domain.ErrInvalidInput
	}
	if room.Capacity <= 0 {
		room.Capacity = 1
	}

	row := p.pool.QueryRow(ctx, `
		INSERT INTO rooms (branch_id, name, capacity, is_active)
		VALUES ($1, $2, $3, true)
		RETURNING id::text, branch_id::text, name, capacity, is_active, created_at, updated_at
	`, room.BranchID, strings.TrimSpace(room.Name), room.Capacity)

	created, err := scanRoom(row)
	if err != nil {
		return domain.Room{}, mapPostgresError(err)
	}

	return created, nil
}

func (p *Postgres) UpdateRoom(ctx context.Context, room domain.Room) (domain.Room, error) {
	if strings.TrimSpace(room.Name) == "" {
		return domain.Room{}, domain.ErrInvalidInput
	}
	if room.Capacity <= 0 {
		room.Capacity = 1
	}

	row := p.pool.QueryRow(ctx, `
		UPDATE rooms
		SET name = $2, capacity = $3, updated_at = now()
		WHERE id = $1 AND is_active = true
		RETURNING id::text, branch_id::text, name, capacity, is_active, created_at, updated_at
	`, room.ID, strings.TrimSpace(room.Name), room.Capacity)

	updated, err := scanRoom(row)
	if err != nil {
		return domain.Room{}, mapPostgresError(err)
	}

	return updated, nil
}

func (p *Postgres) ListRooms(ctx context.Context, branchID string) ([]domain.Room, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id::text, branch_id::text, name, capacity, is_active, created_at, updated_at
		FROM rooms
		WHERE is_active = true AND ($1 = '' OR branch_id = $1::uuid)
		ORDER BY name
	`, branchID)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	rooms := make([]domain.Room, 0)
	for rows.Next() {
		room, err := scanRoom(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		rooms = append(rooms, room)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return rooms, nil
}

func (p *Postgres) GetRoom(ctx context.Context, id string) (domain.Room, error) {
	row := p.pool.QueryRow(ctx, `
		SELECT id::text, branch_id::text, name, capacity, is_active, created_at, updated_at
		FROM rooms
		WHERE id = $1
	`, id)

	room, err := scanRoom(row)
	if err != nil {
		return domain.Room{}, mapPostgresError(err)
	}

	return room, nil
}

func (p *Postgres) DeactivateRoom(ctx context.Context, id string) (domain.Room, error) {
	row := p.pool.QueryRow(ctx, `
		UPDATE rooms
		SET is_active = false, updated_at = now()
		WHERE id = $1
		RETURNING id::text, branch_id::text, name, capacity, is_active, created_at, updated_at
	`, id)

	room, err := scanRoom(row)
	if err != nil {
		return domain.Room{}, mapPostgresError(err)
	}

	return room, nil
}

func (p *Postgres) CreateScheduleItem(ctx context.Context, item domain.ScheduleItem) (domain.ScheduleItem, error) {
	if item.Title == "" || !item.Range().IsValid() {
		return domain.ScheduleItem{}, domain.ErrInvalidInput
	}
	if item.ItemType == "" {
		item.ItemType = domain.ScheduleItemLesson
	}

	room, err := p.roomBranch(ctx, item.RoomID)
	if err != nil {
		return domain.ScheduleItem{}, err
	}
	teacher, err := p.GetTeacher(ctx, item.TeacherID)
	if err != nil {
		return domain.ScheduleItem{}, err
	}
	if room != item.BranchID || teacher.BranchID != item.BranchID {
		return domain.ScheduleItem{}, domain.ErrInvalidInput
	}
	if teacher.Status != domain.TeacherStatusActive {
		return domain.ScheduleItem{}, domain.ErrInvalidInput
	}

	row := p.pool.QueryRow(ctx, `
		INSERT INTO schedule_items (
			branch_id, class_id, teacher_id, room_id, item_type, title,
			starts_at, ends_at, created_by_user_id
		)
		VALUES ($1, nullif($2, '')::uuid, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id::text, branch_id::text, coalesce(class_id::text, ''), teacher_id::text, room_id::text,
			item_type, title, starts_at, ends_at, created_by_user_id::text, cancelled_at, created_at, updated_at
	`, item.BranchID, item.ClassID, item.TeacherID, item.RoomID, item.ItemType, item.Title, item.StartsAt, item.EndsAt, item.CreatedByUserID)

	created, err := scanScheduleItem(row)
	if err != nil {
		return domain.ScheduleItem{}, mapPostgresError(err)
	}

	return created, nil
}

func (p *Postgres) ListSchedule(ctx context.Context, branchID string) ([]domain.ScheduleItem, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id::text, branch_id::text, coalesce(class_id::text, ''), teacher_id::text, room_id::text,
			item_type, title, starts_at, ends_at, created_by_user_id::text, cancelled_at, created_at, updated_at
		FROM schedule_items
		WHERE ($1 = '' OR branch_id = $1::uuid)
		ORDER BY starts_at
	`, branchID)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	items := make([]domain.ScheduleItem, 0)
	for rows.Next() {
		item, err := scanScheduleItem(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return items, nil
}

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

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return domain.Payment{}, mapPostgresError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row := tx.QueryRow(ctx, `
		INSERT INTO payments (
			branch_id, student_id, amount_cents, currency, due_date, paid_at,
			status, receipt_file_id, created_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, nullif($8, '')::uuid, $9)
		RETURNING id::text, branch_id::text, student_id::text, amount_cents, currency, due_date,
			paid_at, status, coalesce(receipt_file_id::text, ''), created_by_user_id::text, created_at, updated_at
	`, payment.BranchID, payment.StudentID, payment.AmountCents, strings.ToUpper(payment.Currency), payment.DueDate, payment.PaidAt, payment.Status, payment.ReceiptFileID, payment.CreatedByUserID)

	created, err := scanPayment(row)
	if err != nil {
		return domain.Payment{}, mapPostgresError(err)
	}
	if err := insertOutboxTx(ctx, tx, "finance.payment.created", created); err != nil {
		return domain.Payment{}, mapPostgresError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Payment{}, mapPostgresError(err)
	}

	return created, nil
}

func (p *Postgres) ListPayments(ctx context.Context, branchID string) ([]domain.Payment, error) {
	rows, err := p.pool.Query(ctx, `
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

func (p *Postgres) GetPayment(ctx context.Context, id string) (domain.Payment, error) {
	row := p.pool.QueryRow(ctx, `
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

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return domain.SalaryModel{}, mapPostgresError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

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

	created, err := scanSalaryModel(row)
	if err != nil {
		return domain.SalaryModel{}, mapPostgresError(err)
	}
	if err := insertOutboxTx(ctx, tx, "finance.salary_model.created", created); err != nil {
		return domain.SalaryModel{}, mapPostgresError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.SalaryModel{}, mapPostgresError(err)
	}

	return created, nil
}

func (p *Postgres) ListSalaryModels(ctx context.Context, branchID string) ([]domain.SalaryModel, error) {
	rows, err := p.pool.Query(ctx, `
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

func (p *Postgres) RegisterFile(ctx context.Context, file domain.FileObject) (domain.FileObject, error) {
	if !file.Category.IsValid() || file.OriginalFilename == "" || file.StorageBucket == "" || file.StorageKey == "" {
		return domain.FileObject{}, domain.ErrInvalidInput
	}
	if file.Category.MustPreserveOriginalBytes() && file.StoredSizeBytes != file.OriginalSizeBytes {
		return domain.FileObject{}, domain.ErrInvalidInput
	}
	createdAt := time.Now().UTC()
	if file.RetentionUntil == nil {
		file.RetentionUntil = domain.RetentionUntil(file.Purpose, createdAt)
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return domain.FileObject{}, mapPostgresError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row := tx.QueryRow(ctx, `
		INSERT INTO files (
			branch_id, uploader_user_id, owner_type, owner_id, category, purpose,
			original_filename, mime_type, original_size_bytes, stored_size_bytes,
			original_sha256, storage_bucket, storage_key, retention_until
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id::text, branch_id::text, uploader_user_id::text, owner_type, owner_id::text,
			category, purpose, original_filename, mime_type, original_size_bytes, stored_size_bytes,
			original_sha256, storage_bucket, storage_key, retention_until, deleted_at, created_at
	`, file.BranchID, file.UploaderUserID, file.OwnerType, file.OwnerID, file.Category, file.Purpose, file.OriginalFilename, file.MimeType, file.OriginalSizeBytes, file.StoredSizeBytes, file.OriginalSHA256, file.StorageBucket, file.StorageKey, file.RetentionUntil)

	created, err := scanFileObject(row)
	if err != nil {
		return domain.FileObject{}, mapPostgresError(err)
	}
	if err := insertOutboxTx(ctx, tx, "files.file.registered", created); err != nil {
		return domain.FileObject{}, mapPostgresError(err)
	}
	if created.RetentionUntil != nil {
		if err := insertOutboxTx(ctx, tx, "files.retention.scheduled", map[string]any{
			"file_id":         created.ID,
			"branch_id":       created.BranchID,
			"retention_until": created.RetentionUntil,
		}); err != nil {
			return domain.FileObject{}, mapPostgresError(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.FileObject{}, mapPostgresError(err)
	}

	return created, nil
}

func (p *Postgres) GetFile(ctx context.Context, id string) (domain.FileObject, error) {
	row := p.pool.QueryRow(ctx, `
		SELECT id::text, branch_id::text, uploader_user_id::text, owner_type, owner_id::text,
			category, purpose, original_filename, mime_type, original_size_bytes, stored_size_bytes,
			original_sha256, storage_bucket, storage_key, retention_until, deleted_at, created_at
		FROM files
		WHERE id = $1
	`, id)

	file, err := scanFileObject(row)
	if err != nil {
		return domain.FileObject{}, mapPostgresError(err)
	}

	return file, nil
}

func (p *Postgres) ListFiles(ctx context.Context, branchID string) ([]domain.FileObject, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id::text, branch_id::text, uploader_user_id::text, owner_type, owner_id::text,
			category, purpose, original_filename, mime_type, original_size_bytes, stored_size_bytes,
			original_sha256, storage_bucket, storage_key, retention_until, deleted_at, created_at
		FROM files
		WHERE deleted_at IS NULL AND ($1 = '' OR branch_id = $1::uuid)
		ORDER BY created_at DESC
	`, branchID)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	files := make([]domain.FileObject, 0)
	for rows.Next() {
		file, err := scanFileObject(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return files, nil
}

func (p *Postgres) ListExpiredFiles(ctx context.Context, now time.Time, limit int) ([]domain.FileObject, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id::text, branch_id::text, uploader_user_id::text, owner_type, owner_id::text,
			category, purpose, original_filename, mime_type, original_size_bytes, stored_size_bytes,
			original_sha256, storage_bucket, storage_key, retention_until, deleted_at, created_at
		FROM files
		WHERE deleted_at IS NULL
			AND retention_until IS NOT NULL
			AND retention_until <= $1
		ORDER BY retention_until
		LIMIT $2
	`, now, limit)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	files := make([]domain.FileObject, 0)
	for rows.Next() {
		file, err := scanFileObject(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return files, nil
}

func (p *Postgres) MarkFileDeleted(ctx context.Context, id string, deletedAt time.Time) (domain.FileObject, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return domain.FileObject{}, mapPostgresError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row := tx.QueryRow(ctx, `
		UPDATE files
		SET deleted_at = $2
		WHERE id = $1
		RETURNING id::text, branch_id::text, uploader_user_id::text, owner_type, owner_id::text,
			category, purpose, original_filename, mime_type, original_size_bytes, stored_size_bytes,
			original_sha256, storage_bucket, storage_key, retention_until, deleted_at, created_at
	`, id, deletedAt)

	file, err := scanFileObject(row)
	if err != nil {
		return domain.FileObject{}, mapPostgresError(err)
	}
	if err := insertOutboxTx(ctx, tx, "files.retention.deleted", file); err != nil {
		return domain.FileObject{}, mapPostgresError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.FileObject{}, mapPostgresError(err)
	}

	return file, nil
}

func (p *Postgres) roomBranch(ctx context.Context, roomID string) (string, error) {
	var branchID string
	err := p.pool.QueryRow(ctx, `SELECT branch_id::text FROM rooms WHERE id = $1`, roomID).Scan(&branchID)
	if err != nil {
		return "", mapPostgresError(err)
	}

	return branchID, nil
}

func scanBranch(row scanner) (domain.Branch, error) {
	var branch domain.Branch
	err := row.Scan(&branch.ID, &branch.Name, &branch.Slug, &branch.Address, &branch.OpeningTime, &branch.ClosingTime, &branch.CreatedAt, &branch.UpdatedAt)
	return branch, err
}

func scanUser(row scanner) (domain.User, error) {
	var user domain.User
	err := row.Scan(&user.ID, &user.BranchID, &user.Role, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName, &user.IsActive, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

func staffMemberSelect() string {
	return `
		SELECT sp.id::text, sp.branch_id::text, sp.user_id::text, u.role, u.email,
			u.first_name, u.last_name, coalesce(to_char(sp.birth_date, 'DD/MM/YYYY'), ''),
			coalesce(sp.gender, ''), sp.phone, coalesce(sp.address, ''),
			coalesce(to_char(sp.hired_at, 'DD/MM/YYYY'), ''),
			sp.salary_amount_azn, coalesce(sp.profile_photo_file_id::text, ''),
			u.is_active, coalesce(u.last_login_at::text, ''),
			sp.created_at, sp.updated_at
		FROM staff_profiles sp
		JOIN users u ON u.id = sp.user_id
	`
}

func scanStaffMember(row scanner) (domain.StaffMember, error) {
	var staff domain.StaffMember
	err := row.Scan(
		&staff.ID,
		&staff.BranchID,
		&staff.UserID,
		&staff.Role,
		&staff.Email,
		&staff.FirstName,
		&staff.LastName,
		&staff.BirthDate,
		&staff.Gender,
		&staff.Phone,
		&staff.Address,
		&staff.HiredAt,
		&staff.SalaryAmountAZN,
		&staff.ProfilePhotoFileID,
		&staff.IsActive,
		&staff.LastLoginAt,
		&staff.CreatedAt,
		&staff.UpdatedAt,
	)
	return staff, err
}

func scanStudent(row scanner) (domain.Student, error) {
	var student domain.Student
	var fin string
	err := row.Scan(
		&student.ID,
		&student.BranchID,
		&student.UserID,
		&fin,
		&student.FirstName,
		&student.LastName,
		&student.BirthDate,
		&student.Gender,
		&student.Phone,
		&student.Address,
		&student.ProfilePhotoFileID,
		&student.Status,
		&student.LeftReason,
		&student.CreatedAt,
		&student.UpdatedAt,
	)
	if err != nil {
		return domain.Student{}, err
	}
	parsed, err := domain.ParseFIN(fin)
	if err != nil {
		return domain.Student{}, err
	}
	student.FIN = parsed

	return student, nil
}

func scanStudentAssignmentHubRecord(row scanner) (domain.StudentAssignmentHubRecord, int, error) {
	var record domain.StudentAssignmentHubRecord
	var fin string
	var total int
	err := row.Scan(
		&record.ID,
		&record.BranchID,
		&record.BranchName,
		&record.UserID,
		&fin,
		&record.FirstName,
		&record.LastName,
		&record.ProfilePhotoFileID,
		&record.Status,
		&record.ActiveTeacherID,
		&record.ActiveTeacherFirstName,
		&record.ActiveTeacherLastName,
		&record.RegisteredAt,
		&record.CreatedAt,
		&record.UpdatedAt,
		&total,
	)
	if err != nil {
		return domain.StudentAssignmentHubRecord{}, 0, err
	}
	parsed, err := domain.ParseFIN(fin)
	if err != nil {
		return domain.StudentAssignmentHubRecord{}, 0, err
	}
	record.FIN = parsed

	return record, total, nil
}

func scanTeacher(row scanner) (domain.Teacher, error) {
	var teacher domain.Teacher
	err := row.Scan(
		&teacher.ID,
		&teacher.BranchID,
		&teacher.UserID,
		&teacher.Status,
		&teacher.BirthDate,
		&teacher.Gender,
		&teacher.Phone,
		&teacher.Address,
		&teacher.ProfilePhotoFileID,
		&teacher.CreatedAt,
		&teacher.UpdatedAt,
	)
	return teacher, err
}

func scanTeacherFinanceRecord(row scanner) (domain.TeacherFinanceRecord, error) {
	var record domain.TeacherFinanceRecord
	var salaryType string
	err := row.Scan(
		&record.ID,
		&record.BranchID,
		&record.BranchName,
		&record.UserID,
		&record.Email,
		&record.FirstName,
		&record.LastName,
		&record.ProfilePhotoFileID,
		&record.Subject,
		&record.Status,
		&salaryType,
		&record.CalculatedSalaryAmountCents,
		&record.AssignedStudents,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return domain.TeacherFinanceRecord{}, err
	}
	record.SalaryType = domain.SalaryModelType(salaryType)

	return record, nil
}

func scanCourse(row scanner) (domain.Course, error) {
	var course domain.Course
	err := row.Scan(&course.ID, &course.BranchID, &course.Name, &course.IsActive, &course.CreatedAt, &course.UpdatedAt)
	return course, err
}

func scanScoreCategory(row scanner) (domain.ScoreCategory, error) {
	var category domain.ScoreCategory
	err := row.Scan(
		&category.ID,
		&category.BranchID,
		&category.CourseID,
		&category.Name,
		&category.MinScore,
		&category.MaxScore,
		&category.RequiresFeedback,
		&category.RequiresDocument,
		&category.DocumentPurpose,
		&category.SortOrder,
		&category.CreatedAt,
	)
	return category, err
}

func scanRoom(row scanner) (domain.Room, error) {
	var room domain.Room
	err := row.Scan(&room.ID, &room.BranchID, &room.Name, &room.Capacity, &room.IsActive, &room.CreatedAt, &room.UpdatedAt)
	return room, err
}

func scanScheduleItem(row scanner) (domain.ScheduleItem, error) {
	var item domain.ScheduleItem
	var cancelledAt sql.NullTime
	err := row.Scan(
		&item.ID,
		&item.BranchID,
		&item.ClassID,
		&item.TeacherID,
		&item.RoomID,
		&item.ItemType,
		&item.Title,
		&item.StartsAt,
		&item.EndsAt,
		&item.CreatedByUserID,
		&cancelledAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return domain.ScheduleItem{}, err
	}
	if cancelledAt.Valid {
		item.CancelledAt = &cancelledAt.Time
	}

	return item, nil
}

func scanPayment(row scanner) (domain.Payment, error) {
	var payment domain.Payment
	var paidAt sql.NullTime
	err := row.Scan(
		&payment.ID,
		&payment.BranchID,
		&payment.StudentID,
		&payment.AmountCents,
		&payment.Currency,
		&payment.DueDate,
		&paidAt,
		&payment.Status,
		&payment.ReceiptFileID,
		&payment.CreatedByUserID,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		return domain.Payment{}, err
	}
	if paidAt.Valid {
		payment.PaidAt = &paidAt.Time
	}

	return payment, nil
}

func scanSalaryModel(row scanner) (domain.SalaryModel, error) {
	var model domain.SalaryModel
	var activeTo sql.NullTime
	err := row.Scan(
		&model.ID,
		&model.BranchID,
		&model.TeacherID,
		&model.ModelType,
		&model.FixedMonthlyAmountCents,
		&model.StudentPercentBasisPoints,
		&model.ActiveFrom,
		&activeTo,
		&model.ApprovedByOwnerUserID,
		&model.CreatedAt,
	)
	if err != nil {
		return domain.SalaryModel{}, err
	}
	if activeTo.Valid {
		model.ActiveTo = &activeTo.Time
	}

	return model, nil
}

func scanFileObject(row scanner) (domain.FileObject, error) {
	var file domain.FileObject
	var retentionUntil sql.NullTime
	var deletedAt sql.NullTime
	err := row.Scan(
		&file.ID,
		&file.BranchID,
		&file.UploaderUserID,
		&file.OwnerType,
		&file.OwnerID,
		&file.Category,
		&file.Purpose,
		&file.OriginalFilename,
		&file.MimeType,
		&file.OriginalSizeBytes,
		&file.StoredSizeBytes,
		&file.OriginalSHA256,
		&file.StorageBucket,
		&file.StorageKey,
		&retentionUntil,
		&deletedAt,
		&file.CreatedAt,
	)
	if err != nil {
		return domain.FileObject{}, err
	}
	if retentionUntil.Valid {
		file.RetentionUntil = &retentionUntil.Time
	}
	if deletedAt.Valid {
		file.DeletedAt = &deletedAt.Time
	}

	return file, nil
}

func scanIdempotencyRecord(row scanner) (domain.IdempotencyRecord, error) {
	var record domain.IdempotencyRecord
	var responseBody string
	err := row.Scan(
		&record.ActorUserID,
		&record.Method,
		&record.Path,
		&record.Key,
		&record.RequestHash,
		&record.Status,
		&record.ResponseStatus,
		&responseBody,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.ExpiresAt,
	)
	if err != nil {
		return domain.IdempotencyRecord{}, err
	}
	if responseBody != "" {
		record.ResponseBody = []byte(responseBody)
	}

	return record, nil
}

func fixedSalaryValue(model domain.SalaryModel) any {
	if model.ModelType != domain.SalaryModelFixed && model.ModelType != domain.SalaryModelHybrid {
		return nil
	}

	return model.FixedMonthlyAmountCents
}

func percentSalaryValue(model domain.SalaryModel) any {
	if model.ModelType != domain.SalaryModelPercent && model.ModelType != domain.SalaryModelHybrid {
		return nil
	}

	return model.StudentPercentBasisPoints
}

func mapPostgresError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505", "23P01":
			return domain.ErrConflict
		case "23503":
			return domain.ErrNotFound
		case "23514", "22P02":
			return domain.ErrInvalidInput
		}
	}

	return err
}

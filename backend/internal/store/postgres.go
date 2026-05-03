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
		RETURNING id::text, coalesce(branch_id::text, ''), role, email, password_hash, first_name, last_name, is_active, created_at, updated_at
	`, user.BranchID, user.Role, user.Email, user.PasswordHash, strings.TrimSpace(user.FirstName), strings.TrimSpace(user.LastName)))
	if err != nil {
		return domain.StaffMember{}, mapPostgresError(err)
	}

	var staffID string
	err = tx.QueryRow(ctx, `
		INSERT INTO staff_profiles (branch_id, user_id, role, birth_date, phone, salary_amount_azn, profile_photo_file_id)
		VALUES ($1, $2, $3, CASE WHEN $4 = '' THEN NULL ELSE to_date($4, 'DD/MM/YYYY') END, $5, $6, nullif($7, '')::uuid)
		RETURNING id::text
	`, createdUser.BranchID, createdUser.ID, domain.RoleReceptionist, strings.TrimSpace(staff.BirthDate), strings.TrimSpace(staff.Phone), staff.SalaryAmountAZN, strings.TrimSpace(staff.ProfilePhotoFileID)).Scan(&staffID)
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
		SET email = $2,
			first_name = $3,
			last_name = $4,
			password_hash = CASE WHEN $5 = '' THEN password_hash ELSE $5 END,
			updated_at = now()
		WHERE id = $1
	`, current.UserID, normalizeEmail(staff.Email), strings.TrimSpace(staff.FirstName), strings.TrimSpace(staff.LastName), strings.TrimSpace(passwordHash))
	if err != nil {
		return domain.StaffMember{}, mapPostgresError(err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE staff_profiles
		SET birth_date = CASE WHEN $2 = '' THEN NULL ELSE to_date($2, 'DD/MM/YYYY') END,
			phone = $3,
			salary_amount_azn = $4,
			profile_photo_file_id = nullif($5, '')::uuid,
			updated_at = now()
		WHERE id = $1
	`, staff.ID, strings.TrimSpace(staff.BirthDate), strings.TrimSpace(staff.Phone), staff.SalaryAmountAZN, strings.TrimSpace(staff.ProfilePhotoFileID))
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
		RETURNING id::text, coalesce(branch_id::text, ''), role, email, password_hash, first_name, last_name, is_active, created_at, updated_at
	`, user.BranchID, user.Role, user.Email, user.PasswordHash, strings.TrimSpace(user.FirstName), strings.TrimSpace(user.LastName))

	created, err := scanUser(row)
	if err != nil {
		return domain.User{}, mapPostgresError(err)
	}

	return created, nil
}

func (p *Postgres) GetUser(ctx context.Context, id string) (domain.User, error) {
	row := p.pool.QueryRow(ctx, `
		SELECT id::text, coalesce(branch_id::text, ''), role, email, password_hash, first_name, last_name, is_active, created_at, updated_at
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
		SELECT id::text, coalesce(branch_id::text, ''), role, email, password_hash, first_name, last_name, is_active, created_at, updated_at
		FROM users
		WHERE email = $1
	`, normalizeEmail(email))

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
		INSERT INTO students (branch_id, user_id, fin_code, first_name, last_name, status, left_reason)
		VALUES ($1, nullif($2, '')::uuid, $3, $4, $5, $6, nullif($7, ''))
		RETURNING id::text, branch_id::text, coalesce(user_id::text, ''), fin_code, first_name, last_name, status, coalesce(left_reason, ''), created_at, updated_at
	`, student.BranchID, student.UserID, student.FIN.String(), strings.TrimSpace(student.FirstName), strings.TrimSpace(student.LastName), student.Status, strings.TrimSpace(student.LeftReason))

	created, err := scanStudent(row)
	if err != nil {
		return domain.Student{}, mapPostgresError(err)
	}

	return created, nil
}

func (p *Postgres) GetStudent(ctx context.Context, id string) (domain.Student, error) {
	row := p.pool.QueryRow(ctx, `
		SELECT id::text, branch_id::text, coalesce(user_id::text, ''), fin_code, first_name, last_name, status, coalesce(left_reason, ''), created_at, updated_at
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
		SELECT id::text, branch_id::text, coalesce(user_id::text, ''), fin_code, first_name, last_name, status, coalesce(left_reason, ''), created_at, updated_at
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
		SELECT id::text, branch_id::text, coalesce(user_id::text, ''), fin_code, first_name, last_name, status, coalesce(left_reason, ''), created_at, updated_at
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
			status = $5,
			left_reason = nullif($6, ''),
			updated_at = now()
		WHERE id = $1 AND branch_id = $7
		RETURNING id::text, branch_id::text, coalesce(user_id::text, ''), fin_code, first_name, last_name, status, coalesce(left_reason, ''), created_at, updated_at
	`, student.ID, student.UserID, strings.TrimSpace(student.FirstName), strings.TrimSpace(student.LastName), student.Status, strings.TrimSpace(student.LeftReason), student.BranchID)

	updated, err := scanStudent(row)
	if err != nil {
		return domain.Student{}, mapPostgresError(err)
	}

	return updated, nil
}

func (p *Postgres) ListStudents(ctx context.Context, branchID string) ([]domain.Student, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id::text, branch_id::text, coalesce(user_id::text, ''), fin_code, first_name, last_name, status, coalesce(left_reason, ''), created_at, updated_at
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
		INSERT INTO teachers (branch_id, user_id, status)
		VALUES ($1, $2, $3)
		RETURNING id::text, branch_id::text, user_id::text, status, created_at, updated_at
	`, teacher.BranchID, teacher.UserID, teacher.Status)

	created, err := scanTeacher(row)
	if err != nil {
		return domain.Teacher{}, mapPostgresError(err)
	}

	return created, nil
}

func (p *Postgres) GetTeacher(ctx context.Context, id string) (domain.Teacher, error) {
	row := p.pool.QueryRow(ctx, `
		SELECT id::text, branch_id::text, user_id::text, status, created_at, updated_at
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
		SELECT id::text, branch_id::text, user_id::text, status, created_at, updated_at
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
		SET status = $2, updated_at = now()
		WHERE id = $1
		RETURNING id::text, branch_id::text, user_id::text, status, created_at, updated_at
	`, teacher.ID, teacher.Status)

	updated, err := scanTeacher(row)
	if err != nil {
		return domain.Teacher{}, mapPostgresError(err)
	}

	return updated, nil
}

func (p *Postgres) ListTeachers(ctx context.Context, branchID string) ([]domain.Teacher, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id::text, branch_id::text, user_id::text, status, created_at, updated_at
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
	err := row.Scan(&user.ID, &user.BranchID, &user.Role, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

func staffMemberSelect() string {
	return `
		SELECT sp.id::text, sp.branch_id::text, sp.user_id::text, u.role, u.email,
			u.first_name, u.last_name, coalesce(to_char(sp.birth_date, 'DD/MM/YYYY'), ''),
			sp.phone, sp.salary_amount_azn, coalesce(sp.profile_photo_file_id::text, ''),
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
		&staff.Phone,
		&staff.SalaryAmountAZN,
		&staff.ProfilePhotoFileID,
		&staff.CreatedAt,
		&staff.UpdatedAt,
	)
	return staff, err
}

func scanStudent(row scanner) (domain.Student, error) {
	var student domain.Student
	var fin string
	err := row.Scan(&student.ID, &student.BranchID, &student.UserID, &fin, &student.FirstName, &student.LastName, &student.Status, &student.LeftReason, &student.CreatedAt, &student.UpdatedAt)
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

func scanTeacher(row scanner) (domain.Teacher, error) {
	var teacher domain.Teacher
	err := row.Scan(&teacher.ID, &teacher.BranchID, &teacher.UserID, &teacher.Status, &teacher.CreatedAt, &teacher.UpdatedAt)
	return teacher, err
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

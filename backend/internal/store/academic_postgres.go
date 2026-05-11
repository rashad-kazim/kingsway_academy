package store

import (
	"context"
	"database/sql"
	"strings"

	"github.com/jackc/pgx/v5"

	"kingsway/backend/internal/domain"
)

func (p *Postgres) CreateClass(ctx context.Context, class domain.Class) (domain.Class, error) {
	if strings.TrimSpace(class.Name) == "" || class.CourseID == "" || class.TeacherID == "" || class.StartDate.IsZero() {
		return domain.Class{}, domain.ErrInvalidInput
	}
	if class.EndDate != nil && class.EndDate.Before(class.StartDate) {
		return domain.Class{}, domain.ErrInvalidInput
	}

	if err := p.ensureCourseTeacherBranch(ctx, class.CourseID, class.TeacherID, class.BranchID); err != nil {
		return domain.Class{}, err
	}

	row := p.queryRow(ctx, `
		INSERT INTO classes (branch_id, course_id, teacher_id, name, start_date, end_date, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, true)
		RETURNING id::text, branch_id::text, course_id::text, teacher_id::text, name,
			start_date, end_date, is_active, created_at, updated_at
	`, class.BranchID, class.CourseID, class.TeacherID, strings.TrimSpace(class.Name), class.StartDate, class.EndDate)

	created, err := scanClass(row)
	if err != nil {
		return domain.Class{}, mapPostgresError(err)
	}

	return created, nil
}

func (p *Postgres) GetClass(ctx context.Context, id string) (domain.Class, error) {
	row := p.queryRow(ctx, `
		SELECT id::text, branch_id::text, course_id::text, teacher_id::text, name,
			start_date, end_date, is_active, created_at, updated_at
		FROM classes
		WHERE id = $1
	`, id)

	class, err := scanClass(row)
	if err != nil {
		return domain.Class{}, mapPostgresError(err)
	}

	return class, nil
}

func (p *Postgres) ListClasses(ctx context.Context, branchID string) ([]domain.Class, error) {
	rows, err := p.query(ctx, `
		SELECT id::text, branch_id::text, course_id::text, teacher_id::text, name,
			start_date, end_date, is_active, created_at, updated_at
		FROM classes
		WHERE ($1 = '' OR branch_id = $1::uuid)
		ORDER BY start_date DESC, name
	`, branchID)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	classes := make([]domain.Class, 0)
	for rows.Next() {
		class, err := scanClass(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		classes = append(classes, class)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return classes, nil
}

func (p *Postgres) ListClassesPage(ctx context.Context, branchID string, active *bool, page domain.PageRequest) ([]domain.Class, int, error) {
	page = normalizePage(page)
	activeSet := active != nil
	activeValue := false
	if active != nil {
		activeValue = *active
	}

	var total int
	if err := p.queryRow(ctx, `
		SELECT count(*)
		FROM classes
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = false OR is_active = $3)
	`, branchID, activeSet, activeValue).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	rows, err := p.query(ctx, `
		SELECT id::text, branch_id::text, course_id::text, teacher_id::text, name,
			start_date, end_date, is_active, created_at, updated_at
		FROM classes
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = false OR is_active = $3)
		ORDER BY start_date DESC, name
		LIMIT $4 OFFSET $5
	`, branchID, activeSet, activeValue, page.Limit, page.Offset)
	if err != nil {
		return nil, 0, mapPostgresError(err)
	}
	defer rows.Close()

	classes := make([]domain.Class, 0, page.Limit)
	for rows.Next() {
		class, err := scanClass(rows)
		if err != nil {
			return nil, 0, mapPostgresError(err)
		}
		classes = append(classes, class)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	return classes, total, nil
}

func (p *Postgres) EnrollStudent(ctx context.Context, enrollment domain.ClassStudent) (domain.ClassStudent, error) {
	if enrollment.ClassID == "" || enrollment.StudentID == "" || enrollment.JoinedAt.IsZero() {
		return domain.ClassStudent{}, domain.ErrInvalidInput
	}
	if enrollment.LeftAt != nil && enrollment.LeftAt.Before(enrollment.JoinedAt) {
		return domain.ClassStudent{}, domain.ErrInvalidInput
	}
	if err := p.ensureClassStudentBranch(ctx, enrollment.ClassID, enrollment.StudentID, enrollment.BranchID); err != nil {
		return domain.ClassStudent{}, err
	}

	row := p.queryRow(ctx, `
		INSERT INTO class_students (branch_id, class_id, student_id, joined_at, left_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, branch_id::text, class_id::text, student_id::text, joined_at, left_at, created_at
	`, enrollment.BranchID, enrollment.ClassID, enrollment.StudentID, enrollment.JoinedAt, enrollment.LeftAt)

	created, err := scanClassStudent(row)
	if err != nil {
		return domain.ClassStudent{}, mapPostgresError(err)
	}

	return created, nil
}

func (p *Postgres) ListClassStudents(ctx context.Context, branchID string, classID string) ([]domain.ClassStudent, error) {
	rows, err := p.query(ctx, `
		SELECT id::text, branch_id::text, class_id::text, student_id::text, joined_at, left_at, created_at
		FROM class_students
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR class_id = $2::uuid)
		ORDER BY joined_at DESC, created_at DESC
	`, branchID, classID)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	enrollments := make([]domain.ClassStudent, 0)
	for rows.Next() {
		enrollment, err := scanClassStudent(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		enrollments = append(enrollments, enrollment)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return enrollments, nil
}

func (p *Postgres) ListClassStudentsPage(ctx context.Context, branchID string, classID string, page domain.PageRequest) ([]domain.ClassStudent, int, error) {
	page = normalizePage(page)

	var total int
	if err := p.queryRow(ctx, `
		SELECT count(*)
		FROM class_students
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR class_id = $2::uuid)
	`, branchID, classID).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	rows, err := p.query(ctx, `
		SELECT id::text, branch_id::text, class_id::text, student_id::text, joined_at, left_at, created_at
		FROM class_students
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR class_id = $2::uuid)
		ORDER BY joined_at DESC, created_at DESC
		LIMIT $3 OFFSET $4
	`, branchID, classID, page.Limit, page.Offset)
	if err != nil {
		return nil, 0, mapPostgresError(err)
	}
	defer rows.Close()

	enrollments := make([]domain.ClassStudent, 0, page.Limit)
	for rows.Next() {
		enrollment, err := scanClassStudent(rows)
		if err != nil {
			return nil, 0, mapPostgresError(err)
		}
		enrollments = append(enrollments, enrollment)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	return enrollments, total, nil
}

func (p *Postgres) CreateAssignment(ctx context.Context, assignment domain.Assignment) (domain.Assignment, error) {
	if strings.TrimSpace(assignment.Title) == "" || assignment.ClassID == "" || assignment.CreatedByUserID == "" {
		return domain.Assignment{}, domain.ErrInvalidInput
	}
	class, err := p.GetClass(ctx, assignment.ClassID)
	if err != nil {
		return domain.Assignment{}, err
	}
	if class.BranchID != assignment.BranchID {
		return domain.Assignment{}, domain.ErrInvalidInput
	}

	row := p.queryRow(ctx, `
		INSERT INTO assignments (
			branch_id, class_id, title, description, due_at, material_file_id, created_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5, nullif($6, '')::uuid, $7)
		RETURNING id::text, branch_id::text, class_id::text, title, description,
			due_at, coalesce(material_file_id::text, ''), created_by_user_id::text, created_at, updated_at
	`, assignment.BranchID, assignment.ClassID, strings.TrimSpace(assignment.Title), strings.TrimSpace(assignment.Description), assignment.DueAt, assignment.MaterialFileID, assignment.CreatedByUserID)

	created, err := scanAssignment(row)
	if err != nil {
		return domain.Assignment{}, mapPostgresError(err)
	}

	return created, nil
}

func (p *Postgres) ListAssignments(ctx context.Context, branchID string, classID string) ([]domain.Assignment, error) {
	rows, err := p.query(ctx, `
		SELECT id::text, branch_id::text, class_id::text, title, description,
			due_at, coalesce(material_file_id::text, ''), created_by_user_id::text, created_at, updated_at
		FROM assignments
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR class_id = $2::uuid)
		ORDER BY coalesce(due_at, created_at) DESC
	`, branchID, classID)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	assignments := make([]domain.Assignment, 0)
	for rows.Next() {
		assignment, err := scanAssignment(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		assignments = append(assignments, assignment)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return assignments, nil
}

func (p *Postgres) ListAssignmentsPage(ctx context.Context, branchID string, classID string, page domain.PageRequest) ([]domain.Assignment, int, error) {
	page = normalizePage(page)

	var total int
	if err := p.queryRow(ctx, `
		SELECT count(*)
		FROM assignments
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR class_id = $2::uuid)
	`, branchID, classID).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	rows, err := p.query(ctx, `
		SELECT id::text, branch_id::text, class_id::text, title, description,
			due_at, coalesce(material_file_id::text, ''), created_by_user_id::text, created_at, updated_at
		FROM assignments
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR class_id = $2::uuid)
		ORDER BY coalesce(due_at, created_at) DESC
		LIMIT $3 OFFSET $4
	`, branchID, classID, page.Limit, page.Offset)
	if err != nil {
		return nil, 0, mapPostgresError(err)
	}
	defer rows.Close()

	assignments := make([]domain.Assignment, 0, page.Limit)
	for rows.Next() {
		assignment, err := scanAssignment(rows)
		if err != nil {
			return nil, 0, mapPostgresError(err)
		}
		assignments = append(assignments, assignment)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	return assignments, total, nil
}

func (p *Postgres) CreateExam(ctx context.Context, exam domain.Exam, participantStudentIDs []string) (domain.Exam, []domain.ExamParticipant, error) {
	if strings.TrimSpace(exam.Title) == "" || exam.CourseID == "" || exam.CreatedByUserID == "" {
		return domain.Exam{}, nil, domain.ErrInvalidInput
	}
	if err := p.ensureCourseBranch(ctx, exam.CourseID, exam.BranchID); err != nil {
		return domain.Exam{}, nil, err
	}
	if exam.ClassID != "" {
		class, err := p.GetClass(ctx, exam.ClassID)
		if err != nil {
			return domain.Exam{}, nil, err
		}
		if class.BranchID != exam.BranchID || class.CourseID != exam.CourseID {
			return domain.Exam{}, nil, domain.ErrInvalidInput
		}
	}

	var created domain.Exam
	participants := make([]domain.ExamParticipant, 0, len(participantStudentIDs))
	err := p.withTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			INSERT INTO exams (branch_id, course_id, class_id, schedule_item_id, title, created_by_user_id)
			VALUES ($1, $2, nullif($3, '')::uuid, nullif($4, '')::uuid, $5, $6)
			RETURNING id::text, branch_id::text, course_id::text, coalesce(class_id::text, ''),
				coalesce(schedule_item_id::text, ''), title, created_by_user_id::text, created_at, updated_at
		`, exam.BranchID, exam.CourseID, exam.ClassID, exam.ScheduleItemID, strings.TrimSpace(exam.Title), exam.CreatedByUserID)

		next, err := scanExam(row)
		if err != nil {
			return mapPostgresError(err)
		}
		created = next

		for _, studentID := range participantStudentIDs {
			studentID = strings.TrimSpace(studentID)
			if studentID == "" {
				continue
			}
			if err := p.ensureStudentBranchTx(ctx, tx, studentID, exam.BranchID); err != nil {
				return err
			}
			row := tx.QueryRow(ctx, `
				INSERT INTO exam_participants (exam_id, student_id, branch_id, rsvp_status)
				VALUES ($1, $2, $3, 'pending')
				ON CONFLICT (exam_id, student_id) DO UPDATE SET rsvp_status = exam_participants.rsvp_status
				RETURNING exam_id::text, student_id::text, branch_id::text, rsvp_status
			`, created.ID, studentID, exam.BranchID)
			participant, err := scanExamParticipant(row)
			if err != nil {
				return mapPostgresError(err)
			}
			participants = append(participants, participant)
		}

		return nil
	})
	if err != nil {
		return domain.Exam{}, nil, err
	}

	return created, participants, nil
}

func (p *Postgres) GetExam(ctx context.Context, id string) (domain.Exam, error) {
	row := p.queryRow(ctx, `
		SELECT id::text, branch_id::text, course_id::text, coalesce(class_id::text, ''),
			coalesce(schedule_item_id::text, ''), title, created_by_user_id::text, created_at, updated_at
		FROM exams
		WHERE id = $1
	`, id)

	exam, err := scanExam(row)
	if err != nil {
		return domain.Exam{}, mapPostgresError(err)
	}

	return exam, nil
}

func (p *Postgres) ListExams(ctx context.Context, branchID string, classID string) ([]domain.Exam, error) {
	rows, err := p.query(ctx, `
		SELECT id::text, branch_id::text, course_id::text, coalesce(class_id::text, ''),
			coalesce(schedule_item_id::text, ''), title, created_by_user_id::text, created_at, updated_at
		FROM exams
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR class_id = $2::uuid)
		ORDER BY created_at DESC
	`, branchID, classID)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	exams := make([]domain.Exam, 0)
	for rows.Next() {
		exam, err := scanExam(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		exams = append(exams, exam)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return exams, nil
}

func (p *Postgres) ListExamsPage(ctx context.Context, branchID string, classID string, page domain.PageRequest) ([]domain.Exam, int, error) {
	page = normalizePage(page)

	var total int
	if err := p.queryRow(ctx, `
		SELECT count(*)
		FROM exams
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR class_id = $2::uuid)
	`, branchID, classID).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	rows, err := p.query(ctx, `
		SELECT id::text, branch_id::text, course_id::text, coalesce(class_id::text, ''),
			coalesce(schedule_item_id::text, ''), title, created_by_user_id::text, created_at, updated_at
		FROM exams
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR class_id = $2::uuid)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`, branchID, classID, page.Limit, page.Offset)
	if err != nil {
		return nil, 0, mapPostgresError(err)
	}
	defer rows.Close()

	exams := make([]domain.Exam, 0, page.Limit)
	for rows.Next() {
		exam, err := scanExam(rows)
		if err != nil {
			return nil, 0, mapPostgresError(err)
		}
		exams = append(exams, exam)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	return exams, total, nil
}

func (p *Postgres) CreateExamResult(ctx context.Context, result domain.ExamResult) (domain.ExamResult, error) {
	if result.ExamID == "" || result.StudentID == "" || result.CategoryID == "" || result.EnteredByTeacherID == "" {
		return domain.ExamResult{}, domain.ErrInvalidInput
	}
	if err := p.ensureExamResultBranch(ctx, result); err != nil {
		return domain.ExamResult{}, err
	}

	row := p.queryRow(ctx, `
		INSERT INTO exam_results (
			branch_id, exam_id, student_id, category_id, score, feedback, document_file_id, entered_by_teacher_id
		)
		VALUES ($1, $2, $3, $4, $5, nullif($6, ''), nullif($7, '')::uuid, $8)
		ON CONFLICT (exam_id, student_id, category_id) DO UPDATE SET
			score = EXCLUDED.score,
			feedback = EXCLUDED.feedback,
			document_file_id = EXCLUDED.document_file_id,
			entered_by_teacher_id = EXCLUDED.entered_by_teacher_id,
			updated_at = now()
		RETURNING id::text, branch_id::text, exam_id::text, student_id::text, category_id::text,
			score, coalesce(feedback, ''), coalesce(document_file_id::text, ''), entered_by_teacher_id::text, created_at, updated_at
	`, result.BranchID, result.ExamID, result.StudentID, result.CategoryID, result.Score, strings.TrimSpace(result.Feedback), result.DocumentFileID, result.EnteredByTeacherID)

	created, err := scanExamResult(row)
	if err != nil {
		return domain.ExamResult{}, mapPostgresError(err)
	}

	return created, nil
}

func (p *Postgres) ListExamResults(ctx context.Context, branchID string, examID string, studentID string) ([]domain.ExamResult, error) {
	rows, err := p.query(ctx, `
		SELECT id::text, branch_id::text, exam_id::text, student_id::text, category_id::text,
			score, coalesce(feedback, ''), coalesce(document_file_id::text, ''), entered_by_teacher_id::text, created_at, updated_at
		FROM exam_results
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR exam_id = $2::uuid)
			AND ($3 = '' OR student_id = $3::uuid)
		ORDER BY created_at DESC
	`, branchID, examID, studentID)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()

	results := make([]domain.ExamResult, 0)
	for rows.Next() {
		result, err := scanExamResult(rows)
		if err != nil {
			return nil, mapPostgresError(err)
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}

	return results, nil
}

func (p *Postgres) ListExamResultsPage(ctx context.Context, branchID string, examID string, studentID string, page domain.PageRequest) ([]domain.ExamResult, int, error) {
	page = normalizePage(page)

	var total int
	if err := p.queryRow(ctx, `
		SELECT count(*)
		FROM exam_results
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR exam_id = $2::uuid)
			AND ($3 = '' OR student_id = $3::uuid)
	`, branchID, examID, studentID).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	rows, err := p.query(ctx, `
		SELECT id::text, branch_id::text, exam_id::text, student_id::text, category_id::text,
			score, coalesce(feedback, ''), coalesce(document_file_id::text, ''), entered_by_teacher_id::text, created_at, updated_at
		FROM exam_results
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR exam_id = $2::uuid)
			AND ($3 = '' OR student_id = $3::uuid)
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5
	`, branchID, examID, studentID, page.Limit, page.Offset)
	if err != nil {
		return nil, 0, mapPostgresError(err)
	}
	defer rows.Close()

	results := make([]domain.ExamResult, 0, page.Limit)
	for rows.Next() {
		result, err := scanExamResult(rows)
		if err != nil {
			return nil, 0, mapPostgresError(err)
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	return results, total, nil
}

func (p *Postgres) AcademicDashboard(ctx context.Context, branchID string) (domain.AcademicDashboard, error) {
	var dashboard domain.AcademicDashboard
	dashboard.BranchID = branchID
	err := p.queryRow(ctx, `
		SELECT
			(SELECT count(*) FROM students WHERE ($1 = '' OR branch_id = $1::uuid) AND status = 'active'),
			(SELECT count(*) FROM teachers WHERE ($1 = '' OR branch_id = $1::uuid) AND status = 'active'),
			(SELECT count(*) FROM classes WHERE ($1 = '' OR branch_id = $1::uuid) AND is_active = true),
			(SELECT count(*) FROM schedule_items WHERE ($1 = '' OR branch_id = $1::uuid) AND cancelled_at IS NULL AND starts_at >= now()),
			(SELECT count(*) FROM payments WHERE ($1 = '' OR branch_id = $1::uuid) AND status IN ('pending', 'overdue')),
			(SELECT count(*) FROM files WHERE ($1 = '' OR branch_id = $1::uuid) AND purpose = 'exam_writing' AND deleted_at IS NULL)
	`, branchID).Scan(
		&dashboard.ActiveStudents,
		&dashboard.ActiveTeachers,
		&dashboard.ActiveClasses,
		&dashboard.UpcomingSchedule,
		&dashboard.PendingPayments,
		&dashboard.WritingFilesRetained,
	)
	if err != nil {
		return domain.AcademicDashboard{}, mapPostgresError(err)
	}

	return dashboard, nil
}

func scanClass(row scanner) (domain.Class, error) {
	var class domain.Class
	var endDate sql.NullTime
	err := row.Scan(
		&class.ID,
		&class.BranchID,
		&class.CourseID,
		&class.TeacherID,
		&class.Name,
		&class.StartDate,
		&endDate,
		&class.IsActive,
		&class.CreatedAt,
		&class.UpdatedAt,
	)
	if err != nil {
		return domain.Class{}, err
	}
	if endDate.Valid {
		class.EndDate = &endDate.Time
	}

	return class, nil
}

func scanClassStudent(row scanner) (domain.ClassStudent, error) {
	var enrollment domain.ClassStudent
	var leftAt sql.NullTime
	err := row.Scan(
		&enrollment.ID,
		&enrollment.BranchID,
		&enrollment.ClassID,
		&enrollment.StudentID,
		&enrollment.JoinedAt,
		&leftAt,
		&enrollment.CreatedAt,
	)
	if err != nil {
		return domain.ClassStudent{}, err
	}
	if leftAt.Valid {
		enrollment.LeftAt = &leftAt.Time
	}

	return enrollment, nil
}

func scanAssignment(row scanner) (domain.Assignment, error) {
	var assignment domain.Assignment
	var dueAt sql.NullTime
	err := row.Scan(
		&assignment.ID,
		&assignment.BranchID,
		&assignment.ClassID,
		&assignment.Title,
		&assignment.Description,
		&dueAt,
		&assignment.MaterialFileID,
		&assignment.CreatedByUserID,
		&assignment.CreatedAt,
		&assignment.UpdatedAt,
	)
	if err != nil {
		return domain.Assignment{}, err
	}
	if dueAt.Valid {
		assignment.DueAt = &dueAt.Time
	}

	return assignment, nil
}

func scanExam(row scanner) (domain.Exam, error) {
	var exam domain.Exam
	err := row.Scan(
		&exam.ID,
		&exam.BranchID,
		&exam.CourseID,
		&exam.ClassID,
		&exam.ScheduleItemID,
		&exam.Title,
		&exam.CreatedByUserID,
		&exam.CreatedAt,
		&exam.UpdatedAt,
	)
	return exam, err
}

func scanExamParticipant(row scanner) (domain.ExamParticipant, error) {
	var participant domain.ExamParticipant
	err := row.Scan(&participant.ExamID, &participant.StudentID, &participant.BranchID, &participant.RSVPStatus)
	return participant, err
}

func scanExamResult(row scanner) (domain.ExamResult, error) {
	var result domain.ExamResult
	err := row.Scan(
		&result.ID,
		&result.BranchID,
		&result.ExamID,
		&result.StudentID,
		&result.CategoryID,
		&result.Score,
		&result.Feedback,
		&result.DocumentFileID,
		&result.EnteredByTeacherID,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	return result, err
}

func (p *Postgres) ensureCourseTeacherBranch(ctx context.Context, courseID string, teacherID string, branchID string) error {
	if err := p.ensureCourseBranch(ctx, courseID, branchID); err != nil {
		return err
	}
	teacher, err := p.GetTeacher(ctx, teacherID)
	if err != nil {
		return err
	}
	if teacher.BranchID != branchID || teacher.Status != domain.TeacherStatusActive {
		return domain.ErrInvalidInput
	}

	return nil
}

func (p *Postgres) ensureCourseBranch(ctx context.Context, courseID string, branchID string) error {
	var found string
	err := p.queryRow(ctx, `SELECT branch_id::text FROM courses WHERE id = $1`, courseID).Scan(&found)
	if err != nil {
		return mapPostgresError(err)
	}
	if found != branchID {
		return domain.ErrInvalidInput
	}

	return nil
}

func (p *Postgres) ensureClassStudentBranch(ctx context.Context, classID string, studentID string, branchID string) error {
	class, err := p.GetClass(ctx, classID)
	if err != nil {
		return err
	}
	student, err := p.GetStudent(ctx, studentID)
	if err != nil {
		return err
	}
	if class.BranchID != branchID || student.BranchID != branchID {
		return domain.ErrInvalidInput
	}

	return nil
}

func (p *Postgres) ensureStudentBranchTx(ctx context.Context, tx pgx.Tx, studentID string, branchID string) error {
	var found string
	err := tx.QueryRow(ctx, `SELECT branch_id::text FROM students WHERE id = $1`, studentID).Scan(&found)
	if err != nil {
		return mapPostgresError(err)
	}
	if found != branchID {
		return domain.ErrInvalidInput
	}

	return nil
}

func (p *Postgres) ensureExamResultBranch(ctx context.Context, result domain.ExamResult) error {
	var examBranch string
	var courseID string
	err := p.queryRow(ctx, `SELECT branch_id::text, course_id::text FROM exams WHERE id = $1`, result.ExamID).Scan(&examBranch, &courseID)
	if err != nil {
		return mapPostgresError(err)
	}
	if examBranch != result.BranchID {
		return domain.ErrInvalidInput
	}

	student, err := p.GetStudent(ctx, result.StudentID)
	if err != nil {
		return err
	}
	teacher, err := p.GetTeacher(ctx, result.EnteredByTeacherID)
	if err != nil {
		return err
	}
	if student.BranchID != result.BranchID || teacher.BranchID != result.BranchID {
		return domain.ErrInvalidInput
	}

	var minScore float64
	var maxScore float64
	err = p.queryRow(ctx, `
		SELECT min_score, max_score
		FROM course_score_categories
		WHERE id = $1 AND course_id = $2 AND branch_id = $3
	`, result.CategoryID, courseID, result.BranchID).Scan(&minScore, &maxScore)
	if err != nil {
		return mapPostgresError(err)
	}
	if result.Score < minScore || result.Score > maxScore {
		return domain.ErrInvalidInput
	}

	return nil
}

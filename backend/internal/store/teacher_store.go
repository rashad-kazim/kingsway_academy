package store

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"

	"kingsway/backend/internal/domain"
)

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

	row := p.queryRow(ctx, `
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
	row := p.queryRow(ctx, `
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
	row := p.queryRow(ctx, `
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
	row := p.queryRow(ctx, `
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

	var updatedTeacher domain.Teacher
	var updatedUser domain.User
	err := p.withTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
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
			return mapPostgresError(err)
		}

		nextUser, err := scanUser(tx.QueryRow(ctx, `
			UPDATE users
			SET branch_id = $2,
				email = $3,
				first_name = $4,
				last_name = $5,
				password_hash = CASE WHEN $6 = '' THEN password_hash ELSE $6 END,
				token_version = token_version + CASE WHEN $6 = '' AND branch_id = $2 THEN 0 ELSE 1 END,
				updated_at = now()
			WHERE id = $1
			RETURNING id::text, coalesce(branch_id::text, ''), role, email, password_hash, first_name, last_name, is_active, token_version, coalesce(last_login_at::text, ''), created_at, updated_at
		`, current.UserID, teacher.BranchID, normalizeEmail(user.Email), strings.TrimSpace(user.FirstName), strings.TrimSpace(user.LastName), strings.TrimSpace(passwordHash)))
		if err != nil {
			return mapPostgresError(err)
		}
		updatedUser = nextUser

		nextTeacher, err := scanTeacher(tx.QueryRow(ctx, `
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
			return mapPostgresError(err)
		}
		updatedTeacher = nextTeacher
		return nil
	})
	if err != nil {
		return domain.Teacher{}, domain.User{}, err
	}

	return updatedTeacher, updatedUser, nil
}

func (p *Postgres) DeleteTeacher(ctx context.Context, id string) (domain.Teacher, error) {
	var teacher domain.Teacher
	err := p.withTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
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
		`, id))
		if err != nil {
			return mapPostgresError(err)
		}
		teacher = current

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
			return err
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
			return err
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
			return err
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
			return err
		}
		if err := exec(`
			DELETE FROM assignments
			WHERE created_by_user_id = $2
				OR class_id IN (SELECT c.id FROM classes c WHERE c.teacher_id = $1)
		`, teacherID, userID); err != nil {
			return err
		}
		if err := exec(`DELETE FROM payments WHERE created_by_user_id = $1`, userID); err != nil {
			return err
		}
		if err := exec(`
			DELETE FROM schedule_items
			WHERE teacher_id = $1
				OR created_by_user_id = $2
				OR class_id IN (SELECT c.id FROM classes c WHERE c.teacher_id = $1)
		`, teacherID, userID); err != nil {
			return err
		}
		if err := exec(`
			DELETE FROM class_students
			WHERE class_id IN (SELECT c.id FROM classes c WHERE c.teacher_id = $1)
		`, teacherID); err != nil {
			return err
		}
		if err := exec(`
			DELETE FROM student_teacher_assignments
			WHERE teacher_id = $1
				OR class_id IN (SELECT c.id FROM classes c WHERE c.teacher_id = $1)
		`, teacherID); err != nil {
			return err
		}
		if err := exec(`DELETE FROM salary_models WHERE teacher_id = $1`, teacherID); err != nil {
			return err
		}
		if err := exec(`DELETE FROM teacher_course_specializations WHERE teacher_id = $1`, teacherID); err != nil {
			return err
		}
		if err := exec(`DELETE FROM classes WHERE teacher_id = $1`, teacherID); err != nil {
			return err
		}
		if err := exec(`UPDATE exam_results SET document_file_id = NULL WHERE document_file_id IN (`+teacherFileFilter+`)`, userID, teacherID); err != nil {
			return err
		}
		if err := exec(`UPDATE assignments SET material_file_id = NULL WHERE material_file_id IN (`+teacherFileFilter+`)`, userID, teacherID); err != nil {
			return err
		}
		if err := exec(`UPDATE payments SET receipt_file_id = NULL WHERE receipt_file_id IN (`+teacherFileFilter+`)`, userID, teacherID); err != nil {
			return err
		}
		if err := exec(`UPDATE teachers SET profile_photo_file_id = NULL WHERE id = $1`, teacherID); err != nil {
			return err
		}
		if err := exec(`DELETE FROM files WHERE uploader_user_id = $1 OR (owner_type = 'teacher' AND owner_id = $2)`, userID, teacherID); err != nil {
			return err
		}
		if err := exec(`DELETE FROM teachers WHERE id = $1`, teacherID); err != nil {
			return err
		}
		if err := exec(`UPDATE students SET user_id = NULL WHERE user_id = $1`, userID); err != nil {
			return err
		}
		if err := exec(`DELETE FROM users WHERE id = $1`, userID); err != nil {
			return err
		}
		if err := exec(`
			DELETE FROM outbox_events
			WHERE payload->>'teacher_id' = $1
				OR payload->>'user_id' = $2
				OR payload->'teacher'->>'id' = $1
				OR payload->'teacher'->>'user_id' = $2
		`, teacherID, userID); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return domain.Teacher{}, err
	}

	return teacher, nil
}

func (p *Postgres) ListTeachers(ctx context.Context, branchID string) ([]domain.Teacher, error) {
	rows, err := p.query(ctx, `
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

func (p *Postgres) ListTeachersPage(ctx context.Context, branchID string, status domain.TeacherStatus, page domain.PageRequest) ([]domain.Teacher, int, error) {
	page = normalizePage(page)

	var total int
	if err := p.queryRow(ctx, `
		SELECT count(*)
		FROM teachers
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR status = $2)
	`, branchID, string(status)).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	rows, err := p.query(ctx, `
		SELECT id::text, branch_id::text, user_id::text, status,
			coalesce(to_char(birth_date, 'DD/MM/YYYY'), ''),
			coalesce(gender, ''),
			coalesce(phone, ''),
			coalesce(address, ''),
			coalesce(profile_photo_file_id::text, ''),
			created_at, updated_at
		FROM teachers
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR status = $2)
		ORDER BY created_at, id
		LIMIT $3 OFFSET $4
	`, branchID, string(status), page.Limit, page.Offset)
	if err != nil {
		return nil, 0, mapPostgresError(err)
	}
	defer rows.Close()

	teachers := make([]domain.Teacher, 0, page.Limit)
	for rows.Next() {
		teacher, err := scanTeacher(rows)
		if err != nil {
			return nil, 0, mapPostgresError(err)
		}
		teachers = append(teachers, teacher)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	return teachers, total, nil
}

func (p *Postgres) ReplaceTeacherCourseSpecializations(ctx context.Context, teacherID string, courseIDs []string) error {
	return p.withTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
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

		return nil
	})
}

func (p *Postgres) ListTeacherFinanceRecords(ctx context.Context, branchID string) ([]domain.TeacherFinanceRecord, error) {
	rows, err := p.query(ctx, `
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

func (p *Postgres) ListTeacherFinanceRecordsPage(ctx context.Context, branchID string, subject string, status domain.TeacherStatus, salaryModel domain.SalaryModelType, page domain.PageRequest) ([]domain.TeacherFinanceRecord, int, error) {
	page = normalizePage(page)
	subject = strings.TrimSpace(subject)

	baseSQL := `
		WITH teacher_finance AS (
			SELECT
				t.id::text AS id,
				t.branch_id::text AS branch_id,
				b.name AS branch_name,
				t.user_id::text AS user_id,
				u.email AS email,
				u.first_name AS first_name,
				u.last_name AS last_name,
				coalesce(t.profile_photo_file_id::text, '') AS profile_photo_file_id,
				coalesce(string_agg(DISTINCT c.name, ', '), '') AS subject,
				t.status AS status,
				coalesce(latest_salary.model_type, '') AS salary_type,
				CASE
					WHEN latest_salary.model_type IN ('fixed', 'hybrid') THEN coalesce(latest_salary.fixed_monthly_amount_cents, 0)
					ELSE 0
				END AS calculated_salary_cents,
				coalesce(active_assignments.assigned_students, 0)::integer AS assigned_students,
				t.created_at AS created_at,
				t.updated_at AS updated_at
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
		)
	`

	var total int
	if err := p.queryRow(ctx, baseSQL+`
		SELECT count(*)
		FROM teacher_finance
		WHERE ($2 = '' OR lower(subject) LIKE '%' || lower($2) || '%')
			AND ($3 = '' OR status = $3)
			AND ($4 = '' OR salary_type = $4)
	`, branchID, subject, string(status), string(salaryModel)).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	rows, err := p.query(ctx, baseSQL+`
		SELECT id, branch_id, branch_name, user_id, email, first_name, last_name, profile_photo_file_id,
			subject, status, salary_type, calculated_salary_cents, assigned_students, created_at, updated_at
		FROM teacher_finance
		WHERE ($2 = '' OR lower(subject) LIKE '%' || lower($2) || '%')
			AND ($3 = '' OR status = $3)
			AND ($4 = '' OR salary_type = $4)
		ORDER BY last_name, first_name
		LIMIT $5 OFFSET $6
	`, branchID, subject, string(status), string(salaryModel), page.Limit, page.Offset)
	if err != nil {
		return nil, 0, mapPostgresError(err)
	}
	defer rows.Close()

	records := make([]domain.TeacherFinanceRecord, 0, page.Limit)
	for rows.Next() {
		record, err := scanTeacherFinanceRecord(rows)
		if err != nil {
			return nil, 0, mapPostgresError(err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	return records, total, nil
}

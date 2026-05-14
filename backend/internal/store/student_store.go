package store

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"

	"kingsway/backend/internal/domain"
)

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

	row := p.queryRow(ctx, `
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

	var created domain.Student
	err := p.withTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		next, err := scanStudent(tx.QueryRow(ctx, `
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
			return mapPostgresError(err)
		}
		created = next

		for _, contact := range contacts {
			if contact.BranchID != created.BranchID || strings.TrimSpace(contact.Relation) == "" || strings.TrimSpace(contact.Name) == "" || len(contact.Phones) == 0 {
				return domain.ErrInvalidInput
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO student_parent_contacts (branch_id, student_id, relation, name, phones)
				VALUES ($1, $2, $3, $4, $5)
			`, contact.BranchID, created.ID, strings.TrimSpace(contact.Relation), strings.TrimSpace(contact.Name), contact.Phones); err != nil {
				return mapPostgresError(err)
			}
		}

		for _, registration := range registrations {
			if registration.BranchID != created.BranchID || registration.CourseID == "" || registration.StartDate == "" || registration.MonthlyAmountCents < 0 {
				return domain.ErrInvalidInput
			}
			var courseBranchID string
			var courseActive bool
			if err := tx.QueryRow(ctx, `
				SELECT branch_id::text, is_active
				FROM courses
				WHERE id = $1
			`, registration.CourseID).Scan(&courseBranchID, &courseActive); err != nil {
				return mapPostgresError(err)
			}
			if courseBranchID != created.BranchID || !courseActive {
				return domain.ErrInvalidInput
			}
			if registration.TeacherID != "" {
				var teacherBranchID string
				var teacherStatus domain.TeacherStatus
				if err := tx.QueryRow(ctx, `
					SELECT branch_id::text, status
					FROM teachers
					WHERE id = $1
				`, registration.TeacherID).Scan(&teacherBranchID, &teacherStatus); err != nil {
					return mapPostgresError(err)
				}
				if teacherBranchID != created.BranchID || teacherStatus != domain.TeacherStatusActive {
					return domain.ErrInvalidInput
				}
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO student_course_registrations (
					branch_id, student_id, course_id, teacher_id, monthly_amount_cents, start_date
				)
				VALUES ($1, $2, $3, nullif($4, '')::uuid, $5, to_date($6, 'DD/MM/YYYY'))
			`, registration.BranchID, created.ID, registration.CourseID, registration.TeacherID, registration.MonthlyAmountCents, registration.StartDate); err != nil {
				return mapPostgresError(err)
			}
			if registration.TeacherID != "" {
				if _, err := tx.Exec(ctx, `
					UPDATE student_teacher_assignments
					SET valid_to = to_date($3, 'DD/MM/YYYY')
					WHERE branch_id = $1 AND student_id = $2 AND valid_to IS NULL
				`, registration.BranchID, created.ID, registration.StartDate); err != nil {
					return mapPostgresError(err)
				}
				if _, err := tx.Exec(ctx, `
					INSERT INTO student_teacher_assignments (branch_id, student_id, teacher_id, valid_from, reason)
					VALUES ($1, $2, $3, to_date($4, 'DD/MM/YYYY'), 'initial')
				`, registration.BranchID, created.ID, registration.TeacherID, registration.StartDate); err != nil {
					return mapPostgresError(err)
				}
			}
		}

		return nil
	})
	if err != nil {
		return domain.Student{}, err
	}

	return created, nil
}

func (p *Postgres) CreateStudentParentContacts(ctx context.Context, studentID string, contacts []domain.StudentParentContact) ([]domain.StudentParentContact, error) {
	student, err := p.GetStudent(ctx, studentID)
	if err != nil {
		return nil, err
	}
	created := make([]domain.StudentParentContact, 0, len(contacts))
	err = p.withTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		for _, contact := range contacts {
			if contact.BranchID != student.BranchID || strings.TrimSpace(contact.Relation) == "" || strings.TrimSpace(contact.Name) == "" || len(contact.Phones) == 0 {
				return domain.ErrInvalidInput
			}
			row := tx.QueryRow(ctx, `
				INSERT INTO student_parent_contacts (branch_id, student_id, relation, name, phones)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id::text, branch_id::text, student_id::text, relation, name, phones, created_at
			`, contact.BranchID, student.ID, strings.TrimSpace(contact.Relation), strings.TrimSpace(contact.Name), contact.Phones)
			var next domain.StudentParentContact
			if err := row.Scan(&next.ID, &next.BranchID, &next.StudentID, &next.Relation, &next.Name, &next.Phones, &next.CreatedAt); err != nil {
				return mapPostgresError(err)
			}
			created = append(created, next)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (p *Postgres) CreateStudentCourseRegistrations(ctx context.Context, studentID string, registrations []domain.StudentCourseRegistration) ([]domain.StudentCourseRegistration, error) {
	student, err := p.GetStudent(ctx, studentID)
	if err != nil {
		return nil, err
	}
	created := make([]domain.StudentCourseRegistration, 0, len(registrations))
	err = p.withTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		for _, registration := range registrations {
			if registration.BranchID != student.BranchID || registration.CourseID == "" || registration.StartDate == "" || registration.MonthlyAmountCents < 0 {
				return domain.ErrInvalidInput
			}
			var courseBranchID string
			var courseActive bool
			if err := tx.QueryRow(ctx, `
				SELECT branch_id::text, is_active
				FROM courses
				WHERE id = $1
			`, registration.CourseID).Scan(&courseBranchID, &courseActive); err != nil {
				return mapPostgresError(err)
			}
			if courseBranchID != student.BranchID || !courseActive {
				return domain.ErrInvalidInput
			}
			if registration.TeacherID != "" {
				var teacherBranchID string
				var teacherStatus domain.TeacherStatus
				if err := tx.QueryRow(ctx, `
					SELECT branch_id::text, status
					FROM teachers
					WHERE id = $1
				`, registration.TeacherID).Scan(&teacherBranchID, &teacherStatus); err != nil {
					return mapPostgresError(err)
				}
				if teacherBranchID != student.BranchID || teacherStatus != domain.TeacherStatusActive {
					return domain.ErrInvalidInput
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
				return mapPostgresError(err)
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
					return mapPostgresError(err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (p *Postgres) GetStudent(ctx context.Context, id string) (domain.Student, error) {
	row := p.queryRow(ctx, `
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
	row := p.queryRow(ctx, `
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
	row := p.queryRow(ctx, `
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

	row := p.queryRow(ctx, `
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
	rows, err := p.query(ctx, `
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

func (p *Postgres) ListStudentsPage(ctx context.Context, branchID string, status domain.StudentStatus, page domain.PageRequest) ([]domain.Student, int, error) {
	page = normalizePage(page)

	var total int
	if err := p.queryRow(ctx, `
		SELECT count(*)
		FROM students
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR status = $2)
	`, branchID, string(status)).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	rows, err := p.query(ctx, `
		SELECT id::text, branch_id::text, coalesce(user_id::text, ''), fin_code,
			first_name, last_name, coalesce(to_char(birth_date, 'DD/MM/YYYY'), ''),
			coalesce(gender, ''), phone, address, coalesce(profile_photo_file_id::text, ''),
			status, coalesce(left_reason, ''), created_at, updated_at
		FROM students
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR status = $2)
		ORDER BY last_name, first_name
		LIMIT $3 OFFSET $4
	`, branchID, string(status), page.Limit, page.Offset)
	if err != nil {
		return nil, 0, mapPostgresError(err)
	}
	defer rows.Close()

	students := make([]domain.Student, 0, page.Limit)
	for rows.Next() {
		student, err := scanStudent(rows)
		if err != nil {
			return nil, 0, mapPostgresError(err)
		}
		students = append(students, student)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	return students, total, nil
}

func (p *Postgres) ListStudentAssignmentHub(ctx context.Context, filter domain.StudentAssignmentHubFilter) (domain.StudentAssignmentHubPage, error) {
	rows, err := p.query(ctx, `
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

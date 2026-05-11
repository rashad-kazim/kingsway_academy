package store

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"

	"kingsway/backend/internal/domain"
)

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

	var created domain.Course
	createdCategories := make([]domain.ScoreCategory, 0, len(categories))
	err := p.withTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		nextCourse, err := scanCourse(tx.QueryRow(ctx, `
			INSERT INTO courses (branch_id, name, is_active)
			VALUES ($1, $2, true)
			RETURNING id::text, branch_id::text, name, is_active, created_at, updated_at
		`, course.BranchID, strings.TrimSpace(course.Name)))
		if err != nil {
			return mapPostgresError(err)
		}
		created = nextCourse

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
				return mapPostgresError(err)
			}
			createdCategories = append(createdCategories, createdCategory)
		}

		return nil
	})
	if err != nil {
		return domain.Course{}, nil, err
	}

	return created, createdCategories, nil
}

func (p *Postgres) GetCourse(ctx context.Context, id string) (domain.Course, error) {
	row := p.queryRow(ctx, `
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
	rows, err := p.query(ctx, `
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

func (p *Postgres) ListCoursesPage(ctx context.Context, branchID string, page domain.PageRequest) ([]domain.Course, int, error) {
	page = normalizePage(page)

	var total int
	if err := p.queryRow(ctx, `
		SELECT count(*)
		FROM courses
		WHERE ($1 = '' OR branch_id = $1::uuid)
	`, branchID).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	rows, err := p.query(ctx, `
		SELECT id::text, branch_id::text, name, is_active, created_at, updated_at
		FROM courses
		WHERE ($1 = '' OR branch_id = $1::uuid)
		ORDER BY name
		LIMIT $2 OFFSET $3
	`, branchID, page.Limit, page.Offset)
	if err != nil {
		return nil, 0, mapPostgresError(err)
	}
	defer rows.Close()

	courses := make([]domain.Course, 0, page.Limit)
	for rows.Next() {
		course, err := scanCourse(rows)
		if err != nil {
			return nil, 0, mapPostgresError(err)
		}
		courses = append(courses, course)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	return courses, total, nil
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

	row := p.queryRow(ctx, `
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
	rows, err := p.query(ctx, `
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

func (p *Postgres) ListSchedulePage(ctx context.Context, branchID string, itemType domain.ScheduleItemType, page domain.PageRequest) ([]domain.ScheduleItem, int, error) {
	page = normalizePage(page)

	var total int
	if err := p.queryRow(ctx, `
		SELECT count(*)
		FROM schedule_items
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR item_type = $2)
	`, branchID, string(itemType)).Scan(&total); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	rows, err := p.query(ctx, `
		SELECT id::text, branch_id::text, coalesce(class_id::text, ''), teacher_id::text, room_id::text,
			item_type, title, starts_at, ends_at, created_by_user_id::text, cancelled_at, created_at, updated_at
		FROM schedule_items
		WHERE ($1 = '' OR branch_id = $1::uuid)
			AND ($2 = '' OR item_type = $2)
		ORDER BY starts_at
		LIMIT $3 OFFSET $4
	`, branchID, string(itemType), page.Limit, page.Offset)
	if err != nil {
		return nil, 0, mapPostgresError(err)
	}
	defer rows.Close()

	items := make([]domain.ScheduleItem, 0, page.Limit)
	for rows.Next() {
		item, err := scanScheduleItem(rows)
		if err != nil {
			return nil, 0, mapPostgresError(err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, mapPostgresError(err)
	}

	return items, total, nil
}

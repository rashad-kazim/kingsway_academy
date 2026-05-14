package academic

import (
	"context"
	"strings"

	"kingsway/backend/internal/auth"
	"kingsway/backend/internal/domain"
)

func (s *Service) RegisterTeacher(ctx context.Context, actor domain.Principal, input RegisterTeacherInput) (domain.Teacher, domain.User, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist); err != nil {
		return domain.Teacher{}, domain.User{}, err
	}
	if !actor.IsOwner() {
		input.BranchID = actor.BranchID
	}
	if err := auth.RequireBranch(actor, input.BranchID); err != nil {
		return domain.Teacher{}, domain.User{}, err
	}
	if strings.TrimSpace(input.FirstName) == "" ||
		strings.TrimSpace(input.LastName) == "" ||
		strings.TrimSpace(input.Email) == "" ||
		strings.TrimSpace(input.Password) == "" ||
		strings.TrimSpace(input.Phone) == "" {
		return domain.Teacher{}, domain.User{}, domain.ErrInvalidInput
	}

	user, err := s.auth.CreateUser(ctx, actor, auth.CreateUserInput{
		BranchID:  input.BranchID,
		Role:      domain.RoleTeacher,
		Email:     input.Email,
		Password:  input.Password,
		FirstName: input.FirstName,
		LastName:  input.LastName,
	})
	if err != nil {
		return domain.Teacher{}, domain.User{}, err
	}

	teacher, err := s.teachers.CreateTeacher(ctx, domain.Teacher{
		BranchID:           input.BranchID,
		UserID:             user.ID,
		Status:             domain.TeacherStatusPending,
		BirthDate:          strings.TrimSpace(input.BirthDate),
		Gender:             normalizeOptionalTeacherGender(input.Gender),
		Phone:              strings.TrimSpace(input.Phone),
		Address:            strings.TrimSpace(input.Address),
		ProfilePhotoFileID: strings.TrimSpace(input.ProfilePhotoFileID),
	})
	if err != nil {
		return domain.Teacher{}, domain.User{}, err
	}
	courseIDs, err := s.teacherCourseIDs(ctx, input.BranchID, input.CourseIDs, input.Subjects)
	if err != nil {
		return domain.Teacher{}, domain.User{}, err
	}
	if len(courseIDs) > 0 {
		if err := s.teachers.ReplaceTeacherCourseSpecializations(ctx, teacher.ID, courseIDs); err != nil {
			return domain.Teacher{}, domain.User{}, err
		}
	}

	return teacher, user, nil
}

func (s *Service) UpdateTeacher(ctx context.Context, actor domain.Principal, input UpdateTeacherInput) (domain.Teacher, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist); err != nil {
		return domain.Teacher{}, err
	}
	teacher, err := s.teachers.GetTeacher(ctx, strings.TrimSpace(input.ID))
	if err != nil {
		return domain.Teacher{}, err
	}
	if err := auth.RequireBranch(actor, teacher.BranchID); err != nil {
		return domain.Teacher{}, err
	}
	user, err := s.auth.CurrentUser(ctx, domain.Principal{UserID: teacher.UserID})
	if err != nil {
		return domain.Teacher{}, err
	}
	if actor.IsOwner() && strings.TrimSpace(input.BranchID) != "" {
		teacher.BranchID = strings.TrimSpace(input.BranchID)
	}
	if err := auth.RequireBranch(actor, teacher.BranchID); err != nil {
		return domain.Teacher{}, err
	}
	if strings.TrimSpace(input.Email) != "" {
		user.Email = strings.TrimSpace(input.Email)
	}
	if strings.TrimSpace(input.FirstName) != "" {
		user.FirstName = strings.TrimSpace(input.FirstName)
	}
	if strings.TrimSpace(input.LastName) != "" {
		user.LastName = strings.TrimSpace(input.LastName)
	}
	user.BranchID = teacher.BranchID
	teacher.BirthDate = strings.TrimSpace(input.BirthDate)
	teacher.Gender = normalizeOptionalTeacherGender(input.Gender)
	teacher.Phone = strings.TrimSpace(input.Phone)
	teacher.Address = strings.TrimSpace(input.Address)
	teacher.ProfilePhotoFileID = strings.TrimSpace(input.ProfilePhotoFileID)
	if strings.TrimSpace(user.Email) == "" ||
		strings.TrimSpace(user.FirstName) == "" ||
		strings.TrimSpace(user.LastName) == "" ||
		strings.TrimSpace(teacher.Phone) == "" {
		return domain.Teacher{}, domain.ErrInvalidInput
	}
	passwordHash := ""
	if strings.TrimSpace(input.Password) != "" {
		passwordHash, err = auth.HashPassword(input.Password)
		if err != nil {
			return domain.Teacher{}, err
		}
	}

	updated, _, err := s.teachers.UpdateTeacherAccount(ctx, teacher, user, passwordHash)
	if err != nil {
		return domain.Teacher{}, err
	}
	courseIDs, err := s.teacherCourseIDs(ctx, updated.BranchID, input.CourseIDs, input.Subjects)
	if err != nil {
		return domain.Teacher{}, err
	}
	if len(courseIDs) > 0 {
		if err := s.teachers.ReplaceTeacherCourseSpecializations(ctx, teacher.ID, courseIDs); err != nil {
			return domain.Teacher{}, err
		}
	}

	return updated, nil
}

func (s *Service) ActivateTeacher(ctx context.Context, actor domain.Principal, teacherID string) (domain.Teacher, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner); err != nil {
		return domain.Teacher{}, err
	}

	teacher, err := s.teachers.GetTeacher(ctx, teacherID)
	if err != nil {
		return domain.Teacher{}, err
	}
	teacher.Status = domain.TeacherStatusActive

	return s.teachers.UpdateTeacher(ctx, teacher)
}

func (s *Service) teacherCourseIDs(ctx context.Context, branchID string, rawCourseIDs []string, subjects []string) ([]string, error) {
	seen := make(map[string]struct{})
	courseIDs := make([]string, 0, len(rawCourseIDs)+len(subjects))
	for _, courseID := range rawCourseIDs {
		courseID = strings.TrimSpace(courseID)
		if courseID == "" {
			continue
		}
		course, err := s.courses.GetCourse(ctx, courseID)
		if err != nil {
			return nil, err
		}
		if course.BranchID != branchID {
			return nil, domain.ErrInvalidInput
		}
		if _, exists := seen[course.ID]; !exists {
			seen[course.ID] = struct{}{}
			courseIDs = append(courseIDs, course.ID)
		}
	}

	for _, subject := range subjects {
		subject = strings.TrimSpace(subject)
		if subject == "" {
			continue
		}
		course, err := s.ensureCourseByName(ctx, branchID, subject)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[course.ID]; !exists {
			seen[course.ID] = struct{}{}
			courseIDs = append(courseIDs, course.ID)
		}
	}

	return courseIDs, nil
}

func (s *Service) ensureCourseByName(ctx context.Context, branchID string, name string) (domain.Course, error) {
	courses, err := s.courses.ListCourses(ctx, branchID)
	if err != nil {
		return domain.Course{}, err
	}
	for _, course := range courses {
		if strings.EqualFold(course.Name, name) {
			return course, nil
		}
	}
	course, _, err := s.courses.CreateCourse(ctx, domain.Course{
		BranchID: branchID,
		Name:     name,
	}, nil)
	return course, err
}

func normalizeOptionalTeacherGender(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "male", "female", "other":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func (s *Service) ListTeachers(ctx context.Context, actor domain.Principal, branchID string) ([]domain.Teacher, error) {
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	if actor.IsOwner() && branchID == "" {
		return s.teachers.ListTeachers(ctx, "")
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, err
	}

	teachers, err := s.teachers.ListTeachers(ctx, branchID)
	if err != nil {
		return nil, err
	}
	if actor.Role == domain.RoleTeacher {
		current, err := s.teachers.GetTeacherByUser(ctx, actor.UserID)
		if err != nil {
			return nil, err
		}
		return []domain.Teacher{current}, nil
	}
	if actor.Role == domain.RoleStudent {
		return []domain.Teacher{}, nil
	}

	return teachers, nil
}

func (s *Service) ListTeachersPage(ctx context.Context, actor domain.Principal, branchID string, status domain.TeacherStatus, page domain.PageRequest) ([]domain.Teacher, int, error) {
	if status != "" && status != domain.TeacherStatusPending && status != domain.TeacherStatusActive && status != domain.TeacherStatusTerminated {
		return nil, 0, domain.ErrInvalidInput
	}
	if actor.Role == domain.RoleTeacher || actor.Role == domain.RoleStudent {
		teachers, err := s.ListTeachers(ctx, actor, branchID)
		if err != nil {
			return nil, 0, err
		}
		if status != "" {
			filtered := make([]domain.Teacher, 0, len(teachers))
			for _, teacher := range teachers {
				if teacher.Status == status {
					filtered = append(filtered, teacher)
				}
			}
			teachers = filtered
		}
		items, total := domain.PageSlice(teachers, page)
		return items, total, nil
	}

	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	branchID = strings.TrimSpace(branchID)
	if actor.IsOwner() && branchID == "" {
		return s.teachers.ListTeachersPage(ctx, "", status, page)
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, 0, err
	}
	return s.teachers.ListTeachersPage(ctx, branchID, status, page)
}

func (s *Service) GetTeacher(ctx context.Context, actor domain.Principal, id string) (domain.Teacher, error) {
	teacher, err := s.teachers.GetTeacher(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.Teacher{}, err
	}
	if err := auth.RequireBranch(actor, teacher.BranchID); err != nil {
		return domain.Teacher{}, err
	}
	if actor.Role == domain.RoleTeacher {
		current, err := s.teachers.GetTeacherByUser(ctx, actor.UserID)
		if err != nil {
			return domain.Teacher{}, err
		}
		if current.ID != teacher.ID {
			return domain.Teacher{}, domain.ErrForbidden
		}
	}

	return teacher, nil
}

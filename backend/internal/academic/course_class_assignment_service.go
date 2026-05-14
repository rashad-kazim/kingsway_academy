package academic

import (
	"context"
	"strings"

	"kingsway/backend/internal/auth"
	"kingsway/backend/internal/domain"
)

func (s *Service) CreateCourse(ctx context.Context, actor domain.Principal, input CreateCourseInput) (domain.Course, []domain.ScoreCategory, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist); err != nil {
		return domain.Course{}, nil, err
	}
	if !actor.IsOwner() {
		input.BranchID = actor.BranchID
	}
	if err := auth.RequireBranch(actor, input.BranchID); err != nil {
		return domain.Course{}, nil, err
	}

	categories := make([]domain.ScoreCategory, len(input.Categories))
	for i, category := range input.Categories {
		categories[i] = domain.ScoreCategory{
			Name:             strings.TrimSpace(category.Name),
			MinScore:         category.MinScore,
			MaxScore:         category.MaxScore,
			RequiresFeedback: category.RequiresFeedback,
			RequiresDocument: category.RequiresDocument,
			DocumentPurpose:  strings.TrimSpace(category.DocumentPurpose),
		}
	}

	return s.courses.CreateCourse(ctx, domain.Course{
		BranchID: input.BranchID,
		Name:     strings.TrimSpace(input.Name),
	}, categories)
}

func (s *Service) ListCourses(ctx context.Context, actor domain.Principal, branchID string) ([]domain.Course, error) {
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	if actor.IsOwner() && branchID == "" {
		return s.courses.ListCourses(ctx, "")
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, err
	}

	return s.courses.ListCourses(ctx, branchID)
}

func (s *Service) ListCoursesPage(ctx context.Context, actor domain.Principal, branchID string, page domain.PageRequest) ([]domain.Course, int, error) {
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	branchID = strings.TrimSpace(branchID)
	if actor.IsOwner() && branchID == "" {
		return s.courses.ListCoursesPage(ctx, "", page)
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, 0, err
	}
	return s.courses.ListCoursesPage(ctx, branchID, page)
}

func (s *Service) GetCourse(ctx context.Context, actor domain.Principal, id string) (domain.Course, error) {
	course, err := s.courses.GetCourse(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.Course{}, err
	}
	if err := auth.RequireBranch(actor, course.BranchID); err != nil {
		return domain.Course{}, err
	}

	return course, nil
}

func (s *Service) CreateClass(ctx context.Context, actor domain.Principal, input CreateClassInput) (domain.Class, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist); err != nil {
		return domain.Class{}, err
	}
	if !actor.IsOwner() {
		input.BranchID = actor.BranchID
	}
	if err := auth.RequireBranch(actor, input.BranchID); err != nil {
		return domain.Class{}, err
	}
	if input.StartDate.IsZero() || (input.EndDate != nil && input.EndDate.Before(input.StartDate)) {
		return domain.Class{}, domain.ErrInvalidInput
	}

	return s.classes.CreateClass(ctx, domain.Class{
		BranchID:  input.BranchID,
		CourseID:  strings.TrimSpace(input.CourseID),
		TeacherID: strings.TrimSpace(input.TeacherID),
		Name:      strings.TrimSpace(input.Name),
		StartDate: input.StartDate,
		EndDate:   input.EndDate,
	})
}

func (s *Service) GetClass(ctx context.Context, actor domain.Principal, id string) (domain.Class, error) {
	class, err := s.classes.GetClass(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.Class{}, err
	}
	if err := auth.RequireBranch(actor, class.BranchID); err != nil {
		return domain.Class{}, err
	}
	if !s.canUseClass(ctx, actor, class) {
		return domain.Class{}, domain.ErrForbidden
	}

	return class, nil
}

func (s *Service) ListClasses(ctx context.Context, actor domain.Principal, branchID string) ([]domain.Class, error) {
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	if actor.IsOwner() && branchID == "" {
		return s.classes.ListClasses(ctx, "")
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, err
	}

	classes, err := s.classes.ListClasses(ctx, branchID)
	if err != nil {
		return nil, err
	}

	return s.filterClassesForActor(ctx, actor, classes)
}

func (s *Service) ListClassesPage(ctx context.Context, actor domain.Principal, branchID string, active *bool, page domain.PageRequest) ([]domain.Class, int, error) {
	if actor.Role == domain.RoleTeacher || actor.Role == domain.RoleStudent {
		classes, err := s.ListClasses(ctx, actor, branchID)
		if err != nil {
			return nil, 0, err
		}
		if active != nil {
			filtered := make([]domain.Class, 0, len(classes))
			for _, class := range classes {
				if class.IsActive == *active {
					filtered = append(filtered, class)
				}
			}
			classes = filtered
		}
		items, total := domain.PageSlice(classes, page)
		return items, total, nil
	}

	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	branchID = strings.TrimSpace(branchID)
	if actor.IsOwner() && branchID == "" {
		return s.classes.ListClassesPage(ctx, "", active, page)
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, 0, err
	}
	return s.classes.ListClassesPage(ctx, branchID, active, page)
}

func (s *Service) EnrollStudent(ctx context.Context, actor domain.Principal, input EnrollStudentInput) (domain.ClassStudent, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist); err != nil {
		return domain.ClassStudent{}, err
	}
	if !actor.IsOwner() {
		input.BranchID = actor.BranchID
	}
	if err := auth.RequireBranch(actor, input.BranchID); err != nil {
		return domain.ClassStudent{}, err
	}
	if input.JoinedAt.IsZero() || (input.LeftAt != nil && input.LeftAt.Before(input.JoinedAt)) {
		return domain.ClassStudent{}, domain.ErrInvalidInput
	}

	return s.classes.EnrollStudent(ctx, domain.ClassStudent{
		BranchID:  input.BranchID,
		ClassID:   strings.TrimSpace(input.ClassID),
		StudentID: strings.TrimSpace(input.StudentID),
		JoinedAt:  input.JoinedAt,
		LeftAt:    input.LeftAt,
	})
}

func (s *Service) ListClassStudents(ctx context.Context, actor domain.Principal, classID string) ([]domain.ClassStudent, error) {
	class, err := s.classes.GetClass(ctx, strings.TrimSpace(classID))
	if err != nil {
		return nil, err
	}
	if err := auth.RequireBranch(actor, class.BranchID); err != nil {
		return nil, err
	}

	return s.classes.ListClassStudents(ctx, class.BranchID, class.ID)
}

func (s *Service) ListClassStudentsPage(ctx context.Context, actor domain.Principal, classID string, page domain.PageRequest) ([]domain.ClassStudent, int, error) {
	class, err := s.classes.GetClass(ctx, strings.TrimSpace(classID))
	if err != nil {
		return nil, 0, err
	}
	if err := auth.RequireBranch(actor, class.BranchID); err != nil {
		return nil, 0, err
	}

	return s.classes.ListClassStudentsPage(ctx, class.BranchID, class.ID, page)
}

func (s *Service) CreateAssignment(ctx context.Context, actor domain.Principal, input CreateAssignmentInput) (domain.Assignment, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist, domain.RoleTeacher); err != nil {
		return domain.Assignment{}, err
	}
	if !actor.IsOwner() {
		input.BranchID = actor.BranchID
	}
	if err := auth.RequireBranch(actor, input.BranchID); err != nil {
		return domain.Assignment{}, err
	}
	if strings.TrimSpace(input.Title) == "" {
		return domain.Assignment{}, domain.ErrInvalidInput
	}
	if actor.Role == domain.RoleTeacher {
		class, err := s.classes.GetClass(ctx, input.ClassID)
		if err != nil {
			return domain.Assignment{}, err
		}
		teacher, err := s.teachers.GetTeacherByUser(ctx, actor.UserID)
		if err != nil {
			return domain.Assignment{}, err
		}
		if class.TeacherID != teacher.ID {
			return domain.Assignment{}, domain.ErrForbidden
		}
	}

	return s.assignments.CreateAssignment(ctx, domain.Assignment{
		BranchID:        input.BranchID,
		ClassID:         strings.TrimSpace(input.ClassID),
		Title:           strings.TrimSpace(input.Title),
		Description:     strings.TrimSpace(input.Description),
		DueAt:           input.DueAt,
		MaterialFileID:  strings.TrimSpace(input.MaterialFileID),
		CreatedByUserID: actor.UserID,
	})
}

func (s *Service) ListAssignments(ctx context.Context, actor domain.Principal, branchID string, classID string) ([]domain.Assignment, error) {
	if classID != "" {
		class, err := s.classes.GetClass(ctx, strings.TrimSpace(classID))
		if err != nil {
			return nil, err
		}
		if err := auth.RequireBranch(actor, class.BranchID); err != nil {
			return nil, err
		}
		if !s.canUseClass(ctx, actor, class) {
			return nil, domain.ErrForbidden
		}
		return s.assignments.ListAssignments(ctx, class.BranchID, class.ID)
	}
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	if actor.IsOwner() && branchID == "" {
		return s.assignments.ListAssignments(ctx, "", "")
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, err
	}

	assignments, err := s.assignments.ListAssignments(ctx, branchID, "")
	if err != nil {
		return nil, err
	}

	return s.filterAssignmentsForActor(ctx, actor, assignments)
}

func (s *Service) ListAssignmentsPage(ctx context.Context, actor domain.Principal, branchID string, classID string, page domain.PageRequest) ([]domain.Assignment, int, error) {
	if classID != "" {
		class, err := s.classes.GetClass(ctx, strings.TrimSpace(classID))
		if err != nil {
			return nil, 0, err
		}
		if err := auth.RequireBranch(actor, class.BranchID); err != nil {
			return nil, 0, err
		}
		if !s.canUseClass(ctx, actor, class) {
			return nil, 0, domain.ErrForbidden
		}
		return s.assignments.ListAssignmentsPage(ctx, class.BranchID, class.ID, page)
	}
	if actor.Role == domain.RoleTeacher || actor.Role == domain.RoleStudent {
		assignments, err := s.ListAssignments(ctx, actor, branchID, classID)
		if err != nil {
			return nil, 0, err
		}
		items, total := domain.PageSlice(assignments, page)
		return items, total, nil
	}
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	branchID = strings.TrimSpace(branchID)
	if actor.IsOwner() && branchID == "" {
		return s.assignments.ListAssignmentsPage(ctx, "", "", page)
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, 0, err
	}
	return s.assignments.ListAssignmentsPage(ctx, branchID, "", page)
}

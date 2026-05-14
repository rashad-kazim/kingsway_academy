package academic

import (
	"context"
	"strings"
	"time"

	"kingsway/backend/internal/auth"
	"kingsway/backend/internal/domain"
)

func (s *Service) CreateStudent(ctx context.Context, actor domain.Principal, input CreateStudentInput) (domain.Student, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist); err != nil {
		return domain.Student{}, err
	}
	if !actor.IsOwner() {
		input.BranchID = actor.BranchID
	}
	if err := auth.RequireBranch(actor, input.BranchID); err != nil {
		return domain.Student{}, err
	}

	fin, err := domain.ParseFIN(input.FIN)
	if err != nil {
		return domain.Student{}, err
	}

	if strings.TrimSpace(input.FirstName) == "" || strings.TrimSpace(input.LastName) == "" {
		return domain.Student{}, domain.ErrInvalidInput
	}
	if input.Gender != "" && input.Gender != "male" && input.Gender != "female" && input.Gender != "other" {
		return domain.Student{}, domain.ErrInvalidInput
	}
	if strings.TrimSpace(input.BirthDate) != "" && !validShortDate(input.BirthDate) {
		return domain.Student{}, domain.ErrInvalidInput
	}

	parents, err := buildStudentParentContacts(input.BranchID, input.Parents)
	if err != nil {
		return domain.Student{}, err
	}
	registrations, err := s.buildStudentCourseRegistrations(ctx, input.BranchID, input.Courses)
	if err != nil {
		return domain.Student{}, err
	}

	return s.students.CreateStudentWithDetails(ctx, domain.Student{
		BranchID:           input.BranchID,
		FIN:                fin,
		FirstName:          strings.TrimSpace(input.FirstName),
		LastName:           strings.TrimSpace(input.LastName),
		BirthDate:          strings.TrimSpace(input.BirthDate),
		Gender:             strings.TrimSpace(input.Gender),
		Phone:              strings.TrimSpace(input.Phone),
		Address:            strings.TrimSpace(input.Address),
		ProfilePhotoFileID: strings.TrimSpace(input.ProfilePhotoFileID),
		Status:             input.Status,
		LeftReason:         strings.TrimSpace(input.LeftReason),
	}, parents, registrations)
}

func (s *Service) GetStudentByFIN(ctx context.Context, actor domain.Principal, finValue string) (domain.Student, error) {
	fin, err := domain.ParseFIN(finValue)
	if err != nil {
		return domain.Student{}, err
	}

	student, err := s.students.GetStudentByFIN(ctx, fin)
	if err != nil {
		return domain.Student{}, err
	}
	if err := auth.RequireBranch(actor, student.BranchID); err != nil {
		return domain.Student{}, err
	}

	return student, nil
}

func buildStudentParentContacts(branchID string, input []StudentParentInput) ([]domain.StudentParentContact, error) {
	contacts := make([]domain.StudentParentContact, 0, len(input))
	for _, item := range input {
		relation := strings.TrimSpace(strings.ToLower(item.Relation))
		name := strings.TrimSpace(item.Name)
		if relation == "" && name == "" && len(item.Phones) == 0 {
			continue
		}
		if relation != "father" && relation != "mother" && relation != "sister" && relation != "brother" && relation != "other" {
			return nil, domain.ErrInvalidInput
		}
		if name == "" {
			return nil, domain.ErrInvalidInput
		}
		phones := make([]string, 0, len(item.Phones))
		for _, phone := range item.Phones {
			phone = strings.TrimSpace(phone)
			if phone != "" {
				phones = append(phones, phone)
			}
		}
		if len(phones) == 0 {
			return nil, domain.ErrInvalidInput
		}
		contacts = append(contacts, domain.StudentParentContact{
			BranchID: branchID,
			Relation: relation,
			Name:     name,
			Phones:   phones,
		})
	}

	return contacts, nil
}

func (s *Service) buildStudentCourseRegistrations(ctx context.Context, branchID string, input []StudentCourseAssignInput) ([]domain.StudentCourseRegistration, error) {
	registrations := make([]domain.StudentCourseRegistration, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, item := range input {
		courseID := strings.TrimSpace(item.CourseID)
		if courseID == "" {
			continue
		}
		if _, exists := seen[courseID]; exists {
			return nil, domain.ErrInvalidInput
		}
		seen[courseID] = struct{}{}

		course, err := s.courses.GetCourse(ctx, courseID)
		if err != nil {
			return nil, err
		}
		if course.BranchID != branchID || !course.IsActive || item.MonthlyAmountCents < 0 {
			return nil, domain.ErrInvalidInput
		}
		if !validShortDate(item.StartDate) {
			return nil, domain.ErrInvalidInput
		}

		teacherID := strings.TrimSpace(item.TeacherID)
		if teacherID != "" {
			teacher, err := s.teachers.GetTeacher(ctx, teacherID)
			if err != nil {
				return nil, err
			}
			if teacher.BranchID != branchID || teacher.Status != domain.TeacherStatusActive {
				return nil, domain.ErrInvalidInput
			}
		}

		registrations = append(registrations, domain.StudentCourseRegistration{
			BranchID:           branchID,
			CourseID:           courseID,
			TeacherID:          teacherID,
			MonthlyAmountCents: item.MonthlyAmountCents,
			StartDate:          strings.TrimSpace(item.StartDate),
		})
	}

	return registrations, nil
}

func validShortDate(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	_, err := time.Parse("02/01/2006", value)
	return err == nil
}

func (s *Service) GetStudent(ctx context.Context, actor domain.Principal, id string) (domain.Student, error) {
	student, err := s.students.GetStudent(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.Student{}, err
	}
	if err := auth.RequireBranch(actor, student.BranchID); err != nil {
		return domain.Student{}, err
	}
	if actor.Role == domain.RoleStudent && student.UserID != actor.UserID {
		return domain.Student{}, domain.ErrForbidden
	}

	return student, nil
}

func (s *Service) UpdateStudent(ctx context.Context, actor domain.Principal, id string, input UpdateStudentInput) (domain.Student, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist); err != nil {
		return domain.Student{}, err
	}
	student, err := s.students.GetStudent(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.Student{}, err
	}
	if err := auth.RequireBranch(actor, student.BranchID); err != nil {
		return domain.Student{}, err
	}
	if strings.TrimSpace(input.FirstName) != "" {
		student.FirstName = strings.TrimSpace(input.FirstName)
	}
	if strings.TrimSpace(input.LastName) != "" {
		student.LastName = strings.TrimSpace(input.LastName)
	}
	if strings.TrimSpace(input.BirthDate) != "" && !validShortDate(input.BirthDate) {
		return domain.Student{}, domain.ErrInvalidInput
	}
	if input.Gender != "" && input.Gender != "male" && input.Gender != "female" && input.Gender != "other" {
		return domain.Student{}, domain.ErrInvalidInput
	}
	student.BirthDate = strings.TrimSpace(input.BirthDate)
	student.Gender = strings.TrimSpace(input.Gender)
	student.Phone = strings.TrimSpace(input.Phone)
	student.Address = strings.TrimSpace(input.Address)
	student.ProfilePhotoFileID = strings.TrimSpace(input.ProfilePhotoFileID)
	if input.Status != "" {
		if input.Status != domain.StudentStatusActive && input.Status != domain.StudentStatusLeft && input.Status != domain.StudentStatusGraduated {
			return domain.Student{}, domain.ErrInvalidInput
		}
		student.Status = input.Status
	}
	student.LeftReason = strings.TrimSpace(input.LeftReason)

	return s.students.UpdateStudent(ctx, student)
}

func (s *Service) CreateStudentAccount(ctx context.Context, actor domain.Principal, studentID string, input CreateStudentAccountInput) (domain.Student, domain.User, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist); err != nil {
		return domain.Student{}, domain.User{}, err
	}
	student, err := s.students.GetStudent(ctx, strings.TrimSpace(studentID))
	if err != nil {
		return domain.Student{}, domain.User{}, err
	}
	if err := auth.RequireBranch(actor, student.BranchID); err != nil {
		return domain.Student{}, domain.User{}, err
	}
	if student.UserID != "" {
		return domain.Student{}, domain.User{}, domain.ErrConflict
	}
	if strings.TrimSpace(input.FirstName) == "" {
		input.FirstName = student.FirstName
	}
	if strings.TrimSpace(input.LastName) == "" {
		input.LastName = student.LastName
	}

	user, err := s.auth.CreateUser(ctx, actor, auth.CreateUserInput{
		BranchID:  student.BranchID,
		Role:      domain.RoleStudent,
		Email:     input.Email,
		Password:  input.Password,
		FirstName: input.FirstName,
		LastName:  input.LastName,
	})
	if err != nil {
		return domain.Student{}, domain.User{}, err
	}
	student.UserID = user.ID
	updated, err := s.students.UpdateStudent(ctx, student)
	if err != nil {
		return domain.Student{}, domain.User{}, err
	}

	return updated, user, nil
}

func (s *Service) ListStudents(ctx context.Context, actor domain.Principal, branchID string) ([]domain.Student, error) {
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	if actor.IsOwner() && branchID == "" {
		return s.students.ListStudents(ctx, "")
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, err
	}

	students, err := s.students.ListStudents(ctx, branchID)
	if err != nil {
		return nil, err
	}
	return s.filterStudentsForActor(ctx, actor, students)
}

func (s *Service) ListStudentsPage(ctx context.Context, actor domain.Principal, branchID string, status domain.StudentStatus, page domain.PageRequest) ([]domain.Student, int, error) {
	if status != "" && status != domain.StudentStatusActive && status != domain.StudentStatusLeft && status != domain.StudentStatusGraduated {
		return nil, 0, domain.ErrInvalidInput
	}
	if actor.Role == domain.RoleTeacher || actor.Role == domain.RoleStudent {
		students, err := s.ListStudents(ctx, actor, branchID)
		if err != nil {
			return nil, 0, err
		}
		if status != "" {
			filtered := make([]domain.Student, 0, len(students))
			for _, student := range students {
				if student.Status == status {
					filtered = append(filtered, student)
				}
			}
			students = filtered
		}
		items, total := domain.PageSlice(students, page)
		return items, total, nil
	}

	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	branchID = strings.TrimSpace(branchID)
	if actor.IsOwner() && branchID == "" {
		return s.students.ListStudentsPage(ctx, "", status, page)
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, 0, err
	}
	return s.students.ListStudentsPage(ctx, branchID, status, page)
}

func (s *Service) ListStudentAssignmentHub(ctx context.Context, actor domain.Principal, filter domain.StudentAssignmentHubFilter) (domain.StudentAssignmentHubPage, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist); err != nil {
		return domain.StudentAssignmentHubPage{}, err
	}
	if !actor.IsOwner() {
		filter.BranchID = actor.BranchID
	}
	if filter.BranchID != "" {
		if err := auth.RequireBranch(actor, filter.BranchID); err != nil {
			return domain.StudentAssignmentHubPage{}, err
		}
	}
	filter.Query = strings.TrimSpace(filter.Query)
	filter.TeacherID = strings.TrimSpace(filter.TeacherID)
	filter.BranchID = strings.TrimSpace(filter.BranchID)
	if filter.Status != "" && filter.Status != domain.StudentStatusActive && filter.Status != domain.StudentStatusLeft && filter.Status != domain.StudentStatusGraduated {
		return domain.StudentAssignmentHubPage{}, domain.ErrInvalidInput
	}
	if filter.Limit <= 0 || filter.Limit > 500 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	return s.students.ListStudentAssignmentHub(ctx, filter)
}

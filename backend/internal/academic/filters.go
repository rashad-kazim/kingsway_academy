package academic

import (
	"context"
	"errors"

	"kingsway/backend/internal/domain"
)

func (s *Service) filterClassesForActor(ctx context.Context, actor domain.Principal, classes []domain.Class) ([]domain.Class, error) {
	switch actor.Role {
	case domain.RoleOwner, domain.RoleReceptionist:
		return classes, nil
	case domain.RoleTeacher:
		teacher, err := s.teachers.GetTeacherByUser(ctx, actor.UserID)
		if err != nil {
			return nil, err
		}
		filtered := make([]domain.Class, 0, len(classes))
		for _, class := range classes {
			if class.TeacherID == teacher.ID {
				filtered = append(filtered, class)
			}
		}
		return filtered, nil
	case domain.RoleStudent:
		student, err := s.students.GetStudentByUser(ctx, actor.UserID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return []domain.Class{}, nil
			}
			return nil, err
		}
		enrollments, err := s.classes.ListClassStudents(ctx, student.BranchID, "")
		if err != nil {
			return nil, err
		}
		allowed := make(map[string]bool)
		for _, enrollment := range enrollments {
			if enrollment.StudentID == student.ID && enrollment.LeftAt == nil {
				allowed[enrollment.ClassID] = true
			}
		}
		filtered := make([]domain.Class, 0, len(classes))
		for _, class := range classes {
			if allowed[class.ID] {
				filtered = append(filtered, class)
			}
		}
		return filtered, nil
	default:
		return nil, domain.ErrForbidden
	}
}

func (s *Service) filterStudentsForActor(ctx context.Context, actor domain.Principal, students []domain.Student) ([]domain.Student, error) {
	switch actor.Role {
	case domain.RoleOwner, domain.RoleReceptionist:
		return students, nil
	case domain.RoleTeacher:
		teacher, err := s.teachers.GetTeacherByUser(ctx, actor.UserID)
		if err != nil {
			return nil, err
		}
		classes, err := s.classes.ListClasses(ctx, teacher.BranchID)
		if err != nil {
			return nil, err
		}
		classIDs := make(map[string]bool)
		for _, class := range classes {
			if class.TeacherID == teacher.ID {
				classIDs[class.ID] = true
			}
		}
		enrollments, err := s.classes.ListClassStudents(ctx, teacher.BranchID, "")
		if err != nil {
			return nil, err
		}
		studentIDs := make(map[string]bool)
		for _, enrollment := range enrollments {
			if classIDs[enrollment.ClassID] && enrollment.LeftAt == nil {
				studentIDs[enrollment.StudentID] = true
			}
		}
		filtered := make([]domain.Student, 0, len(students))
		for _, student := range students {
			if studentIDs[student.ID] {
				filtered = append(filtered, student)
			}
		}
		return filtered, nil
	case domain.RoleStudent:
		student, err := s.students.GetStudentByUser(ctx, actor.UserID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return []domain.Student{}, nil
			}
			return nil, err
		}
		return []domain.Student{student}, nil
	default:
		return nil, domain.ErrForbidden
	}
}

func (s *Service) filterAssignmentsForActor(ctx context.Context, actor domain.Principal, assignments []domain.Assignment) ([]domain.Assignment, error) {
	filtered := make([]domain.Assignment, 0, len(assignments))
	for _, assignment := range assignments {
		class, err := s.classes.GetClass(ctx, assignment.ClassID)
		if err != nil {
			return nil, err
		}
		if s.canUseClass(ctx, actor, class) {
			filtered = append(filtered, assignment)
		}
	}

	return filtered, nil
}

func (s *Service) filterScheduleForActor(ctx context.Context, actor domain.Principal, items []domain.ScheduleItem) ([]domain.ScheduleItem, error) {
	switch actor.Role {
	case domain.RoleOwner, domain.RoleReceptionist:
		return items, nil
	case domain.RoleTeacher:
		teacher, err := s.teachers.GetTeacherByUser(ctx, actor.UserID)
		if err != nil {
			return nil, err
		}
		filtered := make([]domain.ScheduleItem, 0, len(items))
		for _, item := range items {
			if item.TeacherID == teacher.ID {
				filtered = append(filtered, item)
			}
		}
		return filtered, nil
	case domain.RoleStudent:
		student, err := s.students.GetStudentByUser(ctx, actor.UserID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return []domain.ScheduleItem{}, nil
			}
			return nil, err
		}
		enrollments, err := s.classes.ListClassStudents(ctx, student.BranchID, "")
		if err != nil {
			return nil, err
		}
		allowed := make(map[string]bool)
		for _, enrollment := range enrollments {
			if enrollment.StudentID == student.ID && enrollment.LeftAt == nil {
				allowed[enrollment.ClassID] = true
			}
		}
		filtered := make([]domain.ScheduleItem, 0, len(items))
		for _, item := range items {
			if item.ClassID != "" && allowed[item.ClassID] {
				filtered = append(filtered, item)
			}
		}
		return filtered, nil
	default:
		return nil, domain.ErrForbidden
	}
}

func (s *Service) filterExamsForActor(ctx context.Context, actor domain.Principal, exams []domain.Exam) ([]domain.Exam, error) {
	switch actor.Role {
	case domain.RoleOwner, domain.RoleReceptionist:
		return exams, nil
	case domain.RoleTeacher:
		teacher, err := s.teachers.GetTeacherByUser(ctx, actor.UserID)
		if err != nil {
			return nil, err
		}
		filtered := make([]domain.Exam, 0, len(exams))
		for _, exam := range exams {
			if exam.CreatedByUserID == actor.UserID {
				filtered = append(filtered, exam)
				continue
			}
			if exam.ClassID == "" {
				continue
			}
			class, err := s.classes.GetClass(ctx, exam.ClassID)
			if err != nil {
				return nil, err
			}
			if class.TeacherID == teacher.ID {
				filtered = append(filtered, exam)
			}
		}
		return filtered, nil
	case domain.RoleStudent:
		student, err := s.students.GetStudentByUser(ctx, actor.UserID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return []domain.Exam{}, nil
			}
			return nil, err
		}
		enrollments, err := s.classes.ListClassStudents(ctx, student.BranchID, "")
		if err != nil {
			return nil, err
		}
		allowed := make(map[string]bool)
		for _, enrollment := range enrollments {
			if enrollment.StudentID == student.ID && enrollment.LeftAt == nil {
				allowed[enrollment.ClassID] = true
			}
		}
		filtered := make([]domain.Exam, 0, len(exams))
		for _, exam := range exams {
			if exam.ClassID != "" && allowed[exam.ClassID] {
				filtered = append(filtered, exam)
			}
		}
		return filtered, nil
	default:
		return nil, domain.ErrForbidden
	}
}

func (s *Service) filterExamResultsForActor(ctx context.Context, actor domain.Principal, results []domain.ExamResult) ([]domain.ExamResult, error) {
	switch actor.Role {
	case domain.RoleOwner, domain.RoleReceptionist:
		return results, nil
	case domain.RoleTeacher:
		teacher, err := s.teachers.GetTeacherByUser(ctx, actor.UserID)
		if err != nil {
			return nil, err
		}
		filtered := make([]domain.ExamResult, 0, len(results))
		for _, result := range results {
			if result.EnteredByTeacherID == teacher.ID {
				filtered = append(filtered, result)
				continue
			}
			exam, err := s.exams.GetExam(ctx, result.ExamID)
			if err != nil {
				return nil, err
			}
			if exam.ClassID == "" {
				continue
			}
			class, err := s.classes.GetClass(ctx, exam.ClassID)
			if err != nil {
				return nil, err
			}
			if class.TeacherID == teacher.ID {
				filtered = append(filtered, result)
			}
		}
		return filtered, nil
	case domain.RoleStudent:
		student, err := s.students.GetStudentByUser(ctx, actor.UserID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return []domain.ExamResult{}, nil
			}
			return nil, err
		}
		filtered := make([]domain.ExamResult, 0, len(results))
		for _, result := range results {
			if result.StudentID == student.ID {
				filtered = append(filtered, result)
			}
		}
		return filtered, nil
	default:
		return nil, domain.ErrForbidden
	}
}

func (s *Service) canUseClass(ctx context.Context, actor domain.Principal, class domain.Class) bool {
	switch actor.Role {
	case domain.RoleOwner, domain.RoleReceptionist:
		return true
	case domain.RoleTeacher:
		teacher, err := s.teachers.GetTeacherByUser(ctx, actor.UserID)
		return err == nil && class.TeacherID == teacher.ID
	case domain.RoleStudent:
		student, err := s.students.GetStudentByUser(ctx, actor.UserID)
		if err != nil {
			return false
		}
		enrollments, err := s.classes.ListClassStudents(ctx, student.BranchID, class.ID)
		if err != nil {
			return false
		}
		for _, enrollment := range enrollments {
			if enrollment.StudentID == student.ID && enrollment.LeftAt == nil {
				return true
			}
		}
		return false
	default:
		return false
	}
}

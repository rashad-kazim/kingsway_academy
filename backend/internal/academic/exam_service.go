package academic

import (
	"context"
	"strings"

	"kingsway/backend/internal/auth"
	"kingsway/backend/internal/domain"
)

func (s *Service) CreateExam(ctx context.Context, actor domain.Principal, input CreateExamInput) (domain.Exam, []domain.ExamParticipant, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist, domain.RoleTeacher); err != nil {
		return domain.Exam{}, nil, err
	}
	if !actor.IsOwner() {
		input.BranchID = actor.BranchID
	}
	if err := auth.RequireBranch(actor, input.BranchID); err != nil {
		return domain.Exam{}, nil, err
	}
	if strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.CourseID) == "" {
		return domain.Exam{}, nil, domain.ErrInvalidInput
	}
	if actor.Role == domain.RoleTeacher && input.ClassID != "" {
		class, err := s.classes.GetClass(ctx, input.ClassID)
		if err != nil {
			return domain.Exam{}, nil, err
		}
		teacher, err := s.teachers.GetTeacherByUser(ctx, actor.UserID)
		if err != nil {
			return domain.Exam{}, nil, err
		}
		if class.TeacherID != teacher.ID {
			return domain.Exam{}, nil, domain.ErrForbidden
		}
	}

	return s.exams.CreateExam(ctx, domain.Exam{
		BranchID:        input.BranchID,
		CourseID:        strings.TrimSpace(input.CourseID),
		ClassID:         strings.TrimSpace(input.ClassID),
		ScheduleItemID:  strings.TrimSpace(input.ScheduleItemID),
		Title:           strings.TrimSpace(input.Title),
		CreatedByUserID: actor.UserID,
	}, input.ParticipantStudentIDs)
}

func (s *Service) GetExam(ctx context.Context, actor domain.Principal, id string) (domain.Exam, error) {
	exam, err := s.exams.GetExam(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.Exam{}, err
	}
	if err := auth.RequireBranch(actor, exam.BranchID); err != nil {
		return domain.Exam{}, err
	}
	filtered, err := s.filterExamsForActor(ctx, actor, []domain.Exam{exam})
	if err != nil {
		return domain.Exam{}, err
	}
	if len(filtered) == 0 {
		return domain.Exam{}, domain.ErrForbidden
	}

	return exam, nil
}

func (s *Service) ListExams(ctx context.Context, actor domain.Principal, branchID string, classID string) ([]domain.Exam, error) {
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
		return s.exams.ListExams(ctx, class.BranchID, class.ID)
	}
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	if actor.IsOwner() && branchID == "" {
		return s.exams.ListExams(ctx, "", "")
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, err
	}

	exams, err := s.exams.ListExams(ctx, branchID, "")
	if err != nil {
		return nil, err
	}

	return s.filterExamsForActor(ctx, actor, exams)
}

func (s *Service) ListExamsPage(ctx context.Context, actor domain.Principal, branchID string, classID string, page domain.PageRequest) ([]domain.Exam, int, error) {
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
		return s.exams.ListExamsPage(ctx, class.BranchID, class.ID, page)
	}
	if actor.Role == domain.RoleTeacher || actor.Role == domain.RoleStudent {
		exams, err := s.ListExams(ctx, actor, branchID, classID)
		if err != nil {
			return nil, 0, err
		}
		items, total := domain.PageSlice(exams, page)
		return items, total, nil
	}
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	branchID = strings.TrimSpace(branchID)
	if actor.IsOwner() && branchID == "" {
		return s.exams.ListExamsPage(ctx, "", "", page)
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, 0, err
	}
	return s.exams.ListExamsPage(ctx, branchID, "", page)
}

func (s *Service) CreateExamResult(ctx context.Context, actor domain.Principal, input CreateExamResultInput) (domain.ExamResult, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleTeacher); err != nil {
		return domain.ExamResult{}, err
	}
	if !actor.IsOwner() {
		input.BranchID = actor.BranchID
	}
	if err := auth.RequireBranch(actor, input.BranchID); err != nil {
		return domain.ExamResult{}, err
	}

	teacherID := ""
	if actor.Role == domain.RoleTeacher {
		teacher, err := s.teachers.GetTeacherByUser(ctx, actor.UserID)
		if err != nil {
			return domain.ExamResult{}, err
		}
		teacherID = teacher.ID
		exam, err := s.exams.GetExam(ctx, input.ExamID)
		if err != nil {
			return domain.ExamResult{}, err
		}
		if exam.ClassID != "" {
			class, err := s.classes.GetClass(ctx, exam.ClassID)
			if err != nil {
				return domain.ExamResult{}, err
			}
			if class.TeacherID != teacher.ID {
				return domain.ExamResult{}, domain.ErrForbidden
			}
		} else if exam.CreatedByUserID != actor.UserID {
			return domain.ExamResult{}, domain.ErrForbidden
		}
	}
	if teacherID == "" {
		teacherID = strings.TrimSpace(input.EnteredByTeacherID)
	}
	if teacherID == "" {
		return domain.ExamResult{}, domain.ErrInvalidInput
	}

	return s.exams.CreateExamResult(ctx, domain.ExamResult{
		BranchID:           input.BranchID,
		ExamID:             strings.TrimSpace(input.ExamID),
		StudentID:          strings.TrimSpace(input.StudentID),
		CategoryID:         strings.TrimSpace(input.CategoryID),
		Score:              input.Score,
		Feedback:           strings.TrimSpace(input.Feedback),
		DocumentFileID:     strings.TrimSpace(input.DocumentFileID),
		EnteredByTeacherID: teacherID,
	})
}

func (s *Service) ListExamResults(ctx context.Context, actor domain.Principal, branchID string, examID string, studentID string) ([]domain.ExamResult, error) {
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	if actor.IsOwner() && branchID == "" {
		return s.exams.ListExamResults(ctx, "", strings.TrimSpace(examID), strings.TrimSpace(studentID))
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, err
	}

	results, err := s.exams.ListExamResults(ctx, branchID, strings.TrimSpace(examID), strings.TrimSpace(studentID))
	if err != nil {
		return nil, err
	}

	return s.filterExamResultsForActor(ctx, actor, results)
}

func (s *Service) ListExamResultsPage(ctx context.Context, actor domain.Principal, branchID string, examID string, studentID string, page domain.PageRequest) ([]domain.ExamResult, int, error) {
	if actor.Role == domain.RoleTeacher || actor.Role == domain.RoleStudent {
		results, err := s.ListExamResults(ctx, actor, branchID, examID, studentID)
		if err != nil {
			return nil, 0, err
		}
		items, total := domain.PageSlice(results, page)
		return items, total, nil
	}
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	branchID = strings.TrimSpace(branchID)
	if actor.IsOwner() && branchID == "" {
		return s.exams.ListExamResultsPage(ctx, "", strings.TrimSpace(examID), strings.TrimSpace(studentID), page)
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, 0, err
	}
	return s.exams.ListExamResultsPage(ctx, branchID, strings.TrimSpace(examID), strings.TrimSpace(studentID), page)
}

func (s *Service) AcademicDashboard(ctx context.Context, actor domain.Principal, branchID string) (domain.AcademicDashboard, error) {
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	if actor.IsOwner() && branchID == "" {
		return s.dashboards.AcademicDashboard(ctx, "")
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return domain.AcademicDashboard{}, err
	}

	return s.dashboards.AcademicDashboard(ctx, branchID)
}

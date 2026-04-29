package store

import (
	"context"
	"sort"
	"strings"
	"time"

	"kingsway/backend/internal/domain"
)

func (m *Memory) CreateClass(_ context.Context, class domain.Class) (domain.Class, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if strings.TrimSpace(class.Name) == "" || class.CourseID == "" || class.TeacherID == "" || class.StartDate.IsZero() {
		return domain.Class{}, domain.ErrInvalidInput
	}
	course, courseOK := m.courses[class.CourseID]
	teacher, teacherOK := m.teachers[class.TeacherID]
	if !courseOK || !teacherOK {
		return domain.Class{}, domain.ErrNotFound
	}
	if course.BranchID != class.BranchID || teacher.BranchID != class.BranchID || teacher.Status != domain.TeacherStatusActive {
		return domain.Class{}, domain.ErrInvalidInput
	}
	for _, existing := range m.classes {
		if existing.BranchID == class.BranchID && strings.EqualFold(existing.Name, class.Name) {
			return domain.Class{}, domain.ErrConflict
		}
	}

	ts := now()
	class.ID = newID()
	class.Name = strings.TrimSpace(class.Name)
	class.IsActive = true
	class.CreatedAt = ts
	class.UpdatedAt = ts
	m.classes[class.ID] = class

	return class, nil
}

func (m *Memory) GetClass(_ context.Context, id string) (domain.Class, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	class, ok := m.classes[id]
	if !ok {
		return domain.Class{}, domain.ErrNotFound
	}

	return class, nil
}

func (m *Memory) ListClasses(_ context.Context, branchID string) ([]domain.Class, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	classes := make([]domain.Class, 0)
	for _, class := range m.classes {
		if branchID == "" || class.BranchID == branchID {
			classes = append(classes, class)
		}
	}
	sort.Slice(classes, func(i, j int) bool { return classes[i].StartDate.After(classes[j].StartDate) })

	return classes, nil
}

func (m *Memory) EnrollStudent(_ context.Context, enrollment domain.ClassStudent) (domain.ClassStudent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	class, classOK := m.classes[enrollment.ClassID]
	student, studentOK := m.students[enrollment.StudentID]
	if !classOK || !studentOK {
		return domain.ClassStudent{}, domain.ErrNotFound
	}
	if class.BranchID != enrollment.BranchID || student.BranchID != enrollment.BranchID || enrollment.JoinedAt.IsZero() {
		return domain.ClassStudent{}, domain.ErrInvalidInput
	}
	if enrollment.LeftAt != nil && enrollment.LeftAt.Before(enrollment.JoinedAt) {
		return domain.ClassStudent{}, domain.ErrInvalidInput
	}
	for _, existing := range m.classStudents {
		if existing.ClassID == enrollment.ClassID && existing.StudentID == enrollment.StudentID && sameDay(existing.JoinedAt, enrollment.JoinedAt) {
			return domain.ClassStudent{}, domain.ErrConflict
		}
	}

	enrollment.ID = newID()
	enrollment.CreatedAt = now()
	m.classStudents[enrollment.ID] = enrollment

	return enrollment, nil
}

func (m *Memory) ListClassStudents(_ context.Context, branchID string, classID string) ([]domain.ClassStudent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	enrollments := make([]domain.ClassStudent, 0)
	for _, enrollment := range m.classStudents {
		if (branchID == "" || enrollment.BranchID == branchID) && (classID == "" || enrollment.ClassID == classID) {
			enrollments = append(enrollments, enrollment)
		}
	}

	return enrollments, nil
}

func (m *Memory) CreateAssignment(_ context.Context, assignment domain.Assignment) (domain.Assignment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	class, ok := m.classes[assignment.ClassID]
	if !ok {
		return domain.Assignment{}, domain.ErrNotFound
	}
	if class.BranchID != assignment.BranchID || strings.TrimSpace(assignment.Title) == "" || assignment.CreatedByUserID == "" {
		return domain.Assignment{}, domain.ErrInvalidInput
	}

	ts := now()
	assignment.ID = newID()
	assignment.Title = strings.TrimSpace(assignment.Title)
	assignment.Description = strings.TrimSpace(assignment.Description)
	assignment.CreatedAt = ts
	assignment.UpdatedAt = ts
	m.assignments[assignment.ID] = assignment

	return assignment, nil
}

func (m *Memory) ListAssignments(_ context.Context, branchID string, classID string) ([]domain.Assignment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	assignments := make([]domain.Assignment, 0)
	for _, assignment := range m.assignments {
		if (branchID == "" || assignment.BranchID == branchID) && (classID == "" || assignment.ClassID == classID) {
			assignments = append(assignments, assignment)
		}
	}

	return assignments, nil
}

func (m *Memory) CreateExam(_ context.Context, exam domain.Exam, participantStudentIDs []string) (domain.Exam, []domain.ExamParticipant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	course, ok := m.courses[exam.CourseID]
	if !ok {
		return domain.Exam{}, nil, domain.ErrNotFound
	}
	if course.BranchID != exam.BranchID || strings.TrimSpace(exam.Title) == "" || exam.CreatedByUserID == "" {
		return domain.Exam{}, nil, domain.ErrInvalidInput
	}
	if exam.ClassID != "" {
		class, ok := m.classes[exam.ClassID]
		if !ok {
			return domain.Exam{}, nil, domain.ErrNotFound
		}
		if class.BranchID != exam.BranchID || class.CourseID != exam.CourseID {
			return domain.Exam{}, nil, domain.ErrInvalidInput
		}
	}

	ts := now()
	exam.ID = newID()
	exam.Title = strings.TrimSpace(exam.Title)
	exam.CreatedAt = ts
	exam.UpdatedAt = ts
	m.exams[exam.ID] = exam

	participants := make([]domain.ExamParticipant, 0, len(participantStudentIDs))
	for _, studentID := range participantStudentIDs {
		studentID = strings.TrimSpace(studentID)
		if studentID == "" {
			continue
		}
		student, ok := m.students[studentID]
		if !ok {
			return domain.Exam{}, nil, domain.ErrNotFound
		}
		if student.BranchID != exam.BranchID {
			return domain.Exam{}, nil, domain.ErrInvalidInput
		}
		participant := domain.ExamParticipant{
			ExamID:     exam.ID,
			StudentID:  studentID,
			BranchID:   exam.BranchID,
			RSVPStatus: domain.RSVPStatusPending,
		}
		m.examParticipants[exam.ID+":"+studentID] = participant
		participants = append(participants, participant)
	}

	return exam, participants, nil
}

func (m *Memory) ListExams(_ context.Context, branchID string, classID string) ([]domain.Exam, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	exams := make([]domain.Exam, 0)
	for _, exam := range m.exams {
		if (branchID == "" || exam.BranchID == branchID) && (classID == "" || exam.ClassID == classID) {
			exams = append(exams, exam)
		}
	}
	sort.Slice(exams, func(i, j int) bool { return exams[i].CreatedAt.After(exams[j].CreatedAt) })

	return exams, nil
}

func (m *Memory) GetExam(_ context.Context, id string) (domain.Exam, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	exam, ok := m.exams[id]
	if !ok {
		return domain.Exam{}, domain.ErrNotFound
	}

	return exam, nil
}

func (m *Memory) CreateExamResult(_ context.Context, result domain.ExamResult) (domain.ExamResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	exam, examOK := m.exams[result.ExamID]
	student, studentOK := m.students[result.StudentID]
	teacher, teacherOK := m.teachers[result.EnteredByTeacherID]
	if !examOK || !studentOK || !teacherOK {
		return domain.ExamResult{}, domain.ErrNotFound
	}
	if exam.BranchID != result.BranchID || student.BranchID != result.BranchID || teacher.BranchID != result.BranchID {
		return domain.ExamResult{}, domain.ErrInvalidInput
	}

	var categoryOK bool
	for _, category := range m.courseCategories[exam.CourseID] {
		if category.ID == result.CategoryID {
			categoryOK = true
			if result.Score < category.MinScore || result.Score > category.MaxScore {
				return domain.ExamResult{}, domain.ErrInvalidInput
			}
		}
	}
	if !categoryOK {
		return domain.ExamResult{}, domain.ErrNotFound
	}

	ts := now()
	key := result.ExamID + ":" + result.StudentID + ":" + result.CategoryID
	if existing, ok := m.examResults[key]; ok {
		result.ID = existing.ID
		result.CreatedAt = existing.CreatedAt
	} else {
		result.ID = newID()
		result.CreatedAt = ts
	}
	result.UpdatedAt = ts
	result.Feedback = strings.TrimSpace(result.Feedback)
	m.examResults[key] = result

	return result, nil
}

func (m *Memory) ListExamResults(_ context.Context, branchID string, examID string, studentID string) ([]domain.ExamResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]domain.ExamResult, 0)
	for _, result := range m.examResults {
		if (branchID == "" || result.BranchID == branchID) && (examID == "" || result.ExamID == examID) && (studentID == "" || result.StudentID == studentID) {
			results = append(results, result)
		}
	}

	return results, nil
}

func (m *Memory) AcademicDashboard(_ context.Context, branchID string) (domain.AcademicDashboard, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dashboard := domain.AcademicDashboard{BranchID: branchID}
	for _, student := range m.students {
		if (branchID == "" || student.BranchID == branchID) && student.Status == domain.StudentStatusActive {
			dashboard.ActiveStudents++
		}
	}
	for _, teacher := range m.teachers {
		if (branchID == "" || teacher.BranchID == branchID) && teacher.Status == domain.TeacherStatusActive {
			dashboard.ActiveTeachers++
		}
	}
	for _, class := range m.classes {
		if (branchID == "" || class.BranchID == branchID) && class.IsActive {
			dashboard.ActiveClasses++
		}
	}
	for _, item := range m.schedules {
		if (branchID == "" || item.BranchID == branchID) && item.CancelledAt == nil && item.StartsAt.After(now()) {
			dashboard.UpcomingSchedule++
		}
	}
	for _, payment := range m.payments {
		if (branchID == "" || payment.BranchID == branchID) && (payment.Status == domain.PaymentStatusPending || payment.Status == domain.PaymentStatusOverdue) {
			dashboard.PendingPayments++
		}
	}
	for _, file := range m.files {
		if (branchID == "" || file.BranchID == branchID) && file.Purpose == domain.FilePurposeExamWriting && file.DeletedAt == nil {
			dashboard.WritingFilesRetained++
		}
	}

	return dashboard, nil
}

func sameDay(a time.Time, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

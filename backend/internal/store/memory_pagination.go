package store

import (
	"context"
	"strings"

	"kingsway/backend/internal/domain"
)

func (m *Memory) ListBranchesPage(ctx context.Context, page domain.PageRequest) ([]domain.Branch, int, error) {
	items, err := m.ListBranches(ctx)
	if err != nil {
		return nil, 0, err
	}
	paged, total := pageSlice(items, page)
	return paged, total, nil
}

func (m *Memory) ListStaffMembersPage(ctx context.Context, branchID string, page domain.PageRequest) ([]domain.StaffMember, int, error) {
	items, err := m.ListStaffMembers(ctx, branchID)
	if err != nil {
		return nil, 0, err
	}
	paged, total := pageSlice(items, page)
	return paged, total, nil
}

func (m *Memory) ListStudentsPage(ctx context.Context, branchID string, status domain.StudentStatus, page domain.PageRequest) ([]domain.Student, int, error) {
	items, err := m.ListStudents(ctx, branchID)
	if err != nil {
		return nil, 0, err
	}
	if status != "" {
		filtered := make([]domain.Student, 0, len(items))
		for _, item := range items {
			if item.Status == status {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	paged, total := pageSlice(items, page)
	return paged, total, nil
}

func (m *Memory) ListTeachersPage(ctx context.Context, branchID string, status domain.TeacherStatus, page domain.PageRequest) ([]domain.Teacher, int, error) {
	items, err := m.ListTeachers(ctx, branchID)
	if err != nil {
		return nil, 0, err
	}
	if status != "" {
		filtered := make([]domain.Teacher, 0, len(items))
		for _, item := range items {
			if item.Status == status {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	paged, total := pageSlice(items, page)
	return paged, total, nil
}

func (m *Memory) ListCoursesPage(ctx context.Context, branchID string, page domain.PageRequest) ([]domain.Course, int, error) {
	items, err := m.ListCourses(ctx, branchID)
	if err != nil {
		return nil, 0, err
	}
	paged, total := pageSlice(items, page)
	return paged, total, nil
}

func (m *Memory) ListClassesPage(ctx context.Context, branchID string, active *bool, page domain.PageRequest) ([]domain.Class, int, error) {
	items, err := m.ListClasses(ctx, branchID)
	if err != nil {
		return nil, 0, err
	}
	if active != nil {
		filtered := make([]domain.Class, 0, len(items))
		for _, item := range items {
			if item.IsActive == *active {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	paged, total := pageSlice(items, page)
	return paged, total, nil
}

func (m *Memory) ListClassStudentsPage(ctx context.Context, branchID string, classID string, page domain.PageRequest) ([]domain.ClassStudent, int, error) {
	items, err := m.ListClassStudents(ctx, branchID, classID)
	if err != nil {
		return nil, 0, err
	}
	paged, total := pageSlice(items, page)
	return paged, total, nil
}

func (m *Memory) ListAssignmentsPage(ctx context.Context, branchID string, classID string, page domain.PageRequest) ([]domain.Assignment, int, error) {
	items, err := m.ListAssignments(ctx, branchID, classID)
	if err != nil {
		return nil, 0, err
	}
	paged, total := pageSlice(items, page)
	return paged, total, nil
}

func (m *Memory) ListRoomsPage(ctx context.Context, branchID string, page domain.PageRequest) ([]domain.Room, int, error) {
	items, err := m.ListRooms(ctx, branchID)
	if err != nil {
		return nil, 0, err
	}
	paged, total := pageSlice(items, page)
	return paged, total, nil
}

func (m *Memory) ListSchedulePage(ctx context.Context, branchID string, itemType domain.ScheduleItemType, page domain.PageRequest) ([]domain.ScheduleItem, int, error) {
	items, err := m.ListSchedule(ctx, branchID)
	if err != nil {
		return nil, 0, err
	}
	if itemType != "" {
		filtered := make([]domain.ScheduleItem, 0, len(items))
		for _, item := range items {
			if item.ItemType == itemType {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	paged, total := pageSlice(items, page)
	return paged, total, nil
}

func (m *Memory) ListExamsPage(ctx context.Context, branchID string, classID string, page domain.PageRequest) ([]domain.Exam, int, error) {
	items, err := m.ListExams(ctx, branchID, classID)
	if err != nil {
		return nil, 0, err
	}
	paged, total := pageSlice(items, page)
	return paged, total, nil
}

func (m *Memory) ListExamResultsPage(ctx context.Context, branchID string, examID string, studentID string, page domain.PageRequest) ([]domain.ExamResult, int, error) {
	items, err := m.ListExamResults(ctx, branchID, examID, studentID)
	if err != nil {
		return nil, 0, err
	}
	paged, total := pageSlice(items, page)
	return paged, total, nil
}

func (m *Memory) ListTeacherFinanceRecordsPage(ctx context.Context, branchID string, subject string, status domain.TeacherStatus, salaryModel domain.SalaryModelType, page domain.PageRequest) ([]domain.TeacherFinanceRecord, int, error) {
	items, err := m.ListTeacherFinanceRecords(ctx, branchID)
	if err != nil {
		return nil, 0, err
	}
	subject = strings.ToLower(strings.TrimSpace(subject))
	if subject != "" || status != "" || salaryModel != "" {
		filtered := make([]domain.TeacherFinanceRecord, 0, len(items))
		for _, item := range items {
			if subject != "" && !strings.Contains(strings.ToLower(item.Subject), subject) {
				continue
			}
			if status != "" && item.Status != status {
				continue
			}
			if salaryModel != "" && item.SalaryType != salaryModel {
				continue
			}
			filtered = append(filtered, item)
		}
		items = filtered
	}
	paged, total := pageSlice(items, page)
	return paged, total, nil
}

func (m *Memory) ListPaymentsPage(ctx context.Context, branchID string, status domain.PaymentStatus, page domain.PageRequest) ([]domain.Payment, int, error) {
	items, err := m.ListPayments(ctx, branchID)
	if err != nil {
		return nil, 0, err
	}
	if status != "" {
		filtered := make([]domain.Payment, 0, len(items))
		for _, item := range items {
			if item.Status == status {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	paged, total := pageSlice(items, page)
	return paged, total, nil
}

func (m *Memory) ListSalaryModelsPage(ctx context.Context, branchID string, teacherID string, page domain.PageRequest) ([]domain.SalaryModel, int, error) {
	items, err := m.ListSalaryModels(ctx, branchID)
	if err != nil {
		return nil, 0, err
	}
	if teacherID != "" {
		filtered := make([]domain.SalaryModel, 0, len(items))
		for _, item := range items {
			if item.TeacherID == teacherID {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	paged, total := pageSlice(items, page)
	return paged, total, nil
}

func (m *Memory) ListFilesPage(ctx context.Context, branchID string, ownerType string, ownerID string, purpose domain.FilePurpose, page domain.PageRequest) ([]domain.FileObject, int, error) {
	items, err := m.ListFiles(ctx, branchID)
	if err != nil {
		return nil, 0, err
	}
	if ownerType != "" || ownerID != "" || purpose != "" {
		filtered := make([]domain.FileObject, 0, len(items))
		for _, item := range items {
			if ownerType != "" && item.OwnerType != ownerType {
				continue
			}
			if ownerID != "" && item.OwnerID != ownerID {
				continue
			}
			if purpose != "" && item.Purpose != purpose {
				continue
			}
			filtered = append(filtered, item)
		}
		items = filtered
	}
	paged, total := pageSlice(items, page)
	return paged, total, nil
}

func (m *Memory) ListNotificationsPage(ctx context.Context, recipientUserID string, unreadOnly bool, page domain.PageRequest) ([]domain.Notification, int, error) {
	items, err := m.ListNotifications(ctx, recipientUserID, unreadOnly)
	if err != nil {
		return nil, 0, err
	}
	paged, total := pageSlice(items, page)
	return paged, total, nil
}

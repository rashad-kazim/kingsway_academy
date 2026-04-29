package store

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"kingsway/backend/internal/domain"
)

type Memory struct {
	mu sync.RWMutex

	branches       map[string]domain.Branch
	branchesBySlug map[string]string

	users        map[string]domain.User
	usersByEmail map[string]string

	students      map[string]domain.Student
	studentsByFIN map[string]string

	teachers       map[string]domain.Teacher
	teachersByUser map[string]string

	courses          map[string]domain.Course
	courseCategories map[string][]domain.ScoreCategory
	classes          map[string]domain.Class
	classStudents    map[string]domain.ClassStudent
	assignments      map[string]domain.Assignment
	rooms            map[string]domain.Room
	schedules        map[string]domain.ScheduleItem
	exams            map[string]domain.Exam
	examParticipants map[string]domain.ExamParticipant
	examResults      map[string]domain.ExamResult
	payments         map[string]domain.Payment
	salaryModels     map[string]domain.SalaryModel
	files            map[string]domain.FileObject
	notifications    map[string]domain.Notification
}

func NewMemory() *Memory {
	return &Memory{
		branches:         make(map[string]domain.Branch),
		branchesBySlug:   make(map[string]string),
		users:            make(map[string]domain.User),
		usersByEmail:     make(map[string]string),
		students:         make(map[string]domain.Student),
		studentsByFIN:    make(map[string]string),
		teachers:         make(map[string]domain.Teacher),
		teachersByUser:   make(map[string]string),
		courses:          make(map[string]domain.Course),
		courseCategories: make(map[string][]domain.ScoreCategory),
		classes:          make(map[string]domain.Class),
		classStudents:    make(map[string]domain.ClassStudent),
		assignments:      make(map[string]domain.Assignment),
		rooms:            make(map[string]domain.Room),
		schedules:        make(map[string]domain.ScheduleItem),
		exams:            make(map[string]domain.Exam),
		examParticipants: make(map[string]domain.ExamParticipant),
		examResults:      make(map[string]domain.ExamResult),
		payments:         make(map[string]domain.Payment),
		salaryModels:     make(map[string]domain.SalaryModel),
		files:            make(map[string]domain.FileObject),
		notifications:    make(map[string]domain.Notification),
	}
}

func newID() string {
	return uuid.NewString()
}

func now() time.Time {
	return time.Now().UTC()
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func normalizeSlug(slug string) string {
	return strings.ToLower(strings.TrimSpace(slug))
}

func (m *Memory) CountOwners(_ context.Context) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, user := range m.users {
		if user.Role == domain.RoleOwner {
			count++
		}
	}

	return count, nil
}

func (m *Memory) CreateBranch(_ context.Context, branch domain.Branch) (domain.Branch, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	branch.Slug = normalizeSlug(branch.Slug)
	if branch.Name == "" || branch.Slug == "" {
		return domain.Branch{}, domain.ErrInvalidInput
	}
	if _, exists := m.branchesBySlug[branch.Slug]; exists {
		return domain.Branch{}, domain.ErrConflict
	}

	ts := now()
	branch.ID = newID()
	branch.CreatedAt = ts
	branch.UpdatedAt = ts
	m.branches[branch.ID] = branch
	m.branchesBySlug[branch.Slug] = branch.ID

	return branch, nil
}

func (m *Memory) GetBranch(_ context.Context, id string) (domain.Branch, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	branch, ok := m.branches[id]
	if !ok {
		return domain.Branch{}, domain.ErrNotFound
	}

	return branch, nil
}

func (m *Memory) ListBranches(_ context.Context) ([]domain.Branch, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	branches := make([]domain.Branch, 0, len(m.branches))
	for _, branch := range m.branches {
		branches = append(branches, branch)
	}
	sort.Slice(branches, func(i, j int) bool { return branches[i].Name < branches[j].Name })

	return branches, nil
}

func (m *Memory) CreateUser(_ context.Context, user domain.User) (domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	user.Email = normalizeEmail(user.Email)
	if user.Email == "" || user.PasswordHash == "" || !user.Role.IsValid() {
		return domain.User{}, domain.ErrInvalidInput
	}
	if user.Role.RequiresBranchScope() {
		if user.BranchID == "" {
			return domain.User{}, domain.ErrInvalidInput
		}
		if _, ok := m.branches[user.BranchID]; !ok {
			return domain.User{}, domain.ErrNotFound
		}
	}
	if _, exists := m.usersByEmail[user.Email]; exists {
		return domain.User{}, domain.ErrConflict
	}

	ts := now()
	user.ID = newID()
	user.IsActive = true
	user.CreatedAt = ts
	user.UpdatedAt = ts
	m.users[user.ID] = user
	m.usersByEmail[user.Email] = user.ID

	return user, nil
}

func (m *Memory) GetUser(_ context.Context, id string) (domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, ok := m.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}

	return user, nil
}

func (m *Memory) GetUserByEmail(_ context.Context, email string) (domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	id, ok := m.usersByEmail[normalizeEmail(email)]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}

	return m.users[id], nil
}

func (m *Memory) CreateStudent(_ context.Context, student domain.Student) (domain.Student, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.branches[student.BranchID]; !ok {
		return domain.Student{}, domain.ErrNotFound
	}
	if student.FirstName == "" || student.LastName == "" || student.FIN == "" {
		return domain.Student{}, domain.ErrInvalidInput
	}
	if student.Status == "" {
		student.Status = domain.StudentStatusActive
	}
	if student.Status == domain.StudentStatusLeft && student.LeftReason == "" {
		return domain.Student{}, domain.ErrInvalidInput
	}
	if _, exists := m.studentsByFIN[student.FIN.String()]; exists {
		return domain.Student{}, domain.ErrConflict
	}

	ts := now()
	student.ID = newID()
	student.CreatedAt = ts
	student.UpdatedAt = ts
	m.students[student.ID] = student
	m.studentsByFIN[student.FIN.String()] = student.ID

	return student, nil
}

func (m *Memory) GetStudent(_ context.Context, id string) (domain.Student, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	student, ok := m.students[id]
	if !ok {
		return domain.Student{}, domain.ErrNotFound
	}

	return student, nil
}

func (m *Memory) GetStudentByUser(_ context.Context, userID string) (domain.Student, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, student := range m.students {
		if student.UserID == userID {
			return student, nil
		}
	}

	return domain.Student{}, domain.ErrNotFound
}

func (m *Memory) GetStudentByFIN(_ context.Context, fin domain.FIN) (domain.Student, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	id, ok := m.studentsByFIN[fin.String()]
	if !ok {
		return domain.Student{}, domain.ErrNotFound
	}

	return m.students[id], nil
}

func (m *Memory) UpdateStudent(_ context.Context, student domain.Student) (domain.Student, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.students[student.ID]
	if !ok {
		return domain.Student{}, domain.ErrNotFound
	}
	if existing.BranchID != student.BranchID || student.FIN != existing.FIN {
		return domain.Student{}, domain.ErrInvalidInput
	}
	if student.UserID != "" {
		user, ok := m.users[student.UserID]
		if !ok {
			return domain.Student{}, domain.ErrNotFound
		}
		if user.BranchID != student.BranchID || user.Role != domain.RoleStudent {
			return domain.Student{}, domain.ErrInvalidInput
		}
	}
	student.CreatedAt = existing.CreatedAt
	student.UpdatedAt = now()
	m.students[student.ID] = student

	return student, nil
}

func (m *Memory) ListStudents(_ context.Context, branchID string) ([]domain.Student, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	students := make([]domain.Student, 0)
	for _, student := range m.students {
		if branchID == "" || student.BranchID == branchID {
			students = append(students, student)
		}
	}
	sort.Slice(students, func(i, j int) bool {
		if students[i].LastName == students[j].LastName {
			return students[i].FirstName < students[j].FirstName
		}
		return students[i].LastName < students[j].LastName
	})

	return students, nil
}

func (m *Memory) CreateTeacher(_ context.Context, teacher domain.Teacher) (domain.Teacher, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.users[teacher.UserID]
	if !ok {
		return domain.Teacher{}, domain.ErrNotFound
	}
	if user.Role != domain.RoleTeacher || user.BranchID != teacher.BranchID {
		return domain.Teacher{}, domain.ErrInvalidInput
	}
	if _, exists := m.teachersByUser[teacher.UserID]; exists {
		return domain.Teacher{}, domain.ErrConflict
	}
	if teacher.Status == "" {
		teacher.Status = domain.TeacherStatusPending
	}

	ts := now()
	teacher.ID = newID()
	teacher.CreatedAt = ts
	teacher.UpdatedAt = ts
	m.teachers[teacher.ID] = teacher
	m.teachersByUser[teacher.UserID] = teacher.ID

	return teacher, nil
}

func (m *Memory) GetTeacher(_ context.Context, id string) (domain.Teacher, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	teacher, ok := m.teachers[id]
	if !ok {
		return domain.Teacher{}, domain.ErrNotFound
	}

	return teacher, nil
}

func (m *Memory) GetTeacherByUser(_ context.Context, userID string) (domain.Teacher, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	id, ok := m.teachersByUser[userID]
	if !ok {
		return domain.Teacher{}, domain.ErrNotFound
	}

	return m.teachers[id], nil
}

func (m *Memory) UpdateTeacher(_ context.Context, teacher domain.Teacher) (domain.Teacher, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.teachers[teacher.ID]; !ok {
		return domain.Teacher{}, domain.ErrNotFound
	}
	teacher.UpdatedAt = now()
	m.teachers[teacher.ID] = teacher

	return teacher, nil
}

func (m *Memory) ListTeachers(_ context.Context, branchID string) ([]domain.Teacher, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	teachers := make([]domain.Teacher, 0)
	for _, teacher := range m.teachers {
		if branchID == "" || teacher.BranchID == branchID {
			teachers = append(teachers, teacher)
		}
	}

	return teachers, nil
}

func (m *Memory) CreateCourse(_ context.Context, course domain.Course, categories []domain.ScoreCategory) (domain.Course, []domain.ScoreCategory, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.branches[course.BranchID]; !ok {
		return domain.Course{}, nil, domain.ErrNotFound
	}
	if strings.TrimSpace(course.Name) == "" {
		return domain.Course{}, nil, domain.ErrInvalidInput
	}
	for _, existing := range m.courses {
		if existing.BranchID == course.BranchID && strings.EqualFold(existing.Name, course.Name) {
			return domain.Course{}, nil, domain.ErrConflict
		}
	}

	ts := now()
	course.ID = newID()
	course.IsActive = true
	course.CreatedAt = ts
	course.UpdatedAt = ts

	for i := range categories {
		if categories[i].Name == "" || categories[i].MinScore > categories[i].MaxScore {
			return domain.Course{}, nil, domain.ErrInvalidInput
		}
		if categories[i].RequiresDocument && categories[i].DocumentPurpose == "" {
			return domain.Course{}, nil, domain.ErrInvalidInput
		}
		categories[i].ID = newID()
		categories[i].BranchID = course.BranchID
		categories[i].CourseID = course.ID
		categories[i].SortOrder = i
		categories[i].CreatedAt = ts
	}

	m.courses[course.ID] = course
	m.courseCategories[course.ID] = categories

	return course, categories, nil
}

func (m *Memory) GetCourse(_ context.Context, id string) (domain.Course, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	course, ok := m.courses[id]
	if !ok {
		return domain.Course{}, domain.ErrNotFound
	}

	return course, nil
}

func (m *Memory) ListCourses(_ context.Context, branchID string) ([]domain.Course, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	courses := make([]domain.Course, 0)
	for _, course := range m.courses {
		if branchID == "" || course.BranchID == branchID {
			courses = append(courses, course)
		}
	}
	sort.Slice(courses, func(i, j int) bool { return courses[i].Name < courses[j].Name })

	return courses, nil
}

func (m *Memory) CreateRoom(_ context.Context, room domain.Room) (domain.Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.branches[room.BranchID]; !ok {
		return domain.Room{}, domain.ErrNotFound
	}
	if room.Name == "" {
		return domain.Room{}, domain.ErrInvalidInput
	}
	if room.Capacity <= 0 {
		room.Capacity = 1
	}
	for _, existing := range m.rooms {
		if existing.BranchID == room.BranchID && strings.EqualFold(existing.Name, room.Name) {
			return domain.Room{}, domain.ErrConflict
		}
	}

	ts := now()
	room.ID = newID()
	room.IsActive = true
	room.CreatedAt = ts
	room.UpdatedAt = ts
	m.rooms[room.ID] = room

	return room, nil
}

func (m *Memory) ListRooms(_ context.Context, branchID string) ([]domain.Room, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rooms := make([]domain.Room, 0)
	for _, room := range m.rooms {
		if branchID == "" || room.BranchID == branchID {
			rooms = append(rooms, room)
		}
	}
	sort.Slice(rooms, func(i, j int) bool { return rooms[i].Name < rooms[j].Name })

	return rooms, nil
}

func (m *Memory) CreateScheduleItem(_ context.Context, item domain.ScheduleItem) (domain.ScheduleItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if item.Title == "" || !item.Range().IsValid() {
		return domain.ScheduleItem{}, domain.ErrInvalidInput
	}
	room, roomOK := m.rooms[item.RoomID]
	teacher, teacherOK := m.teachers[item.TeacherID]
	if !roomOK || !teacherOK {
		return domain.ScheduleItem{}, domain.ErrNotFound
	}
	if room.BranchID != item.BranchID || teacher.BranchID != item.BranchID {
		return domain.ScheduleItem{}, domain.ErrInvalidInput
	}
	if teacher.Status != domain.TeacherStatusActive {
		return domain.ScheduleItem{}, domain.ErrInvalidInput
	}

	for _, existing := range m.schedules {
		if existing.BranchID != item.BranchID || existing.CancelledAt != nil {
			continue
		}
		if !item.Range().Overlaps(existing.Range()) {
			continue
		}
		if existing.RoomID == item.RoomID || existing.TeacherID == item.TeacherID {
			return domain.ScheduleItem{}, domain.ErrConflict
		}
	}

	ts := now()
	item.ID = newID()
	item.CreatedAt = ts
	item.UpdatedAt = ts
	m.schedules[item.ID] = item

	return item, nil
}

func (m *Memory) ListSchedule(_ context.Context, branchID string) ([]domain.ScheduleItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	items := make([]domain.ScheduleItem, 0)
	for _, item := range m.schedules {
		if branchID == "" || item.BranchID == branchID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].StartsAt.Before(items[j].StartsAt) })

	return items, nil
}

func (m *Memory) CreatePayment(_ context.Context, payment domain.Payment) (domain.Payment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	student, ok := m.students[payment.StudentID]
	if !ok {
		return domain.Payment{}, domain.ErrNotFound
	}
	if student.BranchID != payment.BranchID || payment.AmountCents <= 0 {
		return domain.Payment{}, domain.ErrInvalidInput
	}
	if payment.Currency == "" {
		payment.Currency = "AZN"
	}
	if payment.Status == "" {
		payment.Status = domain.PaymentStatusPending
	}

	ts := now()
	payment.ID = newID()
	payment.CreatedAt = ts
	payment.UpdatedAt = ts
	m.payments[payment.ID] = payment

	return payment, nil
}

func (m *Memory) ListPayments(_ context.Context, branchID string) ([]domain.Payment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	payments := make([]domain.Payment, 0)
	for _, payment := range m.payments {
		if branchID == "" || payment.BranchID == branchID {
			payments = append(payments, payment)
		}
	}
	sort.Slice(payments, func(i, j int) bool { return payments[i].DueDate.Before(payments[j].DueDate) })

	return payments, nil
}

func (m *Memory) GetPayment(_ context.Context, id string) (domain.Payment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	payment, ok := m.payments[id]
	if !ok {
		return domain.Payment{}, domain.ErrNotFound
	}

	return payment, nil
}

func (m *Memory) CreateSalaryModel(_ context.Context, model domain.SalaryModel) (domain.SalaryModel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	teacher, ok := m.teachers[model.TeacherID]
	if !ok {
		return domain.SalaryModel{}, domain.ErrNotFound
	}
	if teacher.BranchID != model.BranchID || !model.ModelType.IsValid() || model.ApprovedByOwnerUserID == "" {
		return domain.SalaryModel{}, domain.ErrInvalidInput
	}

	model.ID = newID()
	model.CreatedAt = now()
	m.salaryModels[model.ID] = model

	return model, nil
}

func (m *Memory) ListSalaryModels(_ context.Context, branchID string) ([]domain.SalaryModel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	models := make([]domain.SalaryModel, 0)
	for _, model := range m.salaryModels {
		if branchID == "" || model.BranchID == branchID {
			models = append(models, model)
		}
	}

	return models, nil
}

func (m *Memory) RegisterFile(_ context.Context, file domain.FileObject) (domain.FileObject, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.branches[file.BranchID]; !ok {
		return domain.FileObject{}, domain.ErrNotFound
	}
	if !file.Category.IsValid() || file.OriginalFilename == "" || file.StorageBucket == "" || file.StorageKey == "" {
		return domain.FileObject{}, domain.ErrInvalidInput
	}
	if file.Category.MustPreserveOriginalBytes() && file.StoredSizeBytes != file.OriginalSizeBytes {
		return domain.FileObject{}, domain.ErrInvalidInput
	}
	for _, existing := range m.files {
		if existing.StorageKey == file.StorageKey {
			return domain.FileObject{}, domain.ErrConflict
		}
	}

	file.ID = newID()
	file.CreatedAt = now()
	if file.RetentionUntil == nil {
		file.RetentionUntil = domain.RetentionUntil(file.Purpose, file.CreatedAt)
	}
	m.files[file.ID] = file

	return file, nil
}

func (m *Memory) GetFile(_ context.Context, id string) (domain.FileObject, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	file, ok := m.files[id]
	if !ok {
		return domain.FileObject{}, domain.ErrNotFound
	}

	return file, nil
}

func (m *Memory) ListFiles(_ context.Context, branchID string) ([]domain.FileObject, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	files := make([]domain.FileObject, 0)
	for _, file := range m.files {
		if branchID == "" || file.BranchID == branchID {
			files = append(files, file)
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].CreatedAt.After(files[j].CreatedAt) })

	return files, nil
}

func (m *Memory) ListExpiredFiles(_ context.Context, now time.Time, limit int) ([]domain.FileObject, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	files := make([]domain.FileObject, 0)
	for _, file := range m.files {
		if file.DeletedAt != nil || file.RetentionUntil == nil || file.RetentionUntil.After(now) {
			continue
		}
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].RetentionUntil.Before(*files[j].RetentionUntil)
	})
	if limit > 0 && len(files) > limit {
		files = files[:limit]
	}

	return files, nil
}

func (m *Memory) MarkFileDeleted(_ context.Context, id string, deletedAt time.Time) (domain.FileObject, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	file, ok := m.files[id]
	if !ok {
		return domain.FileObject{}, domain.ErrNotFound
	}
	file.DeletedAt = &deletedAt
	m.files[id] = file

	return file, nil
}

func (m *Memory) ListOutboxEvents(_ context.Context, _ domain.OutboxEventStatus, _ int, _ int) ([]domain.OutboxEvent, int, error) {
	return []domain.OutboxEvent{}, 0, nil
}

func (m *Memory) RetryOutboxEvent(_ context.Context, _ string) (domain.OutboxEvent, error) {
	return domain.OutboxEvent{}, domain.ErrNotFound
}

func IsNotFound(err error) bool {
	return errors.Is(err, domain.ErrNotFound)
}

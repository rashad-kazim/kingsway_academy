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
	staffMembers map[string]domain.StaffMember

	students                  map[string]domain.Student
	studentsByFIN             map[string]string
	studentParents            map[string]domain.StudentParentContact
	studentCourses            map[string]domain.StudentCourseRegistration
	studentTeacherAssignments map[string]domain.StudentTeacherAssignment

	teachers       map[string]domain.Teacher
	teachersByUser map[string]string
	teacherCourses map[string]map[string]struct{}

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
	idempotency      map[string]domain.IdempotencyRecord
}

func NewMemory() *Memory {
	return &Memory{
		branches:                  make(map[string]domain.Branch),
		branchesBySlug:            make(map[string]string),
		users:                     make(map[string]domain.User),
		usersByEmail:              make(map[string]string),
		staffMembers:              make(map[string]domain.StaffMember),
		students:                  make(map[string]domain.Student),
		studentsByFIN:             make(map[string]string),
		studentParents:            make(map[string]domain.StudentParentContact),
		studentCourses:            make(map[string]domain.StudentCourseRegistration),
		studentTeacherAssignments: make(map[string]domain.StudentTeacherAssignment),
		teachers:                  make(map[string]domain.Teacher),
		teachersByUser:            make(map[string]string),
		teacherCourses:            make(map[string]map[string]struct{}),
		courses:                   make(map[string]domain.Course),
		courseCategories:          make(map[string][]domain.ScoreCategory),
		classes:                   make(map[string]domain.Class),
		classStudents:             make(map[string]domain.ClassStudent),
		assignments:               make(map[string]domain.Assignment),
		rooms:                     make(map[string]domain.Room),
		schedules:                 make(map[string]domain.ScheduleItem),
		exams:                     make(map[string]domain.Exam),
		examParticipants:          make(map[string]domain.ExamParticipant),
		examResults:               make(map[string]domain.ExamResult),
		payments:                  make(map[string]domain.Payment),
		salaryModels:              make(map[string]domain.SalaryModel),
		files:                     make(map[string]domain.FileObject),
		notifications:             make(map[string]domain.Notification),
		idempotency:               make(map[string]domain.IdempotencyRecord),
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

func idempotencyMapKey(actorUserID, method, path, key string) string {
	return actorUserID + "\x00" + method + "\x00" + path + "\x00" + key
}

func (m *Memory) BeginIdempotency(_ context.Context, record domain.IdempotencyRecord) (domain.IdempotencyBeginResult, error) {
	if record.ActorUserID == "" || record.Method == "" || record.Path == "" || record.Key == "" || record.RequestHash == "" {
		return domain.IdempotencyBeginResult{}, domain.ErrInvalidInput
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := idempotencyMapKey(record.ActorUserID, record.Method, record.Path, record.Key)
	if existing, ok := m.idempotency[key]; ok {
		if !existing.ExpiresAt.IsZero() && existing.ExpiresAt.Before(now()) {
			delete(m.idempotency, key)
		} else {
			if existing.RequestHash != record.RequestHash {
				return domain.IdempotencyBeginResult{}, domain.ErrConflict
			}
			return domain.IdempotencyBeginResult{Record: existing}, nil
		}
	}

	ts := now()
	record.Status = domain.IdempotencyStatusPending
	record.CreatedAt = ts
	record.UpdatedAt = ts
	if record.ExpiresAt.IsZero() {
		record.ExpiresAt = ts.Add(24 * time.Hour)
	}
	m.idempotency[key] = record

	return domain.IdempotencyBeginResult{Started: true, Record: record}, nil
}

func (m *Memory) CompleteIdempotency(_ context.Context, actorUserID string, method string, path string, key string, responseStatus int, responseBody []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mapKey := idempotencyMapKey(actorUserID, method, path, key)
	record, ok := m.idempotency[mapKey]
	if !ok {
		return domain.ErrNotFound
	}
	record.Status = domain.IdempotencyStatusCompleted
	record.ResponseStatus = responseStatus
	record.ResponseBody = append([]byte(nil), responseBody...)
	record.UpdatedAt = now()
	m.idempotency[mapKey] = record

	return nil
}

func (m *Memory) ClearIdempotency(_ context.Context, actorUserID string, method string, path string, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.idempotency, idempotencyMapKey(actorUserID, method, path, key))
	return nil
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

func (m *Memory) UpdateBranch(_ context.Context, branch domain.Branch) (domain.Branch, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	current, ok := m.branches[branch.ID]
	if !ok {
		return domain.Branch{}, domain.ErrNotFound
	}
	branch.Slug = normalizeSlug(branch.Slug)
	if strings.TrimSpace(branch.Name) == "" || branch.Slug == "" {
		return domain.Branch{}, domain.ErrInvalidInput
	}
	if existingID, exists := m.branchesBySlug[branch.Slug]; exists && existingID != branch.ID {
		return domain.Branch{}, domain.ErrConflict
	}

	if current.Slug != branch.Slug {
		delete(m.branchesBySlug, current.Slug)
		m.branchesBySlug[branch.Slug] = branch.ID
	}
	branch.CreatedAt = current.CreatedAt
	branch.UpdatedAt = now()
	m.branches[branch.ID] = branch

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

func (m *Memory) DeleteBranch(_ context.Context, id string) (domain.Branch, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	branch, ok := m.branches[id]
	if !ok {
		return domain.Branch{}, domain.ErrNotFound
	}

	delete(m.branches, id)
	delete(m.branchesBySlug, branch.Slug)

	deletedUsers := make(map[string]struct{})
	for userID, user := range m.users {
		if user.BranchID == id {
			deletedUsers[userID] = struct{}{}
			delete(m.users, userID)
			delete(m.usersByEmail, user.Email)
		}
	}
	for staffID, staff := range m.staffMembers {
		if staff.BranchID == id {
			delete(m.staffMembers, staffID)
		}
	}
	for studentID, student := range m.students {
		if student.BranchID == id {
			delete(m.students, studentID)
			delete(m.studentsByFIN, student.FIN.String())
		}
	}
	for teacherID, teacher := range m.teachers {
		if teacher.BranchID == id {
			delete(m.teachers, teacherID)
			delete(m.teachersByUser, teacher.UserID)
			delete(m.teacherCourses, teacherID)
		}
	}
	for courseID, course := range m.courses {
		if course.BranchID == id {
			delete(m.courses, courseID)
			delete(m.courseCategories, courseID)
		}
	}
	for classID, class := range m.classes {
		if class.BranchID == id {
			delete(m.classes, classID)
		}
	}
	for enrollmentID, enrollment := range m.classStudents {
		if enrollment.BranchID == id {
			delete(m.classStudents, enrollmentID)
		}
	}
	for assignmentID, assignment := range m.assignments {
		if assignment.BranchID == id {
			delete(m.assignments, assignmentID)
		}
	}
	for roomID, room := range m.rooms {
		if room.BranchID == id {
			delete(m.rooms, roomID)
		}
	}
	for scheduleID, item := range m.schedules {
		if item.BranchID == id {
			delete(m.schedules, scheduleID)
		}
	}
	for examID, exam := range m.exams {
		if exam.BranchID == id {
			delete(m.exams, examID)
		}
	}
	for participantID, participant := range m.examParticipants {
		if participant.BranchID == id {
			delete(m.examParticipants, participantID)
		}
	}
	for resultID, result := range m.examResults {
		if result.BranchID == id {
			delete(m.examResults, resultID)
		}
	}
	for paymentID, payment := range m.payments {
		if payment.BranchID == id {
			delete(m.payments, paymentID)
		}
	}
	for modelID, model := range m.salaryModels {
		if model.BranchID == id {
			delete(m.salaryModels, modelID)
		}
	}
	for fileID, file := range m.files {
		if file.BranchID == id {
			delete(m.files, fileID)
		}
	}
	for notificationID, notification := range m.notifications {
		if notification.BranchID == id {
			delete(m.notifications, notificationID)
			continue
		}
		if _, ok := deletedUsers[notification.RecipientUserID]; ok {
			delete(m.notifications, notificationID)
		}
	}

	return branch, nil
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

func (m *Memory) RecordUserLogin(_ context.Context, id string) (domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	user.LastLoginAt = now().Format(time.RFC3339)
	user.UpdatedAt = now()
	m.users[id] = user

	return user, nil
}

func (m *Memory) CreateStaffMember(_ context.Context, user domain.User, staff domain.StaffMember) (domain.StaffMember, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	user.Email = normalizeEmail(user.Email)
	if user.BranchID == "" || user.Email == "" || user.PasswordHash == "" || user.FirstName == "" || user.LastName == "" || user.Role != domain.RoleReceptionist {
		return domain.StaffMember{}, domain.ErrInvalidInput
	}
	if _, ok := m.branches[user.BranchID]; !ok {
		return domain.StaffMember{}, domain.ErrNotFound
	}
	if _, exists := m.usersByEmail[user.Email]; exists {
		return domain.StaffMember{}, domain.ErrConflict
	}

	ts := now()
	user.ID = newID()
	user.CreatedAt = ts
	user.UpdatedAt = ts
	user.IsActive = true
	m.users[user.ID] = user
	m.usersByEmail[user.Email] = user.ID

	staff.ID = newID()
	staff.BranchID = user.BranchID
	staff.UserID = user.ID
	staff.Role = domain.RoleReceptionist
	staff.Email = user.Email
	staff.FirstName = user.FirstName
	staff.LastName = user.LastName
	staff.IsActive = user.IsActive
	staff.LastLoginAt = user.LastLoginAt
	staff.CreatedAt = ts
	staff.UpdatedAt = ts
	m.staffMembers[staff.ID] = staff

	return staff, nil
}

func (m *Memory) UpdateStaffMember(_ context.Context, staff domain.StaffMember, passwordHash string) (domain.StaffMember, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	current, ok := m.staffMembers[staff.ID]
	if !ok {
		return domain.StaffMember{}, domain.ErrNotFound
	}
	user, ok := m.users[current.UserID]
	if !ok {
		return domain.StaffMember{}, domain.ErrNotFound
	}
	email := normalizeEmail(staff.Email)
	if existingID, exists := m.usersByEmail[email]; exists && existingID != user.ID {
		return domain.StaffMember{}, domain.ErrConflict
	}
	if _, ok := m.branches[staff.BranchID]; !ok {
		return domain.StaffMember{}, domain.ErrNotFound
	}

	delete(m.usersByEmail, user.Email)
	user.BranchID = staff.BranchID
	user.Email = email
	user.FirstName = strings.TrimSpace(staff.FirstName)
	user.LastName = strings.TrimSpace(staff.LastName)
	if strings.TrimSpace(passwordHash) != "" {
		user.PasswordHash = strings.TrimSpace(passwordHash)
	}
	user.IsActive = staff.IsActive
	user.UpdatedAt = now()
	m.users[user.ID] = user
	m.usersByEmail[user.Email] = user.ID

	staff.UserID = current.UserID
	staff.Role = domain.RoleReceptionist
	staff.Email = user.Email
	staff.FirstName = user.FirstName
	staff.LastName = user.LastName
	staff.IsActive = user.IsActive
	staff.LastLoginAt = user.LastLoginAt
	staff.CreatedAt = current.CreatedAt
	staff.UpdatedAt = now()
	m.staffMembers[staff.ID] = staff

	return staff, nil
}

func (m *Memory) DeleteStaffMember(_ context.Context, id string) (domain.StaffMember, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	staff, ok := m.staffMembers[id]
	if !ok {
		return domain.StaffMember{}, domain.ErrNotFound
	}
	delete(m.staffMembers, id)
	if user, ok := m.users[staff.UserID]; ok {
		delete(m.usersByEmail, user.Email)
		delete(m.users, staff.UserID)
	}

	return staff, nil
}

func (m *Memory) GetStaffMember(_ context.Context, id string) (domain.StaffMember, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	staff, ok := m.staffMembers[id]
	if !ok {
		return domain.StaffMember{}, domain.ErrNotFound
	}

	return staff, nil
}

func (m *Memory) ListStaffMembers(_ context.Context, branchID string) ([]domain.StaffMember, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	staff := make([]domain.StaffMember, 0)
	for _, member := range m.staffMembers {
		if member.BranchID == branchID {
			staff = append(staff, member)
		}
	}
	sort.Slice(staff, func(i, j int) bool {
		if staff[i].LastName == staff[j].LastName {
			return staff[i].FirstName < staff[j].FirstName
		}
		return staff[i].LastName < staff[j].LastName
	})

	return staff, nil
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

func (m *Memory) CreateStudentWithDetails(_ context.Context, student domain.Student, contacts []domain.StudentParentContact, registrations []domain.StudentCourseRegistration) (domain.Student, error) {
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
	for _, contact := range contacts {
		if contact.BranchID != student.BranchID || contact.Relation == "" || contact.Name == "" || len(contact.Phones) == 0 {
			return domain.Student{}, domain.ErrInvalidInput
		}
	}
	for _, registration := range registrations {
		course, ok := m.courses[registration.CourseID]
		if !ok {
			return domain.Student{}, domain.ErrNotFound
		}
		if registration.BranchID != student.BranchID || course.BranchID != student.BranchID || registration.MonthlyAmountCents < 0 || registration.StartDate == "" {
			return domain.Student{}, domain.ErrInvalidInput
		}
		if registration.TeacherID != "" {
			teacher, ok := m.teachers[registration.TeacherID]
			if !ok {
				return domain.Student{}, domain.ErrNotFound
			}
			if teacher.BranchID != student.BranchID || teacher.Status != domain.TeacherStatusActive {
				return domain.Student{}, domain.ErrInvalidInput
			}
		}
	}

	ts := now()
	student.ID = newID()
	student.CreatedAt = ts
	student.UpdatedAt = ts
	m.students[student.ID] = student
	m.studentsByFIN[student.FIN.String()] = student.ID

	for _, contact := range contacts {
		contact.ID = newID()
		contact.StudentID = student.ID
		contact.CreatedAt = ts
		m.studentParents[contact.ID] = contact
	}
	for _, registration := range registrations {
		registration.ID = newID()
		registration.StudentID = student.ID
		registration.CreatedAt = ts
		m.studentCourses[registration.ID] = registration
		if registration.TeacherID != "" {
			m.assignStudentTeacher(student, registration, ts)
		}
	}

	return student, nil
}

func (m *Memory) CreateStudentParentContacts(_ context.Context, studentID string, contacts []domain.StudentParentContact) ([]domain.StudentParentContact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	student, ok := m.students[studentID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	created := make([]domain.StudentParentContact, 0, len(contacts))
	for _, contact := range contacts {
		if contact.BranchID != student.BranchID || contact.Relation == "" || contact.Name == "" || len(contact.Phones) == 0 {
			return nil, domain.ErrInvalidInput
		}
		contact.ID = newID()
		contact.StudentID = student.ID
		contact.CreatedAt = now()
		m.studentParents[contact.ID] = contact
		created = append(created, contact)
	}

	return created, nil
}

func (m *Memory) CreateStudentCourseRegistrations(_ context.Context, studentID string, registrations []domain.StudentCourseRegistration) ([]domain.StudentCourseRegistration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	student, ok := m.students[studentID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	created := make([]domain.StudentCourseRegistration, 0, len(registrations))
	ts := now()
	for _, registration := range registrations {
		course, ok := m.courses[registration.CourseID]
		if !ok {
			return nil, domain.ErrNotFound
		}
		if registration.BranchID != student.BranchID || course.BranchID != student.BranchID || registration.MonthlyAmountCents < 0 || registration.StartDate == "" {
			return nil, domain.ErrInvalidInput
		}
		if registration.TeacherID != "" {
			teacher, ok := m.teachers[registration.TeacherID]
			if !ok {
				return nil, domain.ErrNotFound
			}
			if teacher.BranchID != student.BranchID || teacher.Status != domain.TeacherStatusActive {
				return nil, domain.ErrInvalidInput
			}
		}
		registration.ID = newID()
		registration.StudentID = student.ID
		registration.CreatedAt = ts
		m.studentCourses[registration.ID] = registration
		created = append(created, registration)
		if registration.TeacherID != "" {
			m.assignStudentTeacher(student, registration, ts)
		}
	}

	return created, nil
}

func (m *Memory) assignStudentTeacher(student domain.Student, registration domain.StudentCourseRegistration, ts time.Time) {
	for id, assignment := range m.studentTeacherAssignments {
		if assignment.StudentID == student.ID && assignment.BranchID == student.BranchID && assignment.ValidTo == "" {
			assignment.ValidTo = registration.StartDate
			m.studentTeacherAssignments[id] = assignment
		}
	}
	assignment := domain.StudentTeacherAssignment{
		ID:        newID(),
		BranchID:  student.BranchID,
		StudentID: student.ID,
		TeacherID: registration.TeacherID,
		ValidFrom: registration.StartDate,
		Reason:    "initial",
		CreatedAt: ts,
	}
	m.studentTeacherAssignments[assignment.ID] = assignment
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

func (m *Memory) ListStudentAssignmentHub(_ context.Context, filter domain.StudentAssignmentHubFilter) (domain.StudentAssignmentHubPage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	records := make([]domain.StudentAssignmentHubRecord, 0)
	query := strings.ToLower(strings.TrimSpace(filter.Query))
	for _, student := range m.students {
		if filter.BranchID != "" && student.BranchID != filter.BranchID {
			continue
		}
		if filter.Status != "" && student.Status != filter.Status {
			continue
		}
		branch, ok := m.branches[student.BranchID]
		if !ok {
			continue
		}
		teacherID, teacherFirstName, teacherLastName := m.activeTeacherForStudent(student)
		if filter.TeacherID != "" && teacherID != filter.TeacherID {
			continue
		}
		if query != "" && !studentHubRecordMatches(student, branch.Name, teacherFirstName, teacherLastName, query) {
			continue
		}

		records = append(records, domain.StudentAssignmentHubRecord{
			ID:                     student.ID,
			BranchID:               student.BranchID,
			BranchName:             branch.Name,
			UserID:                 student.UserID,
			FIN:                    student.FIN,
			FirstName:              student.FirstName,
			LastName:               student.LastName,
			ProfilePhotoFileID:     student.ProfilePhotoFileID,
			Status:                 student.Status,
			ActiveTeacherID:        teacherID,
			ActiveTeacherFirstName: teacherFirstName,
			ActiveTeacherLastName:  teacherLastName,
			RegisteredAt:           student.CreatedAt,
			CreatedAt:              student.CreatedAt,
			UpdatedAt:              student.UpdatedAt,
		})
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].CreatedAt.Equal(records[j].CreatedAt) {
			if records[i].LastName == records[j].LastName {
				return records[i].FirstName < records[j].FirstName
			}
			return records[i].LastName < records[j].LastName
		}
		return records[i].CreatedAt.After(records[j].CreatedAt)
	})

	total := len(records)
	start := filter.Offset
	if start > total {
		start = total
	}
	end := start + filter.Limit
	if end > total {
		end = total
	}

	return domain.StudentAssignmentHubPage{
		Items:  records[start:end],
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}, nil
}

func (m *Memory) activeTeacherForStudent(student domain.Student) (string, string, string) {
	var latestDirect domain.StudentTeacherAssignment
	var latestDirectTeacher domain.Teacher
	directFound := false
	for _, assignment := range m.studentTeacherAssignments {
		if assignment.StudentID != student.ID || assignment.BranchID != student.BranchID || assignment.ValidTo != "" {
			continue
		}
		teacher, ok := m.teachers[assignment.TeacherID]
		if !ok {
			continue
		}
		if !directFound || assignment.CreatedAt.After(latestDirect.CreatedAt) {
			latestDirect = assignment
			latestDirectTeacher = teacher
			directFound = true
		}
	}
	if directFound {
		user := m.users[latestDirectTeacher.UserID]
		return latestDirect.TeacherID, user.FirstName, user.LastName
	}

	var latest domain.ClassStudent
	var latestTeacher domain.Teacher
	found := false
	for _, enrollment := range m.classStudents {
		if enrollment.StudentID != student.ID || enrollment.BranchID != student.BranchID || enrollment.LeftAt != nil {
			continue
		}
		class, ok := m.classes[enrollment.ClassID]
		if !ok || !class.IsActive {
			continue
		}
		teacher, ok := m.teachers[class.TeacherID]
		if !ok {
			continue
		}
		if !found || enrollment.JoinedAt.After(latest.JoinedAt) || (enrollment.JoinedAt.Equal(latest.JoinedAt) && enrollment.CreatedAt.After(latest.CreatedAt)) {
			latest = enrollment
			latestTeacher = teacher
			found = true
		}
	}
	if !found {
		return "", "", ""
	}
	user, ok := m.users[latestTeacher.UserID]
	if !ok {
		return latestTeacher.ID, "", ""
	}
	return latestTeacher.ID, user.FirstName, user.LastName
}

func studentHubRecordMatches(student domain.Student, branchName string, teacherFirstName string, teacherLastName string, query string) bool {
	values := []string{
		student.FirstName,
		student.LastName,
		student.FirstName + " " + student.LastName,
		student.FIN.String(),
		branchName,
		teacherFirstName,
		teacherLastName,
		teacherFirstName + " " + teacherLastName,
	}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
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

func (m *Memory) UpdateTeacherAccount(_ context.Context, teacher domain.Teacher, user domain.User, passwordHash string) (domain.Teacher, domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	current, ok := m.teachers[teacher.ID]
	if !ok {
		return domain.Teacher{}, domain.User{}, domain.ErrNotFound
	}
	currentUser, ok := m.users[current.UserID]
	if !ok {
		return domain.Teacher{}, domain.User{}, domain.ErrNotFound
	}
	email := normalizeEmail(user.Email)
	if email == "" || strings.TrimSpace(user.FirstName) == "" || strings.TrimSpace(user.LastName) == "" {
		return domain.Teacher{}, domain.User{}, domain.ErrInvalidInput
	}
	if existingID, exists := m.usersByEmail[email]; exists && existingID != currentUser.ID {
		return domain.Teacher{}, domain.User{}, domain.ErrConflict
	}
	if _, exists := m.branches[teacher.BranchID]; !exists {
		return domain.Teacher{}, domain.User{}, domain.ErrNotFound
	}

	delete(m.usersByEmail, currentUser.Email)
	currentUser.BranchID = teacher.BranchID
	currentUser.Email = email
	currentUser.FirstName = strings.TrimSpace(user.FirstName)
	currentUser.LastName = strings.TrimSpace(user.LastName)
	if strings.TrimSpace(passwordHash) != "" {
		currentUser.PasswordHash = strings.TrimSpace(passwordHash)
	}
	currentUser.UpdatedAt = now()
	m.users[currentUser.ID] = currentUser
	m.usersByEmail[currentUser.Email] = currentUser.ID

	teacher.UserID = current.UserID
	teacher.CreatedAt = current.CreatedAt
	teacher.UpdatedAt = now()
	m.teachers[teacher.ID] = teacher

	return teacher, currentUser, nil
}

func (m *Memory) DeleteTeacher(_ context.Context, id string) (domain.Teacher, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	teacher, ok := m.teachers[id]
	if !ok {
		return domain.Teacher{}, domain.ErrNotFound
	}

	classIDs := make(map[string]struct{})
	for classID, class := range m.classes {
		if class.TeacherID == teacher.ID {
			classIDs[classID] = struct{}{}
		}
	}

	scheduleIDs := make(map[string]struct{})
	for scheduleID, item := range m.schedules {
		_, classBelongsToTeacher := classIDs[item.ClassID]
		if item.TeacherID == teacher.ID || item.CreatedByUserID == teacher.UserID || classBelongsToTeacher {
			scheduleIDs[scheduleID] = struct{}{}
		}
	}

	examIDs := make(map[string]struct{})
	for examID, exam := range m.exams {
		_, classBelongsToTeacher := classIDs[exam.ClassID]
		_, scheduleBelongsToTeacher := scheduleIDs[exam.ScheduleItemID]
		if exam.CreatedByUserID == teacher.UserID || classBelongsToTeacher || scheduleBelongsToTeacher {
			examIDs[examID] = struct{}{}
		}
	}

	for resultID, result := range m.examResults {
		if result.EnteredByTeacherID == teacher.ID {
			delete(m.examResults, resultID)
			continue
		}
		if _, ok := examIDs[result.ExamID]; ok {
			delete(m.examResults, resultID)
		}
	}
	for participantID, participant := range m.examParticipants {
		if _, ok := examIDs[participant.ExamID]; ok {
			delete(m.examParticipants, participantID)
		}
	}
	for examID := range examIDs {
		delete(m.exams, examID)
	}
	for assignmentID, assignment := range m.assignments {
		if assignment.CreatedByUserID == teacher.UserID {
			delete(m.assignments, assignmentID)
			continue
		}
		if _, ok := classIDs[assignment.ClassID]; ok {
			delete(m.assignments, assignmentID)
		}
	}
	for paymentID, payment := range m.payments {
		if payment.CreatedByUserID == teacher.UserID {
			delete(m.payments, paymentID)
		}
	}
	for scheduleID := range scheduleIDs {
		delete(m.schedules, scheduleID)
	}
	for enrollmentID, enrollment := range m.classStudents {
		if _, ok := classIDs[enrollment.ClassID]; ok {
			delete(m.classStudents, enrollmentID)
		}
	}
	for classID := range classIDs {
		delete(m.classes, classID)
	}
	for modelID, model := range m.salaryModels {
		if model.TeacherID == teacher.ID {
			delete(m.salaryModels, modelID)
		}
	}
	for fileID, file := range m.files {
		if file.UploaderUserID == teacher.UserID || (file.OwnerType == "teacher" && file.OwnerID == teacher.ID) {
			delete(m.files, fileID)
		}
	}
	for notificationID, notification := range m.notifications {
		if notification.RecipientUserID == teacher.UserID {
			delete(m.notifications, notificationID)
		}
	}

	delete(m.teachers, teacher.ID)
	delete(m.teachersByUser, teacher.UserID)
	delete(m.teacherCourses, teacher.ID)
	for studentID, student := range m.students {
		if student.UserID == teacher.UserID {
			student.UserID = ""
			student.UpdatedAt = now()
			m.students[studentID] = student
		}
	}
	if user, ok := m.users[teacher.UserID]; ok {
		delete(m.usersByEmail, user.Email)
	}
	delete(m.users, teacher.UserID)

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

func (m *Memory) ReplaceTeacherCourseSpecializations(_ context.Context, teacherID string, courseIDs []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	teacher, ok := m.teachers[teacherID]
	if !ok {
		return domain.ErrNotFound
	}
	next := make(map[string]struct{}, len(courseIDs))
	for _, courseID := range courseIDs {
		courseID = strings.TrimSpace(courseID)
		if courseID == "" {
			continue
		}
		course, ok := m.courses[courseID]
		if !ok {
			return domain.ErrNotFound
		}
		if course.BranchID != teacher.BranchID {
			return domain.ErrInvalidInput
		}
		next[courseID] = struct{}{}
	}
	m.teacherCourses[teacherID] = next

	return nil
}

func (m *Memory) ListTeacherFinanceRecords(_ context.Context, branchID string) ([]domain.TeacherFinanceRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	records := make([]domain.TeacherFinanceRecord, 0)
	for _, teacher := range m.teachers {
		if branchID != "" && teacher.BranchID != branchID {
			continue
		}
		user, userOK := m.users[teacher.UserID]
		branch, branchOK := m.branches[teacher.BranchID]
		if !userOK || !branchOK {
			continue
		}
		salaryType, calculated := m.latestTeacherSalary(teacher.ID)
		records = append(records, domain.TeacherFinanceRecord{
			ID:                          teacher.ID,
			BranchID:                    teacher.BranchID,
			BranchName:                  branch.Name,
			UserID:                      teacher.UserID,
			Email:                       user.Email,
			FirstName:                   user.FirstName,
			LastName:                    user.LastName,
			ProfilePhotoFileID:          teacher.ProfilePhotoFileID,
			Subject:                     m.subjectsForTeacher(teacher.ID),
			Status:                      teacher.Status,
			SalaryType:                  salaryType,
			AssignedStudents:            m.assignedStudentsForTeacher(teacher.ID),
			CalculatedSalaryAmountCents: calculated,
			CreatedAt:                   teacher.CreatedAt,
			UpdatedAt:                   teacher.UpdatedAt,
		})
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].LastName == records[j].LastName {
			return records[i].FirstName < records[j].FirstName
		}
		return records[i].LastName < records[j].LastName
	})

	return records, nil
}

func (m *Memory) latestTeacherSalary(teacherID string) (domain.SalaryModelType, int64) {
	var latest domain.SalaryModel
	found := false
	for _, model := range m.salaryModels {
		if model.TeacherID != teacherID {
			continue
		}
		if !found || model.CreatedAt.After(latest.CreatedAt) {
			latest = model
			found = true
		}
	}
	if !found {
		return "", 0
	}
	if latest.ModelType == domain.SalaryModelFixed || latest.ModelType == domain.SalaryModelHybrid {
		return latest.ModelType, latest.FixedMonthlyAmountCents
	}

	return latest.ModelType, 0
}

func (m *Memory) subjectsForTeacher(teacherID string) string {
	courseIDs := m.teacherCourses[teacherID]
	if len(courseIDs) == 0 {
		return ""
	}
	names := make([]string, 0, len(courseIDs))
	for courseID := range courseIDs {
		if course, ok := m.courses[courseID]; ok {
			names = append(names, course.Name)
		}
	}
	sort.Strings(names)

	return strings.Join(names, ", ")
}

func (m *Memory) assignedStudentsForTeacher(teacherID string) int {
	classIDs := make(map[string]struct{})
	for _, class := range m.classes {
		if class.TeacherID == teacherID && class.IsActive {
			classIDs[class.ID] = struct{}{}
		}
	}
	studentIDs := make(map[string]struct{})
	for _, enrollment := range m.classStudents {
		if enrollment.LeftAt != nil {
			continue
		}
		if _, ok := classIDs[enrollment.ClassID]; ok {
			studentIDs[enrollment.StudentID] = struct{}{}
		}
	}

	return len(studentIDs)
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

func (m *Memory) UpdateRoom(_ context.Context, room domain.Room) (domain.Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	current, ok := m.rooms[room.ID]
	if !ok || !current.IsActive {
		return domain.Room{}, domain.ErrNotFound
	}
	if strings.TrimSpace(room.Name) == "" {
		return domain.Room{}, domain.ErrInvalidInput
	}
	for _, existing := range m.rooms {
		if existing.ID != room.ID && existing.IsActive && existing.BranchID == current.BranchID && strings.EqualFold(existing.Name, room.Name) {
			return domain.Room{}, domain.ErrConflict
		}
	}
	if room.Capacity <= 0 {
		room.Capacity = 1
	}
	current.Name = strings.TrimSpace(room.Name)
	current.Capacity = room.Capacity
	current.UpdatedAt = now()
	m.rooms[current.ID] = current

	return current, nil
}

func (m *Memory) ListRooms(_ context.Context, branchID string) ([]domain.Room, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rooms := make([]domain.Room, 0)
	for _, room := range m.rooms {
		if room.IsActive && (branchID == "" || room.BranchID == branchID) {
			rooms = append(rooms, room)
		}
	}
	sort.Slice(rooms, func(i, j int) bool { return rooms[i].Name < rooms[j].Name })

	return rooms, nil
}

func (m *Memory) GetRoom(_ context.Context, id string) (domain.Room, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	room, ok := m.rooms[id]
	if !ok {
		return domain.Room{}, domain.ErrNotFound
	}

	return room, nil
}

func (m *Memory) DeactivateRoom(_ context.Context, id string) (domain.Room, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, ok := m.rooms[id]
	if !ok {
		return domain.Room{}, domain.ErrNotFound
	}
	room.IsActive = false
	room.UpdatedAt = now()
	m.rooms[id] = room

	return room, nil
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
		if file.DeletedAt == nil && (branchID == "" || file.BranchID == branchID) {
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

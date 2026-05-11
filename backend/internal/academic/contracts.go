package academic

import (
	"context"

	"kingsway/backend/internal/auth"
	"kingsway/backend/internal/domain"
)

type BranchStore interface {
	CreateBranch(ctx context.Context, branch domain.Branch) (domain.Branch, error)
	GetBranch(ctx context.Context, id string) (domain.Branch, error)
	UpdateBranch(ctx context.Context, branch domain.Branch) (domain.Branch, error)
	DeleteBranch(ctx context.Context, id string) (domain.Branch, error)
	ListBranches(ctx context.Context) ([]domain.Branch, error)
	ListBranchesPage(ctx context.Context, page domain.PageRequest) ([]domain.Branch, int, error)
}

type StaffStore interface {
	CreateStaffMember(ctx context.Context, user domain.User, staff domain.StaffMember) (domain.StaffMember, error)
	UpdateStaffMember(ctx context.Context, staff domain.StaffMember, passwordHash string) (domain.StaffMember, error)
	DeleteStaffMember(ctx context.Context, id string) (domain.StaffMember, error)
	GetStaffMember(ctx context.Context, id string) (domain.StaffMember, error)
	ListStaffMembers(ctx context.Context, branchID string) ([]domain.StaffMember, error)
	ListStaffMembersPage(ctx context.Context, branchID string, page domain.PageRequest) ([]domain.StaffMember, int, error)
}

type StudentStore interface {
	CreateStudent(ctx context.Context, student domain.Student) (domain.Student, error)
	CreateStudentWithDetails(ctx context.Context, student domain.Student, parents []domain.StudentParentContact, registrations []domain.StudentCourseRegistration) (domain.Student, error)
	GetStudent(ctx context.Context, id string) (domain.Student, error)
	GetStudentByUser(ctx context.Context, userID string) (domain.Student, error)
	GetStudentByFIN(ctx context.Context, fin domain.FIN) (domain.Student, error)
	UpdateStudent(ctx context.Context, student domain.Student) (domain.Student, error)
	ListStudents(ctx context.Context, branchID string) ([]domain.Student, error)
	ListStudentsPage(ctx context.Context, branchID string, status domain.StudentStatus, page domain.PageRequest) ([]domain.Student, int, error)
	ListStudentAssignmentHub(ctx context.Context, filter domain.StudentAssignmentHubFilter) (domain.StudentAssignmentHubPage, error)
	CreateStudentParentContacts(ctx context.Context, studentID string, contacts []domain.StudentParentContact) ([]domain.StudentParentContact, error)
	CreateStudentCourseRegistrations(ctx context.Context, studentID string, registrations []domain.StudentCourseRegistration) ([]domain.StudentCourseRegistration, error)
}

type TeacherStore interface {
	CreateTeacher(ctx context.Context, teacher domain.Teacher) (domain.Teacher, error)
	GetTeacher(ctx context.Context, id string) (domain.Teacher, error)
	GetTeacherByUser(ctx context.Context, userID string) (domain.Teacher, error)
	UpdateTeacher(ctx context.Context, teacher domain.Teacher) (domain.Teacher, error)
	UpdateTeacherAccount(ctx context.Context, teacher domain.Teacher, user domain.User, passwordHash string) (domain.Teacher, domain.User, error)
	ListTeachers(ctx context.Context, branchID string) ([]domain.Teacher, error)
	ListTeachersPage(ctx context.Context, branchID string, status domain.TeacherStatus, page domain.PageRequest) ([]domain.Teacher, int, error)
	ReplaceTeacherCourseSpecializations(ctx context.Context, teacherID string, courseIDs []string) error
}

type CourseStore interface {
	CreateCourse(ctx context.Context, course domain.Course, categories []domain.ScoreCategory) (domain.Course, []domain.ScoreCategory, error)
	GetCourse(ctx context.Context, id string) (domain.Course, error)
	ListCourses(ctx context.Context, branchID string) ([]domain.Course, error)
	ListCoursesPage(ctx context.Context, branchID string, page domain.PageRequest) ([]domain.Course, int, error)
}

type ClassStore interface {
	CreateClass(ctx context.Context, class domain.Class) (domain.Class, error)
	GetClass(ctx context.Context, id string) (domain.Class, error)
	ListClasses(ctx context.Context, branchID string) ([]domain.Class, error)
	ListClassesPage(ctx context.Context, branchID string, active *bool, page domain.PageRequest) ([]domain.Class, int, error)
	EnrollStudent(ctx context.Context, enrollment domain.ClassStudent) (domain.ClassStudent, error)
	ListClassStudents(ctx context.Context, branchID string, classID string) ([]domain.ClassStudent, error)
	ListClassStudentsPage(ctx context.Context, branchID string, classID string, page domain.PageRequest) ([]domain.ClassStudent, int, error)
}

type AssignmentStore interface {
	CreateAssignment(ctx context.Context, assignment domain.Assignment) (domain.Assignment, error)
	ListAssignments(ctx context.Context, branchID string, classID string) ([]domain.Assignment, error)
	ListAssignmentsPage(ctx context.Context, branchID string, classID string, page domain.PageRequest) ([]domain.Assignment, int, error)
}

type RoomStore interface {
	CreateRoom(ctx context.Context, room domain.Room) (domain.Room, error)
	UpdateRoom(ctx context.Context, room domain.Room) (domain.Room, error)
	GetRoom(ctx context.Context, id string) (domain.Room, error)
	DeactivateRoom(ctx context.Context, id string) (domain.Room, error)
	ListRooms(ctx context.Context, branchID string) ([]domain.Room, error)
	ListRoomsPage(ctx context.Context, branchID string, page domain.PageRequest) ([]domain.Room, int, error)
}

type ScheduleStore interface {
	CreateScheduleItem(ctx context.Context, item domain.ScheduleItem) (domain.ScheduleItem, error)
	ListSchedule(ctx context.Context, branchID string) ([]domain.ScheduleItem, error)
	ListSchedulePage(ctx context.Context, branchID string, itemType domain.ScheduleItemType, page domain.PageRequest) ([]domain.ScheduleItem, int, error)
}

type ExamStore interface {
	CreateExam(ctx context.Context, exam domain.Exam, participantStudentIDs []string) (domain.Exam, []domain.ExamParticipant, error)
	GetExam(ctx context.Context, id string) (domain.Exam, error)
	ListExams(ctx context.Context, branchID string, classID string) ([]domain.Exam, error)
	ListExamsPage(ctx context.Context, branchID string, classID string, page domain.PageRequest) ([]domain.Exam, int, error)
	CreateExamResult(ctx context.Context, result domain.ExamResult) (domain.ExamResult, error)
	ListExamResults(ctx context.Context, branchID string, examID string, studentID string) ([]domain.ExamResult, error)
	ListExamResultsPage(ctx context.Context, branchID string, examID string, studentID string, page domain.PageRequest) ([]domain.ExamResult, int, error)
}

type DashboardStore interface {
	AcademicDashboard(ctx context.Context, branchID string) (domain.AcademicDashboard, error)
}

type Store interface {
	BranchStore
	StaffStore
	StudentStore
	TeacherStore
	CourseStore
	ClassStore
	AssignmentStore
	RoomStore
	ScheduleStore
	ExamStore
	DashboardStore
}

type AuthService interface {
	CreateUser(ctx context.Context, actor domain.Principal, input auth.CreateUserInput) (domain.User, error)
	CurrentUser(ctx context.Context, principal domain.Principal) (domain.User, error)
}

type Service struct {
	branches    BranchStore
	staff       StaffStore
	students    StudentStore
	teachers    TeacherStore
	courses     CourseStore
	classes     ClassStore
	assignments AssignmentStore
	rooms       RoomStore
	schedules   ScheduleStore
	exams       ExamStore
	dashboards  DashboardStore
	auth        AuthService
}

func NewService(store Store, authService AuthService) *Service {
	return &Service{
		branches:    store,
		staff:       store,
		students:    store,
		teachers:    store,
		courses:     store,
		classes:     store,
		assignments: store,
		rooms:       store,
		schedules:   store,
		exams:       store,
		dashboards:  store,
		auth:        authService,
	}
}

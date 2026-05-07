package academic

import (
	"context"
	"errors"
	"strings"
	"time"

	"kingsway/backend/internal/auth"
	"kingsway/backend/internal/domain"
)

type Store interface {
	CreateBranch(ctx context.Context, branch domain.Branch) (domain.Branch, error)
	GetBranch(ctx context.Context, id string) (domain.Branch, error)
	UpdateBranch(ctx context.Context, branch domain.Branch) (domain.Branch, error)
	DeleteBranch(ctx context.Context, id string) (domain.Branch, error)
	ListBranches(ctx context.Context) ([]domain.Branch, error)
	CreateStaffMember(ctx context.Context, user domain.User, staff domain.StaffMember) (domain.StaffMember, error)
	UpdateStaffMember(ctx context.Context, staff domain.StaffMember, passwordHash string) (domain.StaffMember, error)
	DeleteStaffMember(ctx context.Context, id string) (domain.StaffMember, error)
	GetStaffMember(ctx context.Context, id string) (domain.StaffMember, error)
	ListStaffMembers(ctx context.Context, branchID string) ([]domain.StaffMember, error)
	CreateStudent(ctx context.Context, student domain.Student) (domain.Student, error)
	CreateStudentWithDetails(ctx context.Context, student domain.Student, parents []domain.StudentParentContact, registrations []domain.StudentCourseRegistration) (domain.Student, error)
	GetStudent(ctx context.Context, id string) (domain.Student, error)
	GetStudentByUser(ctx context.Context, userID string) (domain.Student, error)
	GetStudentByFIN(ctx context.Context, fin domain.FIN) (domain.Student, error)
	UpdateStudent(ctx context.Context, student domain.Student) (domain.Student, error)
	ListStudents(ctx context.Context, branchID string) ([]domain.Student, error)
	ListStudentAssignmentHub(ctx context.Context, filter domain.StudentAssignmentHubFilter) (domain.StudentAssignmentHubPage, error)
	CreateStudentParentContacts(ctx context.Context, studentID string, contacts []domain.StudentParentContact) ([]domain.StudentParentContact, error)
	CreateStudentCourseRegistrations(ctx context.Context, studentID string, registrations []domain.StudentCourseRegistration) ([]domain.StudentCourseRegistration, error)
	CreateTeacher(ctx context.Context, teacher domain.Teacher) (domain.Teacher, error)
	GetTeacher(ctx context.Context, id string) (domain.Teacher, error)
	GetTeacherByUser(ctx context.Context, userID string) (domain.Teacher, error)
	UpdateTeacher(ctx context.Context, teacher domain.Teacher) (domain.Teacher, error)
	UpdateTeacherAccount(ctx context.Context, teacher domain.Teacher, user domain.User, passwordHash string) (domain.Teacher, domain.User, error)
	ListTeachers(ctx context.Context, branchID string) ([]domain.Teacher, error)
	ReplaceTeacherCourseSpecializations(ctx context.Context, teacherID string, courseIDs []string) error
	CreateCourse(ctx context.Context, course domain.Course, categories []domain.ScoreCategory) (domain.Course, []domain.ScoreCategory, error)
	GetCourse(ctx context.Context, id string) (domain.Course, error)
	ListCourses(ctx context.Context, branchID string) ([]domain.Course, error)
	CreateClass(ctx context.Context, class domain.Class) (domain.Class, error)
	GetClass(ctx context.Context, id string) (domain.Class, error)
	ListClasses(ctx context.Context, branchID string) ([]domain.Class, error)
	EnrollStudent(ctx context.Context, enrollment domain.ClassStudent) (domain.ClassStudent, error)
	ListClassStudents(ctx context.Context, branchID string, classID string) ([]domain.ClassStudent, error)
	CreateAssignment(ctx context.Context, assignment domain.Assignment) (domain.Assignment, error)
	ListAssignments(ctx context.Context, branchID string, classID string) ([]domain.Assignment, error)
	CreateRoom(ctx context.Context, room domain.Room) (domain.Room, error)
	UpdateRoom(ctx context.Context, room domain.Room) (domain.Room, error)
	GetRoom(ctx context.Context, id string) (domain.Room, error)
	DeactivateRoom(ctx context.Context, id string) (domain.Room, error)
	ListRooms(ctx context.Context, branchID string) ([]domain.Room, error)
	CreateScheduleItem(ctx context.Context, item domain.ScheduleItem) (domain.ScheduleItem, error)
	ListSchedule(ctx context.Context, branchID string) ([]domain.ScheduleItem, error)
	CreateExam(ctx context.Context, exam domain.Exam, participantStudentIDs []string) (domain.Exam, []domain.ExamParticipant, error)
	GetExam(ctx context.Context, id string) (domain.Exam, error)
	ListExams(ctx context.Context, branchID string, classID string) ([]domain.Exam, error)
	CreateExamResult(ctx context.Context, result domain.ExamResult) (domain.ExamResult, error)
	ListExamResults(ctx context.Context, branchID string, examID string, studentID string) ([]domain.ExamResult, error)
	AcademicDashboard(ctx context.Context, branchID string) (domain.AcademicDashboard, error)
}

type AuthService interface {
	CreateUser(ctx context.Context, actor domain.Principal, input auth.CreateUserInput) (domain.User, error)
	CurrentUser(ctx context.Context, principal domain.Principal) (domain.User, error)
}

type Service struct {
	store Store
	auth  AuthService
}

type CreateBranchInput struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Address     string `json:"address"`
	OpeningTime string `json:"opening_time"`
	ClosingTime string `json:"closing_time"`
}

type UpdateBranchInput struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Address     string `json:"address"`
	OpeningTime string `json:"opening_time"`
	ClosingTime string `json:"closing_time"`
}

type CreateStaffInput struct {
	BranchID           string `json:"branch_id"`
	FirstName          string `json:"first_name"`
	LastName           string `json:"last_name"`
	BirthDate          string `json:"birth_date"`
	Gender             string `json:"gender"`
	Phone              string `json:"phone"`
	Address            string `json:"address"`
	HiredAt            string `json:"hired_at"`
	SalaryAmountAZN    int    `json:"salary_amount_azn"`
	Email              string `json:"email"`
	Password           string `json:"password"`
	ProfilePhotoFileID string `json:"profile_photo_file_id"`
}

type UpdateStaffInput struct {
	ID                 string `json:"id"`
	BranchID           string `json:"branch_id"`
	FirstName          string `json:"first_name"`
	LastName           string `json:"last_name"`
	BirthDate          string `json:"birth_date"`
	Gender             string `json:"gender"`
	Phone              string `json:"phone"`
	Address            string `json:"address"`
	HiredAt            string `json:"hired_at"`
	SalaryAmountAZN    int    `json:"salary_amount_azn"`
	Email              string `json:"email"`
	Password           string `json:"password"`
	ProfilePhotoFileID string `json:"profile_photo_file_id"`
	IsActive           *bool  `json:"is_active"`
}

type CreateStudentInput struct {
	BranchID           string                     `json:"branch_id"`
	FIN                string                     `json:"fin"`
	FirstName          string                     `json:"first_name"`
	LastName           string                     `json:"last_name"`
	BirthDate          string                     `json:"birth_date"`
	Gender             string                     `json:"gender"`
	Phone              string                     `json:"phone"`
	Address            string                     `json:"address"`
	ProfilePhotoFileID string                     `json:"profile_photo_file_id"`
	Status             domain.StudentStatus       `json:"status"`
	LeftReason         string                     `json:"left_reason"`
	Parents            []StudentParentInput       `json:"parents"`
	Courses            []StudentCourseAssignInput `json:"courses"`
}

type StudentParentInput struct {
	Relation string   `json:"relation"`
	Name     string   `json:"name"`
	Phones   []string `json:"phones"`
}

type StudentCourseAssignInput struct {
	CourseID           string `json:"course_id"`
	TeacherID          string `json:"teacher_id"`
	MonthlyAmountCents int64  `json:"monthly_amount_cents"`
	StartDate          string `json:"start_date"`
}

type CreateStudentAccountInput struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type UpdateStudentInput struct {
	FirstName          string               `json:"first_name"`
	LastName           string               `json:"last_name"`
	BirthDate          string               `json:"birth_date"`
	Gender             string               `json:"gender"`
	Phone              string               `json:"phone"`
	Address            string               `json:"address"`
	ProfilePhotoFileID string               `json:"profile_photo_file_id"`
	Status             domain.StudentStatus `json:"status"`
	LeftReason         string               `json:"left_reason"`
}

type RegisterTeacherInput struct {
	BranchID           string   `json:"branch_id"`
	Email              string   `json:"email"`
	Password           string   `json:"password"`
	FirstName          string   `json:"first_name"`
	LastName           string   `json:"last_name"`
	BirthDate          string   `json:"birth_date"`
	Gender             string   `json:"gender"`
	Phone              string   `json:"phone"`
	Address            string   `json:"address"`
	ProfilePhotoFileID string   `json:"profile_photo_file_id"`
	CourseIDs          []string `json:"course_ids"`
	Subjects           []string `json:"subjects"`
}

type UpdateTeacherInput struct {
	ID                 string   `json:"id"`
	BranchID           string   `json:"branch_id"`
	Email              string   `json:"email"`
	Password           string   `json:"password"`
	FirstName          string   `json:"first_name"`
	LastName           string   `json:"last_name"`
	BirthDate          string   `json:"birth_date"`
	Gender             string   `json:"gender"`
	Phone              string   `json:"phone"`
	Address            string   `json:"address"`
	ProfilePhotoFileID string   `json:"profile_photo_file_id"`
	CourseIDs          []string `json:"course_ids"`
	Subjects           []string `json:"subjects"`
}

type CourseCategoryInput struct {
	Name             string  `json:"name"`
	MinScore         float64 `json:"min_score"`
	MaxScore         float64 `json:"max_score"`
	RequiresFeedback bool    `json:"requires_feedback"`
	RequiresDocument bool    `json:"requires_document"`
	DocumentPurpose  string  `json:"document_purpose"`
}

type CreateCourseInput struct {
	BranchID   string                `json:"branch_id"`
	Name       string                `json:"name"`
	Categories []CourseCategoryInput `json:"categories"`
}

type CreateClassInput struct {
	BranchID  string     `json:"branch_id"`
	CourseID  string     `json:"course_id"`
	TeacherID string     `json:"teacher_id"`
	Name      string     `json:"name"`
	StartDate time.Time  `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
}

type EnrollStudentInput struct {
	BranchID  string     `json:"branch_id"`
	ClassID   string     `json:"class_id"`
	StudentID string     `json:"student_id"`
	JoinedAt  time.Time  `json:"joined_at"`
	LeftAt    *time.Time `json:"left_at"`
}

type CreateAssignmentInput struct {
	BranchID       string     `json:"branch_id"`
	ClassID        string     `json:"class_id"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	DueAt          *time.Time `json:"due_at"`
	MaterialFileID string     `json:"material_file_id"`
}

type CreateRoomInput struct {
	BranchID string `json:"branch_id"`
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
}

type UpdateRoomInput struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
}

type CreateScheduleItemInput struct {
	BranchID  string                  `json:"branch_id"`
	ClassID   string                  `json:"class_id"`
	TeacherID string                  `json:"teacher_id"`
	RoomID    string                  `json:"room_id"`
	ItemType  domain.ScheduleItemType `json:"item_type"`
	Title     string                  `json:"title"`
	StartsAt  time.Time               `json:"starts_at"`
	EndsAt    time.Time               `json:"ends_at"`
}

type CreateExamInput struct {
	BranchID              string   `json:"branch_id"`
	CourseID              string   `json:"course_id"`
	ClassID               string   `json:"class_id"`
	ScheduleItemID        string   `json:"schedule_item_id"`
	Title                 string   `json:"title"`
	ParticipantStudentIDs []string `json:"participant_student_ids"`
}

type CreateExamResultInput struct {
	BranchID           string  `json:"branch_id"`
	ExamID             string  `json:"exam_id"`
	StudentID          string  `json:"student_id"`
	CategoryID         string  `json:"category_id"`
	Score              float64 `json:"score"`
	Feedback           string  `json:"feedback"`
	DocumentFileID     string  `json:"document_file_id"`
	EnteredByTeacherID string  `json:"entered_by_teacher_id"`
}

func NewService(store Store, authService AuthService) *Service {
	return &Service{store: store, auth: authService}
}

func (s *Service) CreateBranch(ctx context.Context, actor domain.Principal, input CreateBranchInput) (domain.Branch, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner); err != nil {
		return domain.Branch{}, err
	}

	openingTime, err := normalizeClockTime(input.OpeningTime)
	if err != nil {
		return domain.Branch{}, err
	}
	closingTime, err := normalizeClockTime(input.ClosingTime)
	if err != nil {
		return domain.Branch{}, err
	}

	return s.store.CreateBranch(ctx, domain.Branch{
		Name:        strings.TrimSpace(input.Name),
		Slug:        strings.TrimSpace(input.Slug),
		Address:     strings.TrimSpace(input.Address),
		OpeningTime: openingTime,
		ClosingTime: closingTime,
	})
}

func (s *Service) UpdateBranch(ctx context.Context, actor domain.Principal, input UpdateBranchInput) (domain.Branch, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner); err != nil {
		return domain.Branch{}, err
	}

	branch, err := s.store.GetBranch(ctx, input.ID)
	if err != nil {
		return domain.Branch{}, err
	}
	openingTime, err := normalizeClockTime(input.OpeningTime)
	if err != nil {
		return domain.Branch{}, err
	}
	closingTime, err := normalizeClockTime(input.ClosingTime)
	if err != nil {
		return domain.Branch{}, err
	}
	branch.Name = strings.TrimSpace(input.Name)
	branch.Slug = strings.TrimSpace(input.Slug)
	branch.Address = strings.TrimSpace(input.Address)
	branch.OpeningTime = openingTime
	branch.ClosingTime = closingTime

	return s.store.UpdateBranch(ctx, branch)
}

func (s *Service) DeleteBranch(ctx context.Context, actor domain.Principal, id string) (domain.Branch, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner); err != nil {
		return domain.Branch{}, err
	}
	if strings.TrimSpace(id) == "" {
		return domain.Branch{}, domain.ErrInvalidInput
	}

	return s.store.DeleteBranch(ctx, strings.TrimSpace(id))
}

func (s *Service) ListBranches(ctx context.Context, actor domain.Principal) ([]domain.Branch, error) {
	if actor.IsOwner() {
		return s.store.ListBranches(ctx)
	}
	if actor.BranchID == "" {
		return nil, domain.ErrForbidden
	}

	branch, err := s.store.GetBranch(ctx, actor.BranchID)
	if err != nil {
		return nil, err
	}

	return []domain.Branch{branch}, nil
}

func (s *Service) CreateStaffMember(ctx context.Context, actor domain.Principal, input CreateStaffInput) (domain.StaffMember, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner); err != nil {
		return domain.StaffMember{}, err
	}
	branchID := strings.TrimSpace(input.BranchID)
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return domain.StaffMember{}, err
	}
	if _, err := s.store.GetBranch(ctx, branchID); err != nil {
		return domain.StaffMember{}, err
	}
	if strings.TrimSpace(input.FirstName) == "" || strings.TrimSpace(input.LastName) == "" || strings.TrimSpace(input.Email) == "" || input.Password == "" {
		return domain.StaffMember{}, domain.ErrInvalidInput
	}
	birthDate, err := normalizeBirthDate(input.BirthDate)
	if err != nil {
		return domain.StaffMember{}, err
	}
	hiredAt, err := normalizeBirthDate(input.HiredAt)
	if err != nil {
		return domain.StaffMember{}, err
	}
	if !isHireDateOnOrAfterBirthDate(birthDate, hiredAt) {
		return domain.StaffMember{}, domain.ErrInvalidInput
	}
	if input.SalaryAmountAZN < 0 {
		return domain.StaffMember{}, domain.ErrInvalidInput
	}
	passwordHash, err := auth.HashPassword(input.Password)
	if err != nil {
		return domain.StaffMember{}, err
	}

	return s.store.CreateStaffMember(ctx, domain.User{
		BranchID:     branchID,
		Role:         domain.RoleReceptionist,
		Email:        strings.TrimSpace(input.Email),
		PasswordHash: passwordHash,
		FirstName:    strings.TrimSpace(input.FirstName),
		LastName:     strings.TrimSpace(input.LastName),
	}, domain.StaffMember{
		BranchID:           branchID,
		Role:               domain.RoleReceptionist,
		BirthDate:          birthDate,
		Gender:             normalizeOptionalTeacherGender(input.Gender),
		Phone:              strings.TrimSpace(input.Phone),
		Address:            strings.TrimSpace(input.Address),
		HiredAt:            hiredAt,
		SalaryAmountAZN:    input.SalaryAmountAZN,
		ProfilePhotoFileID: strings.TrimSpace(input.ProfilePhotoFileID),
	})
}

func (s *Service) UpdateStaffMember(ctx context.Context, actor domain.Principal, input UpdateStaffInput) (domain.StaffMember, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner); err != nil {
		return domain.StaffMember{}, err
	}
	current, err := s.store.GetStaffMember(ctx, strings.TrimSpace(input.ID))
	if err != nil {
		return domain.StaffMember{}, err
	}
	if err := auth.RequireBranch(actor, current.BranchID); err != nil {
		return domain.StaffMember{}, err
	}
	nextBranchID := strings.TrimSpace(input.BranchID)
	if nextBranchID != "" && nextBranchID != current.BranchID {
		if err := auth.RequireBranch(actor, nextBranchID); err != nil {
			return domain.StaffMember{}, err
		}
		if _, err := s.store.GetBranch(ctx, nextBranchID); err != nil {
			return domain.StaffMember{}, err
		}
		current.BranchID = nextBranchID
	}
	if strings.TrimSpace(input.FirstName) == "" || strings.TrimSpace(input.LastName) == "" || strings.TrimSpace(input.Email) == "" {
		return domain.StaffMember{}, domain.ErrInvalidInput
	}
	birthDate, err := normalizeBirthDate(input.BirthDate)
	if err != nil {
		return domain.StaffMember{}, err
	}
	hiredAt, err := normalizeBirthDate(input.HiredAt)
	if err != nil {
		return domain.StaffMember{}, err
	}
	if !isHireDateOnOrAfterBirthDate(birthDate, hiredAt) {
		return domain.StaffMember{}, domain.ErrInvalidInput
	}
	if input.SalaryAmountAZN < 0 {
		return domain.StaffMember{}, domain.ErrInvalidInput
	}
	passwordHash := ""
	if strings.TrimSpace(input.Password) != "" {
		passwordHash, err = auth.HashPassword(input.Password)
		if err != nil {
			return domain.StaffMember{}, err
		}
	}

	current.FirstName = strings.TrimSpace(input.FirstName)
	current.LastName = strings.TrimSpace(input.LastName)
	current.Email = strings.TrimSpace(input.Email)
	current.BirthDate = birthDate
	current.Gender = normalizeOptionalTeacherGender(input.Gender)
	current.Phone = strings.TrimSpace(input.Phone)
	current.Address = strings.TrimSpace(input.Address)
	current.HiredAt = hiredAt
	current.SalaryAmountAZN = input.SalaryAmountAZN
	current.ProfilePhotoFileID = strings.TrimSpace(input.ProfilePhotoFileID)
	if input.IsActive != nil {
		current.IsActive = *input.IsActive
	}
	return s.store.UpdateStaffMember(ctx, current, passwordHash)
}

func (s *Service) DeleteStaffMember(ctx context.Context, actor domain.Principal, id string) (domain.StaffMember, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner); err != nil {
		return domain.StaffMember{}, err
	}
	staff, err := s.store.GetStaffMember(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.StaffMember{}, err
	}
	if err := auth.RequireBranch(actor, staff.BranchID); err != nil {
		return domain.StaffMember{}, err
	}
	return s.store.DeleteStaffMember(ctx, staff.ID)
}

func (s *Service) ListStaffMembers(ctx context.Context, actor domain.Principal, branchID string) ([]domain.StaffMember, error) {
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	if err := auth.RequireBranch(actor, strings.TrimSpace(branchID)); err != nil {
		return nil, err
	}
	return s.store.ListStaffMembers(ctx, strings.TrimSpace(branchID))
}

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

	return s.store.CreateStudentWithDetails(ctx, domain.Student{
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

	student, err := s.store.GetStudentByFIN(ctx, fin)
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

		course, err := s.store.GetCourse(ctx, courseID)
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
			teacher, err := s.store.GetTeacher(ctx, teacherID)
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
	student, err := s.store.GetStudent(ctx, strings.TrimSpace(id))
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
	student, err := s.store.GetStudent(ctx, strings.TrimSpace(id))
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

	return s.store.UpdateStudent(ctx, student)
}

func (s *Service) CreateStudentAccount(ctx context.Context, actor domain.Principal, studentID string, input CreateStudentAccountInput) (domain.Student, domain.User, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist); err != nil {
		return domain.Student{}, domain.User{}, err
	}
	student, err := s.store.GetStudent(ctx, strings.TrimSpace(studentID))
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
	updated, err := s.store.UpdateStudent(ctx, student)
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
		return s.store.ListStudents(ctx, "")
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, err
	}

	students, err := s.store.ListStudents(ctx, branchID)
	if err != nil {
		return nil, err
	}
	return s.filterStudentsForActor(ctx, actor, students)
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

	return s.store.ListStudentAssignmentHub(ctx, filter)
}

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

	teacher, err := s.store.CreateTeacher(ctx, domain.Teacher{
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
		if err := s.store.ReplaceTeacherCourseSpecializations(ctx, teacher.ID, courseIDs); err != nil {
			return domain.Teacher{}, domain.User{}, err
		}
	}

	return teacher, user, nil
}

func (s *Service) UpdateTeacher(ctx context.Context, actor domain.Principal, input UpdateTeacherInput) (domain.Teacher, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist); err != nil {
		return domain.Teacher{}, err
	}
	teacher, err := s.store.GetTeacher(ctx, strings.TrimSpace(input.ID))
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

	updated, _, err := s.store.UpdateTeacherAccount(ctx, teacher, user, passwordHash)
	if err != nil {
		return domain.Teacher{}, err
	}
	courseIDs, err := s.teacherCourseIDs(ctx, updated.BranchID, input.CourseIDs, input.Subjects)
	if err != nil {
		return domain.Teacher{}, err
	}
	if len(courseIDs) > 0 {
		if err := s.store.ReplaceTeacherCourseSpecializations(ctx, teacher.ID, courseIDs); err != nil {
			return domain.Teacher{}, err
		}
	}

	return updated, nil
}

func (s *Service) ActivateTeacher(ctx context.Context, actor domain.Principal, teacherID string) (domain.Teacher, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner); err != nil {
		return domain.Teacher{}, err
	}

	teacher, err := s.store.GetTeacher(ctx, teacherID)
	if err != nil {
		return domain.Teacher{}, err
	}
	teacher.Status = domain.TeacherStatusActive

	return s.store.UpdateTeacher(ctx, teacher)
}

func (s *Service) teacherCourseIDs(ctx context.Context, branchID string, rawCourseIDs []string, subjects []string) ([]string, error) {
	seen := make(map[string]struct{})
	courseIDs := make([]string, 0, len(rawCourseIDs)+len(subjects))
	for _, courseID := range rawCourseIDs {
		courseID = strings.TrimSpace(courseID)
		if courseID == "" {
			continue
		}
		course, err := s.store.GetCourse(ctx, courseID)
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
	courses, err := s.store.ListCourses(ctx, branchID)
	if err != nil {
		return domain.Course{}, err
	}
	for _, course := range courses {
		if strings.EqualFold(course.Name, name) {
			return course, nil
		}
	}
	course, _, err := s.store.CreateCourse(ctx, domain.Course{
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
		return s.store.ListTeachers(ctx, "")
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, err
	}

	teachers, err := s.store.ListTeachers(ctx, branchID)
	if err != nil {
		return nil, err
	}
	if actor.Role == domain.RoleTeacher {
		current, err := s.store.GetTeacherByUser(ctx, actor.UserID)
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

func (s *Service) GetTeacher(ctx context.Context, actor domain.Principal, id string) (domain.Teacher, error) {
	teacher, err := s.store.GetTeacher(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.Teacher{}, err
	}
	if err := auth.RequireBranch(actor, teacher.BranchID); err != nil {
		return domain.Teacher{}, err
	}
	if actor.Role == domain.RoleTeacher {
		current, err := s.store.GetTeacherByUser(ctx, actor.UserID)
		if err != nil {
			return domain.Teacher{}, err
		}
		if current.ID != teacher.ID {
			return domain.Teacher{}, domain.ErrForbidden
		}
	}

	return teacher, nil
}

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

	return s.store.CreateCourse(ctx, domain.Course{
		BranchID: input.BranchID,
		Name:     strings.TrimSpace(input.Name),
	}, categories)
}

func (s *Service) ListCourses(ctx context.Context, actor domain.Principal, branchID string) ([]domain.Course, error) {
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	if actor.IsOwner() && branchID == "" {
		return s.store.ListCourses(ctx, "")
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, err
	}

	return s.store.ListCourses(ctx, branchID)
}

func (s *Service) GetCourse(ctx context.Context, actor domain.Principal, id string) (domain.Course, error) {
	course, err := s.store.GetCourse(ctx, strings.TrimSpace(id))
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

	return s.store.CreateClass(ctx, domain.Class{
		BranchID:  input.BranchID,
		CourseID:  strings.TrimSpace(input.CourseID),
		TeacherID: strings.TrimSpace(input.TeacherID),
		Name:      strings.TrimSpace(input.Name),
		StartDate: input.StartDate,
		EndDate:   input.EndDate,
	})
}

func (s *Service) GetClass(ctx context.Context, actor domain.Principal, id string) (domain.Class, error) {
	class, err := s.store.GetClass(ctx, strings.TrimSpace(id))
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
		return s.store.ListClasses(ctx, "")
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, err
	}

	classes, err := s.store.ListClasses(ctx, branchID)
	if err != nil {
		return nil, err
	}

	return s.filterClassesForActor(ctx, actor, classes)
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

	return s.store.EnrollStudent(ctx, domain.ClassStudent{
		BranchID:  input.BranchID,
		ClassID:   strings.TrimSpace(input.ClassID),
		StudentID: strings.TrimSpace(input.StudentID),
		JoinedAt:  input.JoinedAt,
		LeftAt:    input.LeftAt,
	})
}

func (s *Service) ListClassStudents(ctx context.Context, actor domain.Principal, classID string) ([]domain.ClassStudent, error) {
	class, err := s.store.GetClass(ctx, strings.TrimSpace(classID))
	if err != nil {
		return nil, err
	}
	if err := auth.RequireBranch(actor, class.BranchID); err != nil {
		return nil, err
	}

	return s.store.ListClassStudents(ctx, class.BranchID, class.ID)
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
		class, err := s.store.GetClass(ctx, input.ClassID)
		if err != nil {
			return domain.Assignment{}, err
		}
		teacher, err := s.store.GetTeacherByUser(ctx, actor.UserID)
		if err != nil {
			return domain.Assignment{}, err
		}
		if class.TeacherID != teacher.ID {
			return domain.Assignment{}, domain.ErrForbidden
		}
	}

	return s.store.CreateAssignment(ctx, domain.Assignment{
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
		class, err := s.store.GetClass(ctx, strings.TrimSpace(classID))
		if err != nil {
			return nil, err
		}
		if err := auth.RequireBranch(actor, class.BranchID); err != nil {
			return nil, err
		}
		if !s.canUseClass(ctx, actor, class) {
			return nil, domain.ErrForbidden
		}
		return s.store.ListAssignments(ctx, class.BranchID, class.ID)
	}
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	if actor.IsOwner() && branchID == "" {
		return s.store.ListAssignments(ctx, "", "")
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, err
	}

	assignments, err := s.store.ListAssignments(ctx, branchID, "")
	if err != nil {
		return nil, err
	}

	return s.filterAssignmentsForActor(ctx, actor, assignments)
}

func (s *Service) CreateRoom(ctx context.Context, actor domain.Principal, input CreateRoomInput) (domain.Room, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist); err != nil {
		return domain.Room{}, err
	}
	if !actor.IsOwner() {
		input.BranchID = actor.BranchID
	}
	if err := auth.RequireBranch(actor, input.BranchID); err != nil {
		return domain.Room{}, err
	}

	return s.store.CreateRoom(ctx, domain.Room{
		BranchID: input.BranchID,
		Name:     strings.TrimSpace(input.Name),
		Capacity: input.Capacity,
	})
}

func (s *Service) UpdateRoom(ctx context.Context, actor domain.Principal, input UpdateRoomInput) (domain.Room, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist); err != nil {
		return domain.Room{}, err
	}
	room, err := s.store.GetRoom(ctx, strings.TrimSpace(input.ID))
	if err != nil {
		return domain.Room{}, err
	}
	if err := auth.RequireBranch(actor, room.BranchID); err != nil {
		return domain.Room{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domain.Room{}, domain.ErrInvalidInput
	}
	capacity := input.Capacity
	if capacity <= 0 {
		capacity = 1
	}
	room.Name = name
	room.Capacity = capacity

	return s.store.UpdateRoom(ctx, room)
}

func (s *Service) RemoveRoom(ctx context.Context, actor domain.Principal, id string) (domain.Room, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist); err != nil {
		return domain.Room{}, err
	}
	room, err := s.store.GetRoom(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.Room{}, err
	}
	if err := auth.RequireBranch(actor, room.BranchID); err != nil {
		return domain.Room{}, err
	}

	return s.store.DeactivateRoom(ctx, room.ID)
}

func (s *Service) ListRooms(ctx context.Context, actor domain.Principal, branchID string) ([]domain.Room, error) {
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	if actor.IsOwner() && branchID == "" {
		return s.store.ListRooms(ctx, "")
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, err
	}

	return s.store.ListRooms(ctx, branchID)
}

func normalizeClockTime(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if _, err := time.Parse("15:04", value); err != nil {
		return "", domain.ErrInvalidInput
	}

	return value, nil
}

func normalizeBirthDate(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	parsed, err := time.Parse("02/01/2006", value)
	if err != nil {
		return "", domain.ErrInvalidInput
	}

	return parsed.Format("02/01/2006"), nil
}

func isHireDateOnOrAfterBirthDate(birthDate string, hiredAt string) bool {
	if birthDate == "" || hiredAt == "" {
		return true
	}
	birth, err := time.Parse("02/01/2006", birthDate)
	if err != nil {
		return false
	}
	hire, err := time.Parse("02/01/2006", hiredAt)
	if err != nil {
		return false
	}

	return !hire.Before(birth)
}

func (s *Service) CreateScheduleItem(ctx context.Context, actor domain.Principal, input CreateScheduleItemInput) (domain.ScheduleItem, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist, domain.RoleTeacher); err != nil {
		return domain.ScheduleItem{}, err
	}
	if !actor.IsOwner() {
		input.BranchID = actor.BranchID
	}
	if err := auth.RequireBranch(actor, input.BranchID); err != nil {
		return domain.ScheduleItem{}, err
	}
	if actor.Role == domain.RoleTeacher {
		teacher, err := s.store.GetTeacherByUser(ctx, actor.UserID)
		if err != nil {
			return domain.ScheduleItem{}, err
		}
		input.TeacherID = teacher.ID
		if input.ItemType == domain.ScheduleItemLesson {
			return domain.ScheduleItem{}, domain.ErrForbidden
		}
	}
	if input.ItemType == "" {
		input.ItemType = domain.ScheduleItemLesson
	}

	return s.store.CreateScheduleItem(ctx, domain.ScheduleItem{
		BranchID:        input.BranchID,
		ClassID:         input.ClassID,
		TeacherID:       input.TeacherID,
		RoomID:          input.RoomID,
		ItemType:        input.ItemType,
		Title:           strings.TrimSpace(input.Title),
		StartsAt:        input.StartsAt,
		EndsAt:          input.EndsAt,
		CreatedByUserID: actor.UserID,
	})
}

func (s *Service) ListSchedule(ctx context.Context, actor domain.Principal, branchID string) ([]domain.ScheduleItem, error) {
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	if actor.IsOwner() && branchID == "" {
		return s.store.ListSchedule(ctx, "")
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, err
	}

	items, err := s.store.ListSchedule(ctx, branchID)
	if err != nil {
		return nil, err
	}

	return s.filterScheduleForActor(ctx, actor, items)
}

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
		class, err := s.store.GetClass(ctx, input.ClassID)
		if err != nil {
			return domain.Exam{}, nil, err
		}
		teacher, err := s.store.GetTeacherByUser(ctx, actor.UserID)
		if err != nil {
			return domain.Exam{}, nil, err
		}
		if class.TeacherID != teacher.ID {
			return domain.Exam{}, nil, domain.ErrForbidden
		}
	}

	return s.store.CreateExam(ctx, domain.Exam{
		BranchID:        input.BranchID,
		CourseID:        strings.TrimSpace(input.CourseID),
		ClassID:         strings.TrimSpace(input.ClassID),
		ScheduleItemID:  strings.TrimSpace(input.ScheduleItemID),
		Title:           strings.TrimSpace(input.Title),
		CreatedByUserID: actor.UserID,
	}, input.ParticipantStudentIDs)
}

func (s *Service) GetExam(ctx context.Context, actor domain.Principal, id string) (domain.Exam, error) {
	exam, err := s.store.GetExam(ctx, strings.TrimSpace(id))
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
		class, err := s.store.GetClass(ctx, strings.TrimSpace(classID))
		if err != nil {
			return nil, err
		}
		if err := auth.RequireBranch(actor, class.BranchID); err != nil {
			return nil, err
		}
		if !s.canUseClass(ctx, actor, class) {
			return nil, domain.ErrForbidden
		}
		return s.store.ListExams(ctx, class.BranchID, class.ID)
	}
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	if actor.IsOwner() && branchID == "" {
		return s.store.ListExams(ctx, "", "")
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, err
	}

	exams, err := s.store.ListExams(ctx, branchID, "")
	if err != nil {
		return nil, err
	}

	return s.filterExamsForActor(ctx, actor, exams)
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
		teacher, err := s.store.GetTeacherByUser(ctx, actor.UserID)
		if err != nil {
			return domain.ExamResult{}, err
		}
		teacherID = teacher.ID
		exam, err := s.store.GetExam(ctx, input.ExamID)
		if err != nil {
			return domain.ExamResult{}, err
		}
		if exam.ClassID != "" {
			class, err := s.store.GetClass(ctx, exam.ClassID)
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

	return s.store.CreateExamResult(ctx, domain.ExamResult{
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
		return s.store.ListExamResults(ctx, "", strings.TrimSpace(examID), strings.TrimSpace(studentID))
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, err
	}

	results, err := s.store.ListExamResults(ctx, branchID, strings.TrimSpace(examID), strings.TrimSpace(studentID))
	if err != nil {
		return nil, err
	}

	return s.filterExamResultsForActor(ctx, actor, results)
}

func (s *Service) AcademicDashboard(ctx context.Context, actor domain.Principal, branchID string) (domain.AcademicDashboard, error) {
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	if actor.IsOwner() && branchID == "" {
		return s.store.AcademicDashboard(ctx, "")
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return domain.AcademicDashboard{}, err
	}

	return s.store.AcademicDashboard(ctx, branchID)
}

func (s *Service) filterClassesForActor(ctx context.Context, actor domain.Principal, classes []domain.Class) ([]domain.Class, error) {
	switch actor.Role {
	case domain.RoleOwner, domain.RoleReceptionist:
		return classes, nil
	case domain.RoleTeacher:
		teacher, err := s.store.GetTeacherByUser(ctx, actor.UserID)
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
		student, err := s.store.GetStudentByUser(ctx, actor.UserID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return []domain.Class{}, nil
			}
			return nil, err
		}
		enrollments, err := s.store.ListClassStudents(ctx, student.BranchID, "")
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
		teacher, err := s.store.GetTeacherByUser(ctx, actor.UserID)
		if err != nil {
			return nil, err
		}
		classes, err := s.store.ListClasses(ctx, teacher.BranchID)
		if err != nil {
			return nil, err
		}
		classIDs := make(map[string]bool)
		for _, class := range classes {
			if class.TeacherID == teacher.ID {
				classIDs[class.ID] = true
			}
		}
		enrollments, err := s.store.ListClassStudents(ctx, teacher.BranchID, "")
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
		student, err := s.store.GetStudentByUser(ctx, actor.UserID)
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
		class, err := s.store.GetClass(ctx, assignment.ClassID)
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
		teacher, err := s.store.GetTeacherByUser(ctx, actor.UserID)
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
		student, err := s.store.GetStudentByUser(ctx, actor.UserID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return []domain.ScheduleItem{}, nil
			}
			return nil, err
		}
		enrollments, err := s.store.ListClassStudents(ctx, student.BranchID, "")
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
		teacher, err := s.store.GetTeacherByUser(ctx, actor.UserID)
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
			class, err := s.store.GetClass(ctx, exam.ClassID)
			if err != nil {
				return nil, err
			}
			if class.TeacherID == teacher.ID {
				filtered = append(filtered, exam)
			}
		}
		return filtered, nil
	case domain.RoleStudent:
		student, err := s.store.GetStudentByUser(ctx, actor.UserID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return []domain.Exam{}, nil
			}
			return nil, err
		}
		enrollments, err := s.store.ListClassStudents(ctx, student.BranchID, "")
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
		teacher, err := s.store.GetTeacherByUser(ctx, actor.UserID)
		if err != nil {
			return nil, err
		}
		filtered := make([]domain.ExamResult, 0, len(results))
		for _, result := range results {
			if result.EnteredByTeacherID == teacher.ID {
				filtered = append(filtered, result)
				continue
			}
			exam, err := s.store.GetExam(ctx, result.ExamID)
			if err != nil {
				return nil, err
			}
			if exam.ClassID == "" {
				continue
			}
			class, err := s.store.GetClass(ctx, exam.ClassID)
			if err != nil {
				return nil, err
			}
			if class.TeacherID == teacher.ID {
				filtered = append(filtered, result)
			}
		}
		return filtered, nil
	case domain.RoleStudent:
		student, err := s.store.GetStudentByUser(ctx, actor.UserID)
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
		teacher, err := s.store.GetTeacherByUser(ctx, actor.UserID)
		return err == nil && class.TeacherID == teacher.ID
	case domain.RoleStudent:
		student, err := s.store.GetStudentByUser(ctx, actor.UserID)
		if err != nil {
			return false
		}
		enrollments, err := s.store.ListClassStudents(ctx, student.BranchID, class.ID)
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

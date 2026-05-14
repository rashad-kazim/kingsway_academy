package academic

import (
	"time"

	"kingsway/backend/internal/domain"
)

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

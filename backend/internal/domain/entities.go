package domain

import "time"

type IdempotencyStatus string

const (
	IdempotencyStatusPending   IdempotencyStatus = "pending"
	IdempotencyStatusCompleted IdempotencyStatus = "completed"
)

type IdempotencyRecord struct {
	ActorUserID    string            `json:"actor_user_id"`
	Method         string            `json:"method"`
	Path           string            `json:"path"`
	Key            string            `json:"key"`
	RequestHash    string            `json:"request_hash"`
	Status         IdempotencyStatus `json:"status"`
	ResponseStatus int               `json:"response_status"`
	ResponseBody   []byte            `json:"response_body"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	ExpiresAt      time.Time         `json:"expires_at"`
}

type IdempotencyBeginResult struct {
	Started bool              `json:"started"`
	Record  IdempotencyRecord `json:"record"`
}

type Branch struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Address     string    `json:"address,omitempty"`
	OpeningTime string    `json:"opening_time,omitempty"`
	ClosingTime string    `json:"closing_time,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type User struct {
	ID           string    `json:"id"`
	BranchID     string    `json:"branch_id,omitempty"`
	Role         Role      `json:"role"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	IsActive     bool      `json:"is_active"`
	LastLoginAt  string    `json:"last_login_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type StaffMember struct {
	ID                 string    `json:"id"`
	BranchID           string    `json:"branch_id"`
	UserID             string    `json:"user_id"`
	Role               Role      `json:"role"`
	Email              string    `json:"email"`
	FirstName          string    `json:"first_name"`
	LastName           string    `json:"last_name"`
	BirthDate          string    `json:"birth_date,omitempty"`
	Gender             string    `json:"gender,omitempty"`
	Phone              string    `json:"phone,omitempty"`
	Address            string    `json:"address,omitempty"`
	HiredAt            string    `json:"hired_at,omitempty"`
	SalaryAmountAZN    int       `json:"salary_amount_azn"`
	ProfilePhotoFileID string    `json:"profile_photo_file_id,omitempty"`
	IsActive           bool      `json:"is_active"`
	LastLoginAt        string    `json:"last_login_at,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type Principal struct {
	UserID   string `json:"user_id"`
	BranchID string `json:"branch_id,omitempty"`
	Role     Role   `json:"role"`
}

func (p Principal) IsOwner() bool {
	return p.Role == RoleOwner
}

func (p Principal) CanUseBranch(branchID string) bool {
	return p.IsOwner() || (p.BranchID != "" && p.BranchID == branchID)
}

type StudentStatus string

const (
	StudentStatusActive    StudentStatus = "active"
	StudentStatusGraduated StudentStatus = "graduated"
	StudentStatusLeft      StudentStatus = "left"
)

type Student struct {
	ID                 string        `json:"id"`
	BranchID           string        `json:"branch_id"`
	UserID             string        `json:"user_id,omitempty"`
	FIN                FIN           `json:"fin"`
	FirstName          string        `json:"first_name"`
	LastName           string        `json:"last_name"`
	BirthDate          string        `json:"birth_date,omitempty"`
	Gender             string        `json:"gender,omitempty"`
	Phone              string        `json:"phone,omitempty"`
	Address            string        `json:"address,omitempty"`
	ProfilePhotoFileID string        `json:"profile_photo_file_id,omitempty"`
	Status             StudentStatus `json:"status"`
	LeftReason         string        `json:"left_reason,omitempty"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}

type StudentParentContact struct {
	ID        string    `json:"id"`
	BranchID  string    `json:"branch_id"`
	StudentID string    `json:"student_id"`
	Relation  string    `json:"relation"`
	Name      string    `json:"name"`
	Phones    []string  `json:"phones"`
	CreatedAt time.Time `json:"created_at"`
}

type StudentCourseRegistration struct {
	ID                 string    `json:"id"`
	BranchID           string    `json:"branch_id"`
	StudentID          string    `json:"student_id"`
	CourseID           string    `json:"course_id"`
	TeacherID          string    `json:"teacher_id,omitempty"`
	MonthlyAmountCents int64     `json:"monthly_amount_cents"`
	StartDate          string    `json:"start_date"`
	CreatedAt          time.Time `json:"created_at"`
}

type StudentTeacherAssignment struct {
	ID        string    `json:"id"`
	BranchID  string    `json:"branch_id"`
	StudentID string    `json:"student_id"`
	TeacherID string    `json:"teacher_id"`
	ClassID   string    `json:"class_id,omitempty"`
	ValidFrom string    `json:"valid_from"`
	ValidTo   string    `json:"valid_to,omitempty"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

type StudentAssignmentHubFilter struct {
	BranchID  string        `json:"branch_id,omitempty"`
	Status    StudentStatus `json:"status,omitempty"`
	TeacherID string        `json:"teacher_id,omitempty"`
	Query     string        `json:"q,omitempty"`
	Limit     int           `json:"limit"`
	Offset    int           `json:"offset"`
}

type StudentAssignmentHubPage struct {
	Items  []StudentAssignmentHubRecord `json:"items"`
	Total  int                          `json:"total"`
	Limit  int                          `json:"limit"`
	Offset int                          `json:"offset"`
}

type StudentAssignmentHubRecord struct {
	ID                     string        `json:"id"`
	BranchID               string        `json:"branch_id"`
	BranchName             string        `json:"branch_name"`
	UserID                 string        `json:"user_id,omitempty"`
	FIN                    FIN           `json:"fin"`
	FirstName              string        `json:"first_name"`
	LastName               string        `json:"last_name"`
	ProfilePhotoFileID     string        `json:"profile_photo_file_id,omitempty"`
	Status                 StudentStatus `json:"status"`
	ActiveTeacherID        string        `json:"active_teacher_id,omitempty"`
	ActiveTeacherFirstName string        `json:"active_teacher_first_name,omitempty"`
	ActiveTeacherLastName  string        `json:"active_teacher_last_name,omitempty"`
	RegisteredAt           time.Time     `json:"registered_at"`
	CreatedAt              time.Time     `json:"created_at"`
	UpdatedAt              time.Time     `json:"updated_at"`
}

type TeacherStatus string

const (
	TeacherStatusPending    TeacherStatus = "pending_owner_approval"
	TeacherStatusActive     TeacherStatus = "active"
	TeacherStatusTerminated TeacherStatus = "terminated"
)

type Teacher struct {
	ID                 string        `json:"id"`
	BranchID           string        `json:"branch_id"`
	UserID             string        `json:"user_id"`
	Status             TeacherStatus `json:"status"`
	BirthDate          string        `json:"birth_date,omitempty"`
	Gender             string        `json:"gender,omitempty"`
	Phone              string        `json:"phone,omitempty"`
	Address            string        `json:"address,omitempty"`
	ProfilePhotoFileID string        `json:"profile_photo_file_id,omitempty"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}

type TeacherFinanceRecord struct {
	ID                          string          `json:"id"`
	BranchID                    string          `json:"branch_id"`
	BranchName                  string          `json:"branch_name"`
	UserID                      string          `json:"user_id"`
	Email                       string          `json:"email"`
	FirstName                   string          `json:"first_name"`
	LastName                    string          `json:"last_name"`
	ProfilePhotoFileID          string          `json:"profile_photo_file_id,omitempty"`
	Subject                     string          `json:"subject"`
	Status                      TeacherStatus   `json:"status"`
	SalaryType                  SalaryModelType `json:"salary_type,omitempty"`
	AssignedStudents            int             `json:"assigned_students"`
	CalculatedSalaryAmountCents int64           `json:"calculated_salary_amount_cents"`
	CreatedAt                   time.Time       `json:"created_at"`
	UpdatedAt                   time.Time       `json:"updated_at"`
}

type Course struct {
	ID        string    `json:"id"`
	BranchID  string    `json:"branch_id"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ScoreCategory struct {
	ID               string    `json:"id"`
	BranchID         string    `json:"branch_id"`
	CourseID         string    `json:"course_id"`
	Name             string    `json:"name"`
	MinScore         float64   `json:"min_score"`
	MaxScore         float64   `json:"max_score"`
	RequiresFeedback bool      `json:"requires_feedback"`
	RequiresDocument bool      `json:"requires_document"`
	DocumentPurpose  string    `json:"document_purpose,omitempty"`
	SortOrder        int       `json:"sort_order"`
	CreatedAt        time.Time `json:"created_at"`
}

type Class struct {
	ID        string     `json:"id"`
	BranchID  string     `json:"branch_id"`
	CourseID  string     `json:"course_id"`
	TeacherID string     `json:"teacher_id"`
	Name      string     `json:"name"`
	StartDate time.Time  `json:"start_date"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type ClassStudent struct {
	ID        string     `json:"id"`
	BranchID  string     `json:"branch_id"`
	ClassID   string     `json:"class_id"`
	StudentID string     `json:"student_id"`
	JoinedAt  time.Time  `json:"joined_at"`
	LeftAt    *time.Time `json:"left_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type Assignment struct {
	ID              string     `json:"id"`
	BranchID        string     `json:"branch_id"`
	ClassID         string     `json:"class_id"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	DueAt           *time.Time `json:"due_at,omitempty"`
	MaterialFileID  string     `json:"material_file_id,omitempty"`
	CreatedByUserID string     `json:"created_by_user_id"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type Room struct {
	ID        string    `json:"id"`
	BranchID  string    `json:"branch_id"`
	Name      string    `json:"name"`
	Capacity  int       `json:"capacity"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ScheduleItemType string

const (
	ScheduleItemLesson ScheduleItemType = "lesson"
	ScheduleItemExam   ScheduleItemType = "exam"
	ScheduleItemEvent  ScheduleItemType = "event"
)

type ScheduleItem struct {
	ID              string           `json:"id"`
	BranchID        string           `json:"branch_id"`
	ClassID         string           `json:"class_id,omitempty"`
	TeacherID       string           `json:"teacher_id"`
	RoomID          string           `json:"room_id"`
	ItemType        ScheduleItemType `json:"item_type"`
	Title           string           `json:"title"`
	StartsAt        time.Time        `json:"starts_at"`
	EndsAt          time.Time        `json:"ends_at"`
	CreatedByUserID string           `json:"created_by_user_id"`
	CancelledAt     *time.Time       `json:"cancelled_at,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

func (s ScheduleItem) Range() TimeRange {
	return TimeRange{Start: s.StartsAt, End: s.EndsAt}
}

type RSVPStatus string

const (
	RSVPStatusPending  RSVPStatus = "pending"
	RSVPStatusAccepted RSVPStatus = "accepted"
	RSVPStatusDeclined RSVPStatus = "declined"
)

type Exam struct {
	ID              string    `json:"id"`
	BranchID        string    `json:"branch_id"`
	CourseID        string    `json:"course_id"`
	ClassID         string    `json:"class_id,omitempty"`
	ScheduleItemID  string    `json:"schedule_item_id,omitempty"`
	Title           string    `json:"title"`
	CreatedByUserID string    `json:"created_by_user_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ExamParticipant struct {
	ExamID     string     `json:"exam_id"`
	StudentID  string     `json:"student_id"`
	BranchID   string     `json:"branch_id"`
	RSVPStatus RSVPStatus `json:"rsvp_status"`
}

type ExamResult struct {
	ID                 string    `json:"id"`
	BranchID           string    `json:"branch_id"`
	ExamID             string    `json:"exam_id"`
	StudentID          string    `json:"student_id"`
	CategoryID         string    `json:"category_id"`
	Score              float64   `json:"score"`
	Feedback           string    `json:"feedback,omitempty"`
	DocumentFileID     string    `json:"document_file_id,omitempty"`
	EnteredByTeacherID string    `json:"entered_by_teacher_id"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusPaid      PaymentStatus = "paid"
	PaymentStatusOverdue   PaymentStatus = "overdue"
	PaymentStatusCancelled PaymentStatus = "cancelled"
)

type Payment struct {
	ID              string        `json:"id"`
	BranchID        string        `json:"branch_id"`
	StudentID       string        `json:"student_id"`
	AmountCents     int64         `json:"amount_cents"`
	Currency        string        `json:"currency"`
	DueDate         time.Time     `json:"due_date"`
	PaidAt          *time.Time    `json:"paid_at,omitempty"`
	Status          PaymentStatus `json:"status"`
	ReceiptFileID   string        `json:"receipt_file_id,omitempty"`
	CreatedByUserID string        `json:"created_by_user_id"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type SalaryModel struct {
	ID                        string          `json:"id"`
	BranchID                  string          `json:"branch_id"`
	TeacherID                 string          `json:"teacher_id"`
	ModelType                 SalaryModelType `json:"model_type"`
	FixedMonthlyAmountCents   int64           `json:"fixed_monthly_amount_cents,omitempty"`
	StudentPercentBasisPoints int             `json:"student_percent_basis_points,omitempty"`
	ActiveFrom                time.Time       `json:"active_from"`
	ActiveTo                  *time.Time      `json:"active_to,omitempty"`
	ApprovedByOwnerUserID     string          `json:"approved_by_owner_user_id"`
	CreatedAt                 time.Time       `json:"created_at"`
}

type FileObject struct {
	ID                string       `json:"id"`
	BranchID          string       `json:"branch_id"`
	UploaderUserID    string       `json:"uploader_user_id"`
	OwnerType         string       `json:"owner_type"`
	OwnerID           string       `json:"owner_id"`
	Category          FileCategory `json:"category"`
	Purpose           FilePurpose  `json:"purpose"`
	OriginalFilename  string       `json:"original_filename"`
	MimeType          string       `json:"mime_type"`
	OriginalSizeBytes int64        `json:"original_size_bytes"`
	StoredSizeBytes   int64        `json:"stored_size_bytes"`
	OriginalSHA256    string       `json:"original_sha256"`
	StorageBucket     string       `json:"storage_bucket"`
	StorageKey        string       `json:"storage_key"`
	RetentionUntil    *time.Time   `json:"retention_until,omitempty"`
	DeletedAt         *time.Time   `json:"deleted_at,omitempty"`
	CreatedAt         time.Time    `json:"created_at"`
}

type Notification struct {
	ID              string         `json:"id"`
	BranchID        string         `json:"branch_id,omitempty"`
	RecipientUserID string         `json:"recipient_user_id"`
	Type            string         `json:"type"`
	Payload         map[string]any `json:"payload"`
	DedupeKey       string         `json:"dedupe_key,omitempty"`
	ReadAt          *time.Time     `json:"read_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
}

type OutboxEventStatus string

const (
	OutboxEventPending    OutboxEventStatus = "pending"
	OutboxEventPublishing OutboxEventStatus = "publishing"
	OutboxEventPublished  OutboxEventStatus = "published"
	OutboxEventFailed     OutboxEventStatus = "failed"
)

func (s OutboxEventStatus) IsValid() bool {
	switch s {
	case OutboxEventPending, OutboxEventPublishing, OutboxEventPublished, OutboxEventFailed:
		return true
	default:
		return false
	}
}

type OutboxEvent struct {
	ID            string            `json:"id"`
	Topic         string            `json:"topic"`
	Payload       map[string]any    `json:"payload"`
	Status        OutboxEventStatus `json:"status"`
	Attempts      int               `json:"attempts"`
	NextAttemptAt time.Time         `json:"next_attempt_at"`
	LockedAt      *time.Time        `json:"locked_at,omitempty"`
	LastError     string            `json:"last_error,omitempty"`
	PublishedAt   *time.Time        `json:"published_at,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
}

type AcademicDashboard struct {
	BranchID             string `json:"branch_id,omitempty"`
	ActiveStudents       int    `json:"active_students"`
	ActiveTeachers       int    `json:"active_teachers"`
	ActiveClasses        int    `json:"active_classes"`
	UpcomingSchedule     int    `json:"upcoming_schedule"`
	PendingPayments      int    `json:"pending_payments"`
	WritingFilesRetained int    `json:"writing_files_retained"`
}

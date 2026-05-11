package finance

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kingsway/backend/internal/auth"
	"kingsway/backend/internal/domain"
)

type Store interface {
	GetStudent(ctx context.Context, id string) (domain.Student, error)
	GetStudentByUser(ctx context.Context, userID string) (domain.Student, error)
	GetTeacher(ctx context.Context, id string) (domain.Teacher, error)
	GetTeacherByUser(ctx context.Context, userID string) (domain.Teacher, error)
	UpdateTeacher(ctx context.Context, teacher domain.Teacher) (domain.Teacher, error)
	DeleteTeacher(ctx context.Context, id string) (domain.Teacher, error)
	ListTeacherFinanceRecords(ctx context.Context, branchID string) ([]domain.TeacherFinanceRecord, error)
	ListTeacherFinanceRecordsPage(ctx context.Context, branchID string, subject string, status domain.TeacherStatus, salaryModel domain.SalaryModelType, page domain.PageRequest) ([]domain.TeacherFinanceRecord, int, error)
	CreatePayment(ctx context.Context, payment domain.Payment) (domain.Payment, error)
	GetPayment(ctx context.Context, id string) (domain.Payment, error)
	ListPayments(ctx context.Context, branchID string) ([]domain.Payment, error)
	ListPaymentsPage(ctx context.Context, branchID string, status domain.PaymentStatus, page domain.PageRequest) ([]domain.Payment, int, error)
	CreateSalaryModel(ctx context.Context, model domain.SalaryModel) (domain.SalaryModel, error)
	ListSalaryModels(ctx context.Context, branchID string) ([]domain.SalaryModel, error)
	ListSalaryModelsPage(ctx context.Context, branchID string, teacherID string, page domain.PageRequest) ([]domain.SalaryModel, int, error)
}

type JSONCache interface {
	Get(ctx context.Context, key string, out any) (bool, error)
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
}

type EventPublisher interface {
	Publish(ctx context.Context, topic string, payload any) error
}

type Service struct {
	store  Store
	cache  JSONCache
	events EventPublisher
}

type Option func(*Service)

func WithCache(cache JSONCache) Option {
	return func(s *Service) {
		s.cache = cache
	}
}

func WithEventPublisher(events EventPublisher) Option {
	return func(s *Service) {
		s.events = events
	}
}

type CreatePaymentInput struct {
	BranchID      string               `json:"branch_id"`
	StudentID     string               `json:"student_id"`
	AmountCents   int64                `json:"amount_cents"`
	Currency      string               `json:"currency"`
	DueDate       time.Time            `json:"due_date"`
	PaidAt        *time.Time           `json:"paid_at"`
	Status        domain.PaymentStatus `json:"status"`
	ReceiptFileID string               `json:"receipt_file_id"`
}

type CreateSalaryModelInput struct {
	BranchID                  string                 `json:"branch_id"`
	TeacherID                 string                 `json:"teacher_id"`
	ModelType                 domain.SalaryModelType `json:"model_type"`
	FixedMonthlyAmountCents   int64                  `json:"fixed_monthly_amount_cents"`
	StudentPercentBasisPoints int                    `json:"student_percent_basis_points"`
	ActiveFrom                time.Time              `json:"active_from"`
	ActiveTo                  *time.Time             `json:"active_to"`
}

type SwapAllocationInput struct {
	TotalAmountCents  int64 `json:"total_amount_cents"`
	FirstTeacherDays  int   `json:"first_teacher_days"`
	SecondTeacherDays int   `json:"second_teacher_days"`
}

type SwapAllocationResult struct {
	FirstTeacherAmountCents  int64 `json:"first_teacher_amount_cents"`
	SecondTeacherAmountCents int64 `json:"second_teacher_amount_cents"`
}

type TeacherFinanceFilter struct {
	BranchID    string                 `json:"branch_id"`
	Subject     string                 `json:"subject"`
	Status      domain.TeacherStatus   `json:"status"`
	SalaryModel domain.SalaryModelType `json:"salary_model"`
}

func NewService(store Store, options ...Option) *Service {
	service := &Service{store: store}
	for _, option := range options {
		option(service)
	}

	return service
}

func (s *Service) CreatePayment(ctx context.Context, actor domain.Principal, input CreatePaymentInput) (domain.Payment, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist); err != nil {
		return domain.Payment{}, err
	}
	if !actor.IsOwner() {
		input.BranchID = actor.BranchID
	}
	if err := auth.RequireBranch(actor, input.BranchID); err != nil {
		return domain.Payment{}, err
	}

	student, err := s.store.GetStudent(ctx, input.StudentID)
	if err != nil {
		return domain.Payment{}, err
	}
	if student.BranchID != input.BranchID {
		return domain.Payment{}, domain.ErrForbidden
	}

	payment, err := s.store.CreatePayment(ctx, domain.Payment{
		BranchID:        input.BranchID,
		StudentID:       input.StudentID,
		AmountCents:     input.AmountCents,
		Currency:        input.Currency,
		DueDate:         input.DueDate,
		PaidAt:          input.PaidAt,
		Status:          input.Status,
		ReceiptFileID:   input.ReceiptFileID,
		CreatedByUserID: actor.UserID,
	})
	if err != nil {
		return domain.Payment{}, err
	}
	s.publishBestEffort(ctx, "finance.payment.created", payment)

	return payment, nil
}

func (s *Service) ListPayments(ctx context.Context, actor domain.Principal, branchID string) ([]domain.Payment, error) {
	if actor.Role == domain.RoleTeacher {
		return nil, domain.ErrForbidden
	}
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	if actor.IsOwner() && branchID == "" {
		return s.store.ListPayments(ctx, "")
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, err
	}

	payments, err := s.store.ListPayments(ctx, branchID)
	if err != nil {
		return nil, err
	}
	if actor.Role != domain.RoleStudent {
		return payments, nil
	}

	student, err := s.store.GetStudentByUser(ctx, actor.UserID)
	if err != nil {
		return nil, err
	}
	filtered := make([]domain.Payment, 0, len(payments))
	for _, payment := range payments {
		if payment.StudentID == student.ID {
			filtered = append(filtered, payment)
		}
	}

	return filtered, nil
}

func (s *Service) ListPaymentsPage(ctx context.Context, actor domain.Principal, branchID string, status domain.PaymentStatus, page domain.PageRequest) ([]domain.Payment, int, error) {
	if actor.Role == domain.RoleTeacher {
		return nil, 0, domain.ErrForbidden
	}
	if actor.Role == domain.RoleStudent {
		payments, err := s.ListPayments(ctx, actor, branchID)
		if err != nil {
			return nil, 0, err
		}
		if status != "" {
			filtered := make([]domain.Payment, 0, len(payments))
			for _, payment := range payments {
				if payment.Status == status {
					filtered = append(filtered, payment)
				}
			}
			payments = filtered
		}
		items, total := domain.PageSlice(payments, page)
		return items, total, nil
	}
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	branchID = strings.TrimSpace(branchID)
	if actor.IsOwner() && branchID == "" {
		return s.store.ListPaymentsPage(ctx, "", status, page)
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, 0, err
	}
	return s.store.ListPaymentsPage(ctx, branchID, status, page)
}

func (s *Service) GetPayment(ctx context.Context, actor domain.Principal, id string) (domain.Payment, error) {
	payment, err := s.store.GetPayment(ctx, id)
	if err != nil {
		return domain.Payment{}, err
	}
	if err := auth.RequireBranch(actor, payment.BranchID); err != nil {
		return domain.Payment{}, err
	}
	if actor.Role == domain.RoleStudent {
		student, err := s.store.GetStudent(ctx, payment.StudentID)
		if err != nil {
			return domain.Payment{}, err
		}
		if student.UserID != actor.UserID {
			return domain.Payment{}, domain.ErrForbidden
		}
	}

	return payment, nil
}

func (s *Service) CreateSalaryModel(ctx context.Context, actor domain.Principal, input CreateSalaryModelInput) (domain.SalaryModel, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner); err != nil {
		return domain.SalaryModel{}, err
	}
	if input.BranchID == "" || input.TeacherID == "" || !input.ModelType.IsValid() {
		return domain.SalaryModel{}, domain.ErrInvalidInput
	}

	teacher, err := s.store.GetTeacher(ctx, input.TeacherID)
	if err != nil {
		return domain.SalaryModel{}, err
	}
	if teacher.BranchID != input.BranchID {
		return domain.SalaryModel{}, domain.ErrInvalidInput
	}
	if input.ModelType == domain.SalaryModelFixed || input.ModelType == domain.SalaryModelHybrid {
		if input.FixedMonthlyAmountCents <= 0 {
			return domain.SalaryModel{}, domain.ErrInvalidInput
		}
	}
	if input.ModelType == domain.SalaryModelPercent || input.ModelType == domain.SalaryModelHybrid {
		if input.StudentPercentBasisPoints < 0 || input.StudentPercentBasisPoints > 10000 {
			return domain.SalaryModel{}, domain.ErrInvalidInput
		}
	}

	model, err := s.store.CreateSalaryModel(ctx, domain.SalaryModel{
		BranchID:                  input.BranchID,
		TeacherID:                 input.TeacherID,
		ModelType:                 input.ModelType,
		FixedMonthlyAmountCents:   input.FixedMonthlyAmountCents,
		StudentPercentBasisPoints: input.StudentPercentBasisPoints,
		ActiveFrom:                input.ActiveFrom,
		ActiveTo:                  input.ActiveTo,
		ApprovedByOwnerUserID:     actor.UserID,
	})
	if err != nil {
		return domain.SalaryModel{}, err
	}
	if teacher.Status == domain.TeacherStatusPending {
		teacher.Status = domain.TeacherStatusActive
		if _, err := s.store.UpdateTeacher(ctx, teacher); err != nil {
			return domain.SalaryModel{}, err
		}
	}
	s.publishBestEffort(ctx, "finance.salary_model.created", model)

	return model, nil
}

func (s *Service) ListSalaryModels(ctx context.Context, actor domain.Principal, branchID string) ([]domain.SalaryModel, error) {
	if actor.Role == domain.RoleReceptionist || actor.Role == domain.RoleStudent {
		return nil, domain.ErrForbidden
	}
	if actor.Role == domain.RoleTeacher {
		teacher, err := s.store.GetTeacherByUser(ctx, actor.UserID)
		if err != nil {
			return nil, err
		}
		models, err := s.store.ListSalaryModels(ctx, teacher.BranchID)
		if err != nil {
			return nil, err
		}
		filtered := make([]domain.SalaryModel, 0)
		for _, model := range models {
			if model.TeacherID == teacher.ID {
				filtered = append(filtered, model)
			}
		}
		return filtered, nil
	}
	if actor.IsOwner() && branchID == "" {
		return s.store.ListSalaryModels(ctx, "")
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, err
	}

	return s.store.ListSalaryModels(ctx, branchID)
}

func (s *Service) ListSalaryModelsPage(ctx context.Context, actor domain.Principal, branchID string, page domain.PageRequest) ([]domain.SalaryModel, int, error) {
	if actor.Role == domain.RoleReceptionist || actor.Role == domain.RoleStudent {
		return nil, 0, domain.ErrForbidden
	}
	teacherID := ""
	if actor.Role == domain.RoleTeacher {
		teacher, err := s.store.GetTeacherByUser(ctx, actor.UserID)
		if err != nil {
			return nil, 0, err
		}
		branchID = teacher.BranchID
		teacherID = teacher.ID
	}
	if actor.IsOwner() {
		branchID = strings.TrimSpace(branchID)
	}
	if actor.IsOwner() && branchID == "" {
		return s.store.ListSalaryModelsPage(ctx, "", teacherID, page)
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, 0, err
	}
	return s.store.ListSalaryModelsPage(ctx, branchID, teacherID, page)
}

func (s *Service) ListTeacherFinanceRecords(ctx context.Context, actor domain.Principal, filter TeacherFinanceFilter) ([]domain.TeacherFinanceRecord, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner); err != nil {
		return nil, err
	}
	branchID := strings.TrimSpace(filter.BranchID)
	if branchID != "" {
		if err := auth.RequireBranch(actor, branchID); err != nil {
			return nil, err
		}
	}

	records, err := s.store.ListTeacherFinanceRecords(ctx, branchID)
	if err != nil {
		return nil, err
	}

	subject := normalizeFilter(filter.Subject)
	filtered := make([]domain.TeacherFinanceRecord, 0, len(records))
	for _, record := range records {
		if subject != "" && !strings.Contains(normalizeFilter(record.Subject), subject) {
			continue
		}
		if filter.Status != "" && record.Status != filter.Status {
			continue
		}
		if filter.SalaryModel != "" && record.SalaryType != filter.SalaryModel {
			continue
		}
		filtered = append(filtered, record)
	}

	return filtered, nil
}

func (s *Service) ListTeacherFinanceRecordsPage(ctx context.Context, actor domain.Principal, filter TeacherFinanceFilter, page domain.PageRequest) ([]domain.TeacherFinanceRecord, int, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner); err != nil {
		return nil, 0, err
	}
	branchID := strings.TrimSpace(filter.BranchID)
	if branchID != "" {
		if err := auth.RequireBranch(actor, branchID); err != nil {
			return nil, 0, err
		}
	}
	return s.store.ListTeacherFinanceRecordsPage(ctx, branchID, normalizeFilter(filter.Subject), filter.Status, filter.SalaryModel, page)
}

func (s *Service) DeleteTeacher(ctx context.Context, actor domain.Principal, teacherID string) (domain.Teacher, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner); err != nil {
		return domain.Teacher{}, err
	}
	teacherID = strings.TrimSpace(teacherID)
	if teacherID == "" {
		return domain.Teacher{}, domain.ErrInvalidInput
	}
	teacher, err := s.store.GetTeacher(ctx, teacherID)
	if err != nil {
		return domain.Teacher{}, err
	}

	records, err := s.store.ListTeacherFinanceRecords(ctx, teacher.BranchID)
	if err != nil {
		return domain.Teacher{}, err
	}
	for _, record := range records {
		if record.ID == teacher.ID && record.AssignedStudents > 0 {
			return domain.Teacher{}, domain.ErrConflict
		}
	}

	deleted, err := s.store.DeleteTeacher(ctx, teacher.ID)
	if err != nil {
		return domain.Teacher{}, err
	}
	s.publishBestEffort(ctx, "finance.teacher.deleted", deleted)

	return deleted, nil
}

func (s *Service) CalculateSwapAllocation(ctx context.Context, actor domain.Principal, input SwapAllocationInput) (SwapAllocationResult, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleTeacher); err != nil {
		return SwapAllocationResult{}, err
	}

	cacheKey := fmt.Sprintf("salary:swap:%d:%d:%d", input.TotalAmountCents, input.FirstTeacherDays, input.SecondTeacherDays)
	if s.cache != nil {
		var cached SwapAllocationResult
		if ok, err := s.cache.Get(ctx, cacheKey, &cached); err == nil && ok {
			return cached, nil
		}
	}

	first, second := domain.CalculateSwapAllocation(input.TotalAmountCents, input.FirstTeacherDays, input.SecondTeacherDays)
	result := SwapAllocationResult{FirstTeacherAmountCents: first, SecondTeacherAmountCents: second}
	if s.cache != nil {
		_ = s.cache.Set(ctx, cacheKey, result, 24*time.Hour)
	}

	return result, nil
}

func (s *Service) publishBestEffort(ctx context.Context, topic string, payload any) {
	if s.events == nil {
		return
	}

	_ = s.events.Publish(ctx, topic, payload)
}

func normalizeFilter(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

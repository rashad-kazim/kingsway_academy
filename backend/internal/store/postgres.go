package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"kingsway/backend/internal/domain"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

type scanner interface {
	Scan(dest ...any) error
}

func scanBranch(row scanner) (domain.Branch, error) {
	var branch domain.Branch
	err := row.Scan(&branch.ID, &branch.Name, &branch.Slug, &branch.Address, &branch.OpeningTime, &branch.ClosingTime, &branch.CreatedAt, &branch.UpdatedAt)
	return branch, err
}

func scanUser(row scanner) (domain.User, error) {
	var user domain.User
	err := row.Scan(&user.ID, &user.BranchID, &user.Role, &user.Email, &user.PasswordHash, &user.FirstName, &user.LastName, &user.IsActive, &user.TokenVersion, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

func scanAuditLog(row scanner) (domain.AuditLog, error) {
	var log domain.AuditLog
	var metadata []byte
	var beforeJSON []byte
	var afterJSON []byte
	err := row.Scan(
		&log.ID,
		&log.ActorUserID,
		&log.ActorRole,
		&log.ActorBranchID,
		&log.Action,
		&log.EntityType,
		&log.EntityID,
		&log.EntityBranchID,
		&log.RequestID,
		&log.IdempotencyKey,
		&beforeJSON,
		&afterJSON,
		&metadata,
		&log.CreatedAt,
	)
	if err != nil {
		return domain.AuditLog{}, err
	}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &log.Metadata); err != nil {
			return domain.AuditLog{}, err
		}
	}
	if len(beforeJSON) > 0 {
		if err := json.Unmarshal(beforeJSON, &log.BeforeJSON); err != nil {
			return domain.AuditLog{}, err
		}
	}
	if len(afterJSON) > 0 {
		if err := json.Unmarshal(afterJSON, &log.AfterJSON); err != nil {
			return domain.AuditLog{}, err
		}
	}

	return log, nil
}

func staffMemberSelect() string {
	return `
		SELECT sp.id::text, sp.branch_id::text, sp.user_id::text, u.role, u.email,
			u.first_name, u.last_name, coalesce(to_char(sp.birth_date, 'DD/MM/YYYY'), ''),
			coalesce(sp.gender, ''), sp.phone, coalesce(sp.address, ''),
			coalesce(to_char(sp.hired_at, 'DD/MM/YYYY'), ''),
			sp.salary_amount_azn, coalesce(sp.profile_photo_file_id::text, ''),
			u.is_active, coalesce(u.last_login_at::text, ''),
			sp.created_at, sp.updated_at
		FROM staff_profiles sp
		JOIN users u ON u.id = sp.user_id
	`
}

func scanStaffMember(row scanner) (domain.StaffMember, error) {
	var staff domain.StaffMember
	err := row.Scan(
		&staff.ID,
		&staff.BranchID,
		&staff.UserID,
		&staff.Role,
		&staff.Email,
		&staff.FirstName,
		&staff.LastName,
		&staff.BirthDate,
		&staff.Gender,
		&staff.Phone,
		&staff.Address,
		&staff.HiredAt,
		&staff.SalaryAmountAZN,
		&staff.ProfilePhotoFileID,
		&staff.IsActive,
		&staff.LastLoginAt,
		&staff.CreatedAt,
		&staff.UpdatedAt,
	)
	return staff, err
}

func scanStudent(row scanner) (domain.Student, error) {
	var student domain.Student
	var fin string
	err := row.Scan(
		&student.ID,
		&student.BranchID,
		&student.UserID,
		&fin,
		&student.FirstName,
		&student.LastName,
		&student.BirthDate,
		&student.Gender,
		&student.Phone,
		&student.Address,
		&student.ProfilePhotoFileID,
		&student.Status,
		&student.LeftReason,
		&student.CreatedAt,
		&student.UpdatedAt,
	)
	if err != nil {
		return domain.Student{}, err
	}
	parsed, err := domain.ParseFIN(fin)
	if err != nil {
		return domain.Student{}, err
	}
	student.FIN = parsed

	return student, nil
}

func scanStudentAssignmentHubRecord(row scanner) (domain.StudentAssignmentHubRecord, int, error) {
	var record domain.StudentAssignmentHubRecord
	var fin string
	var total int
	err := row.Scan(
		&record.ID,
		&record.BranchID,
		&record.BranchName,
		&record.UserID,
		&fin,
		&record.FirstName,
		&record.LastName,
		&record.ProfilePhotoFileID,
		&record.Status,
		&record.ActiveTeacherID,
		&record.ActiveTeacherFirstName,
		&record.ActiveTeacherLastName,
		&record.RegisteredAt,
		&record.CreatedAt,
		&record.UpdatedAt,
		&total,
	)
	if err != nil {
		return domain.StudentAssignmentHubRecord{}, 0, err
	}
	parsed, err := domain.ParseFIN(fin)
	if err != nil {
		return domain.StudentAssignmentHubRecord{}, 0, err
	}
	record.FIN = parsed

	return record, total, nil
}

func scanTeacher(row scanner) (domain.Teacher, error) {
	var teacher domain.Teacher
	err := row.Scan(
		&teacher.ID,
		&teacher.BranchID,
		&teacher.UserID,
		&teacher.Status,
		&teacher.BirthDate,
		&teacher.Gender,
		&teacher.Phone,
		&teacher.Address,
		&teacher.ProfilePhotoFileID,
		&teacher.CreatedAt,
		&teacher.UpdatedAt,
	)
	return teacher, err
}

func scanTeacherFinanceRecord(row scanner) (domain.TeacherFinanceRecord, error) {
	var record domain.TeacherFinanceRecord
	var salaryType string
	err := row.Scan(
		&record.ID,
		&record.BranchID,
		&record.BranchName,
		&record.UserID,
		&record.Email,
		&record.FirstName,
		&record.LastName,
		&record.ProfilePhotoFileID,
		&record.Subject,
		&record.Status,
		&salaryType,
		&record.CalculatedSalaryAmountCents,
		&record.AssignedStudents,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return domain.TeacherFinanceRecord{}, err
	}
	record.SalaryType = domain.SalaryModelType(salaryType)

	return record, nil
}

func scanCourse(row scanner) (domain.Course, error) {
	var course domain.Course
	err := row.Scan(&course.ID, &course.BranchID, &course.Name, &course.IsActive, &course.CreatedAt, &course.UpdatedAt)
	return course, err
}

func scanScoreCategory(row scanner) (domain.ScoreCategory, error) {
	var category domain.ScoreCategory
	err := row.Scan(
		&category.ID,
		&category.BranchID,
		&category.CourseID,
		&category.Name,
		&category.MinScore,
		&category.MaxScore,
		&category.RequiresFeedback,
		&category.RequiresDocument,
		&category.DocumentPurpose,
		&category.SortOrder,
		&category.CreatedAt,
	)
	return category, err
}

func scanRoom(row scanner) (domain.Room, error) {
	var room domain.Room
	err := row.Scan(&room.ID, &room.BranchID, &room.Name, &room.Capacity, &room.IsActive, &room.CreatedAt, &room.UpdatedAt)
	return room, err
}

func scanScheduleItem(row scanner) (domain.ScheduleItem, error) {
	var item domain.ScheduleItem
	var cancelledAt sql.NullTime
	err := row.Scan(
		&item.ID,
		&item.BranchID,
		&item.ClassID,
		&item.TeacherID,
		&item.RoomID,
		&item.ItemType,
		&item.Title,
		&item.StartsAt,
		&item.EndsAt,
		&item.CreatedByUserID,
		&cancelledAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return domain.ScheduleItem{}, err
	}
	if cancelledAt.Valid {
		item.CancelledAt = &cancelledAt.Time
	}

	return item, nil
}

func scanPayment(row scanner) (domain.Payment, error) {
	var payment domain.Payment
	var paidAt sql.NullTime
	err := row.Scan(
		&payment.ID,
		&payment.BranchID,
		&payment.StudentID,
		&payment.AmountCents,
		&payment.Currency,
		&payment.DueDate,
		&paidAt,
		&payment.Status,
		&payment.ReceiptFileID,
		&payment.CreatedByUserID,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		return domain.Payment{}, err
	}
	if paidAt.Valid {
		payment.PaidAt = &paidAt.Time
	}

	return payment, nil
}

func scanSalaryModel(row scanner) (domain.SalaryModel, error) {
	var model domain.SalaryModel
	var activeTo sql.NullTime
	err := row.Scan(
		&model.ID,
		&model.BranchID,
		&model.TeacherID,
		&model.ModelType,
		&model.FixedMonthlyAmountCents,
		&model.StudentPercentBasisPoints,
		&model.ActiveFrom,
		&activeTo,
		&model.ApprovedByOwnerUserID,
		&model.CreatedAt,
	)
	if err != nil {
		return domain.SalaryModel{}, err
	}
	if activeTo.Valid {
		model.ActiveTo = &activeTo.Time
	}

	return model, nil
}

func scanFileObject(row scanner) (domain.FileObject, error) {
	var file domain.FileObject
	var retentionUntil sql.NullTime
	var deletedAt sql.NullTime
	err := row.Scan(
		&file.ID,
		&file.BranchID,
		&file.UploaderUserID,
		&file.OwnerType,
		&file.OwnerID,
		&file.Category,
		&file.Policy,
		&file.Purpose,
		&file.OriginalFilename,
		&file.MimeType,
		&file.OriginalSizeBytes,
		&file.StoredSizeBytes,
		&file.OriginalSHA256,
		&file.StorageBucket,
		&file.StorageKey,
		&retentionUntil,
		&deletedAt,
		&file.CreatedAt,
	)
	if err != nil {
		return domain.FileObject{}, err
	}
	if retentionUntil.Valid {
		file.RetentionUntil = &retentionUntil.Time
	}
	if deletedAt.Valid {
		file.DeletedAt = &deletedAt.Time
	}

	return file, nil
}

func scanIdempotencyRecord(row scanner) (domain.IdempotencyRecord, error) {
	var record domain.IdempotencyRecord
	var responseBody string
	err := row.Scan(
		&record.ActorUserID,
		&record.Method,
		&record.Path,
		&record.Key,
		&record.RequestHash,
		&record.Status,
		&record.ResponseStatus,
		&responseBody,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.ExpiresAt,
	)
	if err != nil {
		return domain.IdempotencyRecord{}, err
	}
	if responseBody != "" {
		record.ResponseBody = []byte(responseBody)
	}
	record.RequestHash = strings.TrimSpace(record.RequestHash)

	return record, nil
}

func fixedSalaryValue(model domain.SalaryModel) any {
	if model.ModelType != domain.SalaryModelFixed && model.ModelType != domain.SalaryModelHybrid {
		return nil
	}

	return model.FixedMonthlyAmountCents
}

func percentSalaryValue(model domain.SalaryModel) any {
	if model.ModelType != domain.SalaryModelPercent && model.ModelType != domain.SalaryModelHybrid {
		return nil
	}

	return model.StudentPercentBasisPoints
}

func mapPostgresError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505", "23P01":
			return domain.ErrConflict
		case "23503":
			return domain.ErrNotFound
		case "23514", "22P02":
			return domain.ErrInvalidInput
		}
	}

	return err
}

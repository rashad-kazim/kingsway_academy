package integration_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"kingsway/backend/internal/domain"
	"kingsway/backend/internal/platform/config"
	"kingsway/backend/internal/platform/database"
	"kingsway/backend/internal/store"
)

func TestPostgresStoreCoreWorkflow(t *testing.T) {
	if os.Getenv("KINGSWAY_INTEGRATION") != "1" {
		t.Skip("set KINGSWAY_INTEGRATION=1 to run Docker-backed PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}

	pool, err := database.Connect(ctx, database.DSN(cfg.Postgres()))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	if err := database.ApplyMigrations(ctx, pool, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}

	repo := store.NewPostgres(pool)
	stamp := time.Now().UTC().UnixNano()

	branch, err := repo.CreateBranch(ctx, domain.Branch{
		Name: fmt.Sprintf("Integration %d", stamp),
		Slug: fmt.Sprintf("integration-%d", stamp),
	})
	if err != nil {
		t.Fatal(err)
	}

	owner, err := repo.CreateUser(ctx, domain.User{
		Role:         domain.RoleOwner,
		Email:        fmt.Sprintf("owner-%d@integration.local", stamp),
		PasswordHash: "hash",
		FirstName:    "Integration",
		LastName:     "Owner",
	})
	if err != nil {
		t.Fatal(err)
	}

	teacherUser, err := repo.CreateUser(ctx, domain.User{
		BranchID:     branch.ID,
		Role:         domain.RoleTeacher,
		Email:        fmt.Sprintf("teacher-%d@integration.local", stamp),
		PasswordHash: "hash",
		FirstName:    "Integration",
		LastName:     "Teacher",
	})
	if err != nil {
		t.Fatal(err)
	}

	teacher, err := repo.CreateTeacher(ctx, domain.Teacher{
		BranchID: branch.ID,
		UserID:   teacherUser.ID,
		Status:   domain.TeacherStatusPending,
	})
	if err != nil {
		t.Fatal(err)
	}
	teacher.Status = domain.TeacherStatusActive
	teacher, err = repo.UpdateTeacher(ctx, teacher)
	if err != nil {
		t.Fatal(err)
	}

	room, err := repo.CreateRoom(ctx, domain.Room{
		BranchID: branch.ID,
		Name:     fmt.Sprintf("Room %d", stamp),
		Capacity: 10,
	})
	if err != nil {
		t.Fatal(err)
	}

	student, err := repo.CreateStudent(ctx, domain.Student{
		BranchID:  branch.ID,
		FIN:       integrationFIN(t, stamp),
		FirstName: "Integration",
		LastName:  "Student",
	})
	if err != nil {
		t.Fatal(err)
	}

	course, categories, err := repo.CreateCourse(ctx, domain.Course{
		BranchID: branch.ID,
		Name:     fmt.Sprintf("IELTS %d", stamp),
	}, []domain.ScoreCategory{
		{Name: "Writing", MinScore: 0, MaxScore: 9, RequiresFeedback: true, RequiresDocument: true, DocumentPurpose: string(domain.FilePurposeExamWriting)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(categories) != 1 {
		t.Fatalf("expected one score category, got %d", len(categories))
	}

	class, err := repo.CreateClass(ctx, domain.Class{
		BranchID:  branch.ID,
		CourseID:  course.ID,
		TeacherID: teacher.ID,
		Name:      fmt.Sprintf("IELTS Morning %d", stamp),
		StartDate: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.EnrollStudent(ctx, domain.ClassStudent{
		BranchID:  branch.ID,
		ClassID:   class.ID,
		StudentID: student.ID,
		JoinedAt:  time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateAssignment(ctx, domain.Assignment{
		BranchID:        branch.ID,
		ClassID:         class.ID,
		Title:           "Writing task 1",
		Description:     "Academic writing practice",
		CreatedByUserID: owner.ID,
	}); err != nil {
		t.Fatal(err)
	}

	startsAt := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	schedule, err := repo.CreateScheduleItem(ctx, domain.ScheduleItem{
		BranchID:        branch.ID,
		ClassID:         class.ID,
		TeacherID:       teacher.ID,
		RoomID:          room.ID,
		ItemType:        domain.ScheduleItemLesson,
		Title:           "Integration lesson",
		StartsAt:        startsAt,
		EndsAt:          startsAt.Add(time.Hour),
		CreatedByUserID: owner.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if schedule.ID == "" {
		t.Fatal("expected schedule id")
	}

	_, err = repo.CreateScheduleItem(ctx, domain.ScheduleItem{
		BranchID:        branch.ID,
		TeacherID:       teacher.ID,
		RoomID:          room.ID,
		ItemType:        domain.ScheduleItemLesson,
		Title:           "Conflicting lesson",
		StartsAt:        startsAt.Add(15 * time.Minute),
		EndsAt:          startsAt.Add(45 * time.Minute),
		CreatedByUserID: owner.ID,
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected schedule conflict, got %v", err)
	}

	exam, participants, err := repo.CreateExam(ctx, domain.Exam{
		BranchID:        branch.ID,
		CourseID:        course.ID,
		ClassID:         class.ID,
		ScheduleItemID:  schedule.ID,
		Title:           "Mock exam",
		CreatedByUserID: owner.ID,
	}, []string{student.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(participants) != 1 {
		t.Fatalf("expected one exam participant, got %d", len(participants))
	}
	if _, err := repo.CreateExamResult(ctx, domain.ExamResult{
		BranchID:           branch.ID,
		ExamID:             exam.ID,
		StudentID:          student.ID,
		CategoryID:         categories[0].ID,
		Score:              7.5,
		Feedback:           "Strong structure",
		EnteredByTeacherID: teacher.ID,
	}); err != nil {
		t.Fatal(err)
	}
	dashboard, err := repo.AcademicDashboard(ctx, branch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if dashboard.ActiveClasses == 0 || dashboard.ActiveStudents == 0 || dashboard.ActiveTeachers == 0 {
		t.Fatalf("expected dashboard counts to include new records: %#v", dashboard)
	}

	payment, err := repo.CreatePayment(ctx, domain.Payment{
		BranchID:        branch.ID,
		StudentID:       student.ID,
		AmountCents:     10000,
		Currency:        "AZN",
		DueDate:         time.Now().UTC().AddDate(0, 0, 7),
		Status:          domain.PaymentStatusPending,
		CreatedByUserID: owner.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertOutboxEvent(t, ctx, pool, "finance.payment.created", payment.ID)

	expiredAt := time.Now().UTC().Add(-time.Hour)
	file, err := repo.RegisterFile(ctx, domain.FileObject{
		BranchID:          branch.ID,
		UploaderUserID:    owner.ID,
		OwnerType:         "student",
		OwnerID:           student.ID,
		Category:          domain.FileCategoryStandard,
		Purpose:           domain.FilePurposeMaterial,
		OriginalFilename:  "integration.txt",
		MimeType:          "text/plain",
		OriginalSizeBytes: 10,
		StoredSizeBytes:   10,
		OriginalSHA256:    strings.Repeat("a", 64),
		StorageBucket:     "integration",
		StorageKey:        fmt.Sprintf("integration/%d.txt", stamp),
		RetentionUntil:    &expiredAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertOutboxEvent(t, ctx, pool, "files.file.registered", file.ID)

	expired, err := repo.ListExpiredFiles(ctx, time.Now().UTC(), 20)
	if err != nil {
		t.Fatal(err)
	}
	if !containsFile(expired, file.ID) {
		t.Fatal("expected registered expired file to be returned")
	}

	deleted, err := repo.MarkFileDeleted(ctx, file.ID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if deleted.DeletedAt == nil {
		t.Fatal("expected deleted_at to be set")
	}
	assertOutboxEvent(t, ctx, pool, "files.retention.deleted", file.ID)
}

func integrationFIN(t *testing.T, stamp int64) domain.FIN {
	t.Helper()

	fin, err := domain.ParseFIN(fmt.Sprintf("T%06d", stamp%1000000))
	if err != nil {
		t.Fatal(err)
	}

	return fin
}

func containsFile(files []domain.FileObject, id string) bool {
	for _, file := range files {
		if file.ID == id {
			return true
		}
	}

	return false
}

func assertOutboxEvent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, topic string, id string) {
	t.Helper()

	var count int
	err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM outbox_events
		WHERE topic = $1
			AND payload->>'id' = $2
	`, topic, id).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatalf("expected outbox event %s for id %s", topic, id)
	}
}

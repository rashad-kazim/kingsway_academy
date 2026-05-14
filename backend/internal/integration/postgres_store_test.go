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

func TestPostgresStudentRegistrationWritesColumnsAndRollsBack(t *testing.T) {
	if os.Getenv("KINGSWAY_INTEGRATION") != "1" {
		t.Skip("set KINGSWAY_INTEGRATION=1 to run PostgreSQL integration tests")
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
		Name:    fmt.Sprintf("Student Integration %d", stamp),
		Slug:    fmt.Sprintf("student-integration-%d", stamp),
		Address: "Student Integration Address",
	})
	if err != nil {
		t.Fatal(err)
	}

	teacherUser, err := repo.CreateUser(ctx, domain.User{
		BranchID:     branch.ID,
		Role:         domain.RoleTeacher,
		Email:        fmt.Sprintf("student-teacher-%d@integration.local", stamp),
		PasswordHash: "hash",
		FirstName:    "Column",
		LastName:     "Teacher",
	})
	if err != nil {
		t.Fatal(err)
	}
	teacher, err := repo.CreateTeacher(ctx, domain.Teacher{
		BranchID: branch.ID,
		UserID:   teacherUser.ID,
		Status:   domain.TeacherStatusActive,
	})
	if err != nil {
		t.Fatal(err)
	}

	course, _, err := repo.CreateCourse(ctx, domain.Course{
		BranchID: branch.ID,
		Name:     fmt.Sprintf("SAT Columns %d", stamp),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	fin := integrationFINWithPrefix(t, "S", stamp)
	student, err := repo.CreateStudentWithDetails(
		ctx,
		domain.Student{
			BranchID:  branch.ID,
			FIN:       fin,
			FirstName: "Leyla",
			LastName:  "Aliyeva",
			BirthDate: "15/07/1998",
			Gender:    "female",
			Phone:     "+994501112233",
			Address:   "Gence, Main Street",
			Status:    domain.StudentStatusActive,
		},
		[]domain.StudentParentContact{
			{
				BranchID: branch.ID,
				Relation: "mother",
				Name:     "Aynur Aliyeva",
				Phones:   []string{"+994502223344", "+994503334455"},
			},
		},
		[]domain.StudentCourseRegistration{
			{
				BranchID:           branch.ID,
				CourseID:           course.ID,
				TeacherID:          teacher.ID,
				MonthlyAmountCents: 25000,
				StartDate:          "07/05/2026",
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	var gotFIN, firstName, lastName, birthDate, gender, phone, address, status string
	err = pool.QueryRow(ctx, `
		SELECT fin_code, first_name, last_name, coalesce(to_char(birth_date, 'DD/MM/YYYY'), ''),
			coalesce(gender, ''), phone, address, status
		FROM students
		WHERE id = $1
	`, student.ID).Scan(&gotFIN, &firstName, &lastName, &birthDate, &gender, &phone, &address, &status)
	if err != nil {
		t.Fatal(err)
	}
	if gotFIN != fin.String() || firstName != "Leyla" || lastName != "Aliyeva" || birthDate != "15/07/1998" ||
		gender != "female" || phone != "+994501112233" || address != "Gence, Main Street" || status != string(domain.StudentStatusActive) {
		t.Fatalf("student DB columns mismatch: fin=%q first=%q last=%q birth=%q gender=%q phone=%q address=%q status=%q",
			gotFIN, firstName, lastName, birthDate, gender, phone, address, status)
	}

	var relation, parentName, phones string
	err = pool.QueryRow(ctx, `
		SELECT relation, name, array_to_string(phones, ',')
		FROM student_parent_contacts
		WHERE student_id = $1
	`, student.ID).Scan(&relation, &parentName, &phones)
	if err != nil {
		t.Fatal(err)
	}
	if relation != "mother" || parentName != "Aynur Aliyeva" || phones != "+994502223344,+994503334455" {
		t.Fatalf("parent contact mismatch: relation=%q name=%q phones=%q", relation, parentName, phones)
	}

	var gotCourseID, gotTeacherID, startDate string
	var monthlyAmount int64
	err = pool.QueryRow(ctx, `
		SELECT course_id::text, coalesce(teacher_id::text, ''), monthly_amount_cents,
			to_char(start_date, 'DD/MM/YYYY')
		FROM student_course_registrations
		WHERE student_id = $1
	`, student.ID).Scan(&gotCourseID, &gotTeacherID, &monthlyAmount, &startDate)
	if err != nil {
		t.Fatal(err)
	}
	if gotCourseID != course.ID || gotTeacherID != teacher.ID || monthlyAmount != 25000 || startDate != "07/05/2026" {
		t.Fatalf("course registration mismatch: course=%q teacher=%q monthly=%d start=%q",
			gotCourseID, gotTeacherID, monthlyAmount, startDate)
	}

	var assignmentCount int
	err = pool.QueryRow(ctx, `
		SELECT count(*)
		FROM student_teacher_assignments
		WHERE student_id = $1
			AND teacher_id = $2
			AND valid_from = to_date('07/05/2026', 'DD/MM/YYYY')
			AND reason = 'initial'
	`, student.ID, teacher.ID).Scan(&assignmentCount)
	if err != nil {
		t.Fatal(err)
	}
	if assignmentCount != 1 {
		t.Fatalf("expected one active teacher assignment, got %d", assignmentCount)
	}

	rollbackFIN := integrationFINWithPrefix(t, "R", stamp+1)
	_, err = repo.CreateStudentWithDetails(
		ctx,
		domain.Student{
			BranchID:  branch.ID,
			FIN:       rollbackFIN,
			FirstName: "Rollback",
			LastName:  "Student",
			Status:    domain.StudentStatusActive,
		},
		nil,
		[]domain.StudentCourseRegistration{
			{
				BranchID:           branch.ID,
				CourseID:           "00000000-0000-0000-0000-000000000000",
				MonthlyAmountCents: 10000,
				StartDate:          "07/05/2026",
			},
		},
	)
	if err == nil {
		t.Fatal("expected invalid course registration to fail")
	}

	var rollbackCount int
	err = pool.QueryRow(ctx, `
		SELECT count(*)
		FROM students
		WHERE fin_code = $1
	`, rollbackFIN.String()).Scan(&rollbackCount)
	if err != nil {
		t.Fatal(err)
	}
	if rollbackCount != 0 {
		t.Fatalf("expected failed student registration to roll back, got %d rows", rollbackCount)
	}
}

func TestPostgresBranchRoomTeacherAndReceptionistWritesColumns(t *testing.T) {
	if os.Getenv("KINGSWAY_INTEGRATION") != "1" {
		t.Skip("set KINGSWAY_INTEGRATION=1 to run PostgreSQL integration tests")
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

	owner, err := repo.CreateUser(ctx, domain.User{
		Role:         domain.RoleOwner,
		Email:        fmt.Sprintf("module-owner-%d@integration.local", stamp),
		PasswordHash: "hash",
		FirstName:    "Module",
		LastName:     "Owner",
	})
	if err != nil {
		t.Fatal(err)
	}

	branch, err := repo.CreateBranch(ctx, domain.Branch{
		Name:    fmt.Sprintf("Module Branch %d", stamp),
		Slug:    fmt.Sprintf("module-branch-%d", stamp),
		Address: "Old Address",
	})
	if err != nil {
		t.Fatal(err)
	}
	branch.Name = fmt.Sprintf("Module Branch Updated %d", stamp)
	branch.Slug = fmt.Sprintf("module-branch-updated-%d", stamp)
	branch.Address = "Updated Branch Address"
	branch.OpeningTime = "09:00"
	branch.ClosingTime = "18:30"
	branch, err = repo.UpdateBranch(ctx, branch)
	if err != nil {
		t.Fatal(err)
	}

	var branchName, branchSlug, branchAddress, openingTime, closingTime string
	err = pool.QueryRow(ctx, `
		SELECT name, slug, coalesce(address, ''), opening_time, closing_time
		FROM branches
		WHERE id = $1
	`, branch.ID).Scan(&branchName, &branchSlug, &branchAddress, &openingTime, &closingTime)
	if err != nil {
		t.Fatal(err)
	}
	if branchName != branch.Name || branchSlug != branch.Slug || branchAddress != "Updated Branch Address" ||
		openingTime != "09:00" || closingTime != "18:30" {
		t.Fatalf("branch columns mismatch: name=%q slug=%q address=%q opening=%q closing=%q",
			branchName, branchSlug, branchAddress, openingTime, closingTime)
	}

	room, err := repo.CreateRoom(ctx, domain.Room{
		BranchID: branch.ID,
		Name:     fmt.Sprintf("Module Room %d", stamp),
		Capacity: 12,
	})
	if err != nil {
		t.Fatal(err)
	}
	room.Name = fmt.Sprintf("Module Room Updated %d", stamp)
	room.Capacity = 18
	room, err = repo.UpdateRoom(ctx, room)
	if err != nil {
		t.Fatal(err)
	}
	room, err = repo.DeactivateRoom(ctx, room.ID)
	if err != nil {
		t.Fatal(err)
	}

	var roomName string
	var roomCapacity int
	var roomActive bool
	err = pool.QueryRow(ctx, `
		SELECT name, capacity, is_active
		FROM rooms
		WHERE id = $1
	`, room.ID).Scan(&roomName, &roomCapacity, &roomActive)
	if err != nil {
		t.Fatal(err)
	}
	if roomName != room.Name || roomCapacity != 18 || roomActive {
		t.Fatalf("room columns mismatch: name=%q capacity=%d active=%v", roomName, roomCapacity, roomActive)
	}

	course, _, err := repo.CreateCourse(ctx, domain.Course{
		BranchID: branch.ID,
		Name:     fmt.Sprintf("IELTS Module %d", stamp),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	teacherUser, err := repo.CreateUser(ctx, domain.User{
		BranchID:     branch.ID,
		Role:         domain.RoleTeacher,
		Email:        fmt.Sprintf("module-teacher-%d@integration.local", stamp),
		PasswordHash: "hash",
		FirstName:    "Nigar",
		LastName:     "Teacher",
		IsActive:     true,
	})
	if err != nil {
		t.Fatal(err)
	}
	teacher, err := repo.CreateTeacher(ctx, domain.Teacher{
		BranchID:  branch.ID,
		UserID:    teacherUser.ID,
		Status:    domain.TeacherStatusActive,
		BirthDate: "10/10/1990",
		Gender:    "female",
		Phone:     "+994551112233",
		Address:   "Teacher Address",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.ReplaceTeacherCourseSpecializations(ctx, teacher.ID, []string{course.ID}); err != nil {
		t.Fatal(err)
	}
	salary, err := repo.CreateSalaryModel(ctx, domain.SalaryModel{
		BranchID:                  branch.ID,
		TeacherID:                 teacher.ID,
		ModelType:                 domain.SalaryModelHybrid,
		FixedMonthlyAmountCents:   90000,
		StudentPercentBasisPoints: 2500,
		ActiveFrom:                time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC),
		ApprovedByOwnerUserID:     owner.ID,
	})
	if err != nil {
		t.Fatal(err)
	}

	var teacherEmail, teacherFirstName, teacherLastName, teacherStatus, teacherBirthDate, teacherGender, teacherPhone, teacherAddress string
	err = pool.QueryRow(ctx, `
		SELECT u.email, u.first_name, u.last_name, t.status,
			coalesce(to_char(t.birth_date, 'DD/MM/YYYY'), ''), coalesce(t.gender, ''),
			t.phone, t.address
		FROM teachers t
		JOIN users u ON u.id = t.user_id
		WHERE t.id = $1
	`, teacher.ID).Scan(&teacherEmail, &teacherFirstName, &teacherLastName, &teacherStatus, &teacherBirthDate, &teacherGender, &teacherPhone, &teacherAddress)
	if err != nil {
		t.Fatal(err)
	}
	if teacherEmail != teacherUser.Email || teacherFirstName != "Nigar" || teacherLastName != "Teacher" ||
		teacherStatus != string(domain.TeacherStatusActive) || teacherBirthDate != "10/10/1990" ||
		teacherGender != "female" || teacherPhone != "+994551112233" || teacherAddress != "Teacher Address" {
		t.Fatalf("teacher columns mismatch: email=%q first=%q last=%q status=%q birth=%q gender=%q phone=%q address=%q",
			teacherEmail, teacherFirstName, teacherLastName, teacherStatus, teacherBirthDate, teacherGender, teacherPhone, teacherAddress)
	}

	var salaryType string
	var fixedCents int64
	var percentBasisPoints int
	err = pool.QueryRow(ctx, `
		SELECT model_type, coalesce(fixed_monthly_amount_cents, 0), coalesce(student_percent_basis_points, 0)
		FROM salary_models
		WHERE id = $1
	`, salary.ID).Scan(&salaryType, &fixedCents, &percentBasisPoints)
	if err != nil {
		t.Fatal(err)
	}
	if salaryType != string(domain.SalaryModelHybrid) || fixedCents != 90000 || percentBasisPoints != 2500 {
		t.Fatalf("salary columns mismatch: type=%q fixed=%d percent=%d", salaryType, fixedCents, percentBasisPoints)
	}

	var specializationCount int
	err = pool.QueryRow(ctx, `
		SELECT count(*)
		FROM teacher_course_specializations
		WHERE teacher_id = $1 AND course_id = $2
	`, teacher.ID, course.ID).Scan(&specializationCount)
	if err != nil {
		t.Fatal(err)
	}
	if specializationCount != 1 {
		t.Fatalf("expected one teacher specialization, got %d", specializationCount)
	}

	staff, err := repo.CreateStaffMember(ctx, domain.User{
		BranchID:     branch.ID,
		Role:         domain.RoleReceptionist,
		Email:        fmt.Sprintf("module-receptionist-%d@integration.local", stamp),
		PasswordHash: "hash",
		FirstName:    "Rana",
		LastName:     "Reception",
		IsActive:     true,
	}, domain.StaffMember{
		BirthDate:       "20/08/1995",
		Gender:          "female",
		Phone:           "+994552223344",
		Address:         "Reception Address",
		HiredAt:         "01/05/2026",
		SalaryAmountAZN: 700,
	})
	if err != nil {
		t.Fatal(err)
	}
	staff.FirstName = "Rana Updated"
	staff.LastName = "Reception Updated"
	staff.Email = fmt.Sprintf("module-receptionist-updated-%d@integration.local", stamp)
	staff.Phone = "+994553334455"
	staff.Address = "Reception Updated Address"
	staff.SalaryAmountAZN = 800
	staff.IsActive = false
	staff, err = repo.UpdateStaffMember(ctx, staff, "")
	if err != nil {
		t.Fatal(err)
	}

	var staffEmail, staffFirstName, staffLastName, staffBirthDate, staffGender, staffPhone, staffAddress, staffHiredAt string
	var staffSalary int
	var staffActive bool
	err = pool.QueryRow(ctx, `
		SELECT u.email, u.first_name, u.last_name, u.is_active,
			coalesce(to_char(sp.birth_date, 'DD/MM/YYYY'), ''), sp.gender, sp.phone,
			sp.address, coalesce(to_char(sp.hired_at, 'DD/MM/YYYY'), ''), sp.salary_amount_azn
		FROM staff_profiles sp
		JOIN users u ON u.id = sp.user_id
		WHERE sp.id = $1
	`, staff.ID).Scan(&staffEmail, &staffFirstName, &staffLastName, &staffActive, &staffBirthDate, &staffGender, &staffPhone, &staffAddress, &staffHiredAt, &staffSalary)
	if err != nil {
		t.Fatal(err)
	}
	if staffEmail != staff.Email || staffFirstName != "Rana Updated" || staffLastName != "Reception Updated" ||
		staffActive || staffBirthDate != "20/08/1995" || staffGender != "female" ||
		staffPhone != "+994553334455" || staffAddress != "Reception Updated Address" ||
		staffHiredAt != "01/05/2026" || staffSalary != 800 {
		t.Fatalf("staff columns mismatch: email=%q first=%q last=%q active=%v birth=%q gender=%q phone=%q address=%q hired=%q salary=%d",
			staffEmail, staffFirstName, staffLastName, staffActive, staffBirthDate, staffGender, staffPhone, staffAddress, staffHiredAt, staffSalary)
	}

	if _, err := repo.DeleteStaffMember(ctx, staff.ID); err != nil {
		t.Fatal(err)
	}
	assertZeroRows(t, ctx, pool, "staff_profiles", "id", staff.ID)
	assertZeroRows(t, ctx, pool, "users", "id", staff.UserID)

	if _, err := repo.DeleteTeacher(ctx, teacher.ID); err != nil {
		t.Fatal(err)
	}
	assertZeroRows(t, ctx, pool, "teachers", "id", teacher.ID)
	assertZeroRows(t, ctx, pool, "users", "id", teacher.UserID)
	assertZeroRows(t, ctx, pool, "salary_models", "teacher_id", teacher.ID)
	assertZeroRows(t, ctx, pool, "teacher_course_specializations", "teacher_id", teacher.ID)

	if _, err := repo.DeleteBranch(ctx, branch.ID); err != nil {
		t.Fatal(err)
	}
	assertZeroRows(t, ctx, pool, "branches", "id", branch.ID)
	assertZeroRows(t, ctx, pool, "rooms", "id", room.ID)
	assertZeroRows(t, ctx, pool, "courses", "id", course.ID)
}

func integrationFIN(t *testing.T, stamp int64) domain.FIN {
	t.Helper()

	fin, err := domain.ParseFIN(fmt.Sprintf("T%06d", stamp%1000000))
	if err != nil {
		t.Fatal(err)
	}

	return fin
}

func integrationFINWithPrefix(t *testing.T, prefix string, stamp int64) domain.FIN {
	t.Helper()

	fin, err := domain.ParseFIN(fmt.Sprintf("%s%06d", prefix, stamp%1000000))
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

func assertZeroRows(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table string, idColumn string, id string) {
	t.Helper()

	allowed := map[string]map[string]bool{
		"branches":                       {"id": true},
		"courses":                        {"id": true},
		"rooms":                          {"id": true},
		"salary_models":                  {"teacher_id": true},
		"staff_profiles":                 {"id": true},
		"teacher_course_specializations": {"teacher_id": true},
		"teachers":                       {"id": true},
		"users":                          {"id": true},
	}
	if !allowed[table][idColumn] {
		t.Fatalf("unsafe assertZeroRows target %s.%s", table, idColumn)
	}

	var count int
	err := pool.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FROM %s WHERE %s = $1", table, idColumn), id).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected zero rows in %s where %s=%s, got %d", table, idColumn, id, count)
	}
}

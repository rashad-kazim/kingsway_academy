package academic_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"kingsway/backend/internal/academic"
	"kingsway/backend/internal/domain"
	"kingsway/backend/internal/store"
)

func TestRoleScopedAcademicLists(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := store.NewMemory()
	service := academic.NewService(repo, nil)
	data := seedAcademicAccessData(t, ctx, repo)

	teacherClasses, err := service.ListClasses(ctx, data.teacherPrincipal, data.branch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(teacherClasses) != 1 || teacherClasses[0].ID != data.teacherClass.ID {
		t.Fatalf("teacher should only see own class, got %#v", teacherClasses)
	}

	studentAssignments, err := service.ListAssignments(ctx, data.studentPrincipal, data.branch.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(studentAssignments) != 1 || studentAssignments[0].ClassID != data.teacherClass.ID {
		t.Fatalf("student should only see enrolled class assignments, got %#v", studentAssignments)
	}

	studentResults, err := service.ListExamResults(ctx, data.studentPrincipal, data.branch.ID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(studentResults) != 1 || studentResults[0].StudentID != data.student.ID {
		t.Fatalf("student should only see own exam results, got %#v", studentResults)
	}
}

func TestBranchIsolationAndTeacherResultOwnership(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := store.NewMemory()
	service := academic.NewService(repo, nil)
	data := seedAcademicAccessData(t, ctx, repo)

	_, err := service.GetClass(ctx, data.otherBranchReceptionist, data.teacherClass.ID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected branch isolation forbidden error, got %v", err)
	}

	_, err = service.CreateExamResult(ctx, data.otherTeacherPrincipal, academic.CreateExamResultInput{
		BranchID:   data.branch.ID,
		ExamID:     data.exam.ID,
		StudentID:  data.student.ID,
		CategoryID: data.category.ID,
		Score:      6,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected teacher result ownership forbidden error, got %v", err)
	}
}

type academicAccessData struct {
	branch                  domain.Branch
	teacherPrincipal        domain.Principal
	otherTeacherPrincipal   domain.Principal
	studentPrincipal        domain.Principal
	otherBranchReceptionist domain.Principal
	teacherClass            domain.Class
	student                 domain.Student
	exam                    domain.Exam
	category                domain.ScoreCategory
}

func seedAcademicAccessData(t *testing.T, ctx context.Context, repo *store.Memory) academicAccessData {
	t.Helper()

	branch, err := repo.CreateBranch(ctx, domain.Branch{Name: "Main", Slug: "main"})
	if err != nil {
		t.Fatal(err)
	}
	otherBranch, err := repo.CreateBranch(ctx, domain.Branch{Name: "Other", Slug: "other"})
	if err != nil {
		t.Fatal(err)
	}
	owner, err := repo.CreateUser(ctx, domain.User{Role: domain.RoleOwner, Email: "owner@test.local", PasswordHash: "hash", FirstName: "Owner", LastName: "User"})
	if err != nil {
		t.Fatal(err)
	}
	teacherUser, teacher := createTeacher(t, ctx, repo, branch, "teacher@test.local")
	otherTeacherUser, otherTeacher := createTeacher(t, ctx, repo, branch, "teacher2@test.local")
	studentUser, err := repo.CreateUser(ctx, domain.User{BranchID: branch.ID, Role: domain.RoleStudent, Email: "student@test.local", PasswordHash: "hash", FirstName: "Student", LastName: "User"})
	if err != nil {
		t.Fatal(err)
	}
	fin, err := domain.ParseFIN("ABC1234")
	if err != nil {
		t.Fatal(err)
	}
	student, err := repo.CreateStudent(ctx, domain.Student{BranchID: branch.ID, UserID: studentUser.ID, FIN: fin, FirstName: "Student", LastName: "One"})
	if err != nil {
		t.Fatal(err)
	}
	course, categories, err := repo.CreateCourse(ctx, domain.Course{BranchID: branch.ID, Name: "IELTS"}, []domain.ScoreCategory{
		{Name: "Writing", MinScore: 0, MaxScore: 9},
	})
	if err != nil {
		t.Fatal(err)
	}
	class, err := repo.CreateClass(ctx, domain.Class{BranchID: branch.ID, CourseID: course.ID, TeacherID: teacher.ID, Name: "Teacher Class", StartDate: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	otherClass, err := repo.CreateClass(ctx, domain.Class{BranchID: branch.ID, CourseID: course.ID, TeacherID: otherTeacher.ID, Name: "Other Class", StartDate: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.EnrollStudent(ctx, domain.ClassStudent{BranchID: branch.ID, ClassID: class.ID, StudentID: student.ID, JoinedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateAssignment(ctx, domain.Assignment{BranchID: branch.ID, ClassID: class.ID, Title: "Visible", CreatedByUserID: owner.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateAssignment(ctx, domain.Assignment{BranchID: branch.ID, ClassID: otherClass.ID, Title: "Hidden", CreatedByUserID: owner.ID}); err != nil {
		t.Fatal(err)
	}
	exam, _, err := repo.CreateExam(ctx, domain.Exam{BranchID: branch.ID, CourseID: course.ID, ClassID: class.ID, Title: "Mock", CreatedByUserID: owner.ID}, []string{student.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateExamResult(ctx, domain.ExamResult{
		BranchID:           branch.ID,
		ExamID:             exam.ID,
		StudentID:          student.ID,
		CategoryID:         categories[0].ID,
		Score:              7,
		EnteredByTeacherID: teacher.ID,
	}); err != nil {
		t.Fatal(err)
	}

	return academicAccessData{
		branch:                  branch,
		teacherPrincipal:        domain.Principal{UserID: teacherUser.ID, BranchID: branch.ID, Role: domain.RoleTeacher},
		otherTeacherPrincipal:   domain.Principal{UserID: otherTeacherUser.ID, BranchID: branch.ID, Role: domain.RoleTeacher},
		studentPrincipal:        domain.Principal{UserID: studentUser.ID, BranchID: branch.ID, Role: domain.RoleStudent},
		otherBranchReceptionist: domain.Principal{UserID: "receptionist", BranchID: otherBranch.ID, Role: domain.RoleReceptionist},
		teacherClass:            class,
		student:                 student,
		exam:                    exam,
		category:                categories[0],
	}
}

func createTeacher(t *testing.T, ctx context.Context, repo *store.Memory, branch domain.Branch, email string) (domain.User, domain.Teacher) {
	t.Helper()

	user, err := repo.CreateUser(ctx, domain.User{BranchID: branch.ID, Role: domain.RoleTeacher, Email: email, PasswordHash: "hash", FirstName: "Teacher", LastName: "User"})
	if err != nil {
		t.Fatal(err)
	}
	teacher, err := repo.CreateTeacher(ctx, domain.Teacher{BranchID: branch.ID, UserID: user.ID, Status: domain.TeacherStatusActive})
	if err != nil {
		t.Fatal(err)
	}

	return user, teacher
}

package finance_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"kingsway/backend/internal/domain"
	"kingsway/backend/internal/finance"
	"kingsway/backend/internal/store"
)

func TestStudentOnlySeesOwnPayments(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := store.NewMemory()
	service := finance.NewService(repo)
	data := seedFinanceAccessData(t, ctx, repo)

	payments, err := service.ListPayments(ctx, data.firstStudentPrincipal, data.branch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(payments) != 1 || payments[0].ID != data.firstPayment.ID {
		t.Fatalf("student should only see own payment, got %#v", payments)
	}

	_, err = service.GetPayment(ctx, data.firstStudentPrincipal, data.secondPayment.ID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden when student reads another student's payment, got %v", err)
	}
}

func TestTeacherCannotListPayments(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := store.NewMemory()
	service := finance.NewService(repo)
	data := seedFinanceAccessData(t, ctx, repo)

	_, err := service.ListPayments(ctx, data.teacherPrincipal, data.branch.ID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden for teacher payment list, got %v", err)
	}
}

type financeAccessData struct {
	branch                domain.Branch
	teacherPrincipal      domain.Principal
	firstStudentPrincipal domain.Principal
	firstPayment          domain.Payment
	secondPayment         domain.Payment
}

func seedFinanceAccessData(t *testing.T, ctx context.Context, repo *store.Memory) financeAccessData {
	t.Helper()

	branch, err := repo.CreateBranch(ctx, domain.Branch{Name: "Main", Slug: "finance-main"})
	if err != nil {
		t.Fatal(err)
	}
	owner, err := repo.CreateUser(ctx, domain.User{Role: domain.RoleOwner, Email: "finance-owner@test.local", PasswordHash: "hash", FirstName: "Owner", LastName: "User"})
	if err != nil {
		t.Fatal(err)
	}
	teacherUser, err := repo.CreateUser(ctx, domain.User{BranchID: branch.ID, Role: domain.RoleTeacher, Email: "finance-teacher@test.local", PasswordHash: "hash", FirstName: "Teacher", LastName: "User"})
	if err != nil {
		t.Fatal(err)
	}
	firstStudentUser, firstStudent := createFinanceStudent(t, ctx, repo, branch, "finance-student1@test.local", "FNC1001")
	_, secondStudent := createFinanceStudent(t, ctx, repo, branch, "finance-student2@test.local", "FNC1002")

	firstPayment, err := repo.CreatePayment(ctx, domain.Payment{
		BranchID:        branch.ID,
		StudentID:       firstStudent.ID,
		AmountCents:     10000,
		DueDate:         time.Now().UTC().AddDate(0, 0, 7),
		CreatedByUserID: owner.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	secondPayment, err := repo.CreatePayment(ctx, domain.Payment{
		BranchID:        branch.ID,
		StudentID:       secondStudent.ID,
		AmountCents:     20000,
		DueDate:         time.Now().UTC().AddDate(0, 0, 8),
		CreatedByUserID: owner.ID,
	})
	if err != nil {
		t.Fatal(err)
	}

	return financeAccessData{
		branch:                branch,
		teacherPrincipal:      domain.Principal{UserID: teacherUser.ID, BranchID: branch.ID, Role: domain.RoleTeacher},
		firstStudentPrincipal: domain.Principal{UserID: firstStudentUser.ID, BranchID: branch.ID, Role: domain.RoleStudent},
		firstPayment:          firstPayment,
		secondPayment:         secondPayment,
	}
}

func createFinanceStudent(t *testing.T, ctx context.Context, repo *store.Memory, branch domain.Branch, email string, rawFIN string) (domain.User, domain.Student) {
	t.Helper()

	user, err := repo.CreateUser(ctx, domain.User{BranchID: branch.ID, Role: domain.RoleStudent, Email: email, PasswordHash: "hash", FirstName: "Student", LastName: "User"})
	if err != nil {
		t.Fatal(err)
	}
	fin, err := domain.ParseFIN(rawFIN)
	if err != nil {
		t.Fatal(err)
	}
	student, err := repo.CreateStudent(ctx, domain.Student{BranchID: branch.ID, UserID: user.ID, FIN: fin, FirstName: "Student", LastName: "User"})
	if err != nil {
		t.Fatal(err)
	}

	return user, student
}

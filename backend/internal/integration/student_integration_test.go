package integration_test

import (
	"fmt"
	"testing"

	"kingsway/backend/internal/domain"
)

func TestStudentIntegrationSuite(t *testing.T) {
	fx := newIntegrationFixture(t)
	branch := integrationBranch(t, fx, "student-suite")
	course, _, err := fx.repo.CreateCourse(fx.ctx, domain.Course{
		BranchID: branch.ID,
		Name:     fmt.Sprintf("SAT %d", fx.stamp),
		IsActive: true,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	student, err := fx.repo.CreateStudentWithDetails(
		fx.ctx,
		domain.Student{
			BranchID:  branch.ID,
			FIN:       integrationFINWithPrefix(t, "S", fx.stamp),
			FirstName: "Integration",
			LastName:  "Student",
			BirthDate: "15/07/2008",
			Status:    domain.StudentStatusActive,
		},
		[]domain.StudentParentContact{{
			BranchID: branch.ID,
			Relation: "father",
			Name:     "Parent",
			Phones:   []string{"+994501111111"},
		}},
		[]domain.StudentCourseRegistration{{
			BranchID:           branch.ID,
			CourseID:           course.ID,
			MonthlyAmountCents: 25000,
			StartDate:          "01/06/2026",
		}},
	)
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := fx.repo.GetStudent(fx.ctx, student.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.FIN != student.FIN || loaded.Status != domain.StudentStatusActive {
		t.Fatalf("student create/read mismatch: %+v", loaded)
	}
}

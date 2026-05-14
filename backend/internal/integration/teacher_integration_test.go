package integration_test

import (
	"fmt"
	"testing"

	"kingsway/backend/internal/domain"
)

func TestTeacherIntegrationSuite(t *testing.T) {
	fx := newIntegrationFixture(t)
	branch := integrationBranch(t, fx, "teacher-suite")
	user := integrationUser(t, fx, branch.ID, domain.RoleTeacher, "teacher-suite")

	teacher, err := fx.repo.CreateTeacher(fx.ctx, domain.Teacher{
		BranchID: branch.ID,
		UserID:   user.ID,
		Status:   domain.TeacherStatusPending,
		Phone:    "+994551112233",
		Address:  "Teacher address",
	})
	if err != nil {
		t.Fatal(err)
	}
	course, _, err := fx.repo.CreateCourse(fx.ctx, domain.Course{
		BranchID: branch.ID,
		Name:     fmt.Sprintf("IELTS %d", fx.stamp),
		IsActive: true,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := fx.repo.ReplaceTeacherCourseSpecializations(fx.ctx, teacher.ID, []string{course.ID}); err != nil {
		t.Fatal(err)
	}
	teacher.Phone = "+994559998877"
	updated, err := fx.repo.UpdateTeacher(fx.ctx, teacher)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Phone != teacher.Phone {
		t.Fatalf("teacher update did not persist expected fields: %+v", updated)
	}

	if _, err := fx.repo.DeleteTeacher(fx.ctx, teacher.ID); err != nil {
		t.Fatal(err)
	}
	assertZeroRows(t, fx.ctx, fx.pool, "teachers", "id", teacher.ID)
	assertZeroRows(t, fx.ctx, fx.pool, "users", "id", user.ID)
}

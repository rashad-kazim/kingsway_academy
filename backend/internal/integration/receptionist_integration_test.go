package integration_test

import (
	"fmt"
	"testing"

	"kingsway/backend/internal/domain"
)

func TestReceptionistIntegrationSuite(t *testing.T) {
	fx := newIntegrationFixture(t)
	branch := integrationBranch(t, fx, "receptionist-suite")

	staff, err := fx.repo.CreateStaffMember(fx.ctx, domain.User{
		BranchID:     branch.ID,
		Role:         domain.RoleReceptionist,
		Email:        fmt.Sprintf("receptionist-suite-%d@integration.local", fx.stamp),
		PasswordHash: "hash",
		FirstName:    "Integration",
		LastName:     "Receptionist",
	}, domain.StaffMember{
		BranchID:        branch.ID,
		Role:            domain.RoleReceptionist,
		BirthDate:       "15/07/1998",
		Gender:          "female",
		Phone:           "+994501112233",
		Address:         "Receptionist address",
		HiredAt:         "01/01/2026",
		SalaryAmountAZN: 650,
	})
	if err != nil {
		t.Fatal(err)
	}
	staff.Phone = "+994507778899"
	staff.SalaryAmountAZN = 800
	updated, err := fx.repo.UpdateStaffMember(fx.ctx, staff, "")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Phone != staff.Phone || updated.SalaryAmountAZN != 800 {
		t.Fatalf("receptionist update did not persist expected fields: %+v", updated)
	}

	if _, err := fx.repo.DeleteStaffMember(fx.ctx, staff.ID); err != nil {
		t.Fatal(err)
	}
	assertZeroRows(t, fx.ctx, fx.pool, "staff_profiles", "id", staff.ID)
	assertZeroRows(t, fx.ctx, fx.pool, "users", "id", staff.UserID)
}

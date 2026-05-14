package integration_test

import (
	"strings"
	"testing"
)

func TestBranchIntegrationSuite(t *testing.T) {
	fx := newIntegrationFixture(t)
	branch := integrationBranch(t, fx, "branch-suite")

	branch.Name = branch.Name + " Updated"
	branch.Address = "Updated integration address"
	updated, err := fx.repo.UpdateBranch(fx.ctx, branch)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(updated.Name, "Updated") || updated.Address != "Updated integration address" {
		t.Fatalf("branch update did not persist expected fields: %+v", updated)
	}

	if _, err := fx.repo.DeleteBranch(fx.ctx, branch.ID); err != nil {
		t.Fatal(err)
	}
	assertZeroRows(t, fx.ctx, fx.pool, "branches", "id", branch.ID)
}

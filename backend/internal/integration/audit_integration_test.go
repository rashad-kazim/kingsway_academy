package integration_test

import (
	"testing"

	"kingsway/backend/internal/domain"
)

func TestAuditIntegrationSuite(t *testing.T) {
	fx := newIntegrationFixture(t)
	branch := integrationBranch(t, fx, "audit-suite")
	user := integrationUser(t, fx, branch.ID, domain.RoleOwner, "audit-suite")

	log, err := fx.repo.CreateAuditLog(fx.ctx, domain.AuditLog{
		ActorUserID:    user.ID,
		ActorRole:      domain.RoleOwner,
		ActorBranchID:  branch.ID,
		Action:         "patch.branches",
		EntityType:     "branches",
		EntityID:       branch.ID,
		EntityBranchID: branch.ID,
		RequestID:      "audit-suite-request",
		BeforeJSON:     map[string]any{"name": "before"},
		AfterJSON:      map[string]any{"name": "after"},
		Metadata:       map[string]any{"source": "integration"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if log.BeforeJSON["name"] != "before" || log.AfterJSON["name"] != "after" {
		t.Fatalf("audit before/after json did not persist: %+v", log)
	}
}

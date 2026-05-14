package integration_test

import (
	"fmt"
	"testing"

	"kingsway/backend/internal/domain"
)

func TestRoomIntegrationSuite(t *testing.T) {
	fx := newIntegrationFixture(t)
	branch := integrationBranch(t, fx, "room-suite")

	room, err := fx.repo.CreateRoom(fx.ctx, domain.Room{
		BranchID: branch.ID,
		Name:     fmt.Sprintf("Room %d", fx.stamp),
		Capacity: 12,
	})
	if err != nil {
		t.Fatal(err)
	}
	room.Name = room.Name + " Updated"
	room.Capacity = 18
	updated, err := fx.repo.UpdateRoom(fx.ctx, room)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Capacity != 18 || updated.Name != room.Name {
		t.Fatalf("room update did not persist expected fields: %+v", updated)
	}

	deactivated, err := fx.repo.DeactivateRoom(fx.ctx, room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if deactivated.IsActive {
		t.Fatalf("room should be inactive after delete/deactivate: %+v", deactivated)
	}
}

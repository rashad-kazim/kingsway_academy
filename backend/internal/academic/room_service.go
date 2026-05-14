package academic

import (
	"context"
	"strings"

	"kingsway/backend/internal/auth"
	"kingsway/backend/internal/domain"
)

func (s *Service) CreateRoom(ctx context.Context, actor domain.Principal, input CreateRoomInput) (domain.Room, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist); err != nil {
		return domain.Room{}, err
	}
	if !actor.IsOwner() {
		input.BranchID = actor.BranchID
	}
	if err := auth.RequireBranch(actor, input.BranchID); err != nil {
		return domain.Room{}, err
	}

	return s.rooms.CreateRoom(ctx, domain.Room{
		BranchID: input.BranchID,
		Name:     strings.TrimSpace(input.Name),
		Capacity: input.Capacity,
	})
}

func (s *Service) UpdateRoom(ctx context.Context, actor domain.Principal, input UpdateRoomInput) (domain.Room, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist); err != nil {
		return domain.Room{}, err
	}
	room, err := s.rooms.GetRoom(ctx, strings.TrimSpace(input.ID))
	if err != nil {
		return domain.Room{}, err
	}
	if err := auth.RequireBranch(actor, room.BranchID); err != nil {
		return domain.Room{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domain.Room{}, domain.ErrInvalidInput
	}
	capacity := input.Capacity
	if capacity <= 0 {
		capacity = 1
	}
	room.Name = name
	room.Capacity = capacity

	return s.rooms.UpdateRoom(ctx, room)
}

func (s *Service) RemoveRoom(ctx context.Context, actor domain.Principal, id string) (domain.Room, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist); err != nil {
		return domain.Room{}, err
	}
	room, err := s.rooms.GetRoom(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.Room{}, err
	}
	if err := auth.RequireBranch(actor, room.BranchID); err != nil {
		return domain.Room{}, err
	}

	return s.rooms.DeactivateRoom(ctx, room.ID)
}

func (s *Service) GetRoom(ctx context.Context, actor domain.Principal, id string) (domain.Room, error) {
	room, err := s.rooms.GetRoom(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.Room{}, err
	}
	if err := auth.RequireBranch(actor, room.BranchID); err != nil {
		return domain.Room{}, err
	}

	return room, nil
}

func (s *Service) ListRooms(ctx context.Context, actor domain.Principal, branchID string) ([]domain.Room, error) {
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	if actor.IsOwner() && branchID == "" {
		return s.rooms.ListRooms(ctx, "")
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, err
	}

	return s.rooms.ListRooms(ctx, branchID)
}

func (s *Service) ListRoomsPage(ctx context.Context, actor domain.Principal, branchID string, page domain.PageRequest) ([]domain.Room, int, error) {
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	branchID = strings.TrimSpace(branchID)
	if actor.IsOwner() && branchID == "" {
		return s.rooms.ListRoomsPage(ctx, "", page)
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, 0, err
	}
	return s.rooms.ListRoomsPage(ctx, branchID, page)
}

package academic

import (
	"context"
	"strings"

	"kingsway/backend/internal/auth"
	"kingsway/backend/internal/domain"
)

func (s *Service) CreateScheduleItem(ctx context.Context, actor domain.Principal, input CreateScheduleItemInput) (domain.ScheduleItem, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner, domain.RoleReceptionist, domain.RoleTeacher); err != nil {
		return domain.ScheduleItem{}, err
	}
	if !actor.IsOwner() {
		input.BranchID = actor.BranchID
	}
	if err := auth.RequireBranch(actor, input.BranchID); err != nil {
		return domain.ScheduleItem{}, err
	}
	if actor.Role == domain.RoleTeacher {
		teacher, err := s.teachers.GetTeacherByUser(ctx, actor.UserID)
		if err != nil {
			return domain.ScheduleItem{}, err
		}
		input.TeacherID = teacher.ID
		if input.ItemType == domain.ScheduleItemLesson {
			return domain.ScheduleItem{}, domain.ErrForbidden
		}
	}
	if input.ItemType == "" {
		input.ItemType = domain.ScheduleItemLesson
	}

	return s.schedules.CreateScheduleItem(ctx, domain.ScheduleItem{
		BranchID:        input.BranchID,
		ClassID:         input.ClassID,
		TeacherID:       input.TeacherID,
		RoomID:          input.RoomID,
		ItemType:        input.ItemType,
		Title:           strings.TrimSpace(input.Title),
		StartsAt:        input.StartsAt,
		EndsAt:          input.EndsAt,
		CreatedByUserID: actor.UserID,
	})
}

func (s *Service) ListSchedule(ctx context.Context, actor domain.Principal, branchID string) ([]domain.ScheduleItem, error) {
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	if actor.IsOwner() && branchID == "" {
		return s.schedules.ListSchedule(ctx, "")
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, err
	}

	items, err := s.schedules.ListSchedule(ctx, branchID)
	if err != nil {
		return nil, err
	}

	return s.filterScheduleForActor(ctx, actor, items)
}

func (s *Service) ListSchedulePage(ctx context.Context, actor domain.Principal, branchID string, itemType domain.ScheduleItemType, page domain.PageRequest) ([]domain.ScheduleItem, int, error) {
	if actor.Role == domain.RoleTeacher || actor.Role == domain.RoleStudent {
		items, err := s.ListSchedule(ctx, actor, branchID)
		if err != nil {
			return nil, 0, err
		}
		if itemType != "" {
			filtered := make([]domain.ScheduleItem, 0, len(items))
			for _, item := range items {
				if item.ItemType == itemType {
					filtered = append(filtered, item)
				}
			}
			items = filtered
		}
		paged, total := domain.PageSlice(items, page)
		return paged, total, nil
	}
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	branchID = strings.TrimSpace(branchID)
	if actor.IsOwner() && branchID == "" {
		return s.schedules.ListSchedulePage(ctx, "", itemType, page)
	}
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, 0, err
	}
	return s.schedules.ListSchedulePage(ctx, branchID, itemType, page)
}

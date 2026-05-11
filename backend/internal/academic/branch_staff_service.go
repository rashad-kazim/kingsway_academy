package academic

import (
	"context"
	"strings"

	"kingsway/backend/internal/auth"
	"kingsway/backend/internal/domain"
)

func (s *Service) CreateBranch(ctx context.Context, actor domain.Principal, input CreateBranchInput) (domain.Branch, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner); err != nil {
		return domain.Branch{}, err
	}

	openingTime, err := normalizeClockTime(input.OpeningTime)
	if err != nil {
		return domain.Branch{}, err
	}
	closingTime, err := normalizeClockTime(input.ClosingTime)
	if err != nil {
		return domain.Branch{}, err
	}

	return s.branches.CreateBranch(ctx, domain.Branch{
		Name:        strings.TrimSpace(input.Name),
		Slug:        strings.TrimSpace(input.Slug),
		Address:     strings.TrimSpace(input.Address),
		OpeningTime: openingTime,
		ClosingTime: closingTime,
	})
}

func (s *Service) UpdateBranch(ctx context.Context, actor domain.Principal, input UpdateBranchInput) (domain.Branch, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner); err != nil {
		return domain.Branch{}, err
	}

	branch, err := s.branches.GetBranch(ctx, input.ID)
	if err != nil {
		return domain.Branch{}, err
	}
	openingTime, err := normalizeClockTime(input.OpeningTime)
	if err != nil {
		return domain.Branch{}, err
	}
	closingTime, err := normalizeClockTime(input.ClosingTime)
	if err != nil {
		return domain.Branch{}, err
	}
	branch.Name = strings.TrimSpace(input.Name)
	branch.Slug = strings.TrimSpace(input.Slug)
	branch.Address = strings.TrimSpace(input.Address)
	branch.OpeningTime = openingTime
	branch.ClosingTime = closingTime

	return s.branches.UpdateBranch(ctx, branch)
}

func (s *Service) GetBranch(ctx context.Context, actor domain.Principal, id string) (domain.Branch, error) {
	branch, err := s.branches.GetBranch(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.Branch{}, err
	}
	if err := auth.RequireBranch(actor, branch.ID); err != nil {
		return domain.Branch{}, err
	}

	return branch, nil
}

func (s *Service) DeleteBranch(ctx context.Context, actor domain.Principal, id string) (domain.Branch, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner); err != nil {
		return domain.Branch{}, err
	}
	if strings.TrimSpace(id) == "" {
		return domain.Branch{}, domain.ErrInvalidInput
	}

	return s.branches.DeleteBranch(ctx, strings.TrimSpace(id))
}

func (s *Service) ListBranches(ctx context.Context, actor domain.Principal) ([]domain.Branch, error) {
	if actor.IsOwner() {
		return s.branches.ListBranches(ctx)
	}
	if actor.BranchID == "" {
		return nil, domain.ErrForbidden
	}

	branch, err := s.branches.GetBranch(ctx, actor.BranchID)
	if err != nil {
		return nil, err
	}

	return []domain.Branch{branch}, nil
}

func (s *Service) ListBranchesPage(ctx context.Context, actor domain.Principal, page domain.PageRequest) ([]domain.Branch, int, error) {
	if actor.IsOwner() {
		return s.branches.ListBranchesPage(ctx, page)
	}
	if actor.BranchID == "" {
		return nil, 0, domain.ErrForbidden
	}

	branch, err := s.branches.GetBranch(ctx, actor.BranchID)
	if err != nil {
		return nil, 0, err
	}

	items, total := domain.SinglePage(branch, page)
	return items, total, nil
}

func (s *Service) CreateStaffMember(ctx context.Context, actor domain.Principal, input CreateStaffInput) (domain.StaffMember, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner); err != nil {
		return domain.StaffMember{}, err
	}
	branchID := strings.TrimSpace(input.BranchID)
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return domain.StaffMember{}, err
	}
	if _, err := s.branches.GetBranch(ctx, branchID); err != nil {
		return domain.StaffMember{}, err
	}
	if strings.TrimSpace(input.FirstName) == "" || strings.TrimSpace(input.LastName) == "" || strings.TrimSpace(input.Email) == "" || input.Password == "" {
		return domain.StaffMember{}, domain.ErrInvalidInput
	}
	birthDate, err := normalizeBirthDate(input.BirthDate)
	if err != nil {
		return domain.StaffMember{}, err
	}
	hiredAt, err := normalizeBirthDate(input.HiredAt)
	if err != nil {
		return domain.StaffMember{}, err
	}
	if !isHireDateOnOrAfterBirthDate(birthDate, hiredAt) {
		return domain.StaffMember{}, domain.ErrInvalidInput
	}
	if input.SalaryAmountAZN < 0 {
		return domain.StaffMember{}, domain.ErrInvalidInput
	}
	passwordHash, err := auth.HashPassword(input.Password)
	if err != nil {
		return domain.StaffMember{}, err
	}

	return s.staff.CreateStaffMember(ctx, domain.User{
		BranchID:     branchID,
		Role:         domain.RoleReceptionist,
		Email:        strings.TrimSpace(input.Email),
		PasswordHash: passwordHash,
		FirstName:    strings.TrimSpace(input.FirstName),
		LastName:     strings.TrimSpace(input.LastName),
	}, domain.StaffMember{
		BranchID:           branchID,
		Role:               domain.RoleReceptionist,
		BirthDate:          birthDate,
		Gender:             normalizeOptionalTeacherGender(input.Gender),
		Phone:              strings.TrimSpace(input.Phone),
		Address:            strings.TrimSpace(input.Address),
		HiredAt:            hiredAt,
		SalaryAmountAZN:    input.SalaryAmountAZN,
		ProfilePhotoFileID: strings.TrimSpace(input.ProfilePhotoFileID),
	})
}

func (s *Service) UpdateStaffMember(ctx context.Context, actor domain.Principal, input UpdateStaffInput) (domain.StaffMember, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner); err != nil {
		return domain.StaffMember{}, err
	}
	current, err := s.staff.GetStaffMember(ctx, strings.TrimSpace(input.ID))
	if err != nil {
		return domain.StaffMember{}, err
	}
	if err := auth.RequireBranch(actor, current.BranchID); err != nil {
		return domain.StaffMember{}, err
	}
	nextBranchID := strings.TrimSpace(input.BranchID)
	if nextBranchID != "" && nextBranchID != current.BranchID {
		if err := auth.RequireBranch(actor, nextBranchID); err != nil {
			return domain.StaffMember{}, err
		}
		if _, err := s.branches.GetBranch(ctx, nextBranchID); err != nil {
			return domain.StaffMember{}, err
		}
		current.BranchID = nextBranchID
	}
	if strings.TrimSpace(input.FirstName) == "" || strings.TrimSpace(input.LastName) == "" || strings.TrimSpace(input.Email) == "" {
		return domain.StaffMember{}, domain.ErrInvalidInput
	}
	birthDate, err := normalizeBirthDate(input.BirthDate)
	if err != nil {
		return domain.StaffMember{}, err
	}
	hiredAt, err := normalizeBirthDate(input.HiredAt)
	if err != nil {
		return domain.StaffMember{}, err
	}
	if !isHireDateOnOrAfterBirthDate(birthDate, hiredAt) {
		return domain.StaffMember{}, domain.ErrInvalidInput
	}
	if input.SalaryAmountAZN < 0 {
		return domain.StaffMember{}, domain.ErrInvalidInput
	}
	passwordHash := ""
	if strings.TrimSpace(input.Password) != "" {
		passwordHash, err = auth.HashPassword(input.Password)
		if err != nil {
			return domain.StaffMember{}, err
		}
	}

	current.FirstName = strings.TrimSpace(input.FirstName)
	current.LastName = strings.TrimSpace(input.LastName)
	current.Email = strings.TrimSpace(input.Email)
	current.BirthDate = birthDate
	current.Gender = normalizeOptionalTeacherGender(input.Gender)
	current.Phone = strings.TrimSpace(input.Phone)
	current.Address = strings.TrimSpace(input.Address)
	current.HiredAt = hiredAt
	current.SalaryAmountAZN = input.SalaryAmountAZN
	current.ProfilePhotoFileID = strings.TrimSpace(input.ProfilePhotoFileID)
	if input.IsActive != nil {
		current.IsActive = *input.IsActive
	}
	return s.staff.UpdateStaffMember(ctx, current, passwordHash)
}

func (s *Service) DeleteStaffMember(ctx context.Context, actor domain.Principal, id string) (domain.StaffMember, error) {
	if err := auth.RequireAnyRole(actor, domain.RoleOwner); err != nil {
		return domain.StaffMember{}, err
	}
	staff, err := s.staff.GetStaffMember(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.StaffMember{}, err
	}
	if err := auth.RequireBranch(actor, staff.BranchID); err != nil {
		return domain.StaffMember{}, err
	}
	return s.staff.DeleteStaffMember(ctx, staff.ID)
}

func (s *Service) GetStaffMember(ctx context.Context, actor domain.Principal, id string) (domain.StaffMember, error) {
	staff, err := s.staff.GetStaffMember(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.StaffMember{}, err
	}
	if err := auth.RequireBranch(actor, staff.BranchID); err != nil {
		return domain.StaffMember{}, err
	}

	return staff, nil
}

func (s *Service) ListStaffMembers(ctx context.Context, actor domain.Principal, branchID string) ([]domain.StaffMember, error) {
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	if err := auth.RequireBranch(actor, strings.TrimSpace(branchID)); err != nil {
		return nil, err
	}
	return s.staff.ListStaffMembers(ctx, strings.TrimSpace(branchID))
}

func (s *Service) ListStaffMembersPage(ctx context.Context, actor domain.Principal, branchID string, page domain.PageRequest) ([]domain.StaffMember, int, error) {
	if !actor.IsOwner() {
		branchID = actor.BranchID
	}
	branchID = strings.TrimSpace(branchID)
	if err := auth.RequireBranch(actor, branchID); err != nil {
		return nil, 0, err
	}
	return s.staff.ListStaffMembersPage(ctx, branchID, page)
}

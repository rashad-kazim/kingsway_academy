package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"kingsway/backend/internal/academic"
	"kingsway/backend/internal/domain"
)

func (s *Server) createBranch(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input academic.CreateBranchInput
	s.decodeIdempotentJSON(w, r, principal, &input, func(ctx context.Context) (int, any, error) {
		branch, err := s.academic.CreateBranch(ctx, principal, input)
		if err == nil {
			setAuditEntity(ctx, "branches", branch.ID, branch.ID)
		}
		return http.StatusCreated, branch, err
	})
}

func (s *Server) listBranches(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	page, err := parsePage(r)
	if err != nil {
		writeError(w, err)
		return
	}
	branches, total, err := s.academic.ListBranchesPage(r.Context(), principal, page.domainPage())
	if err != nil {
		writeError(w, err)
		return
	}
	writePageJSON(w, page, total, branches)
}

func (s *Server) branchAction(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/v1/branches/"))
	if len(parts) == 2 && parts[1] == "staff" {
		s.branchStaffAction(w, r, principal, parts[0])
		return
	}
	if len(parts) != 1 || parts[0] == "" {
		writeError(w, domain.ErrNotFound)
		return
	}
	id := parts[0]

	if r.Method == http.MethodDelete {
		s.writeIdempotentNoBodyJSON(w, r, principal, func(ctx context.Context) (int, any, error) {
			beforeBranch, err := s.academic.GetBranch(ctx, principal, id)
			if err != nil {
				return 0, nil, err
			}
			setAuditBefore(ctx, beforeBranch)
			setAuditEntity(ctx, "branches", id, id)

			branchFiles, err := s.files.ListFiles(ctx, principal, id)
			if err != nil {
				return 0, nil, err
			}
			for _, file := range branchFiles {
				if _, err := s.files.DeleteFile(ctx, principal, file.ID); err != nil && !errors.Is(err, domain.ErrNotFound) {
					return 0, nil, err
				}
			}

			branch, err := s.academic.DeleteBranch(ctx, principal, id)
			return http.StatusOK, branch, err
		})
		return
	}

	if r.Method != http.MethodPatch {
		writeError(w, domain.ErrNotFound)
		return
	}

	var input academic.UpdateBranchInput
	s.decodeIdempotentJSON(w, r, principal, &input, func(ctx context.Context) (int, any, error) {
		input.ID = id
		beforeBranch, err := s.academic.GetBranch(ctx, principal, id)
		if err != nil {
			return 0, nil, err
		}
		setAuditBefore(ctx, beforeBranch)
		setAuditEntity(ctx, "branches", id, id)
		branch, err := s.academic.UpdateBranch(ctx, principal, input)
		return http.StatusOK, branch, err
	})
}

func (s *Server) branchStaffAction(w http.ResponseWriter, r *http.Request, principal domain.Principal, branchID string) {
	switch r.Method {
	case http.MethodGet:
		page, err := parsePage(r)
		if err != nil {
			writeError(w, err)
			return
		}
		staff, total, err := s.academic.ListStaffMembersPage(r.Context(), principal, branchID, page.domainPage())
		if err != nil {
			writeError(w, err)
			return
		}
		writePageJSON(w, page, total, staff)
	case http.MethodPost:
		var input academic.CreateStaffInput
		s.decodeIdempotentJSON(w, r, principal, &input, func(ctx context.Context) (int, any, error) {
			input.BranchID = branchID
			staff, err := s.academic.CreateStaffMember(ctx, principal, input)
			if err == nil {
				setAuditEntity(ctx, "staff", staff.ID, staff.BranchID)
			}
			return http.StatusCreated, staff, err
		})
	default:
		writeError(w, domain.ErrNotFound)
	}
}

func (s *Server) staffAction(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/v1/staff/"))
	if len(parts) != 1 || parts[0] == "" {
		writeError(w, domain.ErrNotFound)
		return
	}

	if r.Method == http.MethodDelete {
		s.writeIdempotentNoBodyJSON(w, r, principal, func(ctx context.Context) (int, any, error) {
			beforeStaff, err := s.academic.GetStaffMember(ctx, principal, parts[0])
			if err != nil {
				return 0, nil, err
			}
			setAuditBefore(ctx, beforeStaff)
			setAuditEntity(ctx, "staff", parts[0], beforeStaff.BranchID)

			staff, err := s.academic.DeleteStaffMember(ctx, principal, parts[0])
			if err != nil {
				return 0, nil, err
			}
			if staff.ProfilePhotoFileID != "" {
				if _, err := s.files.DeleteFile(ctx, principal, staff.ProfilePhotoFileID); err != nil && !errors.Is(err, domain.ErrNotFound) {
					return 0, nil, err
				}
			}
			return http.StatusOK, staff, nil
		})
		return
	}

	if r.Method != http.MethodPatch {
		writeError(w, domain.ErrNotFound)
		return
	}

	var input academic.UpdateStaffInput
	s.decodeIdempotentJSON(w, r, principal, &input, func(ctx context.Context) (int, any, error) {
		input.ID = parts[0]
		beforeStaff, err := s.academic.GetStaffMember(ctx, principal, parts[0])
		if err != nil {
			return 0, nil, err
		}
		setAuditBefore(ctx, beforeStaff)
		setAuditEntity(ctx, "staff", parts[0], beforeStaff.BranchID)
		staff, err := s.academic.UpdateStaffMember(ctx, principal, input)
		return http.StatusOK, staff, err
	})
}

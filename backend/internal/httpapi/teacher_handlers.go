package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"kingsway/backend/internal/academic"
	"kingsway/backend/internal/domain"
	"kingsway/backend/internal/finance"
)

type registerTeacherRequest struct {
	academic.RegisterTeacherInput
	SalaryModel               domain.SalaryModelType `json:"salary_model"`
	FixedMonthlyAmountCents   int64                  `json:"fixed_monthly_amount_cents"`
	StudentPercentBasisPoints int                    `json:"student_percent_basis_points"`
}

func (s *Server) registerTeacher(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input registerTeacherRequest
	s.decodeIdempotentJSON(w, r, principal, &input, func(ctx context.Context) (int, any, error) {
		if !principal.IsOwner() && input.SalaryModel != "" {
			return 0, nil, domain.ErrForbidden
		}
		if err := validateTeacherSalaryRequest(input.SalaryModel, input.FixedMonthlyAmountCents, input.StudentPercentBasisPoints); err != nil {
			return 0, nil, err
		}
		teacher, user, err := s.academic.RegisterTeacher(ctx, principal, input.RegisterTeacherInput)
		if err != nil {
			return 0, nil, err
		}
		var salaryModel *domain.SalaryModel
		if input.SalaryModel != "" {
			model, err := s.finance.CreateSalaryModel(ctx, principal, finance.CreateSalaryModelInput{
				BranchID:                  teacher.BranchID,
				TeacherID:                 teacher.ID,
				ModelType:                 input.SalaryModel,
				FixedMonthlyAmountCents:   input.FixedMonthlyAmountCents,
				StudentPercentBasisPoints: input.StudentPercentBasisPoints,
				ActiveFrom:                time.Now().UTC(),
			})
			if err != nil {
				return 0, nil, err
			}
			salaryModel = &model
			teacher, err = s.academic.ActivateTeacher(ctx, principal, teacher.ID)
			if err != nil {
				return 0, nil, err
			}
		}
		return http.StatusCreated, map[string]any{"teacher": teacher, "user": user, "salary_model": salaryModel}, nil
	})
}

func (s *Server) listTeachers(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	page, err := parsePage(r)
	if err != nil {
		writeError(w, err)
		return
	}
	status := domain.TeacherStatus(strings.TrimSpace(r.URL.Query().Get("status")))
	teachers, total, err := s.academic.ListTeachersPage(r.Context(), principal, r.URL.Query().Get("branch_id"), status, page.domainPage())
	if err != nil {
		writeError(w, err)
		return
	}
	writePageJSON(w, page, total, teachers)
}

func (s *Server) listTeacherFinance(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	page, err := parsePage(r)
	if err != nil {
		writeError(w, err)
		return
	}
	records, total, err := s.finance.ListTeacherFinanceRecordsPage(r.Context(), principal, finance.TeacherFinanceFilter{
		BranchID:    r.URL.Query().Get("branch_id"),
		Subject:     r.URL.Query().Get("subject"),
		Status:      domain.TeacherStatus(strings.TrimSpace(r.URL.Query().Get("status"))),
		SalaryModel: domain.SalaryModelType(strings.TrimSpace(r.URL.Query().Get("salary_model"))),
	}, page.domainPage())
	if err != nil {
		writeError(w, err)
		return
	}
	writePageJSON(w, page, total, records)
}

func validateTeacherSalaryRequest(model domain.SalaryModelType, amountCents int64, percentBasisPoints int) error {
	if model == "" {
		return nil
	}
	if !model.IsValid() {
		return domain.ErrInvalidInput
	}
	if model == domain.SalaryModelFixed || model == domain.SalaryModelHybrid {
		if amountCents <= 0 {
			return domain.ErrInvalidInput
		}
	}
	if model == domain.SalaryModelPercent || model == domain.SalaryModelHybrid {
		if percentBasisPoints < 0 || percentBasisPoints > 10000 {
			return domain.ErrInvalidInput
		}
	}

	return nil
}

func (s *Server) teacherAction(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/teachers/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if r.Method == http.MethodGet && len(parts) == 1 && parts[0] != "" {
		teacher, err := s.academic.GetTeacher(r.Context(), principal, parts[0])
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, teacher)
		return
	}
	if r.Method == http.MethodPatch && len(parts) == 1 && parts[0] != "" {
		var input academic.UpdateTeacherInput
		s.decodeIdempotentJSON(w, r, principal, &input, func(ctx context.Context) (int, any, error) {
			input.ID = parts[0]
			beforeTeacher, err := s.academic.GetTeacher(ctx, principal, parts[0])
			if err != nil {
				return 0, nil, err
			}
			setAuditBefore(ctx, beforeTeacher)
			setAuditEntity(ctx, "teachers", parts[0], beforeTeacher.BranchID)
			teacher, err := s.academic.UpdateTeacher(ctx, principal, input)
			return http.StatusOK, teacher, err
		})
		return
	}
	if r.Method == http.MethodDelete && len(parts) == 1 && parts[0] != "" {
		s.writeIdempotentNoBodyJSON(w, r, principal, func(ctx context.Context) (int, any, error) {
			existingTeacher, err := s.academic.GetTeacher(ctx, principal, parts[0])
			if err != nil {
				return 0, nil, err
			}
			setAuditBefore(ctx, existingTeacher)
			setAuditEntity(ctx, "teachers", parts[0], existingTeacher.BranchID)
			records, err := s.finance.ListTeacherFinanceRecords(ctx, principal, finance.TeacherFinanceFilter{
				BranchID: existingTeacher.BranchID,
			})
			if err != nil {
				return 0, nil, err
			}
			for _, record := range records {
				if record.ID == existingTeacher.ID && record.AssignedStudents > 0 {
					return 0, nil, domain.ErrConflict
				}
			}
			branchFiles, err := s.files.ListFiles(ctx, principal, existingTeacher.BranchID)
			if err != nil {
				return 0, nil, err
			}
			for _, file := range branchFiles {
				if file.UploaderUserID != existingTeacher.UserID && (file.OwnerType != "teacher" || file.OwnerID != existingTeacher.ID) {
					continue
				}
				if _, err := s.files.DeleteFile(ctx, principal, file.ID); err != nil && !errors.Is(err, domain.ErrNotFound) {
					return 0, nil, err
				}
			}

			teacher, err := s.finance.DeleteTeacher(ctx, principal, parts[0])
			return http.StatusOK, teacher, err
		})
		return
	}
	if r.Method != http.MethodPost || len(parts) != 2 || parts[1] != "activate" {
		writeError(w, domain.ErrNotFound)
		return
	}
	s.writeIdempotentNoBodyJSON(w, r, principal, func(ctx context.Context) (int, any, error) {
		beforeTeacher, err := s.academic.GetTeacher(ctx, principal, parts[0])
		if err != nil {
			return 0, nil, err
		}
		setAuditBefore(ctx, beforeTeacher)
		setAuditEntity(ctx, "teachers", parts[0], beforeTeacher.BranchID)
		teacher, err := s.academic.ActivateTeacher(ctx, principal, parts[0])
		return http.StatusOK, teacher, err
	})
}

package httpapi

import (
	"context"
	"net/http"
	"strings"

	"kingsway/backend/internal/academic"
	"kingsway/backend/internal/domain"
)

func (s *Server) createStudent(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input academic.CreateStudentInput
	s.decodeIdempotentJSON(w, r, principal, &input, func(ctx context.Context) (int, any, error) {
		student, err := s.academic.CreateStudent(ctx, principal, input)
		return http.StatusCreated, student, err
	})
}

func (s *Server) listStudents(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	page, err := parsePage(r)
	if err != nil {
		writeError(w, err)
		return
	}
	status := domain.StudentStatus(strings.TrimSpace(r.URL.Query().Get("status")))
	students, total, err := s.academic.ListStudentsPage(r.Context(), principal, r.URL.Query().Get("branch_id"), status, page.domainPage())
	if err != nil {
		writeError(w, err)
		return
	}
	writePageJSON(w, page, total, students)
}

func (s *Server) listStudentAssignmentHub(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	page, err := parsePage(r)
	if err != nil {
		writeError(w, err)
		return
	}
	result, err := s.academic.ListStudentAssignmentHub(r.Context(), principal, domain.StudentAssignmentHubFilter{
		BranchID:  r.URL.Query().Get("branch_id"),
		Status:    domain.StudentStatus(strings.TrimSpace(r.URL.Query().Get("status"))),
		TeacherID: r.URL.Query().Get("teacher_id"),
		Query:     r.URL.Query().Get("q"),
		Limit:     page.Limit,
		Offset:    page.Offset,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writePageHeaders(w, page, result.Total)
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) getStudentByFIN(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	fin := strings.TrimPrefix(r.URL.Path, "/v1/students/by-fin/")
	student, err := s.academic.GetStudentByFIN(r.Context(), principal, fin)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, student)
}

func (s *Server) studentAction(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/students/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) == 1 && parts[0] != "" && r.Method == http.MethodGet {
		student, err := s.academic.GetStudent(r.Context(), principal, parts[0])
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, student)
		return
	}
	if len(parts) == 1 && parts[0] != "" && r.Method == http.MethodPatch {
		var input academic.UpdateStudentInput
		s.decodeIdempotentJSON(w, r, principal, &input, func(ctx context.Context) (int, any, error) {
			beforeStudent, err := s.academic.GetStudent(ctx, principal, parts[0])
			if err != nil {
				return 0, nil, err
			}
			setAuditBefore(ctx, beforeStudent)
			setAuditEntity(ctx, "students", parts[0], beforeStudent.BranchID)
			student, err := s.academic.UpdateStudent(ctx, principal, parts[0], input)
			return http.StatusOK, student, err
		})
		return
	}
	if len(parts) == 2 && parts[1] == "account" && r.Method == http.MethodPost {
		var input academic.CreateStudentAccountInput
		s.decodeIdempotentJSON(w, r, principal, &input, func(ctx context.Context) (int, any, error) {
			student, user, err := s.academic.CreateStudentAccount(ctx, principal, parts[0], input)
			return http.StatusCreated, map[string]any{"student": student, "user": user}, err
		})
		return
	}

	if path == "" {
		writeError(w, domain.ErrNotFound)
		return
	}
	writeError(w, domain.ErrNotFound)
}

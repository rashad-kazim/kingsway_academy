package httpapi

import (
	"net/http"

	"kingsway/backend/internal/auth"
	"kingsway/backend/internal/domain"
)

type sessionResponse struct {
	User          domain.User      `json:"user"`
	Principal     domain.Principal `json:"principal"`
	Branch        *domain.Branch   `json:"branch,omitempty"`
	Capabilities  []string         `json:"capabilities"`
	DashboardPath string           `json:"dashboard_path"`
}

func (s *Server) session(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	user, err := s.auth.CurrentUser(r.Context(), principal)
	if err != nil {
		writeError(w, err)
		return
	}

	var branch *domain.Branch
	if user.BranchID != "" {
		branches, err := s.academic.ListBranches(r.Context(), principal)
		if err != nil {
			writeError(w, err)
			return
		}
		if len(branches) > 0 {
			branch = &branches[0]
		}
	}

	writeJSON(w, http.StatusOK, sessionResponse{
		User:          user,
		Principal:     principal,
		Branch:        branch,
		Capabilities:  capabilitiesFor(user.Role),
		DashboardPath: dashboardPathFor(user.Role),
	})
}

func (s *Server) ownerDashboard(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	if err := auth.RequireAnyRole(principal, domain.RoleOwner); err != nil {
		writeError(w, err)
		return
	}
	branchID := r.URL.Query().Get("branch_id")
	dashboard, err := s.academic.AcademicDashboard(r.Context(), principal, branchID)
	if err != nil {
		writeError(w, err)
		return
	}
	branches, err := s.academic.ListBranches(r.Context(), principal)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"role":      domain.RoleOwner,
		"branch_id": branchID,
		"summary":   dashboard,
		"branches":  branches,
	})
}

func (s *Server) receptionistDashboard(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	if err := auth.RequireAnyRole(principal, domain.RoleReceptionist); err != nil {
		writeError(w, err)
		return
	}
	dashboard, err := s.academic.AcademicDashboard(r.Context(), principal, principal.BranchID)
	if err != nil {
		writeError(w, err)
		return
	}
	schedule, err := s.academic.ListSchedule(r.Context(), principal, principal.BranchID)
	if err != nil {
		writeError(w, err)
		return
	}
	payments, err := s.finance.ListPayments(r.Context(), principal, principal.BranchID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"role":      domain.RoleReceptionist,
		"branch_id": principal.BranchID,
		"summary":   dashboard,
		"schedule":  firstN(schedule, 20),
		"payments":  firstN(payments, 20),
	})
}

func (s *Server) teacherDashboard(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	if err := auth.RequireAnyRole(principal, domain.RoleTeacher); err != nil {
		writeError(w, err)
		return
	}
	teachers, err := s.academic.ListTeachers(r.Context(), principal, principal.BranchID)
	if err != nil {
		writeError(w, err)
		return
	}
	classes, err := s.academic.ListClasses(r.Context(), principal, principal.BranchID)
	if err != nil {
		writeError(w, err)
		return
	}
	schedule, err := s.academic.ListSchedule(r.Context(), principal, principal.BranchID)
	if err != nil {
		writeError(w, err)
		return
	}
	assignments, err := s.academic.ListAssignments(r.Context(), principal, principal.BranchID, "")
	if err != nil {
		writeError(w, err)
		return
	}
	var teacher any
	if len(teachers) > 0 {
		teacher = teachers[0]
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"role":        domain.RoleTeacher,
		"branch_id":   principal.BranchID,
		"teacher":     teacher,
		"classes":     firstN(classes, 20),
		"schedule":    firstN(schedule, 20),
		"assignments": firstN(assignments, 20),
	})
}

func (s *Server) studentDashboard(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	if err := auth.RequireAnyRole(principal, domain.RoleStudent); err != nil {
		writeError(w, err)
		return
	}
	students, err := s.academic.ListStudents(r.Context(), principal, principal.BranchID)
	if err != nil {
		writeError(w, err)
		return
	}
	classes, err := s.academic.ListClasses(r.Context(), principal, principal.BranchID)
	if err != nil {
		writeError(w, err)
		return
	}
	assignments, err := s.academic.ListAssignments(r.Context(), principal, principal.BranchID, "")
	if err != nil {
		writeError(w, err)
		return
	}
	exams, err := s.academic.ListExams(r.Context(), principal, principal.BranchID, "")
	if err != nil {
		writeError(w, err)
		return
	}
	results, err := s.academic.ListExamResults(r.Context(), principal, principal.BranchID, "", "")
	if err != nil {
		writeError(w, err)
		return
	}
	var student any
	if len(students) > 0 {
		student = students[0]
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"role":        domain.RoleStudent,
		"branch_id":   principal.BranchID,
		"student":     student,
		"classes":     firstN(classes, 20),
		"assignments": firstN(assignments, 20),
		"exams":       firstN(exams, 20),
		"results":     firstN(results, 20),
	})
}

func capabilitiesFor(role domain.Role) []string {
	switch role {
	case domain.RoleOwner:
		return []string{"branches:all", "students:all", "teachers:all", "finance:all", "salary:all", "files:all", "notifications:read"}
	case domain.RoleReceptionist:
		return []string{"students:branch", "teachers:branch", "schedule:branch", "payments:branch", "files:branch", "notifications:read"}
	case domain.RoleTeacher:
		return []string{"classes:own", "assignments:own", "exams:own", "scores:own", "salary:own", "notifications:read"}
	case domain.RoleStudent:
		return []string{"classes:own", "assignments:own", "exams:own", "payments:own", "notifications:read"}
	default:
		return []string{}
	}
}

func dashboardPathFor(role domain.Role) string {
	switch role {
	case domain.RoleOwner:
		return "/dashboard/owner"
	case domain.RoleReceptionist:
		return "/dashboard/receptionist"
	case domain.RoleTeacher:
		return "/dashboard/teacher"
	case domain.RoleStudent:
		return "/dashboard/student"
	default:
		return "/dashboard"
	}
}

func firstN[T any](items []T, n int) []T {
	if len(items) <= n {
		return items
	}
	return items[:n]
}

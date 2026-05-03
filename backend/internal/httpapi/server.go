package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"kingsway/backend/internal/academic"
	"kingsway/backend/internal/admin"
	"kingsway/backend/internal/auth"
	"kingsway/backend/internal/domain"
	"kingsway/backend/internal/files"
	"kingsway/backend/internal/finance"
	"kingsway/backend/internal/notification"
)

type Server struct {
	auth     *auth.Service
	academic *academic.Service
	finance  *finance.Service
	files    *files.Service
	notify   *notification.Service
	admin    *admin.Service
	logger   *zap.Logger
	mux      *http.ServeMux
	options  Options
	limiter  *rateLimiter
	metrics  *requestMetrics
}

const maxUploadFileBytes int64 = 10 << 20

func New(
	authService *auth.Service,
	academicService *academic.Service,
	financeService *finance.Service,
	filesService *files.Service,
	notificationService *notification.Service,
	adminService *admin.Service,
	logger *zap.Logger,
	opts ...Options,
) *Server {
	options := normalizeOptions(opts...)
	s := &Server{
		auth:     authService,
		academic: academicService,
		finance:  financeService,
		files:    filesService,
		notify:   notificationService,
		admin:    adminService,
		logger:   logger,
		mux:      http.NewServeMux(),
		options:  options,
		limiter:  newRateLimiter(options.RateLimitPerMinute),
		metrics:  newRequestMetrics(),
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	var handler http.Handler = s.mux
	handler = s.withRateLimit(handler)
	handler = s.withMetrics(handler)
	handler = s.withCORS(handler)
	handler = s.withRequestLogging(handler)
	handler = s.withRequestID(handler)
	return handler
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.health)
	s.mux.HandleFunc("GET /readyz", s.ready)
	s.mux.HandleFunc("GET /metrics", s.metricsSnapshot)
	s.mux.HandleFunc("POST /v1/auth/bootstrap-owner", s.bootstrapOwner)
	s.mux.HandleFunc("POST /v1/auth/login", s.login)
	s.mux.HandleFunc("GET /v1/me", s.requireAuth(s.me))
	s.mux.HandleFunc("GET /v1/session", s.requireAuth(s.session))

	s.mux.HandleFunc("GET /v1/branches", s.requireAuth(s.listBranches))
	s.mux.HandleFunc("POST /v1/branches", s.requireAuth(s.createBranch))
	s.mux.HandleFunc("GET /v1/branches/", s.requireAuth(s.branchAction))
	s.mux.HandleFunc("POST /v1/branches/", s.requireAuth(s.branchAction))
	s.mux.HandleFunc("PATCH /v1/branches/", s.requireAuth(s.branchAction))
	s.mux.HandleFunc("DELETE /v1/branches/", s.requireAuth(s.branchAction))
	s.mux.HandleFunc("PATCH /v1/staff/", s.requireAuth(s.staffAction))
	s.mux.HandleFunc("DELETE /v1/staff/", s.requireAuth(s.staffAction))

	s.mux.HandleFunc("GET /v1/students", s.requireAuth(s.listStudents))
	s.mux.HandleFunc("POST /v1/students", s.requireAuth(s.createStudent))
	s.mux.HandleFunc("GET /v1/students/by-fin/", s.requireAuth(s.getStudentByFIN))
	s.mux.HandleFunc("GET /v1/students/", s.requireAuth(s.studentAction))
	s.mux.HandleFunc("POST /v1/students/", s.requireAuth(s.studentAction))

	s.mux.HandleFunc("GET /v1/teachers", s.requireAuth(s.listTeachers))
	s.mux.HandleFunc("POST /v1/teachers", s.requireAuth(s.registerTeacher))
	s.mux.HandleFunc("GET /v1/teachers/", s.requireAuth(s.teacherAction))
	s.mux.HandleFunc("POST /v1/teachers/", s.requireAuth(s.teacherAction))

	s.mux.HandleFunc("GET /v1/courses", s.requireAuth(s.listCourses))
	s.mux.HandleFunc("POST /v1/courses", s.requireAuth(s.createCourse))
	s.mux.HandleFunc("GET /v1/courses/", s.requireAuth(s.courseAction))

	s.mux.HandleFunc("GET /v1/classes", s.requireAuth(s.listClasses))
	s.mux.HandleFunc("POST /v1/classes", s.requireAuth(s.createClass))
	s.mux.HandleFunc("GET /v1/classes/", s.requireAuth(s.classAction))
	s.mux.HandleFunc("POST /v1/classes/", s.requireAuth(s.classAction))

	s.mux.HandleFunc("GET /v1/assignments", s.requireAuth(s.listAssignments))
	s.mux.HandleFunc("POST /v1/assignments", s.requireAuth(s.createAssignment))

	s.mux.HandleFunc("GET /v1/rooms", s.requireAuth(s.listRooms))
	s.mux.HandleFunc("POST /v1/rooms", s.requireAuth(s.createRoom))
	s.mux.HandleFunc("DELETE /v1/rooms/", s.requireAuth(s.roomAction))

	s.mux.HandleFunc("GET /v1/schedules", s.requireAuth(s.listSchedule))
	s.mux.HandleFunc("POST /v1/schedules", s.requireAuth(s.createScheduleItem))

	s.mux.HandleFunc("GET /v1/exams", s.requireAuth(s.listExams))
	s.mux.HandleFunc("POST /v1/exams", s.requireAuth(s.createExam))
	s.mux.HandleFunc("GET /v1/exams/", s.requireAuth(s.examAction))
	s.mux.HandleFunc("POST /v1/exams/", s.requireAuth(s.examAction))
	s.mux.HandleFunc("GET /v1/exam-results", s.requireAuth(s.listExamResults))

	s.mux.HandleFunc("GET /v1/dashboard/academic", s.requireAuth(s.academicDashboard))
	s.mux.HandleFunc("GET /v1/dashboard/owner", s.requireAuth(s.ownerDashboard))
	s.mux.HandleFunc("GET /v1/dashboard/receptionist", s.requireAuth(s.receptionistDashboard))
	s.mux.HandleFunc("GET /v1/dashboard/teacher", s.requireAuth(s.teacherDashboard))
	s.mux.HandleFunc("GET /v1/dashboard/student", s.requireAuth(s.studentDashboard))

	s.mux.HandleFunc("GET /v1/payments", s.requireAuth(s.listPayments))
	s.mux.HandleFunc("POST /v1/payments", s.requireAuth(s.createPayment))
	s.mux.HandleFunc("GET /v1/payments/", s.requireAuth(s.paymentAction))

	s.mux.HandleFunc("GET /v1/salary-models", s.requireAuth(s.listSalaryModels))
	s.mux.HandleFunc("POST /v1/salary-models", s.requireAuth(s.createSalaryModel))
	s.mux.HandleFunc("POST /v1/salary/swap-allocation", s.requireAuth(s.calculateSwapAllocation))

	s.mux.HandleFunc("GET /v1/files", s.requireAuth(s.listFiles))
	s.mux.HandleFunc("POST /v1/files", s.requireAuth(s.registerFile))
	s.mux.HandleFunc("POST /v1/files/upload", s.requireAuth(s.uploadFile))
	s.mux.HandleFunc("POST /v1/files/retention/cleanup", s.requireAuth(s.cleanupFileRetention))
	s.mux.HandleFunc("GET /v1/files/", s.requireAuth(s.fileAction))
	s.mux.HandleFunc("DELETE /v1/files/", s.requireAuth(s.fileAction))

	s.mux.HandleFunc("GET /v1/notifications", s.requireAuth(s.listNotifications))
	s.mux.HandleFunc("POST /v1/notifications/", s.requireAuth(s.notificationAction))

	s.mux.HandleFunc("GET /v1/admin/outbox", s.requireAuth(s.listOutboxEvents))
	s.mux.HandleFunc("POST /v1/admin/outbox/", s.requireAuth(s.outboxAction))
}

type principalKey struct{}

func (s *Server) requireAuth(next func(http.ResponseWriter, *http.Request, domain.Principal)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if raw == "" || raw == r.Header.Get("Authorization") {
			writeError(w, domain.ErrUnauthorized)
			return
		}

		principal, err := s.auth.PrincipalFromToken(raw)
		if err != nil {
			writeError(w, domain.ErrUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), principalKey{}, principal)
		next(w, r.WithContext(ctx), principal)
	}
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Expose-Headers", "X-Total-Count, X-Limit, X-Offset, X-Request-ID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"components": map[string]string{
			"api": "ok",
		},
	})
}

func (s *Server) metricsSnapshot(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.metrics.snapshot())
}

func (s *Server) bootstrapOwner(w http.ResponseWriter, r *http.Request) {
	var input auth.BootstrapOwnerInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := s.auth.BootstrapOwner(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var input auth.LoginInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := s.auth.Login(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	user, err := s.auth.CurrentUser(r.Context(), principal)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) createBranch(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input academic.CreateBranchInput
	if !decodeJSON(w, r, &input) {
		return
	}
	branch, err := s.academic.CreateBranch(r.Context(), principal, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, branch)
}

func (s *Server) listBranches(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	branches, err := s.academic.ListBranches(r.Context(), principal)
	if err != nil {
		writeError(w, err)
		return
	}
	writePagedJSON(w, r, branches)
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
		branchFiles, err := s.files.ListFiles(r.Context(), principal, id)
		if err != nil {
			writeError(w, err)
			return
		}
		for _, file := range branchFiles {
			if _, err := s.files.DeleteFile(r.Context(), principal, file.ID); err != nil && !errors.Is(err, domain.ErrNotFound) {
				writeError(w, err)
				return
			}
		}

		branch, err := s.academic.DeleteBranch(r.Context(), principal, id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, branch)
		return
	}

	if r.Method != http.MethodPatch {
		writeError(w, domain.ErrNotFound)
		return
	}

	var input academic.UpdateBranchInput
	if !decodeJSON(w, r, &input) {
		return
	}
	input.ID = id
	branch, err := s.academic.UpdateBranch(r.Context(), principal, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, branch)
}

func (s *Server) branchStaffAction(w http.ResponseWriter, r *http.Request, principal domain.Principal, branchID string) {
	switch r.Method {
	case http.MethodGet:
		staff, err := s.academic.ListStaffMembers(r.Context(), principal, branchID)
		if err != nil {
			writeError(w, err)
			return
		}
		writePagedJSON(w, r, staff)
	case http.MethodPost:
		var input academic.CreateStaffInput
		if !decodeJSON(w, r, &input) {
			return
		}
		input.BranchID = branchID
		staff, err := s.academic.CreateStaffMember(r.Context(), principal, input)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, staff)
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
		staff, err := s.academic.DeleteStaffMember(r.Context(), principal, parts[0])
		if err != nil {
			writeError(w, err)
			return
		}
		if staff.ProfilePhotoFileID != "" {
			if _, err := s.files.DeleteFile(r.Context(), principal, staff.ProfilePhotoFileID); err != nil && !errors.Is(err, domain.ErrNotFound) {
				writeError(w, err)
				return
			}
		}
		writeJSON(w, http.StatusOK, staff)
		return
	}

	if r.Method != http.MethodPatch {
		writeError(w, domain.ErrNotFound)
		return
	}

	var input academic.UpdateStaffInput
	if !decodeJSON(w, r, &input) {
		return
	}
	input.ID = parts[0]
	staff, err := s.academic.UpdateStaffMember(r.Context(), principal, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, staff)
}

func (s *Server) createStudent(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input academic.CreateStudentInput
	if !decodeJSON(w, r, &input) {
		return
	}
	student, err := s.academic.CreateStudent(r.Context(), principal, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, student)
}

func (s *Server) listStudents(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	students, err := s.academic.ListStudents(r.Context(), principal, r.URL.Query().Get("branch_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	status := domain.StudentStatus(strings.TrimSpace(r.URL.Query().Get("status")))
	if status != "" {
		filtered := make([]domain.Student, 0, len(students))
		for _, student := range students {
			if student.Status == status {
				filtered = append(filtered, student)
			}
		}
		students = filtered
	}
	writePagedJSON(w, r, students)
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
	if len(parts) == 2 && parts[1] == "account" && r.Method == http.MethodPost {
		var input academic.CreateStudentAccountInput
		if !decodeJSON(w, r, &input) {
			return
		}
		student, user, err := s.academic.CreateStudentAccount(r.Context(), principal, parts[0], input)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"student": student, "user": user})
		return
	}

	if path == "" {
		writeError(w, domain.ErrNotFound)
		return
	}
	writeError(w, domain.ErrNotFound)
}

func (s *Server) registerTeacher(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input academic.RegisterTeacherInput
	if !decodeJSON(w, r, &input) {
		return
	}
	teacher, user, err := s.academic.RegisterTeacher(r.Context(), principal, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"teacher": teacher, "user": user})
}

func (s *Server) listTeachers(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	teachers, err := s.academic.ListTeachers(r.Context(), principal, r.URL.Query().Get("branch_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	status := domain.TeacherStatus(strings.TrimSpace(r.URL.Query().Get("status")))
	if status != "" {
		filtered := make([]domain.Teacher, 0, len(teachers))
		for _, teacher := range teachers {
			if teacher.Status == status {
				filtered = append(filtered, teacher)
			}
		}
		teachers = filtered
	}
	writePagedJSON(w, r, teachers)
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
	if r.Method != http.MethodPost || len(parts) != 2 || parts[1] != "activate" {
		writeError(w, domain.ErrNotFound)
		return
	}
	teacher, err := s.academic.ActivateTeacher(r.Context(), principal, parts[0])
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, teacher)
}

func (s *Server) createCourse(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input academic.CreateCourseInput
	if !decodeJSON(w, r, &input) {
		return
	}
	course, categories, err := s.academic.CreateCourse(r.Context(), principal, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"course": course, "categories": categories})
}

func (s *Server) listCourses(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	courses, err := s.academic.ListCourses(r.Context(), principal, r.URL.Query().Get("branch_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writePagedJSON(w, r, courses)
}

func (s *Server) courseAction(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/courses/"), "/")
	if id == "" {
		writeError(w, domain.ErrNotFound)
		return
	}
	course, err := s.academic.GetCourse(r.Context(), principal, id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, course)
}

func (s *Server) createClass(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input academic.CreateClassInput
	if !decodeJSON(w, r, &input) {
		return
	}
	class, err := s.academic.CreateClass(r.Context(), principal, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, class)
}

func (s *Server) listClasses(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	classes, err := s.academic.ListClasses(r.Context(), principal, r.URL.Query().Get("branch_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if rawActive := strings.TrimSpace(r.URL.Query().Get("active")); rawActive != "" {
		active, ok := parseBoolQuery(rawActive)
		if !ok {
			writeError(w, domain.ErrInvalidInput)
			return
		}
		filtered := make([]domain.Class, 0, len(classes))
		for _, class := range classes {
			if class.IsActive == active {
				filtered = append(filtered, class)
			}
		}
		classes = filtered
	}
	writePagedJSON(w, r, classes)
}

func (s *Server) classAction(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/classes/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if r.Method == http.MethodGet && len(parts) == 1 && parts[0] != "" {
		class, err := s.academic.GetClass(r.Context(), principal, parts[0])
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, class)
		return
	}
	if len(parts) != 2 {
		writeError(w, domain.ErrNotFound)
		return
	}

	switch {
	case parts[1] == "students" && r.Method == http.MethodGet:
		enrollments, err := s.academic.ListClassStudents(r.Context(), principal, parts[0])
		if err != nil {
			writeError(w, err)
			return
		}
		writePagedJSON(w, r, enrollments)
	case parts[1] == "students" && r.Method == http.MethodPost:
		var input academic.EnrollStudentInput
		if !decodeJSON(w, r, &input) {
			return
		}
		input.ClassID = parts[0]
		enrollment, err := s.academic.EnrollStudent(r.Context(), principal, input)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, enrollment)
	case parts[1] == "assignments" && r.Method == http.MethodGet:
		assignments, err := s.academic.ListAssignments(r.Context(), principal, "", parts[0])
		if err != nil {
			writeError(w, err)
			return
		}
		writePagedJSON(w, r, assignments)
	case parts[1] == "assignments" && r.Method == http.MethodPost:
		var input academic.CreateAssignmentInput
		if !decodeJSON(w, r, &input) {
			return
		}
		input.ClassID = parts[0]
		assignment, err := s.academic.CreateAssignment(r.Context(), principal, input)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, assignment)
	default:
		writeError(w, domain.ErrNotFound)
	}
}

func (s *Server) createAssignment(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input academic.CreateAssignmentInput
	if !decodeJSON(w, r, &input) {
		return
	}
	assignment, err := s.academic.CreateAssignment(r.Context(), principal, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, assignment)
}

func (s *Server) listAssignments(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	assignments, err := s.academic.ListAssignments(
		r.Context(),
		principal,
		r.URL.Query().Get("branch_id"),
		r.URL.Query().Get("class_id"),
	)
	if err != nil {
		writeError(w, err)
		return
	}
	writePagedJSON(w, r, assignments)
}

func (s *Server) createRoom(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input academic.CreateRoomInput
	if !decodeJSON(w, r, &input) {
		return
	}
	room, err := s.academic.CreateRoom(r.Context(), principal, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, room)
}

func (s *Server) listRooms(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	rooms, err := s.academic.ListRooms(r.Context(), principal, r.URL.Query().Get("branch_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writePagedJSON(w, r, rooms)
}

func (s *Server) roomAction(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/rooms/"), "/")
	if id == "" || r.Method != http.MethodDelete {
		writeError(w, domain.ErrNotFound)
		return
	}

	room, err := s.academic.RemoveRoom(r.Context(), principal, id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, room)
}

func (s *Server) createScheduleItem(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input academic.CreateScheduleItemInput
	if !decodeJSON(w, r, &input) {
		return
	}
	item, err := s.academic.CreateScheduleItem(r.Context(), principal, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) listSchedule(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	items, err := s.academic.ListSchedule(r.Context(), principal, r.URL.Query().Get("branch_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	itemType := domain.ScheduleItemType(strings.TrimSpace(r.URL.Query().Get("item_type")))
	if itemType != "" {
		filtered := make([]domain.ScheduleItem, 0, len(items))
		for _, item := range items {
			if item.ItemType == itemType {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	writePagedJSON(w, r, items)
}

func (s *Server) createExam(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input academic.CreateExamInput
	if !decodeJSON(w, r, &input) {
		return
	}
	exam, participants, err := s.academic.CreateExam(r.Context(), principal, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"exam": exam, "participants": participants})
}

func (s *Server) listExams(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	exams, err := s.academic.ListExams(
		r.Context(),
		principal,
		r.URL.Query().Get("branch_id"),
		r.URL.Query().Get("class_id"),
	)
	if err != nil {
		writeError(w, err)
		return
	}
	writePagedJSON(w, r, exams)
}

func (s *Server) examAction(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/exams/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if r.Method == http.MethodGet && len(parts) == 1 && parts[0] != "" {
		exam, err := s.academic.GetExam(r.Context(), principal, parts[0])
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, exam)
		return
	}
	if len(parts) != 2 || parts[1] != "results" {
		writeError(w, domain.ErrNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		results, err := s.academic.ListExamResults(r.Context(), principal, r.URL.Query().Get("branch_id"), parts[0], r.URL.Query().Get("student_id"))
		if err != nil {
			writeError(w, err)
			return
		}
		writePagedJSON(w, r, results)
	case http.MethodPost:
		var input academic.CreateExamResultInput
		if !decodeJSON(w, r, &input) {
			return
		}
		input.ExamID = parts[0]
		result, err := s.academic.CreateExamResult(r.Context(), principal, input)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, result)
	default:
		writeError(w, domain.ErrNotFound)
	}
}

func (s *Server) listExamResults(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	results, err := s.academic.ListExamResults(
		r.Context(),
		principal,
		r.URL.Query().Get("branch_id"),
		r.URL.Query().Get("exam_id"),
		r.URL.Query().Get("student_id"),
	)
	if err != nil {
		writeError(w, err)
		return
	}
	writePagedJSON(w, r, results)
}

func (s *Server) academicDashboard(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	dashboard, err := s.academic.AcademicDashboard(r.Context(), principal, r.URL.Query().Get("branch_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dashboard)
}

func (s *Server) createPayment(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input finance.CreatePaymentInput
	if !decodeJSON(w, r, &input) {
		return
	}
	payment, err := s.finance.CreatePayment(r.Context(), principal, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, payment)
}

func (s *Server) listPayments(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	payments, err := s.finance.ListPayments(r.Context(), principal, r.URL.Query().Get("branch_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	status := domain.PaymentStatus(strings.TrimSpace(r.URL.Query().Get("status")))
	if status != "" {
		filtered := make([]domain.Payment, 0, len(payments))
		for _, payment := range payments {
			if payment.Status == status {
				filtered = append(filtered, payment)
			}
		}
		payments = filtered
	}
	writePagedJSON(w, r, payments)
}

func (s *Server) paymentAction(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/payments/"), "/")
	if id == "" {
		writeError(w, domain.ErrNotFound)
		return
	}
	payment, err := s.finance.GetPayment(r.Context(), principal, id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, payment)
}

func (s *Server) createSalaryModel(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input finance.CreateSalaryModelInput
	if !decodeJSON(w, r, &input) {
		return
	}
	model, err := s.finance.CreateSalaryModel(r.Context(), principal, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, model)
}

func (s *Server) listSalaryModels(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	models, err := s.finance.ListSalaryModels(r.Context(), principal, r.URL.Query().Get("branch_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writePagedJSON(w, r, models)
}

func (s *Server) calculateSwapAllocation(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input finance.SwapAllocationInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := s.finance.CalculateSwapAllocation(r.Context(), principal, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) registerFile(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input files.RegisterFileInput
	if !decodeJSON(w, r, &input) {
		return
	}
	file, err := s.files.RegisterFile(r.Context(), principal, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, file)
}

func (s *Server) uploadFile(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	r.Body = http.MaxBytesReader(w, r.Body, 12<<20)
	if err := r.ParseMultipartForm(12 << 20); err != nil {
		writeError(w, domain.ErrInvalidInput)
		return
	}

	content, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, domain.ErrInvalidInput)
		return
	}
	defer content.Close()
	if header.Size > maxUploadFileBytes {
		writeError(w, domain.ErrInvalidInput)
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if strings.TrimSpace(mimeType) == "" {
		mimeType = "application/octet-stream"
	}

	file, err := s.files.UploadFile(r.Context(), principal, files.UploadFileInput{
		BranchID:         r.FormValue("branch_id"),
		OwnerType:        r.FormValue("owner_type"),
		OwnerID:          r.FormValue("owner_id"),
		Category:         domain.FileCategory(r.FormValue("category")),
		Purpose:          domain.FilePurpose(r.FormValue("purpose")),
		OriginalFilename: header.Filename,
		MimeType:         mimeType,
		Size:             header.Size,
		Content:          content,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, file)
}

func (s *Server) fileAction(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/files/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if r.Method == http.MethodGet && len(parts) == 1 && parts[0] != "" {
		file, err := s.files.GetFile(r.Context(), principal, parts[0])
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, file)
		return
	}
	if r.Method == http.MethodDelete && len(parts) == 1 && parts[0] != "" {
		file, err := s.files.DeleteFile(r.Context(), principal, parts[0])
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, file)
		return
	}
	if len(parts) != 2 || parts[1] != "download-url" {
		writeError(w, domain.ErrNotFound)
		return
	}

	result, err := s.files.CreateDownloadURL(r.Context(), principal, parts[0])
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) cleanupFileRetention(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	limit := 100
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil {
			writeError(w, domain.ErrInvalidInput)
			return
		}
		limit = parsed
	}

	result, err := s.files.CleanupExpiredFiles(r.Context(), principal, limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) listFiles(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	files, err := s.files.ListFiles(r.Context(), principal, r.URL.Query().Get("branch_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	ownerType := strings.TrimSpace(r.URL.Query().Get("owner_type"))
	ownerID := strings.TrimSpace(r.URL.Query().Get("owner_id"))
	purpose := domain.FilePurpose(strings.TrimSpace(r.URL.Query().Get("purpose")))
	if ownerType != "" || ownerID != "" || purpose != "" {
		filtered := make([]domain.FileObject, 0, len(files))
		for _, file := range files {
			if ownerType != "" && file.OwnerType != ownerType {
				continue
			}
			if ownerID != "" && file.OwnerID != ownerID {
				continue
			}
			if purpose != "" && file.Purpose != purpose {
				continue
			}
			filtered = append(filtered, file)
		}
		files = filtered
	}
	writePagedJSON(w, r, files)
}

func (s *Server) listNotifications(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	unreadOnly := strings.EqualFold(r.URL.Query().Get("unread_only"), "true")
	notifications, err := s.notify.ListNotifications(r.Context(), principal, unreadOnly)
	if err != nil {
		writeError(w, err)
		return
	}
	writePagedJSON(w, r, notifications)
}

func (s *Server) notificationAction(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/notifications/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[1] != "read" {
		writeError(w, domain.ErrNotFound)
		return
	}

	notification, err := s.notify.MarkNotificationRead(r.Context(), principal, parts[0])
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, notification)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, out any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		writeError(w, domain.ErrInvalidInput)
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeError(w, domain.ErrInvalidInput)
		return false
	}

	return true
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "internal error"

	switch {
	case errors.Is(err, domain.ErrInvalidInput), errors.Is(err, domain.ErrInvalidFIN):
		status = http.StatusBadRequest
		message = err.Error()
	case errors.Is(err, domain.ErrUnauthorized), auth.IsAuthError(err):
		status = http.StatusUnauthorized
		message = "unauthorized"
	case errors.Is(err, domain.ErrForbidden):
		status = http.StatusForbidden
		message = "forbidden"
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
		message = "not found"
	case errors.Is(err, domain.ErrConflict):
		status = http.StatusConflict
		message = "conflict"
	}

	writeJSON(w, status, map[string]string{"error": message})
}

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
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
	idem     IdempotencyStore
	audit    AuditStore
	logger   *zap.Logger
	mux      *http.ServeMux
	options  Options
	limiter  *rateLimiter
	metrics  *requestMetrics
}

const maxUploadFileBytes int64 = 15 << 20

func New(
	authService *auth.Service,
	academicService *academic.Service,
	financeService *finance.Service,
	filesService *files.Service,
	notificationService *notification.Service,
	adminService *admin.Service,
	idempotencyStore IdempotencyStore,
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
		idem:     idempotencyStore,
		logger:   logger,
		mux:      http.NewServeMux(),
		options:  options,
		limiter:  newRateLimiter(options.RateLimitPerMinute),
		metrics:  newRequestMetrics(),
	}
	if auditStore, ok := idempotencyStore.(AuditStore); ok {
		s.audit = auditStore
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	var handler http.Handler = s.mux
	handler = s.withRecovery(handler)
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
	s.mux.HandleFunc("POST /v1/auth/logout", s.requireAuth(s.logout))
	s.mux.HandleFunc("GET /v1/me", s.requireAuth(s.me))
	s.mux.HandleFunc("GET /v1/session", s.requireAuth(s.session))
	s.mux.HandleFunc("GET /v1/users/email-availability", s.requireAuth(s.emailAvailability))

	s.mux.HandleFunc("GET /v1/branches", s.requireAuth(s.listBranches))
	s.mux.HandleFunc("POST /v1/branches", s.requireAuth(s.createBranch))
	s.mux.HandleFunc("GET /v1/branches/", s.requireAuth(s.branchAction))
	s.mux.HandleFunc("POST /v1/branches/", s.requireAuth(s.branchAction))
	s.mux.HandleFunc("PATCH /v1/branches/", s.requireAuth(s.branchAction))
	s.mux.HandleFunc("DELETE /v1/branches/", s.requireAuth(s.branchAction))
	s.mux.HandleFunc("PATCH /v1/staff/", s.requireAuth(s.staffAction))
	s.mux.HandleFunc("DELETE /v1/staff/", s.requireAuth(s.staffAction))

	s.mux.HandleFunc("GET /v1/student-assignment-hub", s.requireAuth(s.listStudentAssignmentHub))
	s.mux.HandleFunc("GET /v1/students", s.requireAuth(s.listStudents))
	s.mux.HandleFunc("POST /v1/students", s.requireAuth(s.createStudent))
	s.mux.HandleFunc("GET /v1/students/by-fin/", s.requireAuth(s.getStudentByFIN))
	s.mux.HandleFunc("GET /v1/students/", s.requireAuth(s.studentAction))
	s.mux.HandleFunc("POST /v1/students/", s.requireAuth(s.studentAction))

	s.mux.HandleFunc("GET /v1/teachers", s.requireAuth(s.listTeachers))
	s.mux.HandleFunc("POST /v1/teachers", s.requireAuth(s.registerTeacher))
	s.mux.HandleFunc("GET /v1/teachers/", s.requireAuth(s.teacherAction))
	s.mux.HandleFunc("POST /v1/teachers/", s.requireAuth(s.teacherAction))
	s.mux.HandleFunc("PATCH /v1/teachers/", s.requireAuth(s.teacherAction))
	s.mux.HandleFunc("DELETE /v1/teachers/", s.requireAuth(s.teacherAction))
	s.mux.HandleFunc("GET /v1/teacher-finance", s.requireAuth(s.listTeacherFinance))

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
	s.mux.HandleFunc("PATCH /v1/rooms/", s.requireAuth(s.roomAction))
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

		principal, err := s.auth.AuthenticateToken(r.Context(), raw)
		if err != nil {
			writeError(w, domain.ErrUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), principalKey{}, principal)
		ctx = context.WithValue(ctx, auditDetailsKey{}, &auditDetails{})
		req := r.WithContext(ctx)
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next(recorder, req, principal)
		s.recordAudit(req, principal, recorder.status, recorder.Body())
	}
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" {
			w.Header().Add("Vary", "Origin")
			allowedOrigin := s.allowedCORSOrigin(origin)
			if allowedOrigin == "" {
				if r.Method == http.MethodOptions {
					writeError(w, domain.ErrForbidden)
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID, Idempotency-Key")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Expose-Headers", "X-Total-Count, X-Limit, X-Offset, X-Request-ID")
		}
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

func (s *Server) logout(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	if err := s.auth.Logout(r.Context(), principal); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	user, err := s.auth.CurrentUser(r.Context(), principal)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) emailAvailability(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	result, err := s.auth.CheckEmailAvailability(r.Context(), principal, r.URL.Query().Get("email"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, out any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer func() { _ = r.Body.Close() }()
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

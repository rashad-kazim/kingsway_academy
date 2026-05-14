package httpapi

import (
	"context"
	"net/http"
	"strings"

	"kingsway/backend/internal/academic"
	"kingsway/backend/internal/domain"
)

func (s *Server) createCourse(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input academic.CreateCourseInput
	s.decodeIdempotentJSON(w, r, principal, &input, func(ctx context.Context) (int, any, error) {
		course, categories, err := s.academic.CreateCourse(ctx, principal, input)
		return http.StatusCreated, map[string]any{"course": course, "categories": categories}, err
	})
}

func (s *Server) listCourses(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	page, err := parsePage(r)
	if err != nil {
		writeError(w, err)
		return
	}
	courses, total, err := s.academic.ListCoursesPage(r.Context(), principal, r.URL.Query().Get("branch_id"), page.domainPage())
	if err != nil {
		writeError(w, err)
		return
	}
	writePageJSON(w, page, total, courses)
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
	s.decodeIdempotentJSON(w, r, principal, &input, func(ctx context.Context) (int, any, error) {
		class, err := s.academic.CreateClass(ctx, principal, input)
		return http.StatusCreated, class, err
	})
}

func (s *Server) listClasses(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	page, err := parsePage(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var active *bool
	if rawActive := strings.TrimSpace(r.URL.Query().Get("active")); rawActive != "" {
		parsedActive, ok := parseBoolQuery(rawActive)
		if !ok {
			writeError(w, domain.ErrInvalidInput)
			return
		}
		active = &parsedActive
	}
	classes, total, err := s.academic.ListClassesPage(r.Context(), principal, r.URL.Query().Get("branch_id"), active, page.domainPage())
	if err != nil {
		writeError(w, err)
		return
	}
	writePageJSON(w, page, total, classes)
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
		page, err := parsePage(r)
		if err != nil {
			writeError(w, err)
			return
		}
		enrollments, total, err := s.academic.ListClassStudentsPage(r.Context(), principal, parts[0], page.domainPage())
		if err != nil {
			writeError(w, err)
			return
		}
		writePageJSON(w, page, total, enrollments)
	case parts[1] == "students" && r.Method == http.MethodPost:
		var input academic.EnrollStudentInput
		s.decodeIdempotentJSON(w, r, principal, &input, func(ctx context.Context) (int, any, error) {
			input.ClassID = parts[0]
			enrollment, err := s.academic.EnrollStudent(ctx, principal, input)
			return http.StatusCreated, enrollment, err
		})
	case parts[1] == "assignments" && r.Method == http.MethodGet:
		page, err := parsePage(r)
		if err != nil {
			writeError(w, err)
			return
		}
		assignments, total, err := s.academic.ListAssignmentsPage(r.Context(), principal, "", parts[0], page.domainPage())
		if err != nil {
			writeError(w, err)
			return
		}
		writePageJSON(w, page, total, assignments)
	case parts[1] == "assignments" && r.Method == http.MethodPost:
		var input academic.CreateAssignmentInput
		s.decodeIdempotentJSON(w, r, principal, &input, func(ctx context.Context) (int, any, error) {
			input.ClassID = parts[0]
			assignment, err := s.academic.CreateAssignment(ctx, principal, input)
			return http.StatusCreated, assignment, err
		})
	default:
		writeError(w, domain.ErrNotFound)
	}
}

func (s *Server) createAssignment(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input academic.CreateAssignmentInput
	s.decodeIdempotentJSON(w, r, principal, &input, func(ctx context.Context) (int, any, error) {
		assignment, err := s.academic.CreateAssignment(ctx, principal, input)
		return http.StatusCreated, assignment, err
	})
}

func (s *Server) listAssignments(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	page, err := parsePage(r)
	if err != nil {
		writeError(w, err)
		return
	}
	assignments, total, err := s.academic.ListAssignmentsPage(
		r.Context(),
		principal,
		r.URL.Query().Get("branch_id"),
		r.URL.Query().Get("class_id"),
		page.domainPage(),
	)
	if err != nil {
		writeError(w, err)
		return
	}
	writePageJSON(w, page, total, assignments)
}

func (s *Server) createRoom(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input academic.CreateRoomInput
	s.decodeIdempotentJSON(w, r, principal, &input, func(ctx context.Context) (int, any, error) {
		room, err := s.academic.CreateRoom(ctx, principal, input)
		return http.StatusCreated, room, err
	})
}

func (s *Server) listRooms(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	page, err := parsePage(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rooms, total, err := s.academic.ListRoomsPage(r.Context(), principal, r.URL.Query().Get("branch_id"), page.domainPage())
	if err != nil {
		writeError(w, err)
		return
	}
	writePageJSON(w, page, total, rooms)
}

func (s *Server) roomAction(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v1/rooms/"), "/")
	if id == "" {
		writeError(w, domain.ErrNotFound)
		return
	}

	switch r.Method {
	case http.MethodPatch:
		var input academic.UpdateRoomInput
		s.decodeIdempotentJSON(w, r, principal, &input, func(ctx context.Context) (int, any, error) {
			input.ID = id
			beforeRoom, err := s.academic.GetRoom(ctx, principal, id)
			if err != nil {
				return 0, nil, err
			}
			setAuditBefore(ctx, beforeRoom)
			setAuditEntity(ctx, "rooms", id, beforeRoom.BranchID)
			room, err := s.academic.UpdateRoom(ctx, principal, input)
			return http.StatusOK, room, err
		})
	case http.MethodDelete:
		s.writeIdempotentNoBodyJSON(w, r, principal, func(ctx context.Context) (int, any, error) {
			beforeRoom, err := s.academic.GetRoom(ctx, principal, id)
			if err != nil {
				return 0, nil, err
			}
			setAuditBefore(ctx, beforeRoom)
			setAuditEntity(ctx, "rooms", id, beforeRoom.BranchID)
			room, err := s.academic.RemoveRoom(ctx, principal, id)
			return http.StatusOK, room, err
		})
	default:
		writeError(w, domain.ErrNotFound)
	}
}

func (s *Server) createScheduleItem(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input academic.CreateScheduleItemInput
	s.decodeIdempotentJSON(w, r, principal, &input, func(ctx context.Context) (int, any, error) {
		item, err := s.academic.CreateScheduleItem(ctx, principal, input)
		return http.StatusCreated, item, err
	})
}

func (s *Server) listSchedule(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	page, err := parsePage(r)
	if err != nil {
		writeError(w, err)
		return
	}
	itemType := domain.ScheduleItemType(strings.TrimSpace(r.URL.Query().Get("item_type")))
	items, total, err := s.academic.ListSchedulePage(r.Context(), principal, r.URL.Query().Get("branch_id"), itemType, page.domainPage())
	if err != nil {
		writeError(w, err)
		return
	}
	writePageJSON(w, page, total, items)
}

func (s *Server) createExam(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	var input academic.CreateExamInput
	s.decodeIdempotentJSON(w, r, principal, &input, func(ctx context.Context) (int, any, error) {
		exam, participants, err := s.academic.CreateExam(ctx, principal, input)
		return http.StatusCreated, map[string]any{"exam": exam, "participants": participants}, err
	})
}

func (s *Server) listExams(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	page, err := parsePage(r)
	if err != nil {
		writeError(w, err)
		return
	}
	exams, total, err := s.academic.ListExamsPage(
		r.Context(),
		principal,
		r.URL.Query().Get("branch_id"),
		r.URL.Query().Get("class_id"),
		page.domainPage(),
	)
	if err != nil {
		writeError(w, err)
		return
	}
	writePageJSON(w, page, total, exams)
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
		page, err := parsePage(r)
		if err != nil {
			writeError(w, err)
			return
		}
		results, total, err := s.academic.ListExamResultsPage(r.Context(), principal, r.URL.Query().Get("branch_id"), parts[0], r.URL.Query().Get("student_id"), page.domainPage())
		if err != nil {
			writeError(w, err)
			return
		}
		writePageJSON(w, page, total, results)
	case http.MethodPost:
		var input academic.CreateExamResultInput
		s.decodeIdempotentJSON(w, r, principal, &input, func(ctx context.Context) (int, any, error) {
			input.ExamID = parts[0]
			result, err := s.academic.CreateExamResult(ctx, principal, input)
			return http.StatusCreated, result, err
		})
	default:
		writeError(w, domain.ErrNotFound)
	}
}

func (s *Server) listExamResults(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	page, err := parsePage(r)
	if err != nil {
		writeError(w, err)
		return
	}
	results, total, err := s.academic.ListExamResultsPage(
		r.Context(),
		principal,
		r.URL.Query().Get("branch_id"),
		r.URL.Query().Get("exam_id"),
		r.URL.Query().Get("student_id"),
		page.domainPage(),
	)
	if err != nil {
		writeError(w, err)
		return
	}
	writePageJSON(w, page, total, results)
}

func (s *Server) academicDashboard(w http.ResponseWriter, r *http.Request, principal domain.Principal) {
	dashboard, err := s.academic.AcademicDashboard(r.Context(), principal, r.URL.Query().Get("branch_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dashboard)
}

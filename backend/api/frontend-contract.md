# Kingsway Frontend API Contract

Base URL: `http://127.0.0.1:8080`

Authentication: send `Authorization: Bearer <token>` for every `/v1/*` endpoint except `POST /v1/auth/bootstrap-owner` and `POST /v1/auth/login`.

Every response includes `X-Request-ID`. The frontend may send its own `X-Request-ID` for correlation.

All list endpoints accept:

- `limit`: integer `1..500`, default `100`
- `offset`: integer `>=0`, default `0`

List responses are JSON arrays and include headers:

- `X-Total-Count`
- `X-Limit`
- `X-Offset`

Errors use:

```json
{ "error": "invalid input" }
```

## Auth And Session

- `POST /v1/auth/login`
  - Request: `{ "email": "owner@example.com", "password": "password123" }`
  - Response: `{ "token": "...", "user": User }`
- `GET /v1/me`
  - Response: `User`
- `GET /v1/session`
  - Response: `{ "user": User, "principal": Principal, "branch": Branch?, "capabilities": string[], "dashboard_path": string }`

## Role Dashboards

- `GET /v1/dashboard/owner?branch_id=optional`
- `GET /v1/dashboard/receptionist`
- `GET /v1/dashboard/teacher`
- `GET /v1/dashboard/student`
- `GET /v1/dashboard/academic?branch_id=optional`

## Stable List And Detail Endpoints

- Branches: `GET /v1/branches`
- Students: `GET /v1/students?branch_id=&status=active`, `GET /v1/students/{student_id}`, `GET /v1/students/by-fin/{fin}`
- Student account: `POST /v1/students/{student_id}/account` creates a student login and links it to the student record.
- Teachers: `GET /v1/teachers?branch_id=&status=active`, `GET /v1/teachers/{teacher_id}`
- Courses: `GET /v1/courses?branch_id=`, `GET /v1/courses/{course_id}`
- Classes: `GET /v1/classes?branch_id=&active=true`, `GET /v1/classes/{class_id}`
- Class students: `GET /v1/classes/{class_id}/students`
- Assignments: `GET /v1/assignments?branch_id=&class_id=`, `GET /v1/classes/{class_id}/assignments`
- Rooms: `GET /v1/rooms?branch_id=`
- Schedule: `GET /v1/schedules?branch_id=&item_type=lesson`
- Exams: `GET /v1/exams?branch_id=&class_id=`, `GET /v1/exams/{exam_id}`
- Exam results: `GET /v1/exam-results?branch_id=&exam_id=&student_id=`, `GET /v1/exams/{exam_id}/results`
- Payments: `GET /v1/payments?branch_id=&status=pending`, `GET /v1/payments/{payment_id}`
- Files: `GET /v1/files?branch_id=&owner_type=&owner_id=&purpose=`, `GET /v1/files/{file_id}`
- Notifications: `GET /v1/notifications?unread_only=true`, `POST /v1/notifications/{notification_id}/read`

## Operational Endpoints

- Public: `GET /healthz`, `GET /readyz`, `GET /metrics`
- Owner-only outbox inspection: `GET /v1/admin/outbox?status=failed`
- Owner-only outbox retry: `POST /v1/admin/outbox/{event_id}/retry`

The OpenAPI contract is tracked at `api/openapi.yaml`.

## Frontend Routing Contract

After login, call `GET /v1/session` and route by `dashboard_path`:

- `owner` -> `/dashboard/owner`
- `receptionist` -> `/dashboard/receptionist`
- `teacher` -> `/dashboard/teacher`
- `student` -> `/dashboard/student`

The frontend should treat `capabilities` as feature switches and should not infer access from route names alone.

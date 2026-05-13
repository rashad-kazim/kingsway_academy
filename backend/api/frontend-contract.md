# Kingsway Frontend API Contract

Base URL: `http://127.0.0.1:8080`

Authentication: send `Authorization: Bearer <token>` for every `/v1/*` endpoint except `POST /v1/auth/bootstrap-owner` and `POST /v1/auth/login`.

Every response includes `X-Request-ID`. The frontend may send its own `X-Request-ID` for correlation.

Write/create-style JSON `POST`, critical `PATCH`, critical `DELETE`, and multipart upload endpoints accept optional `Idempotency-Key`. The frontend should generate one key per submit attempt and reuse it for safe retry of the same payload. Repeating the same key and same body returns the original successful response; repeating the same key with a different body returns `409 conflict`.

All list endpoints accept:

- `limit`: integer `1..500`, default `100`
- `offset`: integer `>=0`, default `0`

Most list responses are JSON arrays and include headers:

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
- `POST /v1/auth/logout`
  - Requires bearer token.
  - Response: `{ "ok": true }`
  - Effect: increments/revokes the current user's token version so existing tokens become invalid.
- `GET /v1/me`
  - Response: `User`
- `GET /v1/session`
  - Response: `{ "user": User, "principal": Principal, "branch": Branch?, "capabilities": string[], "dashboard_path": string }`
- `GET /v1/users/email-availability?email=user@example.com`
  - Requires bearer token.
  - Response: `{ "email": "user@example.com", "available": true }`

## Role Dashboards

- `GET /v1/dashboard/owner?branch_id=optional`
- `GET /v1/dashboard/receptionist`
- `GET /v1/dashboard/teacher`
- `GET /v1/dashboard/student`
- `GET /v1/dashboard/academic?branch_id=optional`

## Stable List And Detail Endpoints

- Branches: `GET /v1/branches`, `POST /v1/branches`, `PATCH /v1/branches/{branch_id}`, `DELETE /v1/branches/{branch_id}`
- Branch staff / Receptionist Management: `GET /v1/branches/{branch_id}/staff`, `POST /v1/branches/{branch_id}/staff`, `PATCH /v1/staff/{staff_id}`, `DELETE /v1/staff/{staff_id}`. Staff records include gender, address, hire date, active/inactive status, last login, salary, and profile photo file reference. `PATCH` can update profile fields, branch assignment, and `is_active`; `DELETE` is a hard delete.
- Students: `GET /v1/students?branch_id=&status=active`, `POST /v1/students`, `GET /v1/students/{student_id}`, `PATCH /v1/students/{student_id}`, `GET /v1/students/by-fin/{fin}`. Student create now accepts profile fields, parent contacts, selected course registrations, optional teacher assignment per course, and monthly payment amount per course; student profile photos use `POST /v1/files/upload` with `owner_type=student` and `purpose=profile_photo`.
- Student & Assignment Hub: `GET /v1/student-assignment-hub?branch_id=&status=&teacher_id=&q=&limit=&offset=` returns `{ items, total, limit, offset }` for the combined Owner list. Status values are `active`, `left`, and `graduated`; `q` searches first name, last name, full name, and FIN code. This does not replace create/detail/enrollment endpoints.
- Student account: `POST /v1/students/{student_id}/account` creates a student login and links it to the student record.
- Teachers: `GET /v1/teachers?branch_id=&status=active`, `GET /v1/teachers/{teacher_id}`, `PATCH /v1/teachers/{teacher_id}` updates teacher profile/login/subjects, `DELETE /v1/teachers/{teacher_id}` hard-deletes a teacher when no active students are assigned.
- Teacher Finance & HR: `GET /v1/teacher-finance?branch_id=&subject=&status=&salary_model=` returns the Owner table rows with profile, profile photo file reference, branch, subject, status, salary type, assigned student count, and calculated salary cache value.
- Courses: `GET /v1/courses?branch_id=`, `GET /v1/courses/{course_id}`
- Classes: `GET /v1/classes?branch_id=&active=true`, `GET /v1/classes/{class_id}`
- Class students: `GET /v1/classes/{class_id}/students`
- Assignments: `GET /v1/assignments?branch_id=&class_id=`, `GET /v1/classes/{class_id}/assignments`
- Rooms: `GET /v1/rooms?branch_id=`, `POST /v1/rooms`, `PATCH /v1/rooms/{room_id}`, `DELETE /v1/rooms/{room_id}`
- Schedule: `GET /v1/schedules?branch_id=&item_type=lesson`
- Exams: `GET /v1/exams?branch_id=&class_id=`, `GET /v1/exams/{exam_id}`
- Exam results: `GET /v1/exam-results?branch_id=&exam_id=&student_id=`, `GET /v1/exams/{exam_id}/results`
- Payments: `GET /v1/payments?branch_id=&status=pending`, `GET /v1/payments/{payment_id}`
- Files: `GET /v1/files?branch_id=&owner_type=&owner_id=&purpose=`, `GET /v1/files/{file_id}`, `DELETE /v1/files/{file_id}`, `GET /v1/files/{file_id}/download-url`, `POST /v1/files/upload`
- Notifications: `GET /v1/notifications?unread_only=true`, `POST /v1/notifications/{notification_id}/read`

## File Upload Policy

- Maximum upload size is 15 MB per file.
- Ask before adding any new file upload UI which category the file belongs to, and document upload-time optimization plus download-time behavior before implementation.
- Lossy optimization cannot be reversed. If a file must later download in original visual/content quality, store the original or a content-preserving sanitized object; do not rely on reversing compression.
- Backend stores both legacy `category` and normalized `policy`.
- `standard-ui`: UI/avatar/profile images. Backend keeps only the optimized display asset. No original download promise.
- `standard-downloadable`: normal downloadable file. Backend policy allows original-plus-preview behavior; download should return the original or a content-preserving sanitized object when original quality matters.
- `special`: very important official files. Visible/content data must not be altered. PDFs may have non-content metadata removed; JPG/PNG/WebP special images should not be recompressed or quality-ratio converted. At-rest encryption is required in the target architecture.
- If `policy` is omitted, backend infers it: `special` category -> `special`, `standard + profile_photo` -> `standard-ui`, otherwise `standard-downloadable`.
- `standard-ui` + `profile_photo` image uploads are optimized on the backend for UI use: auto-orientation, center square crop, 768x768 resize/upscale, metadata stripping, and JPEG quality 85 output. This is an optimized display asset and is not intended to recreate the original upload.

## Operational Endpoints

- Public: `GET /healthz`, `GET /readyz`, `GET /metrics`
- Owner-only outbox inspection: `GET /v1/admin/outbox?status=failed`
- Owner-only outbox retry: `POST /v1/admin/outbox/{event_id}/retry`

The OpenAPI contract is tracked at `api/openapi.yaml`.
Current pre-large-update API baseline: `api/baseline-2026-05-13.md`.

## Frontend Routing Contract

After login, call `GET /v1/session` and route by `dashboard_path`:

- `owner` -> `/dashboard/owner`
- `receptionist` -> `/dashboard/receptionist`
- `teacher` -> `/dashboard/teacher`
- `student` -> `/dashboard/student`

The frontend should treat `capabilities` as feature switches and should not infer access from route names alone.

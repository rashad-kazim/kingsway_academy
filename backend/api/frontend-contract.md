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

- Branches: `GET /v1/branches`, `POST /v1/branches`, `PATCH /v1/branches/{branch_id}`, `DELETE /v1/branches/{branch_id}`
- Branch staff: `GET /v1/branches/{branch_id}/staff`, `POST /v1/branches/{branch_id}/staff`, `PATCH /v1/staff/{staff_id}`, `DELETE /v1/staff/{staff_id}`
- Students: `GET /v1/students?branch_id=&status=active`, `GET /v1/students/{student_id}`, `GET /v1/students/by-fin/{fin}`
- Student account: `POST /v1/students/{student_id}/account` creates a student login and links it to the student record.
- Teachers: `GET /v1/teachers?branch_id=&status=active`, `GET /v1/teachers/{teacher_id}`
- Courses: `GET /v1/courses?branch_id=`, `GET /v1/courses/{course_id}`
- Classes: `GET /v1/classes?branch_id=&active=true`, `GET /v1/classes/{class_id}`
- Class students: `GET /v1/classes/{class_id}/students`
- Assignments: `GET /v1/assignments?branch_id=&class_id=`, `GET /v1/classes/{class_id}/assignments`
- Rooms: `GET /v1/rooms?branch_id=`, `POST /v1/rooms`, `DELETE /v1/rooms/{room_id}`
- Schedule: `GET /v1/schedules?branch_id=&item_type=lesson`
- Exams: `GET /v1/exams?branch_id=&class_id=`, `GET /v1/exams/{exam_id}`
- Exam results: `GET /v1/exam-results?branch_id=&exam_id=&student_id=`, `GET /v1/exams/{exam_id}/results`
- Payments: `GET /v1/payments?branch_id=&status=pending`, `GET /v1/payments/{payment_id}`
- Files: `GET /v1/files?branch_id=&owner_type=&owner_id=&purpose=`, `GET /v1/files/{file_id}`, `DELETE /v1/files/{file_id}`, `GET /v1/files/{file_id}/download-url`, `POST /v1/files/upload`
- Notifications: `GET /v1/notifications?unread_only=true`, `POST /v1/notifications/{notification_id}/read`

## File Upload Policy

- Maximum upload size is 10 MB per file.
- Ask before adding any new file upload UI which category the file belongs to, and document upload-time optimization plus download-time behavior before implementation.
- Lossy optimization cannot be reversed. If a file must later download in original visual/content quality, store the original or a content-preserving sanitized object; do not rely on reversing compression.
- `standard` = "Sadece dosya": normal files. These may later use safe optimization/compression such as metadata removal, font subsetting, invisible layer cleanup, and conservative image/PDF downsampling when readability and OCR edge clarity remain intact. If original download quality is required, store original plus optional optimized preview.
- `special` = "Cok onemli dosya": important official files. Visible/content data must not be altered. PDFs may have non-content metadata removed; JPG/PNG/WebP special images should not be recompressed or quality-ratio converted. At-rest encryption is required in the target architecture.
- `standard` + `profile_photo` image uploads are optimized on the backend for UI use: auto-orientation, center square crop, 768x768 resize/upscale, metadata stripping, and JPEG quality 85 output. This is an optimized display asset and is not intended to recreate the original upload.

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

# Kingsway Backend

Go backend workspace for the Kingsway Operation Center.

## Service Boundaries

- `auth-service`: login, JWT sessions, RBAC, branch-scoped access.
- `academic-service`: students, FIN history, teachers, classes, exams, schedules.
- `finance-service`: payments, salary models, swap-based salary allocation.
- `file-service`: Standard/Special file policy, MinIO storage, retention jobs.
- `notification-service`: payment reminders, room changes, file retention notices.

## Infrastructure

Docker usage is currently paused for day-to-day development. The root `start-kingsway.cmd` script now starts the frontend, backend, and local infrastructure without Docker.

For the backend to run with `DATA_STORE=postgres`, the root starter brings up these local/native services:

- PostgreSQL on `localhost:5432`
- Redis on `localhost:6379`
- RabbitMQ on `localhost:5672`
- MinIO on `localhost:9000`

Local data/logs are stored in the root `.runtime/` folder. `backend/.env` is created automatically when missing.

Backend-only Docker dependencies are still defined in this folder's `docker-compose.yml` for later use only:

- PostgreSQL for service schemas.
- Redis for cached salary and dashboard results.
- RabbitMQ for event-driven finance/file/notification workflows.
- MinIO for self-hosted S3-compatible storage.

The API defaults to PostgreSQL persistence and also connects to Redis, RabbitMQ, and MinIO when `DATA_STORE=postgres`.

Docker compose files are retained only for later use if Docker is explicitly re-enabled. Do not use this during the current local workflow:

```bash
docker compose up -d
```

The root starter creates `backend/.env` from local defaults when it is missing.

## Current Backend Foundation

- `internal/domain`: core business rules that can be tested without a database.
  - Roles and salary visibility.
  - FIN parsing and validation.
  - Standard/Special file policy.
  - Writing document 60-day retention.
  - Swap salary allocation.
  - Schedule overlap helper.
- `internal/platform`: thin wrappers for config, logging, migrations, PostgreSQL, Redis, RabbitMQ, MinIO, and validation.
- `migrations/202604290001_initial_core.sql`: first PostgreSQL schema draft for branch isolation, users, students, teachers, courses, rooms, schedules, payments, files, exams, and notifications.
- `internal/store/postgres.go`: PostgreSQL store implementing the current auth, academic, finance, and file service contracts.

## Runnable API

For day-to-day full-stack development while Docker is paused, prefer the root command:

```bash
start-kingsway.cmd
```

Default local owner login after first start:

- Email: `owner@kingsway.local`
- Password: `Kingsway123!`

Stop all local processes started by the root script:

```bash
stop-kingsway.cmd
```

The backend also supports running manually against local/native infrastructure:

```bash
go run ./cmd/api-server
```

Default address: `:8080`

Useful env vars:

```bash
HTTP_ADDR=:8080
JWT_SECRET=change-me
DATA_STORE=postgres
RUN_MIGRATIONS=true
RATE_LIMIT_ENABLED=true
RATE_LIMIT_PER_MINUTE=600
RETENTION_WORKER_ENABLED=true
RETENTION_WORKER_INTERVAL_SECONDS=3600
RETENTION_WORKER_LIMIT=100
OUTBOX_DISPATCHER_ENABLED=true
OUTBOX_DISPATCHER_INTERVAL_SECONDS=5
NOTIFICATION_WORKERS_ENABLED=true
```

For a temporary no-infra run, set `DATA_STORE=memory`.

Public endpoints:

- `GET /healthz`
- `GET /readyz`
- `GET /metrics`
- `POST /v1/auth/bootstrap-owner`
- `POST /v1/auth/login`

Authenticated endpoints:

- `GET /v1/me`
- `GET /v1/session`
- `GET|POST /v1/branches`
- `GET|POST /v1/students`
- `GET /v1/students/{student_id}`
- `POST /v1/students/{student_id}/account`
- `GET /v1/students/by-fin/{fin}`
- `GET|POST /v1/teachers`
- `GET /v1/teachers/{teacher_id}`
- `POST /v1/teachers/{teacher_id}/activate`
- `GET|POST /v1/courses`
- `GET /v1/courses/{course_id}`
- `GET|POST /v1/classes`
- `GET /v1/classes/{class_id}`
- `GET|POST /v1/classes/{class_id}/students`
- `GET|POST /v1/classes/{class_id}/assignments`
- `GET|POST /v1/assignments`
- `GET|POST /v1/rooms`
- `GET|POST /v1/schedules`
- `GET|POST /v1/exams`
- `GET /v1/exams/{exam_id}`
- `GET|POST /v1/exams/{exam_id}/results`
- `GET /v1/exam-results`
- `GET /v1/dashboard/academic`
- `GET /v1/dashboard/owner`
- `GET /v1/dashboard/receptionist`
- `GET /v1/dashboard/teacher`
- `GET /v1/dashboard/student`
- `GET|POST /v1/payments`
- `GET /v1/payments/{payment_id}`
- `GET|POST /v1/salary-models`
- `POST /v1/salary/swap-allocation`
- `GET|POST /v1/files`
- `POST /v1/files/upload` multipart form upload:
  - fields: `branch_id`, `owner_type`, `owner_id`, `category`, `purpose`, `file`
- `GET /v1/files/{file_id}`
- `GET /v1/files/{file_id}/download-url`
- `POST /v1/files/retention/cleanup?limit=100`
- `GET /v1/notifications?unread_only=true`
- `POST /v1/notifications/{notification_id}/read`
- `GET /v1/admin/outbox?status=failed`
- `POST /v1/admin/outbox/{event_id}/retry`

List endpoints accept `limit` and `offset` and return `X-Total-Count`, `X-Limit`, and `X-Offset` headers. Every response includes `X-Request-ID`. The frontend-facing contract is tracked in `api/frontend-contract.md`; OpenAPI is tracked in `api/openapi.yaml`.

Implemented guardrails:

- Branch-scoped users are locked to their branch.
- Owner can list all branches and all branch-scoped resources.
- Teacher/student academic list endpoints are narrowed to own classes, assignments, exams, and results.
- Receptionist cannot access salary models.
- Teacher can only see their own salary models.
- Teacher can create extra exams/events but not normal lessons.
- Room and teacher schedule conflicts return `409 Conflict`.
- FIN codes are normalized and validated.
- Writing files get 60-day retention metadata.
- Special files must preserve original size.
- PostgreSQL migration runs automatically when `RUN_MIGRATIONS=true`.
- HTTP middleware adds request IDs, structured request logging, JSON metrics, CORS exposed pagination/request headers, and per-IP rate limiting.
- Salary swap allocations are cached in Redis.
- Finance/file events are written to PostgreSQL `outbox_events` in the same transaction as key writes, then dispatched to RabbitMQ exchange `kingsway.events`.
- Owner-only outbox admin endpoints can inspect failed/pending/published events and reset failed events to pending retry.
- RabbitMQ notification consumer turns finance/file events into user notifications.
- Payment reminder and file retention notice workers create deduped notifications.
- File upload bytes are stored in MinIO standard/special buckets.
- File download uses short-lived MinIO presigned URLs.
- Owner-triggered retention cleanup deletes expired objects from MinIO and marks files deleted in PostgreSQL.
- Optional retention worker automatically runs the same cleanup on an interval when using PostgreSQL infrastructure.
- Docker-backed PostgreSQL integration tests live in `internal/integration` and run with `KINGSWAY_INTEGRATION=1`.
- Backup/restore procedure is documented in `docs/backup-restore.md`.

## Still Not Production-Final

- Standard file optimization/compression is not implemented yet.
- Notification delivery templates/channels are still in-app only; email/SMS/push require provider choice, credentials, templates, opt-out/bounce handling, and delivery audit tables.
- Persistent audit-log storage is not implemented yet; request logging currently goes to structured application logs with `X-Request-ID`.
- Deep readiness checks for live PostgreSQL/Redis/RabbitMQ/MinIO connections are not wired into `GET /readyz` yet; startup currently fails fast if those dependencies are unavailable.
- More broad integration coverage is still needed for every API workflow.
- gRPC/protobuf transport is deferred.

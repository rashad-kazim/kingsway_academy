# Kingsway Project Memory

This file is the running memory for the project. At the start of each work session, read this file first. At the end of each work session, update it with what changed, what was deferred, and where to continue.

## Current Direction

- Product: smart operation center for IELTS/SAT-focused education centers.
- Frontend: Next.js App Router, TypeScript, Tailwind, shadcn/ui, EN/TR/AZ JSON i18n, dark/light mode.
- Backend: Go, modular service boundaries, PostgreSQL-first data model, event-ready finance/file/notification flows.
- Current build priority: full local development through native services while Docker is paused.
- Historical Docker status: Docker Desktop was installed and verified earlier, but Docker is not used in the active workflow.
- Docker storage: Docker app files are under `D:\Docker\Docker`; Docker WSL/data root is configured as `D:\DockerData`, so images/containers should use D instead of filling C.
- WSL status: no separate Ubuntu distro is required for current Docker use; Docker's own `docker-desktop` WSL2 distro is enough.
- Current checked disk free space after Docker infra pull/start: `C:` about 32.11 GB, `D:` about 191.05 GB, `E:` about 477.42 GB.
- Database direction: PostgreSQL remains the backend persistence target. Local PostgreSQL/Redis/RabbitMQ/MinIO now run from native Scoop-installed binaries under the root `start-kingsway.cmd` workflow.
- Full-stack Docker status: root `compose.yaml` still exists for future full-stack runs, but it is parked until the user explicitly asks to use Docker again.
- Active workflow override as of 2026-05-02: Docker usage is paused until the user explicitly enables it again. Frontend changes should be run and checked through the local Next.js dev server for immediate feedback.

## Active Local Run

Use this workflow while Docker is paused:

```bash
start-kingsway.cmd
```

That script starts, without Docker:

- Frontend: `http://127.0.0.1:3000/en/login`
- Backend: `http://127.0.0.1:8080/healthz`
- PostgreSQL on `127.0.0.1:5432`
- Redis on `127.0.0.1:6379`
- RabbitMQ on `127.0.0.1:5672`
- MinIO API on `127.0.0.1:9000`
- MinIO Console on `127.0.0.1:9001`

Default local login after first start:

- Email: `owner@kingsway.local`
- Password: `Kingsway123!`

Stop all local processes started by the script:

```bash
stop-kingsway.cmd
```

Data and logs live in `.runtime/`, which is ignored by git. `frontend/.env.local` and `backend/.env` are created automatically when missing.

For frontend-only quick UI iteration, use:

```bash
cd frontend
npm run dev -- --hostname 127.0.0.1 --port 3000
```

## Docker Full-Stack Run

Docker is parked until the user explicitly re-enables it. Do not run `docker compose` for local startup or frontend iteration in the current workflow.

Historical command retained only for later:


```bash
docker compose up -d --build
```

Only use Docker again after explicit instruction.

## Role Panels

- Owner: branch setup, profit/loss, teacher salary model approval, branch switching, full branch access.
- Receptionist: student/teacher registration, room/schedule planning, payment collection, assignment/swap, exam visibility.
- Teacher: classes, classroom materials, exams/events, score entry, writing uploads, salary visibility.
- Student: progress analytics, materials, schedule, payments, RSVP.

## Non-Negotiable Product Rules

- Branch isolation: branch users cannot see another branch's students, money, schedules, or files.
- FIN ID: 7-character unique student identity used to restore returning student history.
- Room conflict prevention: same room cannot be booked at overlapping times.
- Teacher conflict prevention: same teacher cannot be booked for overlapping lessons/events.
- Salary privacy: only Owner and the related Teacher can see salary details; Receptionist cannot.
- Swap fairness: teacher salary allocation follows the number of days each teacher taught.
- File policy:
  - Standard files can be optimized/compressed.
  - Special files must preserve original file bytes/hash.
  - Writing exam documents are retained for 60 days; scores remain permanent.

## Completed

- Created `frontend/` and `backend/` folders.
- Read project Word docs:
  - `Roles.docx`
  - `Global Layout Framework.docx`
  - `Detailed Tech stack.docx`
- Frontend initialized:
  - Next.js 16 App Router
  - TypeScript, Tailwind, ESLint
  - shadcn/ui base components
  - `next-intl`, `next-themes`, TanStack Query, Zod, React Hook Form, Recharts, Lucide
  - EN/TR/AZ message files
  - Global app providers
- Frontend architecture pass completed 2026-04-29:
  - Confirmed frontend stack is already installed: Next.js 16.2.4, React 19.2.4, Tailwind 4, shadcn/ui, next-intl, next-themes, TanStack Query, Zod, React Hook Form, Lucide.
  - Added `.env.example` with `KINGSWAY_API_BASE_URL`.
  - Replaced legacy root `middleware.ts` with Next 16 `src/proxy.ts` for locale routing plus optimistic auth redirects.
  - Added backend API client/types, HTTP-only cookie auth session helpers, login/logout server actions, and localized login screen.
  - Added authenticated app shell and role dashboard route architecture at `/[locale]/dashboard/[role]`.
  - Added frontend README documenting stack, runtime contract, and folder architecture.
- Full-stack Docker runtime completed 2026-04-29:
  - Added root `compose.yaml` for frontend, API, PostgreSQL, Redis, RabbitMQ, and MinIO.
  - Added Windows double-click helpers: `start-kingsway.cmd` and `stop-kingsway.cmd`.
  - Added backend and frontend production Dockerfiles plus `.dockerignore` files.
  - Configured Next.js standalone output for the frontend container.
  - Added configurable frontend auth cookie security so local HTTP Docker login can work while production can still use Secure cookies.
- Frontend login redesign completed 2026-04-30:
  - Rebuilt `/[locale]/login` as a full-screen responsive education-themed page with a white login card.
  - Removed the Student/Teacher switcher from the provided reference because Kingsway login is role-neutral.
  - Kept login as a shared multi-role entry point based on `Roles.docx`; backend session still routes users to the correct role dashboard.
  - Added password visibility toggle and localized login copy for EN/TR/AZ.
  - Removed unused image assets; the login scene is now CSS/HTML-based and the only retained app image is `src/app/favicon.ico`.
  - Added global pointer cursor behavior for active clickable controls.
- Frontend login redesign updated 2026-05-01:
  - Rebuilt `/[locale]/login` again against the latest split-screen reference.
  - Added the provided right-side education illustration at `frontend/public/images/login-right-side.png` and renders it as a foreground image, not a dark full-panel background.
  - Added an icon-only transparent Kingsway mark at `frontend/public/images/kingsway-mark.png`; the academy text is excluded from the login logo.
  - Applied the requested login palette: left background `rgb(245,247,250)`, left shapes `rgb(230,234,240)`, right gradient from `rgb(10,40,75)` to `rgb(15,55,100)`, and right shapes `rgb(35,75,120)` at low opacity.
  - Moved animated bubble shapes behind the content, removed the right-side bottom red line, reduced the left form scale, and reduced the right hero headline size.
  - Current retained app images are only the favicon plus the two used login assets.
- Frontend login adjustment and Docker pause completed 2026-05-02:
  - Moved login bubbles into one full-page background layer so bubbles no longer get clipped at the left/right split.
  - Updated login colors: left background `rgb(253,253,253)`, left bubble color `rgb(248,248,249)`, and labels/normal input borders use the right-side navy `#0a284b`.
  - Login inputs now use black text, `rgb(253,253,253)` backgrounds, navy borders by default, and red borders only for invalid login/input states.
  - Regenerated the icon-only Kingsway mark with transparent outer background and increased its displayed size.
  - Disabled the Next.js dev indicator so local frontend previews do not show the bottom-left dev badge.
  - Docker usage is paused until explicitly re-enabled by the user; frontend iteration now uses local `npm run dev`.
- Docker-free local infrastructure completed 2026-05-02:
  - Installed/verified native local PostgreSQL 16, Redis, RabbitMQ/Erlang, and MinIO through Scoop.
  - Reworked `start-kingsway.cmd` to start PostgreSQL, Redis, RabbitMQ, MinIO, backend, and frontend without Docker.
  - Reworked `stop-kingsway.cmd` to stop the local app processes and infrastructure started for this project.
  - Added `.runtime/` for ignored local data/logs/PID files.
  - First local DB start creates default owner login `owner@kingsway.local` / `Kingsway123!` if no owner exists.
- Frontend login bubble correction completed 2026-05-02:
  - Login bubbles now render through separate clipped left/right background layers, so a bubble crossing the center split takes the left bubble color on the left side and the right bubble color on the right side.
  - Left-side bubbles were made more visible.
  - The right-side login illustration now sits at `bottom: 0`.
- Frontend login logo/bubble color adjustment completed 2026-05-02:
  - Left login bubbles now use `#e3e3e8`.
  - Login logo now uses the provided square `frontend/public/images/kingsway-mark.png` dimensions and renders larger.
- Backend initialized:
  - Go module `kingsway/backend`
  - Basic service folders: auth, academic, finance, files, notification
  - Platform folders: config, database, logger, cache, queue, storage
  - Active dependencies now used in code: pgx/PostgreSQL, Redis, RabbitMQ, MinIO, JWT, validator, env config, zap.
- Backend first domain pass completed:
  - Added `internal/domain` rules for roles, FIN, file policy, salary model/swap allocation, and schedule overlap.
  - Added platform wrappers for PostgreSQL, Redis, RabbitMQ, MinIO, validator, config, logger, and JWT tokens.
  - Added first PostgreSQL migration draft: `backend/migrations/202604290001_initial_core.sql`.
  - Migration includes branch isolation, FIN uniqueness, salary model privacy foundation, room/teacher overlap constraints, file retention, payments, exams, and notifications.
- Backend MVP API pass completed:
  - Added runnable API server at `backend/cmd/api-server`.
  - Added in-memory store at `backend/internal/store` so backend can run without Docker/PostgreSQL.
  - Added Auth service: bootstrap owner, login, password hashing, JWT issuing/parsing, current user.
  - Added Academic service: branches, FIN students, teacher registration/activation, courses with dynamic score categories, rooms, schedules.
  - Added Finance service: payments, salary models, swap allocation.
  - Added File service: file metadata registration, Standard/Special policy, Writing retention metadata.
  - Added HTTP API in `backend/internal/httpapi`.
- Backend infrastructure integration pass completed 2026-04-29:
  - Fixed PostgreSQL 18 Docker volume mount to `/var/lib/postgresql`.
  - Started Docker Compose infrastructure: PostgreSQL, Redis, RabbitMQ, and MinIO are healthy.
  - Added automatic PostgreSQL migration runner for goose-style SQL files.
  - Added PostgreSQL store at `backend/internal/store/postgres.go` and switched default API storage to PostgreSQL via `DATA_STORE=postgres`.
  - Added real `.env` loading in backend config.
  - Added Redis JSON cache and connected salary swap allocation cache.
  - Added RabbitMQ topic publisher on durable `kingsway.events` exchange for finance/file events.
  - Added MinIO object storage adapter, bucket setup, and `POST /v1/files/upload` multipart upload endpoint.
- Backend file/integration pass completed 2026-04-29:
  - Added MinIO presigned download URL support and `GET /v1/files/{file_id}/download-url`.
  - Added file retention cleanup support and `POST /v1/files/retention/cleanup?limit=100`.
  - Added store support for file lookup, expired file listing, and marking files deleted.
  - Added Docker-backed PostgreSQL integration test at `backend/internal/integration/postgres_store_test.go`.
- Backend retention worker pass completed 2026-04-29:
  - Added system-level file retention cleanup for internal workers while keeping the HTTP cleanup endpoint owner-only.
  - Added configurable automatic retention worker: `RETENTION_WORKER_ENABLED`, `RETENTION_WORKER_INTERVAL_SECONDS`, and `RETENTION_WORKER_LIMIT`.
  - API server starts the retention worker automatically for PostgreSQL-backed infrastructure and stops it on shutdown.
  - Added unit coverage for system cleanup deletion/marking and owner-only HTTP cleanup authorization.
- Backend academic/workers/outbox pass completed 2026-04-29:
  - Added academic domain/API coverage for classes, enrollments, assignments, exams, exam results, and academic dashboard counts.
  - Added PostgreSQL and in-memory store support for the new academic workflows.
  - Added notification service with `GET /v1/notifications` and mark-read endpoint.
  - Added RabbitMQ topic consumer for finance/file events and notification workers for payment reminders, file retention notices, and salary recalculation requests.
  - Added PostgreSQL-backed transactional outbox table/dispatcher; key finance/file writes now enqueue outbox events in the same database transaction.
  - Added `backend/migrations/202604290002_academic_notifications_outbox.sql` for assignments, notification dedupe keys, and outbox events.
- Backend API hardening/frontend contract pass completed 2026-04-29:
  - Reviewed academic core, notification workers, and outbox; fixed role-scope gaps in academic list/result access.
  - Added role-aware filtering so teachers/students only see own academic data where applicable.
  - Added detail endpoints for students, teachers, courses, classes, exams, payments, and files.
  - Added student account linking endpoint: `POST /v1/students/{student_id}/account`.
  - Added `GET /v1/session` for login/current-user contract with capabilities and dashboard route.
  - Added role dashboards: owner, receptionist, teacher, student.
  - Added list pagination via `limit`/`offset` plus `X-Total-Count`, `X-Limit`, and `X-Offset` headers.
  - Hardened JSON decoding to reject trailing JSON and cap JSON request bodies.
  - Added frontend contract doc at `backend/api/frontend-contract.md`.
- Backend production hardening/docs pass completed 2026-04-29:
  - Added request IDs, structured HTTP request logging, CORS exposure for pagination/request headers, per-IP rate limiting, `GET /readyz`, and `GET /metrics`.
  - Added owner-only outbox admin endpoints: `GET /v1/admin/outbox?status=failed` and `POST /v1/admin/outbox/{event_id}/retry`.
  - Added broader RBAC tests for finance payment privacy, file role restrictions, notification recipient isolation, and outbox admin authorization.
  - Added OpenAPI contract at `backend/api/openapi.yaml`.
  - Added backup/restore runbook at `backend/docs/backup-restore.md`.

## Verification History

- Frontend:
  - `npm run build` passed.
  - `npm run lint` passed.
  - Browser check passed at `http://127.0.0.1:3000/en`.
  - 2026-04-29 frontend architecture verification:
    - `npm run lint` passed.
    - `npm run build` passed.
    - Dev server is running on `http://127.0.0.1:3000`.
    - Temporary memory-mode backend is running on `http://127.0.0.1:8080`.
    - HTTP smoke passed for `/en/login`, auth redirect from `/en/dashboard`, backend owner login/session, and `/en/dashboard/owner` with the backend JWT cookie.
  - 2026-04-29 Docker full-stack verification:
    - `docker compose config --quiet` passed.
    - `docker compose up -d --build` built and started all root services.
    - `docker compose ps` showed frontend and API healthy.
    - `GET http://127.0.0.1:8080/healthz` returned `ok`.
    - `GET http://127.0.0.1:3000/en/login` returned `200 OK`.
    - Owner bootstrap plus `GET http://127.0.0.1:3000/en/dashboard/owner` with the JWT cookie returned `200 OK`.
  - 2026-04-30 login redesign verification:
    - `npm run lint` passed.
    - `npm run build` passed.
    - `docker compose up -d --build frontend` rebuilt and restarted the frontend container.
    - `docker compose ps frontend api` showed both services healthy.
    - `GET http://127.0.0.1:3000/en/login` returned `200 OK`.
    - Chrome headless desktop, tablet, and mobile screenshots were generated for visual QA, then removed so no unused image files remain.
  - 2026-05-01 latest login redesign verification:
    - `npm run lint` passed.
    - `npm run build` passed.
    - `docker compose up -d --build frontend` rebuilt and restarted the frontend container.
    - `docker compose ps frontend api` showed both services healthy.
    - `GET http://127.0.0.1:3000/en/login` returned `200 OK`.
    - Chrome headless desktop and mobile screenshots were generated for visual QA, confirmed no horizontal clipping after adjustment, then removed so no unused verification images remain.
  - 2026-05-02 Docker-paused frontend verification:
    - `npm run lint` passed.
    - `npm run build` passed.
    - Local Next dev server started with `npm run dev -- --hostname 127.0.0.1 --port 3000`.
    - `GET http://127.0.0.1:3000/en/login` returned `200 OK`.
    - Chrome headless mobile screenshot confirmed the dev indicator is hidden and the mobile form no longer clips; screenshot was removed afterward.
  - 2026-05-02 Docker-free full local verification:
    - `start-kingsway.cmd` started frontend, backend, PostgreSQL, Redis, RabbitMQ, and MinIO without Docker.
    - Listening ports verified: `3000`, `8080`, `5432`, `6379`, `5672`, `9000`, and `9001`.
    - `GET http://127.0.0.1:8080/healthz` returned `200 OK`.
    - `GET http://127.0.0.1:3000/en/login` returned `200 OK`.
    - Login with `owner@kingsway.local` / `Kingsway123!` succeeded and `GET /v1/session` returned `/dashboard/owner`.
    - `stop-kingsway.cmd` stopped frontend, backend, PostgreSQL, Redis, RabbitMQ, and MinIO.
  - 2026-05-02 login bubble correction verification:
    - `npm run lint` passed.
    - `GET http://127.0.0.1:3000/en/login` returned `200 OK`.
  - 2026-05-02 login logo/bubble color verification:
    - `npm run lint` passed.
    - `GET http://127.0.0.1:3000/en/login` returned `200 OK`.
- Backend:
  - `go mod verify` passed.
  - `go test ./...` passed.
  - 2026-04-29 update: after adding domain tests, `go test ./...` compiles non-test packages but Windows Application Control blocks execution of Go's generated `domain.test.exe`.
  - `go test -c ./internal/domain -o tmp/domain.test.exe` passed, so the domain test package compiles.
  - 2026-04-29 later update: after adding API/service layers, `go test ./...` passed successfully.
  - `go build -o tmp/api-server.exe ./cmd/api-server` passed.
  - HTTP smoke test passed:
    - `GET /healthz` returned ok.
    - Owner bootstrap succeeded.
    - Branch creation succeeded.
    - Teacher registration and activation succeeded.
    - Room creation succeeded.
    - Schedule creation succeeded.
    - Duplicate overlapping schedule returned `409 Conflict`.
  - 2026-04-29 infrastructure integration verification passed:
    - `docker compose up -d` pulled and started PostgreSQL/Redis/RabbitMQ/MinIO.
    - PostgreSQL migration created 20 public tables including `schema_migrations`.
    - API ran on `http://127.0.0.1:18080` using PostgreSQL store and local infrastructure.
    - Smoke test created owner, branch, teacher, student, room, schedule, payment, salary model, swap allocation, and uploaded a file to MinIO.
    - Duplicate overlapping schedule returned `409 Conflict` through PostgreSQL exclusion constraints.
    - Redis key `kingsway:salary:swap:100000:3:2` was created.
    - RabbitMQ exchange `kingsway.events` exists as a durable topic exchange.
    - MinIO object exists under `kingsway-standard`.
    - `go test ./...` passed after integration.
  - 2026-04-29 file/integration verification passed:
    - `GET /v1/files/{file_id}/download-url` returned a MinIO presigned URL and downloaded an uploaded file with HTTP `200`.
    - Retention cleanup deleted an expired uploaded object and subsequent download URL request returned `404`.
    - `KINGSWAY_INTEGRATION=1 go test ./internal/integration -count=1` passed against Docker PostgreSQL.
    - `go test ./...` and `go build -o tmp/api-server.exe ./cmd/api-server` passed.
    - Windows Application Control blocked direct execution of the newly built `tmp/api-server.exe`; API is running through `go run ./cmd/api-server` on `http://127.0.0.1:18080`.
  - 2026-04-29 retention worker verification:
    - `go build -o tmp/api-server.exe ./cmd/api-server` passed.
    - `go test ./...` was blocked by Windows Application Control when Go tried to execute temporary `files.test.exe` and `integration.test.exe`.
    - `go test -c ./internal/files -o tmp/files.test.exe` and `go test -c ./internal/integration -o tmp/integration.test.exe` passed, confirming the new and integration test packages compile.
  - 2026-04-29 academic/workers/outbox verification:
    - `go test ./...` passed.
    - `KINGSWAY_INTEGRATION=1 go test ./internal/integration -count=1` passed against Docker PostgreSQL and now covers class/enrollment/assignment/exam/result/dashboard plus outbox event rows.
    - `go build -o tmp/api-server.exe ./cmd/api-server` passed.
    - Temporary memory-mode HTTP smoke test on `http://127.0.0.1:18082` passed for owner bootstrap, branch, teacher activation, course, student, class, enrollment, exam result, and academic dashboard.
  - 2026-04-29 API hardening/frontend contract verification:
    - `go test ./internal/academic` passed with RBAC/branch-isolation tests.
    - `go build -o tmp/api-server.exe ./cmd/api-server` passed.
    - `tmp/files.test.exe` and `tmp/integration.test.exe` passed when run from their package directories.
    - `go test ./...` is blocked by Windows Application Control for temporary Go test executables under `%TEMP%`, but the affected test binaries compile and pass when run from `backend/tmp`.
    - Temporary memory-mode HTTP smoke test on `http://127.0.0.1:18085` passed for student account linking, student login, `GET /v1/session`, student dashboard, pagination headers, and trailing JSON validation.
  - 2026-04-29 production hardening/docs verification:
    - `go test ./internal/admin`, `go test ./internal/finance`, and `go test ./internal/files` passed.
    - `go test ./internal/notification` was blocked by Windows Application Control under `%TEMP%`; `go test -c ./internal/notification -o tmp/notification.test.exe` passed and `tmp/notification.test.exe` passed from the package directory.
    - `go test ./...` is still blocked by Windows Application Control for temporary `academic.test.exe`, `notification.test.exe`, and `integration.test.exe`.
    - `go test -c ./internal/academic`, `go test -c ./internal/notification`, and `go test -c ./internal/integration` passed; academic and notification test binaries passed from package directories, while this environment also blocked running the compiled integration test binary.
    - `go build -o tmp/api-server.exe ./cmd/api-server` passed.
    - Temporary memory-mode HTTP smoke test on `http://127.0.0.1:18086` passed for `GET /healthz`, `GET /readyz`, owner bootstrap, owner-only `GET /v1/admin/outbox?status=failed`, `X-Request-ID` propagation, and `GET /metrics`.

## Deferred

- gRPC/protobuf transport is deferred.
- Full goose CLI integration is deferred; backend currently has a built-in minimal runner for existing goose-style SQL files.
- File optimization/compression for Standard files is deferred; uploads currently preserve bytes while recording Standard/Special policy. Safe implementation needs file-type-specific image/PDF/office processing choices and must never alter Special-file bytes.
- Notification delivery channels beyond in-app notifications are deferred. Email/SMS/push require provider selection, credentials, templates, opt-out/bounce handling, and delivery audit tables.
- Persistent audit-log storage is deferred. The backend now has structured request logging with `X-Request-ID`, but immutable per-action audit rows are not implemented yet.
- Deep readiness checks are deferred. `GET /readyz` reports API readiness; live dependency pings for PostgreSQL/Redis/RabbitMQ/MinIO are still not wired after startup.
- Frontend broad screen implementation is still pending beyond login/session and role dashboard shell.
- Docker Desktop installation troubleshooting is complete; Docker remains parked and is not part of current startup.

## Next Steps

1. Continue frontend implementation from the role dashboard shell.
2. Expand role-specific dashboard widgets and list/detail workflows against `backend/api/frontend-contract.md` and `backend/api/openapi.yaml`.
3. When external providers are chosen, add notification delivery channels and persistent delivery/audit records.

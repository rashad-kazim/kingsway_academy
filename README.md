# Kingsway Project Memory

This file is the running memory for the project. At the start of each work session, read this file first. At the end of each work session, update it with what changed, what was deferred, and where to continue.

## Current Direction

- Product: smart operation center for IELTS/SAT-focused education centers.
- Frontend: Next.js App Router, TypeScript, Tailwind, shadcn/ui, EN-first JSON i18n, dark/light mode. AZ/RU/DE JSON files intentionally stay empty until translation work starts.
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
  - Maximum upload size is 10 MB per file.
  - Before adding any new file upload UI, ask which category the upload belongs to and explain the upload-time optimization plus download-time behavior before writing code.
  - Lossy optimization cannot be reversed. If a file must later download in original visual/content quality, store the original or a content-preserving sanitized object; do not rely on reversing compression.
  - `standard` / "Sadece dosya": normal files. Safe optimization/compression may be applied later. If original download quality is required, store original plus optional optimized preview; do not rely on reversing compression.
  - `special` / "Cok onemli dosya": official or critical files. Visible/content data must not be altered. PDFs may have non-content metadata removed; JPG/PNG/WebP special images should not be recompressed or quality-ratio converted. At-rest encryption is required in the target architecture.
  - `standard` + `profile_photo`: UI-only display image. It is optimized for storage/display and is not designed to recreate the original file on download.
  - Writing exam documents are retained for 60 days; scores remain permanent.

## File Architecture Decisions

- Special files are the "vault" class: store encrypted at rest with AES-256-GCM/envelope encryption. For Special PDFs, non-content metadata may be stripped, but page content, text, images, layout, signatures, stamps, and visual meaning must not be changed. For Special JPG/PNG/WebP images, do not recompress or apply 90% quality conversion; keep the image content unchanged and only decrypt on download. "Exact original bytes" is required only when a workflow explicitly needs hash equality; otherwise the required guarantee is content-preserving download.
- Standard downloadable images can have two stored objects when original download quality matters: original object for download plus optimized derivative for preview/UI. If only optimized download is needed, sharpening/enhancement may improve perceived clarity but it never reconstructs lost pixels.
- Standard PDFs/documents can be cleaned/optimized for normal use, but if the user must download original quality, keep the original object and serve it on download.
- UI/avatar/profile images are optimized display assets: no original recovery promise; they prioritize fast loading and low storage.

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
- Owner branch onboarding/shell foundation completed 2026-05-02:
  - Owner login remains `owner@kingsway.local` / `Kingsway123!`.
  - Owner dashboard now shows first-branch setup when no branch is selected/available.
  - Branch setup form includes branch photo upload preview, unique branch-name acceptance icon, address textarea, and backend-backed branch save.
  - After save, owner sees branch cards plus a same-sized add-branch card.
  - Branch cards show circular photo/initials, branch name, address, active student/teacher counts, and a photo-edit icon.
  - Selecting a branch opens the shared dashboard shell.
  - Shared dashboard shell now has fixed top header, hamburger/X sidebar toggle, centered branding, theme toggle, language dropdown, collapsible sidebar with closed-state tooltips, scrollable main content, and footer at the end of content.
- Owner onboarding/header refinement completed 2026-05-02:
  - Owner first-branch page now uses a static navy header (`#0a284b`) aligned with the Kingsway mark.
  - Header right controls are standardized as language dropdown, swipe-style dark/light toggle, and profile dropdown with sign out.
  - Language routing/dropdown now supports English, Azerbaijan, Russian, and German; all active site copy is sourced from `en.json` for now.
  - `az.json`, `ru.json`, and `de.json` are intentionally empty and should not be edited until the user asks for translation work.
  - Owner onboarding, shared shell, and dashboard visible labels now come from JSON message files.
  - First-branch setup uses a fixed desktop no-scroll layout and animated left-side intro copy.
  - First-branch setup badge was scaled up to better match the large onboarding headline.
  - Language dropdown width now matches its header trigger, the dark/light toggle shows the active icon, and theme switching visibly affects the dashboard/onboarding surface.
  - Owner onboarding text/form columns are centered as two balanced halves, with the gap increased by 20%.
  - Profile dropdown sign out now transitions only its text/icon to red on hover/focus while keeping the default hover background, and the owner onboarding Save button now uses a green success accent.
  - The `Kingsway` word in the owner onboarding headline is highlighted in brand red in both light and dark mode.
  - Branch photo upload now has the dashed border on the circular image target only, matching the circular stored branch photo preview.
  - Removed the visible upload helper text from the branch photo upload control, leaving only the circular icon target.
  - Frontend-facing branch save errors are mapped to user-friendly UI messages instead of raw backend errors.
  - Branch photo save was corrected so the file input stays in the form and photos upload through the backend `/v1/files/upload` path into the existing files/MinIO flow instead of being frontend-only.
  - Owner branch lists now attach the latest backend-backed branch profile photo URL when rendering cards, and branch photo edits also upload through the same backend file path.
  - Increased the local Next.js Server Action body limit to 12 MB and added frontend validation so branch photos must be images up to 10 MB.
  - Backend `POST /v1/files/upload` now enforces the 10 MB maximum upload size.
  - Verification after the branch photo fix: `npm run lint` and `npm run build` passed in `frontend/`.
  - File upload policy was clarified: 10 MB max, `standard` means "Sadece dosya" and can later be safely optimized, `special` means "Cok onemli dosya" and must preserve original bytes/hash.
  - Backend image optimization for `standard` + `profile_photo` uploads was implemented: accepted source types are JPG/PNG/WebP, then auto-orientation, metadata stripping, center square crop, forced 768x768 output even for small images, and JPEG quality 85 storage.
  - File download policy was clarified: lossy compression is not reversible. Future downloadable optimized files must store the original object separately and return original bytes on download; profile photos are treated as UI-only optimized images.
  - Special-file policy was refined: Special PDFs may strip only non-content metadata, but page/content/visual meaning must not change; Special JPG/PNG/WebP images are not recompressed or quality-ratio converted.
  - Owner first-branch photo upload was explicitly confirmed as `standard` + `profile_photo` / UI-avatar: JPG/PNG/WebP only, 10 MB max, no original-download promise, backend optimized to 768x768 JPEG quality 85.
  - Owner first-branch photo picker now validates file type/size before previewing the image, and the backend rejects unsupported `profile_photo` image MIME types.
- Owner Branch Management edit flow completed 2026-05-03:
  - Branch detail editing no longer opens in a drawer; clicking a branch row opens a full content-area edit page inside Branch Management.
  - Edit page order is now profile photo, branch name, branch address, operational hours, and room management.
  - Save/Cancel controls were added. Save persists branch name, slug, address, opening/closing time, profile photo replacement/removal, room creation, and room deactivation.
  - Backend now has operational hour fields on branches, `DELETE /v1/rooms/{id}` for room deactivation, and `DELETE /v1/files/{id}` for profile photo removal.
  - Added migration `backend/migrations/202605030001_branch_operational_hours.sql`; next native local start applies it to PostgreSQL.
  - Verification: `frontend npm run lint`, `frontend npm run build`, and `backend go test ./...` passed.
- Backend branch-edit hardening completed 2026-05-03:
  - Added backend tests for owner-only branch profile updates, operational hour validation, room deactivate/list hiding, and profile file deletion.
  - Updated OpenAPI/frontend contract with `PATCH /v1/branches/{branch_id}`, `DELETE /v1/rooms/{room_id}`, and `DELETE /v1/files/{file_id}`.
  - Verification note: `go test ./...` is blocked for one temp academic test binary by Windows Application Control, so `academic` and `files` tests were compiled into `.runtime/*.test.exe` and run directly; both passed. Other backend packages passed through `go test ./...` before the block.
- Branch Management detail UI adjustment completed 2026-05-03:
  - Fixed the branch table header two-tone padding by removing the parent card vertical padding.
  - Branch detail photo area is now centered, without the visible "Branch photo" label, and edit/remove buttons sit under the circular image.
  - Branch name and branch address now sit side by side.
  - Room Management now shows existing rooms first, then a centered Add Room button; clicking it reveals room name/capacity inputs with Save and Cancel controls.
  - Verification: `frontend npm run lint` and `frontend npm run build` passed.
- Branch Detail save/runtime fix completed 2026-05-03:
  - Root cause for Save failure was a stale local backend binary still running without the new `PATCH /v1/branches/{branch_id}` route.
  - Updated `tools/start-local-dev.ps1` so `start-kingsway.cmd` restarts/rebuilds the backend when backend source files are newer than `.runtime/backend-api-server.exe`.
  - Restarted the local backend through the non-Docker local start script and verified `PATCH /v1/branches/{branch_id}` returns success against the running API.
  - Added Edit action before Delete for pending "Rooms to add" rows; editing opens the same Add Room form with the room values prefilled.
  - Verification: `frontend npm run lint`, `frontend npm run build`, direct compiled `academic` backend tests, and live backend PATCH smoke test passed.
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
  - 2026-05-02 owner branch onboarding/shell verification:
    - `npm run lint` passed.
    - `npm run build` passed.
    - Authenticated smoke for `GET http://127.0.0.1:3000/en/dashboard/owner` returned the owner branch setup/list screen.
    - `GET http://127.0.0.1:3000/en/login` returned `200 OK`.
  - 2026-05-02 owner onboarding/header refinement verification:
    - `npm run lint` passed.
    - `npm run build` passed.
    - `git diff --check` passed with only line-ending warnings.
  - 2026-05-02 owner onboarding badge scale verification:
    - `npm run lint` passed.
  - 2026-05-02 owner header/theme/layout refinement verification:
    - `npm run lint` passed.
    - `npm run build` passed.
  - 2026-05-02 local stack startup verification:
    - `start-kingsway.cmd` completed without Docker.
    - Ports `3000`, `8080`, `5432`, `6379`, `5672`, `9000`, and `9001` were listening.
    - `GET http://127.0.0.1:8080/healthz` returned `200`.
    - `GET http://127.0.0.1:3000/en/login` returned `200`.
  - 2026-05-02 profile dropdown/save button refinement verification:
    - `npm run lint` passed.
  - 2026-05-02 sign out hover/save success color verification:
    - `npm run lint` passed.
  - 2026-05-02 owner headline brand highlight verification:
    - `npm run lint` passed.
  - 2026-05-02 branch photo circular upload target verification:
    - `npm run lint` passed.
  - 2026-05-02 branch photo dashed border correction verification:
    - `npm run lint` passed.
  - 2026-05-02 branch photo helper text removal verification:
    - `npm run lint` passed.
  - 2026-05-02 branch photo/backend upload limit verification:
    - `npm run lint` passed.
    - `npm run build` passed.
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
  - 2026-05-02 upload limit verification:
    - `go build -o tmp/api-server.exe ./cmd/api-server` passed.
    - `go test ./internal/httpapi` passed.
  - 2026-05-02 standard profile image optimization verification:
    - `go test ./internal/files` passed.
    - `go build -o tmp/api-server.exe ./cmd/api-server` passed.
    - `npm run lint` passed.
    - `npm run build` passed.
  - 2026-05-02 file compression/download policy verification:
    - `go test ./internal/files` passed.
    - `npm run lint` passed.
  - 2026-05-02 special-file metadata policy verification:
    - `go test ./internal/files` passed.
    - `npm run lint` passed.
  - 2026-05-02 owner branch photo upload policy verification:
    - `npm run lint` passed.
    - `npm run build` passed.
    - `go build -o tmp/api-server.exe ./cmd/api-server` passed.
    - `go test ./internal/files` was blocked by Windows Application Control under `%TEMP%`; `go test -c ./internal/files -o tmp/files.test.exe` passed and `tmp/files.test.exe` passed from the package directory.
  - 2026-05-02 owner branch list refinement:
    - Branch list title was removed; the branch selection instruction is centered and animated.
    - Existing branch and add-branch cards are centered and use the same main card height.
    - Branch card text now shows JSON-backed Branch Name and Branch Address labels, with address clamped to two lines.
    - Branch photo display was enlarged; branch photo edit control is centered 10px below the card at 48x48px.
    - Fixed the topbar theme toggle hydration mismatch by rendering a deterministic first client pass before applying the resolved theme.
    - `npm run lint` passed.
    - `npm run build` passed.
  - 2026-05-02 owner branch list visual polish:
    - Removed animation from the branch selection instruction and promoted it to a centered page-heading style.
    - Existing branch and add-branch columns now reserve the same total height, including the edit-control row.
    - Removed the visible add-branch dashed border and replaced both cards with softer light/dark themed surfaces and shadows.
    - Branch card text, separators, edit control, photo ring, and add card now respond to light/dark mode.
    - `npm run lint` passed.
    - `npm run build` passed.
  - 2026-05-02 owner branch edit flow:
    - Added backend `PATCH /v1/branches/{branch_id}` for owner-only branch name/slug/address updates.
    - Added frontend branch update client/action and wired the branch edit button to open the branch setup form with existing branch data prefilled.
    - Edit mode left copy now shows "Welcome Back," and "Edit your existed Kingsway Branch"; the branch description no longer contains "first".
    - Edit form can also replace the branch profile photo through the existing `standard` + `profile_photo` backend upload path.
    - `go test ./internal/academic` passed.
    - `go test ./internal/httpapi` passed.
    - `go build -o tmp\api-server.exe ./cmd/api-server` passed.
    - `npm run lint` passed.
    - `npm run build` passed.
  - 2026-05-02 owner branch card height correction:
    - Fixed the branch card height mismatch caused by shadcn Card default vertical padding; visible branch and add cards now use the same fixed height.
    - Branch list heading text changed to "Choose a branch".
    - `npm run lint` passed.
    - `npm run build` passed.
  - 2026-05-02 owner branch edit intro polish:
    - Reduced the edit-mode intro weight: smaller setup badge, lighter "Welcome Back," row with a red accent line, and a more controlled edit headline size.
    - Edit-mode description text was tightened to better match the new heading hierarchy.
    - `npm run lint` passed.
    - `npm run build` passed.
  - 2026-05-02 owner choose-branch card hierarchy polish:
    - Choose Branch cards now show the branch name as the main title without label prefixes.
    - Address now appears as a muted MapPin row with a two-line clamp.
    - Student and Teacher counts moved into subtle red-tinted badges.
    - Add Branch card was softened with transparent glass styling and hover icon scale/rotation.
    - Dark owner workspace background now uses Midnight Blue `#0A192F`; cards use glassmorphism surfaces with `border-white/10`.
    - `npm run lint` passed.
    - `npm run build` passed.
  - 2026-05-02 owner choose-branch quick actions:
    - Added Finance, Weekly Program, and Settings circular quick action links under the branch address.
    - Quick action labels stay hidden by default and fade in below the hovered icon.
    - The main branch entry link and quick action links are separated to avoid nested interactive elements.
    - `npm run lint` passed.
    - `npm run build` passed.
  - 2026-05-02 owner branch edit removal:
    - Removed the edit button from the Choose Branch cards.
    - Removed the frontend branch edit mode and branch update action/client.
    - Removed the backend branch update service/store/route and deleted `PATCH /v1/branches/{branch_id}` from the frontend contract/OpenAPI docs.
    - `npm run lint` passed.
    - `go test ./...` passed.
    - `npm run build` passed.
  - 2026-05-02 owner branch list scroll fix:
    - Header remains fixed/static at the top of the owner branch page.
    - Branch list content now owns vertical scrolling when cards wrap into a new row.
    - Two-card branch view remains vertically centered without page scroll.
    - `npm run lint` passed.
    - `npm run build` passed.
  - 2026-05-02 local start backend fix:
    - `start-kingsway.cmd` no longer starts the backend with `go run`, because Windows Application Control blocks Go's temporary `%LOCALAPPDATA%\go-build` executable.
    - `tools/start-local-dev.ps1` now builds `backend` to `.runtime/backend-api-server.exe` from the backend module and starts that executable.
    - Verified Docker-free startup: backend `GET http://127.0.0.1:8080/healthz` returned `{"status":"ok"}` and frontend `http://127.0.0.1:3000/en/login` returned HTTP `200`.
  - 2026-05-02 owner branch grid and scrollbar fix:
    - Branch list grid now supports 3 cards per desktop row.
    - List mode uses a full-width scroll container so the scrollbar sits at the far right of the viewport.
    - Content only switches to top-aligned scroll mode when cards wrap beyond one 3-card row.
    - `npm run lint` passed.
    - `npm run build` passed.
  - 2026-05-02 dark mode palette revision:
    - Applied the softer dark palette to dashboard/common UI outside the login page: page `#101827`, header `#0b2746`, cards `#17243a`, elevated surfaces `#1d2b44`, primary text `#e2e8f0`, body text `#b7c4d6`, muted text `#7e8ea5`, borders `#64748b`, and dark brand red `#ff5a66`.
    - Owner branch setup/list surfaces, inputs, cards, badges, quick actions, header controls, and app shell dark colors now follow the revised palette.
    - Owner branch form errors now use an icon, dark red background `#3b1218`, border `#f87171`, and readable text colors `#ffe4e6` / `#fca5a5`.
    - Login page was intentionally left unchanged.
    - `npm run lint` passed.
    - `npm run build` passed.
  - 2026-05-02 dark mode charcoal revision:
    - Replaced the previous dark palette outside login with the deeper charcoal/navy palette: page `#0f1722` / `#0a111b`, navbar `#151f2c`, card `#1b2635`, card hover `#202d3e`, avatar `#2a3444`, icon surface `#202b3a`, borders `#334155` / `#3a4658`, text `#f3f6fa` / `#a7b0bf` / `#6f7a8a`, and accent red `#ff3b4f`.
  - Dashboard/app shell and owner branch pages now use the charcoal radial background and lower-glare card/header surfaces.
  - Owner branch cards now use subtle dark gradients, muted icon buttons, graphite avatars, and low-opacity red badges.
  - Login page remains unchanged by request.
  - `npm run lint` passed.
  - `npm run build` passed.
  - 2026-05-02 dashboard shell sidebar polish:
    - Replaced the sidebar toggle icon with a custom three-line hamburger that morphs into an X on click.
    - Sidebar background now matches the header color in light and dark modes.
    - Removed the role/user block from the top of the sidebar.
    - Prevented sidebar label overflow during width animation by delaying label reveal until the sidebar expansion finishes.
    - Added a formal footer line with product name and copyright placeholder.
    - `npm run lint` passed.
    - `npm run build` passed.
  - 2026-05-02 dashboard shell header/sidebar refinement:
    - Header branding now uses the Kingsway mark with horizontal `Kingsway Academy` text.
    - Hamburger/X control was slimmed to match the height of header controls.
    - Sidebar labels changed from `Command` to `Dashboard`, with a dashboard icon and active state for selected-branch dashboard pages.
    - Sidebar icons and labels were enlarged, and shrink animation now keeps icons in a stable column while labels fade out.
    - Footer now spans the full content width and resizes with the sidebar.
    - `npm run lint` passed.
    - `npm run build` passed.
  - 2026-05-03 dark mode shell color adjustment:
    - Dark page background changed to `#0B1622`.
    - Dark header/sidebar surfaces changed to `#121F2D`.
    - Applied to global theme variables, dashboard shell, owner branch workspace, and header theme toggle surfaces.
    - `npm run lint` passed.
    - `npm run build` passed.
  - 2026-05-03 header dropdown/sidebar active fix:
    - Header language and profile dropdown menus now render above the fixed header with `z-[1100]`.
    - Active sidebar menu item background changed to `#306186`.
    - `npm run lint` passed.
  - 2026-05-03 owner sidebar label/icon update:
    - Owner sidebar labels changed to: Global Dashboard, Branch Management, Teacher Finance & HR, Receptionist Management, Student Management, Scheduling & Rooms, Assignment & Swap, Payment Hub, Events & Exams.
    - Owner-only icons were aligned with the new labels while preserving already suitable icons.
    - Existing non-owner menu labels remain available for later role work.
    - `npm run lint` passed.
    - `npm run build` passed.
  - 2026-05-03 sidebar width adjustment:
    - Expanded open sidebar width from `w-64` to `w-80` so longer owner menu labels fit better.
    - Updated the main content offset from `left-64` to `left-80`.
    - `npm run lint` passed.
  - 2026-05-03 revised logo asset:
    - `frontend/public/images/kingsway-mark.png` was replaced by the user with the revised logo asset.
    - Existing frontend references already use `/images/kingsway-mark.png` in login, dashboard header, and owner branch header, so no code path change was needed.
  - 2026-05-03 owner global dashboard layout:
    - Removed the old owner dashboard heading/capability/table layout from the owner dashboard view.
    - Added branch scope pills: All Branches plus each branch name, with the active scope highlighted in Kingsway red.
    - Added `branch_id=all` support on the owner dashboard route so aggregate backend dashboard data can render without opening Branch Management.
    - Kept the three existing summary cards and added Debt Tracker and Total Turnover cards in AZN format.
    - Added Course Distribution pie/legend section and Action and Alert Center section.
    - Current financial and course distribution cards render safe zero states until dedicated backend finance/course aggregation fields are added.
    - `npm run lint` passed.
    - `npm run build` passed.
  - 2026-05-03 owner Branch Management view:
    - Sidebar Branch Management now opens `/dashboard/owner?view=branches` inside the AppShell content area instead of the standalone branch picker.
    - Added Branch Management header with Kingsway red `Add New Branch` sheet/drawer.
    - The drawer uses the existing `standard` + `profile_photo` branch create action and 10 MB JPG/PNG/WebP validation path.
    - Added mini stats for Total Branches, Total Classrooms, and Capacity Status.
    - Added a management data table with Branch Name/ID, Address, Rooms, Staff, Status, and Actions columns.
    - Branch management table now uses a light/dark glassmorphism surface with translucent background, subtle border, blur, and deeper shadow.
    - Branch management table header now uses the same glass surface background as the table body.
    - Row click or Edit opens a right-side Branch Detail sheet with Room Management, branch name, address, operational hours, and branch assets fields.
    - Detail edit persistence, room CRUD, branch archival, receptionist assignment avatars, and capacity data remain UI-ready placeholders until backend contracts are finalized.
    - `npm run lint` passed.
    - `npm run build` passed.

## Deferred

- gRPC/protobuf transport is deferred.
- Full goose CLI integration is deferred; backend currently has a built-in minimal runner for existing goose-style SQL files.
- File optimization/compression for general Standard documents is deferred. `standard` + `profile_photo` images are already optimized for UI storage. Future Standard document optimization can remove metadata, subset fonts, clean invisible layers, and conservatively downsample ordinary PDFs/images only when text readability/OCR edge clarity remains safe. Special files must never be altered.
- Notification delivery channels beyond in-app notifications are deferred. Email/SMS/push require provider selection, credentials, templates, opt-out/bounce handling, and delivery audit tables.
- Persistent audit-log storage is deferred. The backend now has structured request logging with `X-Request-ID`, but immutable per-action audit rows are not implemented yet.
- Deep readiness checks are deferred. `GET /readyz` reports API readiness; live dependency pings for PostgreSQL/Redis/RabbitMQ/MinIO are still not wired after startup.
- Frontend broad screen implementation is still pending beyond login/session and role dashboard shell.
  - Owner branch editing is intentionally removed from the current UI/API until the user asks for it again.
  - Docker Desktop installation troubleshooting is complete; Docker remains parked and is not part of current startup.

## Current Work Notes

- 2026-05-03 owner Branch Management save/profile fix:
  - Backend profile-photo upload now normalizes common image MIME aliases (`image/jpg`, `image/x-png`) and sniffs valid image bytes when the browser sends `application/octet-stream` or an empty MIME.
  - Branch Management save now treats already-deleted files/rooms and already-created room drafts as idempotent, preventing the branch save from failing after the branch data itself was already updated.
  - Branch Detail save now pushes the saved branch back into local frontend state before refresh, so returning to the Branch Management table and reopening the branch uses the latest photo/name/address immediately.
  - Branch Management table now labels the first column `Branch Profile` and shows a circular branch photo/initials before the branch name and ID.
  - Verification: `npm run lint`, `npm run build`, and `go test ./internal/files` passed.
- 2026-05-03 branch/room duplicate UX fix:
  - Branch create/edit name fields now show a live availability icon while typing.
  - Room add/edit input now shows a live availability icon and blocks adding duplicate room names from the current branch/pending room list.
  - Backend room conflict during Branch Detail save now maps to `This room already exists.` instead of the branch duplicate message.
  - Verification: `npm run lint` and `npm run build` passed.
- 2026-05-03 Branch Detail profile upload root-cause fix:
  - Root cause: frontend correctly uploaded branch photos as `owner_type=branch`, but PostgreSQL `files_owner_type_check` did not allow `branch`, so valid JPG/PNG/WebP uploads failed with backend `invalid input` and the UI showed the generic invalid-photo message.
  - Added migration `202605030002_allow_branch_file_owner.sql` to allow branch-owned file records.
  - Verified the migration is applied locally and the live API now accepts `kingsway-mark.png`, optimizes it to `image/jpeg`, stores it in `kingsway-standard`, and returns it through `/v1/files`.
  - Frontend now clears the temporary photo preview/file input after invalid photo or oversized photo failures so the user is not left seeing an unsaved preview.
  - Verification: `npm run lint`, `go test ./internal/files`, direct `POST /v1/files/upload` -> `201`, direct `/v1/files` lookup -> saved profile file visible. `go test ./...` is blocked by Windows Application Control for the generated `academic.test.exe`.
- 2026-05-03 local hot-update workflow fix:
  - Added `tools/watch-backend.ps1`, started automatically by `tools/start-local-dev.ps1`.
  - Backend watcher monitors `backend/**/*.go`, `backend/**/*.sql`, `backend/go.mod`, and `backend/go.sum`; on change it rebuilds `.runtime/backend-api-server.exe` and restarts only the backend process without Docker.
  - `tools/stop-local-dev.ps1` now stops the backend watcher before stopping backend, preventing automatic restart during shutdown.
  - Added dev-only `/api/dev-asset/[...path]` route with `no-store` headers for public assets.
  - Static logo images now use dev cache-busting, so replacing `frontend/public/images/kingsway-mark.png` with the same filename refreshes in the running frontend without stale browser/Next image cache.
  - Verification: PowerShell scripts parse cleanly, `npm run lint`, `npm run build`, `go test ./internal/files`, backend watcher timestamp smoke -> rebuild/restart, `GET /api/dev-asset/images/kingsway-mark.png` -> `200` with `Cache-Control: no-store`.
- 2026-05-03 Branch Management header alignment:
  - Aligned the `Add New Branch` button vertically with the Branch Management title/description block by changing the header row from `items-start` to `items-center`.
  - Verification: `npm run lint` passed.
- 2026-05-03 Add New Branch drawer photo preview:
  - Added create-drawer photo preview state and object URL cleanup, so selected branch photos appear immediately after choosing a JPG/PNG/WebP file.
  - Changed the create branch photo upload target from a rectangular drop area to a centered circular upload/photo preview control.
  - Verification: `npm run lint` and `npm run build` passed.
- 2026-05-03 Add New Branch drawer copy cleanup:
  - Removed technical standard/profile-photo optimization wording from the drawer description.
  - Replaced it with user-facing copy: `Create a new branch profile with its photo, name, and address.`
  - Verification: `npm run lint` and `npm run build` passed.
- 2026-05-03 Branch Management delete confirmation UI:
  - Removed `View Reports` from the branch row actions dropdown.
  - Added a darker `Delete` action below `Archive`.
  - Added a two-step delete confirmation: first type the exact branch name, then confirm a warning that related student history, performance records, and teacher history will also be deleted.
  - Cancel closes the full delete confirmation flow.
  - Verification: `npm run lint` and `npm run build` passed.
- 2026-05-03 Branch Management destructive delete:
  - Added backend `DELETE /v1/branches/{branch_id}` for owner-only branch deletion.
  - Delete removes active branch files from object storage first, then hard-deletes branch-owned DB rows in FK-safe order: notifications, exams/results, assignments, schedule, class/student relations, payments, salary models, classes, course metadata, rooms, students, teachers, files, branch users, outbox rows, and the branch.
  - Memory store and academic service tests now cover owner-only branch delete and branch-data cascade cleanup.
  - Frontend Delete confirmation now calls the backend and removes the deleted branch from the table immediately.
  - Archive/Delete dropdown hover/focus keeps the original red icon/text colors instead of turning black.
  - Updated OpenAPI/frontend contract with `DELETE /v1/branches/{branch_id}`.
  - Verification: `go test ./...`, `npm run lint`, `npm run build`, live API smoke create/delete branch, and live API smoke create branch+room/delete branch all passed.
- 2026-05-03 Branch Management staff/receptionist setup:
  - Add New Branch drawer now includes vertical Opening time and Closing time fields.
  - Added backend `staff_profiles` migration plus owner-only receptionist endpoints: `GET/POST /v1/branches/{branch_id}/staff`, `PATCH/DELETE /v1/staff/{staff_id}`.
  - Receptionist records store name, surname, birth date, phone, AZN salary, login email, password hash, role, and optional standard/profile-photo file ID.
  - Branch Detail now shows Staff Management before Room Management with Existing Staff, Add Staff, edit/delete, circular staff photo preview, digit-only salary entry, and the required warning when editing existing staff while a new staff form is open.
  - Branch Management table now shows assigned receptionist avatars/counts per branch instead of a static unassigned placeholder.
  - Updated frontend API types/client/actions and OpenAPI/frontend contract for branch staff.
  - Verification: `npm run lint`, `npm run build`, and `go test ./cmd/api-server` passed. `go test ./internal/academic` is still blocked by Windows Application Control for the generated `academic.test.exe`.

## Next Steps

1. Continue frontend implementation from the role dashboard shell.
2. Expand role-specific dashboard widgets and list/detail workflows against `backend/api/frontend-contract.md` and `backend/api/openapi.yaml`.
3. When external providers are chosen, add notification delivery channels and persistent delivery/audit records.

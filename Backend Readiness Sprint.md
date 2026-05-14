# Backend Readiness Sprint

This file tracks the backend preparation work for the upcoming owner-driven redesign. Read this file only when working on backend architecture/readiness, planning backend changes, or resuming this sprint.

## Goal

Prepare the backend as a modular monolith so large requirement changes can be added without breaking existing modules.

This is not a microservice migration. The target is one Go backend with clear internal module boundaries:

- `auth`
- `branch`
- `student`
- `teacher`
- `receptionist`
- `room`
- `finance`
- `files`
- `audit`
- `idempotency`

## Current Status

### Done

- HTTP handlers were split out of the oversized `backend/internal/httpapi/server.go`.
- Handler files now exist by domain area:
  - `branch_handlers.go`
  - `student_handlers.go`
  - `teacher_handlers.go`
  - `academic_handlers.go`
  - `finance_handlers.go`
  - `file_handlers.go`
  - `notification_handlers.go`
- The oversized Postgres repository file was mostly split out of `backend/internal/store/postgres.go`.
- Store files now exist for major areas:
  - `branch_store.go`
  - `student_store.go`
  - `teacher_store.go`
  - `receptionist_store.go`
  - `room_store.go`
  - `academic_store.go`
  - `finance_store.go`
  - `file_store.go`
  - `idempotency_store.go`
  - `user_store.go`
  - `audit_store.go`
- Shared transaction helper was added in `backend/internal/store/tx.go`.
- Transaction helper now carries the active `pgx.Tx` in context so nested repository/service calls join the current transaction instead of opening a separate pool operation.
- Transaction helper is already used in high-risk flows:
  - branch delete
  - student create with details
  - student parent contact batch create
  - student course registration batch create
- Idempotency base exists:
  - `idempotency_keys` table
  - `Idempotency-Key` CORS support
  - JSON POST idempotency wrapper
  - no-body write helper for DELETE-style requests
  - atomic `RunIdempotent` flow for Postgres-backed writes
- Optional idempotency coverage was extended to several critical PATCH/DELETE paths:
  - branch update/delete
  - staff update/delete
  - teacher update/delete
  - student update
  - room update/delete
  - file delete
- Audit log base infrastructure exists:
  - `audit_logs` migration
  - `domain.AuditLog`
  - Postgres `CreateAuditLog`
  - generic HTTP write audit logging for successful authenticated `POST/PATCH/PUT/DELETE`.
- Regression checks support module targeting through:
  - `check-kingsway.cmd -Module branch`
  - `check-kingsway.cmd -Module teacher`
  - `check-kingsway.cmd -Module student`
  - `check-kingsway.cmd -Module receptionist`
  - `check-kingsway.cmd -Module academic`
  - `check-kingsway.cmd -Module finance`
  - `check-kingsway.cmd -Module files`
  - `check-kingsway.cmd -Module backend`
  - `check-kingsway.cmd -Module frontend`

### Partial

- Repository naming is not fully aligned with the target domain names yet:
  - Receptionist logic is currently in `staff_store.go`; later rename/split to `receptionist_store.go` if no other staff roles are added.
  - Room logic is currently in `academic_store.go`; later split to `room_store.go`.
  - Some academic/class/exam flows still live in `academic_postgres.go`.
- Transaction standard exists, but not every multi-table write has been migrated to the helper yet.
- Idempotency now stores the business write and completed response in one Postgres transaction when an `Idempotency-Key` is present.
  - Still needs final endpoint-by-endpoint audit for coverage gaps.
- Audit log exists, but it is currently generic HTTP-level logging.
  - Sensitive service-level audit records still need richer before/after payloads.
  - Current generic audit does not yet store `before_json` / `after_json`.
- OpenAPI and frontend contract exist, but updates are manual and not yet enforced by CI/contract tests.
- Tests are modular at command level, but full backend integration suites per module are not complete yet.
- File policy exists conceptually and partly in upload logic, but backend policy names should be standardized as:
  - `standard-ui`
  - `standard-downloadable`
  - `special`

### Pending

- Expand `audit_store.go` and service-level audit detail where needed.
- Convert all remaining multi-table writes to the shared transaction helper.
- Add service-level audit logs for critical actions:
  - branch delete
  - teacher salary create/update
  - teacher delete
  - receptionist create/update/delete
  - student create/update/branch transfer
  - room add/edit/delete
  - future assignment/swap
  - future payments
- Add `before_json` and `after_json` fields to audit logs or add a second migration if needed.
- Finish idempotency consistency for all critical mutating endpoints:
  - all `POST`
  - critical `PATCH`
  - critical `DELETE`
- Add module-specific backend integration tests:
  - branch integration
  - teacher integration
  - student integration
  - receptionist integration
  - room integration
  - idempotency/concurrency integration
  - audit log integration
- Strengthen OpenAPI/contract workflow so backend/frontend payload drift is caught earlier.
- Normalize file pipeline policies in backend around `standard-ui`, `standard-downloadable`, and `special`.

## Sprint Plan

### Phase 1: Modular Monolith Cleanup

Goal: make file ownership and module boundaries obvious.

Tasks:

- Keep `server.go` as routing/middleware/bootstrap only.
- Keep domain handler files small and focused.
- Split `staff_store.go` into `receptionist_store.go` when receptionist rules stabilize.
- Split room methods out of `academic_store.go` into `room_store.go`.
- Keep `postgres.go` for shared scan helpers and shared DB utilities only.

Status: partial.

### Phase 2: Transaction Standard

Goal: no partial writes.

Rule:

Any create/update/delete touching more than one table must use a transaction.

Priority flows:

- student create
- teacher create/update/delete
- branch delete
- receptionist create/update/delete
- room add/edit/delete
- future payment writes
- future assignment/swap writes

Status: helper exists, active transaction joining works, migration is partial.

### Phase 3: Idempotency Standard

Goal: repeated clicks, slow internet retries, and duplicate requests must not create duplicate rows.

Rule:

- `POST`: idempotency expected.
- critical `PATCH`: idempotency supported.
- critical `DELETE`: idempotency supported.
- Same key + same body returns same completed response when possible.
- Same key + different body returns conflict.

Status: base exists, final endpoint audit pending.

Latest update:

- `RunIdempotent` makes idempotency row creation, business callback writes, completed response storage, and commit atomic for Postgres.
- A retry with the same key/body replays the stored completed response.
- A retry with same key but different body returns conflict.
- If the business callback fails, the business write and idempotency row roll back together.

### Phase 4: Audit Log

Goal: Owner can know who changed what and when.

Target audit fields:

- `actor_user_id`
- `role`
- `action`
- `entity_type`
- `entity_id`
- `branch_id`
- `before_json`
- `after_json`
- `request_id`
- `created_at`

Status:

- Base table/store/generic HTTP write logging exists.
- Rich service-level before/after logging is pending.

### Phase 5: Contract Discipline

Goal: frontend and backend cannot silently drift.

Tasks:

- Keep `backend/api/openapi.yaml` updated for every endpoint change.
- Keep `backend/api/frontend-contract.md` updated for frontend-facing payloads.
- Later add API contract checks to `check-kingsway.cmd`.

Status: manual, not enforced.

### Phase 6: Modular Tests

Goal: avoid running heavy full tests for every small change.

Command style:

```bash
check-kingsway.cmd -Mode quick -Module student
check-kingsway.cmd -Mode standard -Module teacher
check-kingsway.cmd -Mode full
```

Needed module suites:

- branch integration
- teacher integration
- student integration
- receptionist integration
- room integration
- idempotency/concurrency
- audit log

Status:

- Module command targeting exists.
- Full per-module integration coverage is pending.

### Phase 7: File Policy Standard

Goal: all uploads follow one backend file pipeline.

Target policies:

- `standard-ui`: profile/avatar/display-only optimized file.
- `standard-downloadable`: original stored plus optimized preview when download quality matters.
- `special`: content-preserving, encrypted-at-rest target for official/critical files.

Status: policy exists in project rules, backend naming/pipeline standardization pending.

## Meeting Readiness Notes

The owner meeting may change current business rules. Examples:

- Student status may add `trial`.
- Trial lesson count may control when a student becomes `active`.
- Registration fields may change.
- Assignment/swap rules may become more detailed.
- Payment and attendance state machines may be required.

Architecture rule:

Business-rule changes should happen inside service/domain logic. Infrastructure pieces like idempotency, transaction helper, audit logging, and file policy must stay reusable and should not be rewritten for each new rule.

## Next Recommended Work

1. Split room repository methods from `academic_store.go` into `room_store.go`.
2. Decide whether `staff_store.go` should be renamed to `receptionist_store.go` now or after any future staff roles are confirmed.
3. Add `before_json` / `after_json` support to audit logging.
4. Add service-level audit for branch delete and teacher salary changes first.
5. Run `check-kingsway.cmd -Mode quick -Module backend` only after backend code changes; avoid heavy full checks unless finishing a complete module.

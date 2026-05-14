# DB Migration Discipline

Status: pre-large-update baseline, 2026-05-13.

This project uses SQL migrations in `backend/migrations` with the built-in goose-style runner in `internal/platform/database/migrate.go`.

## Rules

1. Never edit an already-applied migration in normal development.
2. Every schema change must be a new migration file.
3. Migration filenames must use `YYYYMMDDNNNN_snake_case.sql`.
4. Every migration must contain both markers:
   - `-- +goose Up`
   - `-- +goose Down`
5. The `Up` block must be non-empty.
6. Multi-step schema changes must be safe to run once inside one DB transaction.
7. Data backfills must be idempotent where possible:
   - use `IF EXISTS` / `IF NOT EXISTS` when appropriate,
   - avoid assumptions about empty tables,
   - keep destructive updates explicit and documented.
8. New unique business rules must be enforced in DB, not only in frontend/backend validation.
9. New workflow state fields must use DB `CHECK` constraints until a full state-machine table is introduced.
10. When adding a new migration before a major update, update `backend/docs/schema-snapshot-2026-05-13.md` or create a newer snapshot.

## Required Check

Run:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File tools/check-db-migrations.ps1
```

`tools/check-kingsway.ps1` also runs this check when migrations or tooling are touched.

## Future State-Machine Rule

Do not encode complex workflow changes only as loose strings in application code.

The following future areas should get explicit state transition rules before implementation:

- student trial / active / left / graduated lifecycle,
- lesson start / attendance / finish lifecycle,
- payment pending / paid / overdue / cancelled lifecycle,
- teacher assignment / swap lifecycle,
- exam draft / active / submitted / graded lifecycle.

The exact states are intentionally not locked yet because the owner workflow meeting is still pending.

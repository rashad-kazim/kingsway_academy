# Kingsway DB Schema Snapshot

Snapshot date: 2026-05-13  
Source: `backend/migrations` through latest migration `202605110001_user_token_version.sql`  
Purpose: baseline before the next large owner-driven update.

This snapshot is documentation, not an executable migration. New schema changes must be added as new SQL migrations.

## Extensions

- `pgcrypto`
- `btree_gist`

## Migration Chain

| Version | File | Purpose |
| --- | --- | --- |
| 202604290001 | `202604290001_initial_core.sql` | Core branch, users, students, teachers, courses, rooms, classes, schedule, finance, files, exams, notifications |
| 202604290002 | `202604290002_academic_notifications_outbox.sql` | Assignments, notification dedupe, outbox events |
| 202605030001 | `202605030001_branch_operational_hours.sql` | Branch opening/closing hours |
| 202605030002 | `202605030002_allow_branch_file_owner.sql` | Branch-owned file records |
| 202605030003 | `202605030003_staff_profiles.sql` | Receptionist/staff profiles |
| 202605040001 | `202605040001_teacher_profiles.sql` | Teacher profile fields/photo |
| 202605040002 | `202605040002_activate_salary_model_teachers.sql` | Activate teachers that already have salary models |
| 202605040003 | `202605040003_receptionist_profile_fields.sql` | Receptionist gender/address/hire date and user last login |
| 202605050001 | `202605050001_student_hub_statuses.sql` | Student `graduated` status |
| 202605060001 | `202605060001_student_profile_registration_fields.sql` | Student profile fields, parent contacts, course registrations |
| 202605060002 | `202605060002_write_integrity_idempotency.sql` | Idempotency key store |
| 202605080001 | `202605080001_audit_logs.sql` | Audit log table |
| 202605100001 | `202605100001_audit_before_after_and_file_policy.sql` | Audit before/after JSON and file policy |
| 202605110001 | `202605110001_user_token_version.sql` | User token version for logout/revocation |

## Tables

### `branches`

Branch workspace root.

Key fields: `id`, `name`, `slug`, `address`, `logo_file_id`, `opening_time`, `closing_time`, timestamps.  
Constraints/indexes: primary key `id`, unique `slug`, optional `logo_file_id` FK to `files`.

### `users`

Login identity for owner, receptionist, teacher, and student.

Key fields: `id`, `branch_id`, `role`, `email`, `password_hash`, names, `is_active`, `last_login_at`, `token_version`, timestamps.  
Constraints/indexes: unique `email`, role check, non-owner branch required, `branch_id` FK to `branches`.

### `students`

Student profile and lifecycle record.

Key fields: `id`, `branch_id`, `user_id`, `fin_code`, names, `status`, `left_reason`, `birth_date`, `gender`, `phone`, `address`, `profile_photo_file_id`, timestamps.  
Constraints/indexes: unique `fin_code`, unique `user_id`, status check `active|left|graduated`, FIN format check, branch/status and branch/name indexes.

### `student_parent_contacts`

Parent/guardian contacts for students.

Key fields: `id`, `branch_id`, `student_id`, `relation`, `name`, `phones`, `created_at`.  
Constraints/indexes: relation check, FK to branch/student with cascade delete, student lookup index.

### `student_course_registrations`

Student course enrollment/payment metadata for Student & Assignment Hub.

Key fields: `id`, `branch_id`, `student_id`, `course_id`, `teacher_id`, `monthly_amount_cents`, `start_date`, `created_at`.  
Constraints/indexes: unique `(student_id, course_id)`, FK cascade to branch/student/course, teacher set-null, lookup index.

### `teachers`

Teacher profile and status.

Key fields: `id`, `branch_id`, `user_id`, `status`, `birth_date`, `gender`, `phone`, `address`, `profile_photo_file_id`, timestamps.  
Constraints/indexes: unique `user_id`, status check `pending_owner_approval|active|terminated`, branch/status index, profile photo partial index.

### `staff_profiles`

Receptionist profile and HR metadata.

Key fields: `id`, `branch_id`, `user_id`, `role`, `birth_date`, `gender`, `phone`, `address`, `hired_at`, `salary_amount_azn`, `profile_photo_file_id`, timestamps.  
Constraints/indexes: unique `user_id`, role check currently `receptionist`, branch/role index.

### `courses`

Course catalog per branch.

Key fields: `id`, `branch_id`, `name`, `is_active`, timestamps.  
Constraints/indexes: unique `(branch_id, name)`.

### `teacher_course_specializations`

Teacher-to-course specialization join.

Key fields: `teacher_id`, `course_id`, `branch_id`.  
Constraints/indexes: primary key `(teacher_id, course_id)`, cascade delete on teacher/course.

### `course_score_categories`

Exam/result scoring categories per course.

Key fields: `id`, `branch_id`, `course_id`, `name`, score range, document/feedback flags, `sort_order`, `created_at`.  
Constraints/indexes: unique `(course_id, name)`, score range check, document purpose requirement.

### `rooms`

Branch classrooms.

Key fields: `id`, `branch_id`, `name`, `capacity`, `is_active`, timestamps.  
Constraints/indexes: unique `(branch_id, name)`, capacity positive check.

### `classes`

Course class/group.

Key fields: `id`, `branch_id`, `course_id`, `teacher_id`, `name`, `start_date`, `end_date`, `is_active`, timestamps.  
Constraints/indexes: unique `(branch_id, name)`, date range check.

### `class_students`

Class enrollment history.

Key fields: `id`, `branch_id`, `class_id`, `student_id`, `joined_at`, `left_at`, `created_at`.  
Constraints/indexes: unique `(class_id, student_id, joined_at)`, date range check, cascade on class/student.

### `student_teacher_assignments`

Teacher assignment/swap history.

Key fields: `id`, `branch_id`, `student_id`, `teacher_id`, `class_id`, `valid_from`, `valid_to`, `reason`, `created_at`.  
Constraints/indexes: reason check `initial|swap|manual_adjustment`, valid range check, lookup index.

### `salary_models`

Teacher salary model history.

Key fields: `id`, `branch_id`, `teacher_id`, `model_type`, fixed amount, percent basis points, active dates, owner approver, `created_at`.  
Constraints/indexes: model type check `fixed|percent|hybrid`, required fields per model, percent range, active date range.

### `schedule_items`

Lessons, exams, and events.

Key fields: `id`, `branch_id`, `class_id`, `teacher_id`, `room_id`, `item_type`, `title`, start/end, creator, cancellation, timestamps.  
Constraints/indexes: time range check, room no-overlap exclusion, teacher no-overlap exclusion, branch/time index.

### `assignments`

Class assignment/task records.

Key fields: `id`, `branch_id`, `class_id`, `title`, `description`, `due_at`, `material_file_id`, creator, timestamps.  
Constraints/indexes: class cascade delete, branch/class/due index.

### `payments`

Student payment records.

Key fields: `id`, `branch_id`, `student_id`, `amount_cents`, `currency`, `due_date`, `paid_at`, `status`, `receipt_file_id`, creator, timestamps.  
Constraints/indexes: amount positive check, status check `pending|paid|overdue|cancelled`, branch/status/due index.

### `files`

File metadata and object storage location.

Key fields: `id`, `branch_id`, `uploader_user_id`, `owner_type`, `owner_id`, `category`, `policy`, `purpose`, filename/MIME/size/hash/storage, retention, deletion, `created_at`.  
Constraints/indexes: owner type check includes student/teacher/class/exam/payment/event/branch/staff, category check, policy check `standard-ui|standard-downloadable|special`, purpose check, unique `storage_key`, branch/owner index, retention index, policy index.

### `exams`

Exam records.

Key fields: `id`, `branch_id`, `course_id`, `class_id`, `schedule_item_id`, `title`, creator, timestamps.

### `exam_participants`

Exam participant join.

Key fields: `exam_id`, `student_id`, `branch_id`, `rsvp_status`.  
Constraints/indexes: primary key `(exam_id, student_id)`, RSVP status check.

### `exam_results`

Exam category result entries.

Key fields: `id`, `branch_id`, `exam_id`, `student_id`, `category_id`, `score`, `feedback`, `document_file_id`, `entered_by_teacher_id`, timestamps.  
Constraints/indexes: unique `(exam_id, student_id, category_id)`.

### `notifications`

User notifications.

Key fields: `id`, `branch_id`, `recipient_user_id`, `type`, `payload`, `dedupe_key`, `read_at`, `created_at`.  
Constraints/indexes: unread recipient index, partial unique `dedupe_key`.

### `outbox_events`

Async event outbox.

Key fields: `id`, `topic`, `payload`, `status`, `attempts`, `next_attempt_at`, `locked_at`, `last_error`, `published_at`, `created_at`.  
Constraints/indexes: status check, pending/publishing index.

### `idempotency_keys`

HTTP write idempotency records.

Key fields: `actor_user_id`, `method`, `path`, `key`, `request_hash`, `status`, `response_status`, `response_body`, timestamps, `expires_at`.  
Constraints/indexes: primary key `(actor_user_id, method, path, key)`, status check, expiry index.

### `audit_logs`

Security/operation audit trail.

Key fields: `id`, actor user/role/branch, action, entity type/id/branch, request/idempotency IDs, `before_json`, `after_json`, metadata, `created_at`.  
Constraints/indexes: role check, created/actor/entity/branch indexes.

### `schema_migrations`

Built-in migration runner bookkeeping.

Key fields: `version`, `applied_at`.  
Constraints/indexes: primary key `version`.

## Current Workflow Fields That May Become State Machines

These are currently simple constrained statuses and should be revisited after the owner meeting:

- `students.status`: `active`, `left`, `graduated`
- `teachers.status`: `pending_owner_approval`, `active`, `terminated`
- `payments.status`: `pending`, `paid`, `overdue`, `cancelled`
- `schedule_items.item_type` plus `cancelled_at`
- `exam_participants.rsvp_status`
- `outbox_events.status`

Do not add ad hoc extra values to these fields without designing transition rules first.

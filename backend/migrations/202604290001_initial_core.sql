-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE branches (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    slug text NOT NULL UNIQUE,
    address text,
    logo_file_id uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid REFERENCES branches(id),
    role text NOT NULL CHECK (role IN ('owner', 'receptionist', 'teacher', 'student')),
    email text NOT NULL UNIQUE,
    password_hash text NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_branch_scope_required CHECK (role = 'owner' OR branch_id IS NOT NULL)
);

CREATE TABLE students (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches(id),
    user_id uuid UNIQUE REFERENCES users(id),
    fin_code char(7) NOT NULL UNIQUE CHECK (fin_code ~ '^[A-Z0-9]{7}$'),
    first_name text NOT NULL,
    last_name text NOT NULL,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'left')),
    left_reason text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT students_left_reason_required CHECK (status != 'left' OR left_reason IS NOT NULL)
);

CREATE INDEX students_branch_status_idx ON students(branch_id, status);
CREATE INDEX students_branch_name_idx ON students(branch_id, last_name, first_name);

CREATE TABLE teachers (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches(id),
    user_id uuid NOT NULL UNIQUE REFERENCES users(id),
    status text NOT NULL DEFAULT 'pending_owner_approval'
        CHECK (status IN ('pending_owner_approval', 'active', 'terminated')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX teachers_branch_status_idx ON teachers(branch_id, status);

CREATE TABLE courses (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches(id),
    name text NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (branch_id, name)
);

CREATE TABLE teacher_course_specializations (
    teacher_id uuid NOT NULL REFERENCES teachers(id) ON DELETE CASCADE,
    course_id uuid NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    branch_id uuid NOT NULL REFERENCES branches(id),
    PRIMARY KEY (teacher_id, course_id)
);

CREATE TABLE course_score_categories (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches(id),
    course_id uuid NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    name text NOT NULL,
    min_score numeric(5,2) NOT NULL,
    max_score numeric(5,2) NOT NULL,
    requires_feedback boolean NOT NULL DEFAULT false,
    requires_document boolean NOT NULL DEFAULT false,
    document_purpose text CHECK (document_purpose IN ('exam_writing')),
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (course_id, name),
    CONSTRAINT course_score_range_valid CHECK (min_score <= max_score),
    CONSTRAINT course_document_purpose_required CHECK (requires_document = false OR document_purpose IS NOT NULL)
);

CREATE TABLE rooms (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches(id),
    name text NOT NULL,
    capacity integer NOT NULL DEFAULT 1 CHECK (capacity > 0),
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (branch_id, name)
);

CREATE TABLE classes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches(id),
    course_id uuid NOT NULL REFERENCES courses(id),
    teacher_id uuid NOT NULL REFERENCES teachers(id),
    name text NOT NULL,
    start_date date NOT NULL,
    end_date date,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (branch_id, name),
    CONSTRAINT classes_date_range_valid CHECK (end_date IS NULL OR start_date <= end_date)
);

CREATE TABLE class_students (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches(id),
    class_id uuid NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    student_id uuid NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    joined_at date NOT NULL,
    left_at date,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (class_id, student_id, joined_at),
    CONSTRAINT class_students_date_range_valid CHECK (left_at IS NULL OR joined_at <= left_at)
);

CREATE TABLE student_teacher_assignments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches(id),
    student_id uuid NOT NULL REFERENCES students(id),
    teacher_id uuid NOT NULL REFERENCES teachers(id),
    class_id uuid REFERENCES classes(id),
    valid_from date NOT NULL,
    valid_to date,
    reason text NOT NULL DEFAULT 'initial' CHECK (reason IN ('initial', 'swap', 'manual_adjustment')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT student_teacher_assignment_range_valid CHECK (valid_to IS NULL OR valid_from <= valid_to)
);

CREATE INDEX student_teacher_assignments_lookup_idx
    ON student_teacher_assignments(branch_id, student_id, valid_from, valid_to);

CREATE TABLE salary_models (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches(id),
    teacher_id uuid NOT NULL REFERENCES teachers(id),
    model_type text NOT NULL CHECK (model_type IN ('fixed', 'percent', 'hybrid')),
    fixed_monthly_amount_cents bigint,
    student_percent_basis_points integer,
    active_from date NOT NULL,
    active_to date,
    approved_by_owner_user_id uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT salary_model_range_valid CHECK (active_to IS NULL OR active_from <= active_to),
    CONSTRAINT salary_model_fixed_required CHECK (
        model_type NOT IN ('fixed', 'hybrid') OR fixed_monthly_amount_cents IS NOT NULL
    ),
    CONSTRAINT salary_model_percent_required CHECK (
        model_type NOT IN ('percent', 'hybrid') OR student_percent_basis_points IS NOT NULL
    ),
    CONSTRAINT salary_model_percent_range CHECK (
        student_percent_basis_points IS NULL
        OR (student_percent_basis_points >= 0 AND student_percent_basis_points <= 10000)
    )
);

CREATE TABLE schedule_items (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches(id),
    class_id uuid REFERENCES classes(id),
    teacher_id uuid NOT NULL REFERENCES teachers(id),
    room_id uuid NOT NULL REFERENCES rooms(id),
    item_type text NOT NULL CHECK (item_type IN ('lesson', 'exam', 'event')),
    title text NOT NULL,
    starts_at timestamptz NOT NULL,
    ends_at timestamptz NOT NULL,
    created_by_user_id uuid NOT NULL REFERENCES users(id),
    cancelled_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT schedule_item_time_range_valid CHECK (starts_at < ends_at),
    CONSTRAINT schedule_room_no_overlap EXCLUDE USING gist (
        branch_id WITH =,
        room_id WITH =,
        tstzrange(starts_at, ends_at, '[)') WITH &&
    ) WHERE (cancelled_at IS NULL),
    CONSTRAINT schedule_teacher_no_overlap EXCLUDE USING gist (
        branch_id WITH =,
        teacher_id WITH =,
        tstzrange(starts_at, ends_at, '[)') WITH &&
    ) WHERE (cancelled_at IS NULL)
);

CREATE INDEX schedule_items_branch_time_idx ON schedule_items(branch_id, starts_at, ends_at);

CREATE TABLE payments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches(id),
    student_id uuid NOT NULL REFERENCES students(id),
    amount_cents bigint NOT NULL CHECK (amount_cents > 0),
    currency char(3) NOT NULL DEFAULT 'AZN',
    due_date date NOT NULL,
    paid_at timestamptz,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid', 'overdue', 'cancelled')),
    receipt_file_id uuid,
    created_by_user_id uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX payments_branch_status_due_idx ON payments(branch_id, status, due_date);

CREATE TABLE files (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches(id),
    uploader_user_id uuid NOT NULL REFERENCES users(id),
    owner_type text NOT NULL CHECK (owner_type IN ('student', 'teacher', 'class', 'exam', 'payment', 'event')),
    owner_id uuid NOT NULL,
    category text NOT NULL CHECK (category IN ('standard', 'special')),
    purpose text NOT NULL CHECK (
        purpose IN ('material', 'payment_receipt', 'event', 'passport', 'diploma', 'profile_photo', 'exam_writing')
    ),
    original_filename text NOT NULL,
    mime_type text NOT NULL,
    original_size_bytes bigint NOT NULL CHECK (original_size_bytes >= 0),
    stored_size_bytes bigint NOT NULL CHECK (stored_size_bytes >= 0),
    original_sha256 char(64) NOT NULL,
    storage_bucket text NOT NULL,
    storage_key text NOT NULL UNIQUE,
    retention_until timestamptz,
    deleted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT special_files_hash_size_preserved CHECK (
        category != 'special' OR stored_size_bytes = original_size_bytes
    ),
    CONSTRAINT writing_retention_required CHECK (
        purpose != 'exam_writing' OR retention_until IS NOT NULL
    )
);

CREATE INDEX files_branch_owner_idx ON files(branch_id, owner_type, owner_id);
CREATE INDEX files_retention_idx ON files(retention_until) WHERE deleted_at IS NULL;

ALTER TABLE branches
    ADD CONSTRAINT branches_logo_file_fk FOREIGN KEY (logo_file_id) REFERENCES files(id);

ALTER TABLE payments
    ADD CONSTRAINT payments_receipt_file_fk FOREIGN KEY (receipt_file_id) REFERENCES files(id);

CREATE TABLE exams (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches(id),
    course_id uuid NOT NULL REFERENCES courses(id),
    class_id uuid REFERENCES classes(id),
    schedule_item_id uuid REFERENCES schedule_items(id),
    title text NOT NULL,
    created_by_user_id uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE exam_participants (
    exam_id uuid NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    student_id uuid NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    branch_id uuid NOT NULL REFERENCES branches(id),
    rsvp_status text NOT NULL DEFAULT 'pending' CHECK (rsvp_status IN ('pending', 'accepted', 'declined')),
    PRIMARY KEY (exam_id, student_id)
);

CREATE TABLE exam_results (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches(id),
    exam_id uuid NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    student_id uuid NOT NULL REFERENCES students(id),
    category_id uuid NOT NULL REFERENCES course_score_categories(id),
    score numeric(5,2) NOT NULL,
    feedback text,
    document_file_id uuid REFERENCES files(id),
    entered_by_teacher_id uuid NOT NULL REFERENCES teachers(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (exam_id, student_id, category_id)
);

CREATE TABLE notifications (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid REFERENCES branches(id),
    recipient_user_id uuid NOT NULL REFERENCES users(id),
    type text NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    read_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX notifications_recipient_unread_idx
    ON notifications(recipient_user_id, created_at DESC)
    WHERE read_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS exam_results;
DROP TABLE IF EXISTS exam_participants;
DROP TABLE IF EXISTS exams;
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_receipt_file_fk;
ALTER TABLE branches DROP CONSTRAINT IF EXISTS branches_logo_file_fk;
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS schedule_items;
DROP TABLE IF EXISTS salary_models;
DROP TABLE IF EXISTS student_teacher_assignments;
DROP TABLE IF EXISTS class_students;
DROP TABLE IF EXISTS classes;
DROP TABLE IF EXISTS rooms;
DROP TABLE IF EXISTS course_score_categories;
DROP TABLE IF EXISTS teacher_course_specializations;
DROP TABLE IF EXISTS courses;
DROP TABLE IF EXISTS teachers;
DROP TABLE IF EXISTS students;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS branches;

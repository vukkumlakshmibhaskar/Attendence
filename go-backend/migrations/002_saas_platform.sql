BEGIN;

CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(180) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(160) NOT NULL,
    email VARCHAR(180) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role VARCHAR(40) NOT NULL CHECK (role IN ('admin', 'teacher')),
    status VARCHAR(40) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS academic_classes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(120) NOT NULL,
    section VARCHAR(40) NOT NULL DEFAULT '',
    grade_level VARCHAR(40) NOT NULL DEFAULT '',
    status VARCHAR(40) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS subjects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(120) NOT NULL,
    code VARCHAR(40) NOT NULL DEFAULT '',
    status VARCHAR(40) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS teacher_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    teacher_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    class_id UUID NOT NULL REFERENCES academic_classes(id) ON DELETE CASCADE,
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (teacher_id, class_id, subject_id)
);

ALTER TABLE students ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE students ADD COLUMN IF NOT EXISTS student_id VARCHAR(80);
ALTER TABLE students ADD COLUMN IF NOT EXISTS class_id UUID REFERENCES academic_classes(id) ON DELETE SET NULL;
ALTER TABLE students ADD COLUMN IF NOT EXISTS consent_status VARCHAR(40) NOT NULL DEFAULT 'pending';
ALTER TABLE students ADD COLUMN IF NOT EXISTS status VARCHAR(40) NOT NULL DEFAULT 'active';

CREATE TABLE IF NOT EXISTS face_enrollments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    pose VARCHAR(40) NOT NULL,
    image_sha256 CHAR(64) NOT NULL,
    embedding_json JSONB NOT NULL,
    model_path TEXT NOT NULL,
    model_mode VARCHAR(80) NOT NULL,
    is_demo BOOLEAN NOT NULL DEFAULT TRUE,
    quality_score NUMERIC(6, 5) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (student_id, pose)
);

CREATE TABLE IF NOT EXISTS teacher_face_enrollments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    teacher_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    pose VARCHAR(40) NOT NULL,
    image_sha256 CHAR(64) NOT NULL,
    embedding_json JSONB NOT NULL,
    model_path TEXT NOT NULL,
    model_mode VARCHAR(80) NOT NULL,
    is_demo BOOLEAN NOT NULL DEFAULT TRUE,
    quality_score NUMERIC(6, 5) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (teacher_id, pose)
);

CREATE TABLE IF NOT EXISTS cameras (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(120) NOT NULL,
    room VARCHAR(120) NOT NULL DEFAULT '',
    rtsp_url TEXT NOT NULL DEFAULT '',
    class_id UUID REFERENCES academic_classes(id) ON DELETE SET NULL,
    health_status VARCHAR(40) NOT NULL DEFAULT 'unknown',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS attendance_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    title VARCHAR(180) NOT NULL,
    class_id UUID NOT NULL REFERENCES academic_classes(id) ON DELETE CASCADE,
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    teacher_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    late_threshold_minutes INTEGER NOT NULL DEFAULT 10,
    status VARCHAR(40) NOT NULL DEFAULT 'scheduled',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS attendance_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    session_id UUID NOT NULL REFERENCES attendance_sessions(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    status VARCHAR(40) NOT NULL CHECK (status IN ('present', 'late', 'review', 'absent')),
    confidence NUMERIC(6, 5) NOT NULL DEFAULT 0,
    source VARCHAR(40) NOT NULL DEFAULT 'manual',
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (session_id, student_id)
);

CREATE TABLE IF NOT EXISTS manual_corrections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    attendance_event_id UUID NOT NULL REFERENCES attendance_events(id) ON DELETE CASCADE,
    corrected_status VARCHAR(40) NOT NULL CHECK (corrected_status IN ('present', 'late', 'review', 'absent')),
    reason TEXT NOT NULL DEFAULT '',
    corrected_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(120) NOT NULL,
    entity VARCHAR(120) NOT NULL,
    entity_id VARCHAR(120) NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_users_org_role ON users (organization_id, role, status);
CREATE INDEX IF NOT EXISTS idx_students_org_class ON students (organization_id, class_id, status);
CREATE INDEX IF NOT EXISTS idx_face_enrollments_student ON face_enrollments (student_id, pose);
CREATE INDEX IF NOT EXISTS idx_teacher_face_enrollments_teacher ON teacher_face_enrollments (teacher_id, pose);
CREATE INDEX IF NOT EXISTS idx_attendance_sessions_org_time ON attendance_sessions (organization_id, start_time DESC);
CREATE INDEX IF NOT EXISTS idx_attendance_events_session_status ON attendance_events (session_id, status);
CREATE INDEX IF NOT EXISTS idx_cameras_org_health ON cameras (organization_id, health_status);

COMMIT;

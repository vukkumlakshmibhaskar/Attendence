-- Classroom Attendance and Cognitive Analysis System
-- Step 1: PostgreSQL database schema
--
-- CPU-friendly architecture note:
-- Face matching is intentionally NOT done inside PostgreSQL during live frame
-- processing. The Go backend should load active 128-dimensional embeddings at
-- startup and match in memory.

BEGIN;

-- pgcrypto is used for gen_random_uuid().
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- pgvector stores fixed-size 128d face embeddings cleanly.
-- Install package first if needed:
--   Ubuntu/Debian: sudo apt install postgresql-16-pgvector
--   Docker: use image pgvector/pgvector:pg16
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TYPE attendance_status AS ENUM (
    'present',
    'late',
    'absent',
    'excused'
);

CREATE TYPE cognitive_emotion AS ENUM (
    'attentive',
    'tired',
    'distracted',
    'unknown'
);

CREATE TYPE gaze_state AS ENUM (
    'screen',
    'away',
    'unknown'
);

CREATE TABLE classrooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(120) NOT NULL,
    code VARCHAR(40) UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE students (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_student_id VARCHAR(80) UNIQUE NOT NULL,
    full_name VARCHAR(160) NOT NULL,
    email VARCHAR(180) UNIQUE,
    classroom_id UUID REFERENCES classrooms(id) ON DELETE SET NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One student can have multiple enrollment embeddings captured under different
-- lighting/pose conditions. The Go service should load active rows from here.
CREATE TABLE student_face_embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    embedding vector(128) NOT NULL,
    source_image_sha256 CHAR(64),
    model_name VARCHAR(120) NOT NULL DEFAULT 'dlib_resnet_128d_cpu',
    model_version VARCHAR(80) NOT NULL DEFAULT '1.0',
    quality_score NUMERIC(5, 4) CHECK (quality_score IS NULL OR quality_score BETWEEN 0 AND 1),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    captured_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- A class session represents one attendance window.
CREATE TABLE class_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    classroom_id UUID REFERENCES classrooms(id) ON DELETE SET NULL,
    title VARCHAR(180),
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (ends_at IS NULL OR ends_at > starts_at)
);

-- One attendance row per student per class session.
CREATE TABLE attendance_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES class_sessions(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    status attendance_status NOT NULL DEFAULT 'present',
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    recognition_confidence NUMERIC(6, 5) CHECK (
        recognition_confidence IS NULL OR recognition_confidence BETWEEN 0 AND 1
    ),
    matched_embedding_id UUID REFERENCES student_face_embeddings(id) ON DELETE SET NULL,
    frames_matched INTEGER NOT NULL DEFAULT 1 CHECK (frames_matched >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (session_id, student_id)
);

-- Cognitive snapshots are sampled observations attached to a session/student.
-- The Python service owns these labels; the Go service persists them.
CREATE TABLE cognitive_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES class_sessions(id) ON DELETE CASCADE,
    student_id UUID REFERENCES students(id) ON DELETE SET NULL,
    attendance_log_id UUID REFERENCES attendance_logs(id) ON DELETE SET NULL,
    captured_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    emotion cognitive_emotion NOT NULL DEFAULT 'unknown',
    emotion_confidence NUMERIC(6, 5) CHECK (
        emotion_confidence IS NULL OR emotion_confidence BETWEEN 0 AND 1
    ),

    gaze gaze_state NOT NULL DEFAULT 'unknown',
    gaze_confidence NUMERIC(6, 5) CHECK (
        gaze_confidence IS NULL OR gaze_confidence BETWEEN 0 AND 1
    ),

    is_live BOOLEAN,
    liveness_confidence NUMERIC(6, 5) CHECK (
        liveness_confidence IS NULL OR liveness_confidence BETWEEN 0 AND 1
    ),

    face_box_x INTEGER CHECK (face_box_x IS NULL OR face_box_x >= 0),
    face_box_y INTEGER CHECK (face_box_y IS NULL OR face_box_y >= 0),
    face_box_width INTEGER CHECK (face_box_width IS NULL OR face_box_width >= 0),
    face_box_height INTEGER CHECK (face_box_height IS NULL OR face_box_height >= 0),

    processing_ms INTEGER CHECK (processing_ms IS NULL OR processing_ms >= 0),
    service_version VARCHAR(80),
    raw_response JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Frames that fail recognition can be logged without storing the full image.
CREATE TABLE unknown_face_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID REFERENCES class_sessions(id) ON DELETE CASCADE,
    captured_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reason VARCHAR(160) NOT NULL,
    best_distance NUMERIC(8, 6),
    face_box_x INTEGER CHECK (face_box_x IS NULL OR face_box_x >= 0),
    face_box_y INTEGER CHECK (face_box_y IS NULL OR face_box_y >= 0),
    face_box_width INTEGER CHECK (face_box_width IS NULL OR face_box_width >= 0),
    face_box_height INTEGER CHECK (face_box_height IS NULL OR face_box_height >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_students_classroom_active
    ON students (classroom_id, is_active);

CREATE INDEX idx_face_embeddings_student_active
    ON student_face_embeddings (student_id, is_active);

-- Optional vector index for offline enrollment/search/admin tools.
-- Live API matching should still use the Go in-memory cache.
CREATE INDEX idx_face_embeddings_embedding_hnsw
    ON student_face_embeddings
    USING hnsw (embedding vector_l2_ops);

CREATE INDEX idx_class_sessions_classroom_time
    ON class_sessions (classroom_id, starts_at DESC);

CREATE INDEX idx_attendance_logs_session
    ON attendance_logs (session_id, status);

CREATE INDEX idx_attendance_logs_student_time
    ON attendance_logs (student_id, first_seen_at DESC);

CREATE INDEX idx_cognitive_snapshots_session_time
    ON cognitive_snapshots (session_id, captured_at DESC);

CREATE INDEX idx_cognitive_snapshots_student_time
    ON cognitive_snapshots (student_id, captured_at DESC);

CREATE INDEX idx_unknown_face_events_session_time
    ON unknown_face_events (session_id, captured_at DESC);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_classrooms_updated_at
BEFORE UPDATE ON classrooms
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_students_updated_at
BEFORE UPDATE ON students
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_class_sessions_updated_at
BEFORE UPDATE ON class_sessions
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_attendance_logs_updated_at
BEFORE UPDATE ON attendance_logs
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;

-- Startup query for the Go backend in-memory face cache:
--
-- SELECT
--     s.id AS student_id,
--     s.external_student_id,
--     s.full_name,
--     e.id AS embedding_id,
--     e.embedding::text AS embedding
-- FROM students s
-- JOIN student_face_embeddings e ON e.student_id = s.id
-- WHERE s.is_active = TRUE
--   AND e.is_active = TRUE;

-- Attendance upsert pattern for repeated sightings:
--
-- INSERT INTO attendance_logs (
--     session_id,
--     student_id,
--     status,
--     first_seen_at,
--     last_seen_at,
--     recognition_confidence,
--     matched_embedding_id,
--     frames_matched
-- )
-- VALUES ($1, $2, 'present', now(), now(), $3, $4, 1)
-- ON CONFLICT (session_id, student_id)
-- DO UPDATE SET
--     last_seen_at = EXCLUDED.last_seen_at,
--     recognition_confidence = GREATEST(
--         attendance_logs.recognition_confidence,
--         EXCLUDED.recognition_confidence
--     ),
--     matched_embedding_id = EXCLUDED.matched_embedding_id,
--     frames_matched = attendance_logs.frames_matched + 1
-- RETURNING id;

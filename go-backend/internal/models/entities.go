package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

type Organization struct {
	ID        string    `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name      string    `gorm:"column:name" json:"name"`
	Slug      string    `gorm:"column:slug;uniqueIndex" json:"slug"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Organization) TableName() string {
	return "organizations"
}

type User struct {
	ID             string    `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID string    `gorm:"column:organization_id;type:uuid;index" json:"organization_id"`
	Name           string    `gorm:"column:name" json:"name"`
	Email          string    `gorm:"column:email;uniqueIndex" json:"email"`
	PasswordHash   string    `gorm:"column:password_hash" json:"-"`
	Role           string    `gorm:"column:role;index" json:"role"`
	Status         string    `gorm:"column:status;index" json:"status"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}

type AcademicClass struct {
	ID             string    `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID string    `gorm:"column:organization_id;type:uuid;index" json:"organization_id"`
	Name           string    `gorm:"column:name" json:"name"`
	Section        string    `gorm:"column:section" json:"section"`
	GradeLevel     string    `gorm:"column:grade_level" json:"grade_level"`
	Status         string    `gorm:"column:status" json:"status"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (AcademicClass) TableName() string {
	return "academic_classes"
}

type Subject struct {
	ID             string    `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID string    `gorm:"column:organization_id;type:uuid;index" json:"organization_id"`
	Name           string    `gorm:"column:name" json:"name"`
	Code           string    `gorm:"column:code" json:"code"`
	Status         string    `gorm:"column:status" json:"status"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Subject) TableName() string {
	return "subjects"
}

type TeacherAssignment struct {
	ID             string    `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID string    `gorm:"column:organization_id;type:uuid;index" json:"organization_id"`
	TeacherID      string    `gorm:"column:teacher_id;type:uuid;index" json:"teacher_id"`
	ClassID        string    `gorm:"column:class_id;type:uuid;index" json:"class_id"`
	SubjectID      string    `gorm:"column:subject_id;type:uuid;index" json:"subject_id"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
}

func (TeacherAssignment) TableName() string {
	return "teacher_assignments"
}

type Student struct {
	ID                string          `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID    string          `gorm:"column:organization_id;type:uuid;index" json:"organization_id"`
	StudentID         string          `gorm:"column:student_id;index" json:"student_id"`
	ExternalStudentID string          `gorm:"column:external_student_id;unique" json:"external_student_id"`
	FullName          string          `gorm:"column:full_name" json:"full_name"`
	Email             *string         `gorm:"column:email;unique" json:"email"`
	ClassID           *string         `gorm:"column:class_id;type:uuid" json:"class_id"`
	ClassroomID       *string         `gorm:"column:classroom_id;type:uuid" json:"classroom_id"`
	ConsentStatus     string          `gorm:"column:consent_status" json:"consent_status"`
	Status            string          `gorm:"column:status" json:"status"`
	IsActive          bool            `gorm:"column:is_active" json:"is_active"`
	CreatedAt         time.Time       `gorm:"column:created_at" json:"created_at"`
	UpdatedAt         time.Time       `gorm:"column:updated_at" json:"updated_at"`
	Embeddings        []FaceEmbedding `gorm:"-"`
}

func (Student) TableName() string {
	return "students"
}

type FaceEmbedding struct {
	ID                string    `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	StudentID         string    `gorm:"column:student_id;type:uuid"`
	Embedding         string    `gorm:"column:embedding;type:vector(128)"`
	SourceImageSHA256 *string   `gorm:"column:source_image_sha256"`
	ModelName         string    `gorm:"column:model_name"`
	ModelVersion      string    `gorm:"column:model_version"`
	QualityScore      *float64  `gorm:"column:quality_score"`
	IsActive          bool      `gorm:"column:is_active"`
	CapturedAt        time.Time `gorm:"column:captured_at"`
	CreatedAt         time.Time `gorm:"column:created_at"`
}

func (FaceEmbedding) TableName() string {
	return "student_face_embeddings"
}

type FaceEnrollment struct {
	ID             string    `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID string    `gorm:"column:organization_id;type:uuid;index" json:"organization_id"`
	StudentID      string    `gorm:"column:student_id;type:uuid;index" json:"student_id"`
	Pose           string    `gorm:"column:pose;index" json:"pose"`
	ImageSHA256    string    `gorm:"column:image_sha256" json:"image_sha256"`
	EmbeddingJSON  JSONB     `gorm:"column:embedding_json;type:jsonb" json:"embedding_json"`
	ModelPath      string    `gorm:"column:model_path" json:"model_path"`
	ModelMode      string    `gorm:"column:model_mode" json:"model_mode"`
	IsDemo         bool      `gorm:"column:is_demo" json:"is_demo"`
	QualityScore   float64   `gorm:"column:quality_score" json:"quality_score"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
}

func (FaceEnrollment) TableName() string {
	return "face_enrollments"
}

type TeacherFaceEnrollment struct {
	ID             string    `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID string    `gorm:"column:organization_id;type:uuid;index" json:"organization_id"`
	TeacherID      string    `gorm:"column:teacher_id;type:uuid;index" json:"teacher_id"`
	Pose           string    `gorm:"column:pose;index" json:"pose"`
	ImageSHA256    string    `gorm:"column:image_sha256" json:"image_sha256"`
	EmbeddingJSON  JSONB     `gorm:"column:embedding_json;type:jsonb" json:"embedding_json"`
	ModelPath      string    `gorm:"column:model_path" json:"model_path"`
	ModelMode      string    `gorm:"column:model_mode" json:"model_mode"`
	IsDemo         bool      `gorm:"column:is_demo" json:"is_demo"`
	QualityScore   float64   `gorm:"column:quality_score" json:"quality_score"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
}

func (TeacherFaceEnrollment) TableName() string {
	return "teacher_face_enrollments"
}

type Camera struct {
	ID             string    `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID string    `gorm:"column:organization_id;type:uuid;index" json:"organization_id"`
	Name           string    `gorm:"column:name" json:"name"`
	Room           string    `gorm:"column:room" json:"room"`
	RTSPURL        string    `gorm:"column:rtsp_url" json:"rtsp_url"`
	ClassID        *string   `gorm:"column:class_id;type:uuid" json:"class_id"`
	HealthStatus   string    `gorm:"column:health_status" json:"health_status"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Camera) TableName() string {
	return "cameras"
}

type AttendanceSession struct {
	ID             string    `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID string    `gorm:"column:organization_id;type:uuid;index" json:"organization_id"`
	Title          string    `gorm:"column:title" json:"title"`
	ClassID        string    `gorm:"column:class_id;type:uuid;index" json:"class_id"`
	SubjectID      string    `gorm:"column:subject_id;type:uuid;index" json:"subject_id"`
	TeacherID      string    `gorm:"column:teacher_id;type:uuid;index" json:"teacher_id"`
	StartTime      time.Time `gorm:"column:start_time" json:"start_time"`
	EndTime        time.Time `gorm:"column:end_time" json:"end_time"`
	LateThreshold  int       `gorm:"column:late_threshold_minutes" json:"late_threshold_minutes"`
	Status         string    `gorm:"column:status" json:"status"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (AttendanceSession) TableName() string {
	return "attendance_sessions"
}

type AttendanceEvent struct {
	ID             string    `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID string    `gorm:"column:organization_id;type:uuid;index" json:"organization_id"`
	SessionID      string    `gorm:"column:session_id;type:uuid;index" json:"session_id"`
	StudentID      string    `gorm:"column:student_id;type:uuid;index" json:"student_id"`
	Status         string    `gorm:"column:status" json:"status"`
	Confidence     float64   `gorm:"column:confidence" json:"confidence"`
	Source         string    `gorm:"column:source" json:"source"`
	OccurredAt     time.Time `gorm:"column:occurred_at" json:"occurred_at"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (AttendanceEvent) TableName() string {
	return "attendance_events"
}

type ManualCorrection struct {
	ID                string    `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID    string    `gorm:"column:organization_id;type:uuid;index" json:"organization_id"`
	AttendanceEventID string    `gorm:"column:attendance_event_id;type:uuid;index" json:"attendance_event_id"`
	CorrectedStatus   string    `gorm:"column:corrected_status" json:"corrected_status"`
	Reason            string    `gorm:"column:reason" json:"reason"`
	CorrectedBy       string    `gorm:"column:corrected_by;type:uuid" json:"corrected_by"`
	CreatedAt         time.Time `gorm:"column:created_at" json:"created_at"`
}

func (ManualCorrection) TableName() string {
	return "manual_corrections"
}

type AuditLog struct {
	ID             string    `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrganizationID string    `gorm:"column:organization_id;type:uuid;index" json:"organization_id"`
	ActorID        string    `gorm:"column:actor_id;type:uuid;index" json:"actor_id"`
	Action         string    `gorm:"column:action" json:"action"`
	Entity         string    `gorm:"column:entity" json:"entity"`
	EntityID       string    `gorm:"column:entity_id" json:"entity_id"`
	Metadata       JSONB     `gorm:"column:metadata;type:jsonb" json:"metadata"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

type AttendanceLog struct {
	ID                    string    `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	SessionID             string    `gorm:"column:session_id;type:uuid"`
	StudentID             string    `gorm:"column:student_id;type:uuid"`
	Status                string    `gorm:"column:status"`
	FirstSeenAt           time.Time `gorm:"column:first_seen_at"`
	LastSeenAt            time.Time `gorm:"column:last_seen_at"`
	RecognitionConfidence *float64  `gorm:"column:recognition_confidence"`
	MatchedEmbeddingID    *string   `gorm:"column:matched_embedding_id;type:uuid"`
	FramesMatched         int       `gorm:"column:frames_matched"`
	CreatedAt             time.Time `gorm:"column:created_at"`
	UpdatedAt             time.Time `gorm:"column:updated_at"`
}

func (AttendanceLog) TableName() string {
	return "attendance_logs"
}

type CognitiveSnapshot struct {
	ID                 string    `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	SessionID          string    `gorm:"column:session_id;type:uuid"`
	StudentID          *string   `gorm:"column:student_id;type:uuid"`
	AttendanceLogID    *string   `gorm:"column:attendance_log_id;type:uuid"`
	CapturedAt         time.Time `gorm:"column:captured_at"`
	Emotion            string    `gorm:"column:emotion"`
	EmotionConfidence  float64   `gorm:"column:emotion_confidence"`
	Gaze               string    `gorm:"column:gaze"`
	GazeConfidence     float64   `gorm:"column:gaze_confidence"`
	IsLive             bool      `gorm:"column:is_live"`
	LivenessConfidence float64   `gorm:"column:liveness_confidence"`
	FaceBoxX           int       `gorm:"column:face_box_x"`
	FaceBoxY           int       `gorm:"column:face_box_y"`
	FaceBoxWidth       int       `gorm:"column:face_box_width"`
	FaceBoxHeight      int       `gorm:"column:face_box_height"`
	ProcessingMS       int       `gorm:"column:processing_ms"`
	ServiceVersion     string    `gorm:"column:service_version"`
	RawResponse        JSONB     `gorm:"column:raw_response;type:jsonb"`
	CreatedAt          time.Time `gorm:"column:created_at"`
}

func (CognitiveSnapshot) TableName() string {
	return "cognitive_snapshots"
}

type UnknownFaceEvent struct {
	ID            string    `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	SessionID     *string   `gorm:"column:session_id;type:uuid"`
	CapturedAt    time.Time `gorm:"column:captured_at"`
	Reason        string    `gorm:"column:reason"`
	BestDistance  *float64  `gorm:"column:best_distance"`
	FaceBoxX      int       `gorm:"column:face_box_x"`
	FaceBoxY      int       `gorm:"column:face_box_y"`
	FaceBoxWidth  int       `gorm:"column:face_box_width"`
	FaceBoxHeight int       `gorm:"column:face_box_height"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

func (UnknownFaceEvent) TableName() string {
	return "unknown_face_events"
}

type JSONB []byte

func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "{}", nil
	}
	return string(j), nil
}

func (j *JSONB) Scan(value any) error {
	switch typed := value.(type) {
	case nil:
		*j = nil
	case []byte:
		*j = append((*j)[0:0], typed...)
	case string:
		*j = append((*j)[0:0], typed...)
	default:
		return fmt.Errorf("unsupported jsonb scan type %T", value)
	}
	return nil
}

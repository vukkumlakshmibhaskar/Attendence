package models

import (
	"time"

	"classroom-attendance/go-backend/internal/recognition"
)

type FrameRequest struct {
	SessionID   string `json:"session_id"`
	ImageBase64 string `json:"image_base64"`
	CapturedAt  string `json:"captured_at,omitempty"`
}

type FaceResult struct {
	PersonType            string          `json:"person_type,omitempty"`
	StudentID             string          `json:"student_id,omitempty"`
	TeacherID             string          `json:"teacher_id,omitempty"`
	ExternalStudentID     string          `json:"external_student_id,omitempty"`
	FullName              string          `json:"full_name,omitempty"`
	Email                 string          `json:"email,omitempty"`
	MatchedFaceID         string          `json:"matched_face_id,omitempty"`
	MatchedFaceSource     string          `json:"matched_face_source,omitempty"`
	AttendanceLogID       string          `json:"attendance_log_id,omitempty"`
	AttendanceStatus      string          `json:"attendance_status"`
	RecognitionDistance   float64         `json:"recognition_distance"`
	RecognitionConfidence float64         `json:"recognition_confidence"`
	Box                   recognition.Box `json:"box"`
	Cognitive             any             `json:"cognitive,omitempty"`
	CognitiveError        string          `json:"cognitive_error,omitempty"`
}

type FrameResponse struct {
	SessionID       string       `json:"session_id"`
	ProcessedAt     time.Time    `json:"processed_at"`
	CacheSize       int          `json:"cache_size"`
	FacesDetected   int          `json:"faces_detected"`
	FacesRecognized int          `json:"faces_recognized"`
	UnknownCount    int          `json:"unknown_count"`
	Results         []FaceResult `json:"results"`
}

type CognitiveSnapshotInput struct {
	SessionID          string
	StudentID          string
	AttendanceLogID    string
	Emotion            string
	EmotionConfidence  float64
	Gaze               string
	GazeConfidence     float64
	IsLive             bool
	LivenessConfidence float64
	Box                recognition.Box
	ProcessingMS       int
	ServiceVersion     string
	RawResponseJSON    []byte
}

type SetupRequest struct {
	OrganizationName string `json:"organization_name"`
	AdminName        string `json:"admin_name"`
	Email            string `json:"email"`
	Password         string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token        string       `json:"token"`
	User         User         `json:"user"`
	Organization Organization `json:"organization"`
}

type TeacherCreateRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Status   string `json:"status"`
}

type StudentCreateRequest struct {
	FullName      string `json:"full_name"`
	StudentID     string `json:"student_id"`
	Email         string `json:"email"`
	ClassID       string `json:"class_id"`
	ConsentStatus string `json:"consent_status"`
	Status        string `json:"status"`
}

type ClassCreateRequest struct {
	Name       string `json:"name"`
	Section    string `json:"section"`
	GradeLevel string `json:"grade_level"`
}

type SubjectCreateRequest struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type AssignmentCreateRequest struct {
	TeacherID string `json:"teacher_id"`
	ClassID   string `json:"class_id"`
	SubjectID string `json:"subject_id"`
}

type AttendanceSessionCreateRequest struct {
	Title         string `json:"title"`
	ClassID       string `json:"class_id"`
	SubjectID     string `json:"subject_id"`
	TeacherID     string `json:"teacher_id"`
	StartTime     string `json:"start_time"`
	EndTime       string `json:"end_time"`
	LateThreshold int    `json:"late_threshold_minutes"`
}

type ManualCorrectionRequest struct {
	CorrectedStatus string `json:"corrected_status"`
	Reason          string `json:"reason"`
}

type RecognitionSimulationRequest struct {
	SessionID  string  `json:"session_id"`
	StudentID  string  `json:"student_id"`
	Status     string  `json:"status"`
	Confidence float64 `json:"confidence"`
}

type CameraCreateRequest struct {
	Name         string `json:"name"`
	Room         string `json:"room"`
	RTSPURL      string `json:"rtsp_url"`
	ClassID      string `json:"class_id"`
	HealthStatus string `json:"health_status"`
}

type EnrollmentRequest struct {
	EntityID    string `json:"entity_id"`
	Pose        string `json:"pose"`
	ImageBase64 string `json:"image_base64"`
}

type EnrollmentResponse struct {
	EntityID     string  `json:"entity_id"`
	Pose         string  `json:"pose"`
	ImageSHA256  string  `json:"image_sha256"`
	ModelPath    string  `json:"model_path"`
	ModelMode    string  `json:"model_mode"`
	IsDemo       bool    `json:"is_demo"`
	QualityScore float64 `json:"quality_score"`
	VariantCount int     `json:"variant_count"`
}

type DashboardStats struct {
	TotalStudents    int64               `json:"total_students"`
	TotalTeachers    int64               `json:"total_teachers"`
	Classes          int64               `json:"classes"`
	Subjects         int64               `json:"subjects"`
	TodaySessions    int64               `json:"today_sessions"`
	ReviewQueue      int64               `json:"review_queue"`
	WeeklyOperations []ChartPoint        `json:"weekly_classroom_operations"`
	AttendanceMix    []ChartPoint        `json:"attendance_mix"`
	StudentsByClass  []ChartPoint        `json:"students_by_class"`
	SubjectCoverage  []ChartPoint        `json:"subject_coverage"`
	ActiveSessions   []AttendanceSession `json:"active_sessions"`
	TeacherWorkload  []ChartPoint        `json:"teacher_workload"`
}

type ChartPoint struct {
	Label string `json:"label"`
	Value int64  `json:"value"`
}

type TeacherRow struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	Email                string `json:"email"`
	Status               string `json:"status"`
	FaceEnrollmentStatus string `json:"face_enrollment_status"`
	FacePoseCount        int64  `json:"face_pose_count"`
}

type StudentRow struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	StudentID     string `json:"student_id"`
	ClassName     string `json:"class_name"`
	Section       string `json:"section"`
	ConsentStatus string `json:"consent_status"`
	Status        string `json:"status"`
	FacePoseCount int64  `json:"face_pose_count"`
}

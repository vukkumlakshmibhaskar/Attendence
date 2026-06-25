package services

import (
	"context"
	"errors"
	"time"

	"classroom-attendance/go-backend/internal/models"
	"classroom-attendance/go-backend/internal/repositories"
	"classroom-attendance/go-backend/internal/security"
)

type PlatformService struct {
	repository *repositories.PlatformRepository
}

func NewPlatformService(repository *repositories.PlatformRepository) *PlatformService {
	return &PlatformService{repository: repository}
}

func (s *PlatformService) Dashboard(ctx context.Context, orgID string) (models.DashboardStats, error) {
	return s.repository.Dashboard(ctx, orgID)
}

func (s *PlatformService) ListTeachers(ctx context.Context, orgID string) ([]models.TeacherRow, error) {
	return s.repository.ListTeachers(ctx, orgID)
}

func (s *PlatformService) CreateTeacher(ctx context.Context, orgID string, request models.TeacherCreateRequest) (models.User, error) {
	if request.Name == "" || request.Email == "" || len(request.Password) < 8 {
		return models.User{}, errors.New("name, email, and password with at least 8 characters are required")
	}
	hash, err := security.HashPassword(request.Password)
	if err != nil {
		return models.User{}, err
	}
	status := defaultString(request.Status, "active")
	return s.repository.CreateTeacher(ctx, models.User{
		OrganizationID: orgID,
		Name:           request.Name,
		Email:          request.Email,
		PasswordHash:   hash,
		Role:           "teacher",
		Status:         status,
	})
}

func (s *PlatformService) DeleteTeacher(ctx context.Context, orgID string, teacherID string) error {
	if teacherID == "" {
		return errors.New("teacher id is required")
	}
	return s.repository.DeleteTeacher(ctx, orgID, teacherID)
}

func (s *PlatformService) ListStudents(ctx context.Context, orgID string) ([]models.StudentRow, error) {
	return s.repository.ListStudents(ctx, orgID)
}

func (s *PlatformService) CreateStudent(ctx context.Context, orgID string, request models.StudentCreateRequest) (models.Student, error) {
	if request.FullName == "" || request.StudentID == "" {
		return models.Student{}, errors.New("student name and student id are required")
	}
	email := nullableString(request.Email)
	classID := nullableString(request.ClassID)
	student := models.Student{
		OrganizationID:    orgID,
		StudentID:         request.StudentID,
		ExternalStudentID: request.StudentID,
		FullName:          request.FullName,
		Email:             email,
		ClassID:           classID,
		ConsentStatus:     defaultString(request.ConsentStatus, "pending"),
		Status:            defaultString(request.Status, "active"),
		IsActive:          true,
	}
	return s.repository.CreateStudent(ctx, student)
}

func (s *PlatformService) DeleteStudent(ctx context.Context, orgID string, studentID string) error {
	if studentID == "" {
		return errors.New("student id is required")
	}
	return s.repository.DeleteStudent(ctx, orgID, studentID)
}

func (s *PlatformService) CreateClass(ctx context.Context, orgID string, request models.ClassCreateRequest) (models.AcademicClass, error) {
	if request.Name == "" {
		return models.AcademicClass{}, errors.New("class name is required")
	}
	return s.repository.CreateClass(ctx, models.AcademicClass{
		OrganizationID: orgID,
		Name:           request.Name,
		Section:        request.Section,
		GradeLevel:     request.GradeLevel,
		Status:         "active",
	})
}

func (s *PlatformService) ListClasses(ctx context.Context, orgID string) ([]models.AcademicClass, error) {
	return s.repository.ListClasses(ctx, orgID)
}

func (s *PlatformService) CreateSubject(ctx context.Context, orgID string, request models.SubjectCreateRequest) (models.Subject, error) {
	if request.Name == "" {
		return models.Subject{}, errors.New("subject name is required")
	}
	return s.repository.CreateSubject(ctx, models.Subject{
		OrganizationID: orgID,
		Name:           request.Name,
		Code:           request.Code,
		Status:         "active",
	})
}

func (s *PlatformService) ListSubjects(ctx context.Context, orgID string) ([]models.Subject, error) {
	return s.repository.ListSubjects(ctx, orgID)
}

func (s *PlatformService) CreateAssignment(ctx context.Context, orgID string, request models.AssignmentCreateRequest) (models.TeacherAssignment, error) {
	if request.TeacherID == "" || request.ClassID == "" || request.SubjectID == "" {
		return models.TeacherAssignment{}, errors.New("teacher, class, and subject are required")
	}
	return s.repository.CreateAssignment(ctx, models.TeacherAssignment{
		OrganizationID: orgID,
		TeacherID:      request.TeacherID,
		ClassID:        request.ClassID,
		SubjectID:      request.SubjectID,
	})
}

func (s *PlatformService) ListAssignments(ctx context.Context, orgID string) ([]models.TeacherAssignment, error) {
	return s.repository.ListAssignments(ctx, orgID)
}

func (s *PlatformService) CreateCamera(ctx context.Context, orgID string, request models.CameraCreateRequest) (models.Camera, error) {
	return s.repository.CreateCamera(ctx, models.Camera{
		OrganizationID: orgID,
		Name:           request.Name,
		Room:           request.Room,
		RTSPURL:        request.RTSPURL,
		ClassID:        nullableString(request.ClassID),
		HealthStatus:   defaultString(request.HealthStatus, "unknown"),
	})
}

func (s *PlatformService) ListCameras(ctx context.Context, orgID string) ([]models.Camera, error) {
	return s.repository.ListCameras(ctx, orgID)
}

func (s *PlatformService) CreateAttendanceSession(ctx context.Context, orgID string, request models.AttendanceSessionCreateRequest) (models.AttendanceSession, error) {
	start, err := parseTime(request.StartTime)
	if err != nil {
		return models.AttendanceSession{}, err
	}
	end, err := parseTime(request.EndTime)
	if err != nil {
		return models.AttendanceSession{}, err
	}
	if request.LateThreshold == 0 {
		request.LateThreshold = 10
	}
	return s.repository.CreateAttendanceSession(ctx, models.AttendanceSession{
		OrganizationID: orgID,
		Title:          request.Title,
		ClassID:        request.ClassID,
		SubjectID:      request.SubjectID,
		TeacherID:      request.TeacherID,
		StartTime:      start,
		EndTime:        end,
		LateThreshold:  request.LateThreshold,
		Status:         "scheduled",
	})
}

func (s *PlatformService) ListAttendanceSessions(ctx context.Context, orgID string, userID string, role string) ([]models.AttendanceSession, error) {
	return s.repository.ListAttendanceSessions(ctx, orgID, userID, role == "teacher")
}

func (s *PlatformService) ListAttendanceEvents(ctx context.Context, orgID string, userID string, role string) ([]models.AttendanceEvent, error) {
	return s.repository.ListAttendanceEvents(ctx, orgID, userID, role == "teacher")
}

func (s *PlatformService) SimulateRecognition(ctx context.Context, orgID string, request models.RecognitionSimulationRequest) (models.AttendanceEvent, error) {
	status := defaultString(request.Status, "review")
	if request.Confidence == 0 {
		request.Confidence = 0.82
	}
	return s.repository.UpsertAttendanceEvent(ctx, models.AttendanceEvent{
		OrganizationID: orgID,
		SessionID:      request.SessionID,
		StudentID:      request.StudentID,
		Status:         status,
		Confidence:     request.Confidence,
		Source:         "simulation",
		OccurredAt:     time.Now().UTC(),
	})
}

func (s *PlatformService) CorrectEvent(ctx context.Context, orgID string, actorID string, eventID string, request models.ManualCorrectionRequest) (models.ManualCorrection, error) {
	return s.repository.CreateManualCorrection(ctx, models.ManualCorrection{
		OrganizationID:    orgID,
		AttendanceEventID: eventID,
		CorrectedStatus:   request.CorrectedStatus,
		Reason:            request.Reason,
		CorrectedBy:       actorID,
	})
}

func parseTime(value string) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}
	return time.Parse("2006-01-02T15:04", value)
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

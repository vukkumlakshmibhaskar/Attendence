package repositories

import (
	"context"
	"errors"
	"time"

	"classroom-attendance/go-backend/internal/models"

	"gorm.io/gorm"
)

type PlatformRepository struct {
	db *gorm.DB
}

func NewPlatformRepository(db *gorm.DB) *PlatformRepository {
	return &PlatformRepository{db: db}
}

func (r *PlatformRepository) SetupComplete(ctx context.Context) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("role = ?", "admin").Count(&count).Error
	return count > 0, err
}

func (r *PlatformRepository) CreateOrganizationAndAdmin(ctx context.Context, org models.Organization, admin models.User) (models.Organization, models.User, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&org).Error; err != nil {
			return err
		}
		admin.OrganizationID = org.ID
		return tx.Create(&admin).Error
	})
	return org, admin, err
}

func (r *PlatformRepository) FindUserByEmail(ctx context.Context, email string) (models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	return user, err
}

func (r *PlatformRepository) FindUserByID(ctx context.Context, id string) (models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	return user, err
}

func (r *PlatformRepository) FindOrganizationByID(ctx context.Context, id string) (models.Organization, error) {
	var org models.Organization
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&org).Error
	return org, err
}

func (r *PlatformRepository) CreateTeacher(ctx context.Context, teacher models.User) (models.User, error) {
	err := r.db.WithContext(ctx).Create(&teacher).Error
	return teacher, err
}

func (r *PlatformRepository) DeleteTeacher(ctx context.Context, orgID string, teacherID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		sessionIDs := tx.Model(&models.AttendanceSession{}).
			Select("id").
			Where("organization_id = ? AND teacher_id = ?", orgID, teacherID)
		eventIDs := tx.Model(&models.AttendanceEvent{}).
			Select("id").
			Where("organization_id = ? AND session_id IN (?)", orgID, sessionIDs)

		if err := tx.Where("organization_id = ? AND teacher_id = ?", orgID, teacherID).Delete(&models.TeacherFaceEnrollment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("organization_id = ? AND teacher_id = ?", orgID, teacherID).Delete(&models.TeacherAssignment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("organization_id = ? AND attendance_event_id IN (?)", orgID, eventIDs).Delete(&models.ManualCorrection{}).Error; err != nil {
			return err
		}
		if err := tx.Where("organization_id = ? AND session_id IN (?)", orgID, sessionIDs).Delete(&models.AttendanceEvent{}).Error; err != nil {
			return err
		}
		if err := tx.Where("organization_id = ? AND teacher_id = ?", orgID, teacherID).Delete(&models.AttendanceSession{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.ManualCorrection{}).Where("corrected_by = ?", teacherID).Update("corrected_by", nil).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.AuditLog{}).Where("actor_id = ?", teacherID).Update("actor_id", nil).Error; err != nil {
			return err
		}

		result := tx.Where("id = ? AND organization_id = ? AND role = ?", teacherID, orgID, "teacher").Delete(&models.User{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *PlatformRepository) ListTeachers(ctx context.Context, orgID string) ([]models.TeacherRow, error) {
	var teachers []models.User
	if err := r.db.WithContext(ctx).Where("organization_id = ? AND role = ?", orgID, "teacher").Order("created_at DESC").Find(&teachers).Error; err != nil {
		return nil, err
	}

	rows := make([]models.TeacherRow, 0, len(teachers))
	for _, teacher := range teachers {
		var poseCount int64
		_ = r.db.WithContext(ctx).Model(&models.TeacherFaceEnrollment{}).Where("teacher_id = ?", teacher.ID).Count(&poseCount).Error
		status := "pending"
		if poseCount >= 5 {
			status = "enrolled"
		}
		rows = append(rows, models.TeacherRow{
			ID:                   teacher.ID,
			Name:                 teacher.Name,
			Email:                teacher.Email,
			Status:               teacher.Status,
			FaceEnrollmentStatus: status,
			FacePoseCount:        poseCount,
		})
	}
	return rows, nil
}

func (r *PlatformRepository) CreateStudent(ctx context.Context, student models.Student) (models.Student, error) {
	err := r.db.WithContext(ctx).Create(&student).Error
	return student, err
}

func (r *PlatformRepository) DeleteStudent(ctx context.Context, orgID string, studentID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		eventIDs := tx.Model(&models.AttendanceEvent{}).
			Select("id").
			Where("organization_id = ? AND student_id = ?", orgID, studentID)

		if err := tx.Where("organization_id = ? AND student_id = ?", orgID, studentID).Delete(&models.FaceEnrollment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("student_id = ?", studentID).Delete(&models.FaceEmbedding{}).Error; err != nil {
			return err
		}
		if err := tx.Where("organization_id = ? AND attendance_event_id IN (?)", orgID, eventIDs).Delete(&models.ManualCorrection{}).Error; err != nil {
			return err
		}
		if err := tx.Where("student_id = ?", studentID).Delete(&models.CognitiveSnapshot{}).Error; err != nil {
			return err
		}
		if err := tx.Where("student_id = ?", studentID).Delete(&models.AttendanceLog{}).Error; err != nil {
			return err
		}
		if err := tx.Where("organization_id = ? AND student_id = ?", orgID, studentID).Delete(&models.AttendanceEvent{}).Error; err != nil {
			return err
		}

		result := tx.Where("id = ? AND organization_id = ?", studentID, orgID).Delete(&models.Student{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *PlatformRepository) ListStudents(ctx context.Context, orgID string) ([]models.StudentRow, error) {
	var students []models.Student
	if err := r.db.WithContext(ctx).Where("organization_id = ?", orgID).Order("created_at DESC").Find(&students).Error; err != nil {
		return nil, err
	}

	rows := make([]models.StudentRow, 0, len(students))
	for _, student := range students {
		var poseCount int64
		_ = r.db.WithContext(ctx).Model(&models.FaceEnrollment{}).Where("student_id = ?", student.ID).Count(&poseCount).Error
		className := ""
		section := ""
		if student.ClassID != nil && *student.ClassID != "" {
			var class models.AcademicClass
			if err := r.db.WithContext(ctx).Where("id = ?", *student.ClassID).First(&class).Error; err == nil {
				className = class.Name
				section = class.Section
			}
		}
		rows = append(rows, models.StudentRow{
			ID:            student.ID,
			Name:          student.FullName,
			StudentID:     student.StudentID,
			ClassName:     className,
			Section:       section,
			ConsentStatus: student.ConsentStatus,
			Status:        student.Status,
			FacePoseCount: poseCount,
		})
	}
	return rows, nil
}

func (r *PlatformRepository) CreateClass(ctx context.Context, class models.AcademicClass) (models.AcademicClass, error) {
	err := r.db.WithContext(ctx).Create(&class).Error
	return class, err
}

func (r *PlatformRepository) ListClasses(ctx context.Context, orgID string) ([]models.AcademicClass, error) {
	var classes []models.AcademicClass
	err := r.db.WithContext(ctx).Where("organization_id = ?", orgID).Order("name ASC, section ASC").Find(&classes).Error
	return classes, err
}

func (r *PlatformRepository) CreateSubject(ctx context.Context, subject models.Subject) (models.Subject, error) {
	err := r.db.WithContext(ctx).Create(&subject).Error
	return subject, err
}

func (r *PlatformRepository) ListSubjects(ctx context.Context, orgID string) ([]models.Subject, error) {
	var subjects []models.Subject
	err := r.db.WithContext(ctx).Where("organization_id = ?", orgID).Order("name ASC").Find(&subjects).Error
	return subjects, err
}

func (r *PlatformRepository) CreateAssignment(ctx context.Context, assignment models.TeacherAssignment) (models.TeacherAssignment, error) {
	var existing models.TeacherAssignment
	err := r.db.WithContext(ctx).
		Where("teacher_id = ? AND class_id = ? AND subject_id = ?", assignment.TeacherID, assignment.ClassID, assignment.SubjectID).
		First(&existing).Error
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return assignment, err
	}
	err = r.db.WithContext(ctx).Create(&assignment).Error
	return assignment, err
}

func (r *PlatformRepository) ListAssignments(ctx context.Context, orgID string) ([]models.TeacherAssignment, error) {
	var assignments []models.TeacherAssignment
	err := r.db.WithContext(ctx).Where("organization_id = ?", orgID).Order("created_at DESC").Find(&assignments).Error
	return assignments, err
}

func (r *PlatformRepository) CreateCamera(ctx context.Context, camera models.Camera) (models.Camera, error) {
	err := r.db.WithContext(ctx).Create(&camera).Error
	return camera, err
}

func (r *PlatformRepository) ListCameras(ctx context.Context, orgID string) ([]models.Camera, error) {
	var cameras []models.Camera
	err := r.db.WithContext(ctx).Where("organization_id = ?", orgID).Order("created_at DESC").Find(&cameras).Error
	return cameras, err
}

func (r *PlatformRepository) CreateAttendanceSession(ctx context.Context, session models.AttendanceSession) (models.AttendanceSession, error) {
	err := r.db.WithContext(ctx).Create(&session).Error
	return session, err
}

func (r *PlatformRepository) ListAttendanceSessions(ctx context.Context, orgID string, teacherID string, teacherOnly bool) ([]models.AttendanceSession, error) {
	var sessions []models.AttendanceSession
	query := r.db.WithContext(ctx).Where("organization_id = ?", orgID)
	if teacherOnly {
		query = query.Where("teacher_id = ?", teacherID)
	}
	err := query.Order("start_time DESC").Find(&sessions).Error
	return sessions, err
}

func (r *PlatformRepository) UpsertAttendanceEvent(ctx context.Context, event models.AttendanceEvent) (models.AttendanceEvent, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.AttendanceEvent
		err := tx.Where("session_id = ? AND student_id = ?", event.SessionID, event.StudentID).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&event).Error
		}
		if err != nil {
			return err
		}
		event.ID = existing.ID
		return tx.Model(&existing).Updates(map[string]any{
			"status":      event.Status,
			"confidence":  event.Confidence,
			"source":      event.Source,
			"occurred_at": event.OccurredAt,
			"updated_at":  time.Now().UTC(),
		}).Error
	})
	return event, err
}

func (r *PlatformRepository) ListAttendanceEvents(ctx context.Context, orgID string, teacherID string, teacherOnly bool) ([]models.AttendanceEvent, error) {
	var events []models.AttendanceEvent
	query := r.db.WithContext(ctx).Where("organization_id = ?", orgID)
	if teacherOnly {
		query = query.Where("session_id IN (?)",
			r.db.Model(&models.AttendanceSession{}).
				Select("id").
				Where("organization_id = ? AND teacher_id = ?", orgID, teacherID),
		)
	}
	err := query.Order("occurred_at DESC").Limit(250).Find(&events).Error
	return events, err
}

func (r *PlatformRepository) CreateManualCorrection(ctx context.Context, correction models.ManualCorrection) (models.ManualCorrection, error) {
	err := r.db.WithContext(ctx).Create(&correction).Error
	if err != nil {
		return correction, err
	}
	err = r.db.WithContext(ctx).Model(&models.AttendanceEvent{}).Where("id = ?", correction.AttendanceEventID).Updates(map[string]any{
		"status":     correction.CorrectedStatus,
		"source":     "manual_correction",
		"updated_at": time.Now().UTC(),
	}).Error
	return correction, err
}

func (r *PlatformRepository) SaveStudentEnrollment(ctx context.Context, enrollment models.FaceEnrollment) (models.FaceEnrollment, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("student_id = ? AND pose = ?", enrollment.StudentID, enrollment.Pose).Delete(&models.FaceEnrollment{}).Error; err != nil {
			return err
		}
		return tx.Create(&enrollment).Error
	})
	return enrollment, err
}

func (r *PlatformRepository) DeleteStudentEnrollment(ctx context.Context, orgID string, studentID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.Student{}).Where("id = ? AND organization_id = ?", studentID, orgID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return gorm.ErrRecordNotFound
		}
		if err := tx.Where("organization_id = ? AND student_id = ?", orgID, studentID).Delete(&models.FaceEnrollment{}).Error; err != nil {
			return err
		}
		return tx.Where("student_id = ?", studentID).Delete(&models.FaceEmbedding{}).Error
	})
}

func (r *PlatformRepository) SaveTeacherEnrollment(ctx context.Context, enrollment models.TeacherFaceEnrollment) (models.TeacherFaceEnrollment, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("teacher_id = ? AND pose = ?", enrollment.TeacherID, enrollment.Pose).Delete(&models.TeacherFaceEnrollment{}).Error; err != nil {
			return err
		}
		return tx.Create(&enrollment).Error
	})
	return enrollment, err
}

func (r *PlatformRepository) DeleteTeacherEnrollment(ctx context.Context, orgID string, teacherID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.User{}).Where("id = ? AND organization_id = ? AND role = ?", teacherID, orgID, "teacher").Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Where("organization_id = ? AND teacher_id = ?", orgID, teacherID).Delete(&models.TeacherFaceEnrollment{}).Error
	})
}

func (r *PlatformRepository) Dashboard(ctx context.Context, orgID string) (models.DashboardStats, error) {
	stats := models.DashboardStats{
		WeeklyOperations: []models.ChartPoint{},
		AttendanceMix:    []models.ChartPoint{},
		StudentsByClass:  []models.ChartPoint{},
		SubjectCoverage:  []models.ChartPoint{},
		ActiveSessions:   []models.AttendanceSession{},
		TeacherWorkload:  []models.ChartPoint{},
	}
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.Add(24 * time.Hour)

	if err := r.db.WithContext(ctx).Model(&models.Student{}).Where("organization_id = ?", orgID).Count(&stats.TotalStudents).Error; err != nil {
		return stats, err
	}
	if err := r.db.WithContext(ctx).Model(&models.User{}).Where("organization_id = ? AND role = ?", orgID, "teacher").Count(&stats.TotalTeachers).Error; err != nil {
		return stats, err
	}
	if err := r.db.WithContext(ctx).Model(&models.AcademicClass{}).Where("organization_id = ?", orgID).Count(&stats.Classes).Error; err != nil {
		return stats, err
	}
	if err := r.db.WithContext(ctx).Model(&models.Subject{}).Where("organization_id = ?", orgID).Count(&stats.Subjects).Error; err != nil {
		return stats, err
	}
	_ = r.db.WithContext(ctx).Model(&models.AttendanceSession{}).Where("organization_id = ? AND start_time >= ? AND start_time < ?", orgID, todayStart, todayEnd).Count(&stats.TodaySessions).Error
	_ = r.db.WithContext(ctx).Model(&models.AttendanceEvent{}).Where("organization_id = ? AND status = ?", orgID, "review").Count(&stats.ReviewQueue).Error

	stats.AttendanceMix = r.countBy(ctx, orgID, &models.AttendanceEvent{}, "status")
	stats.StudentsByClass = r.studentsByClass(ctx, orgID)
	stats.SubjectCoverage = r.countBy(ctx, orgID, &models.TeacherAssignment{}, "subject_id")
	stats.TeacherWorkload = r.countBy(ctx, orgID, &models.AttendanceSession{}, "teacher_id")
	stats.WeeklyOperations = r.weeklySessions(ctx, orgID)
	_ = r.db.WithContext(ctx).Where("organization_id = ? AND status IN ?", orgID, []string{"scheduled", "active"}).Order("start_time ASC").Limit(5).Find(&stats.ActiveSessions).Error
	return stats, nil
}

func (r *PlatformRepository) countBy(ctx context.Context, orgID string, model any, field string) []models.ChartPoint {
	var rows []models.ChartPoint
	_ = r.db.WithContext(ctx).Model(model).
		Select(field+" AS label, count(*) AS value").
		Where("organization_id = ?", orgID).
		Group(field).
		Scan(&rows).Error
	if rows == nil {
		return []models.ChartPoint{}
	}
	return rows
}

func (r *PlatformRepository) studentsByClass(ctx context.Context, orgID string) []models.ChartPoint {
	var rows []models.ChartPoint
	_ = r.db.WithContext(ctx).Table("students s").
		Select("COALESCE(c.name || ' ' || c.section, 'Unassigned') AS label, count(*) AS value").
		Joins("LEFT JOIN academic_classes c ON c.id = s.class_id").
		Where("s.organization_id = ?", orgID).
		Group("label").
		Scan(&rows).Error
	if rows == nil {
		return []models.ChartPoint{}
	}
	return rows
}

func (r *PlatformRepository) weeklySessions(ctx context.Context, orgID string) []models.ChartPoint {
	var rows []models.ChartPoint
	_ = r.db.WithContext(ctx).Table("attendance_sessions").
		Select("to_char(date_trunc('day', start_time), 'Dy') AS label, count(*) AS value").
		Where("organization_id = ? AND start_time >= ?", orgID, time.Now().AddDate(0, 0, -7)).
		Group("date_trunc('day', start_time)").
		Order("date_trunc('day', start_time)").
		Scan(&rows).Error
	if rows == nil {
		return []models.ChartPoint{}
	}
	return rows
}

func (r *PlatformRepository) Audit(ctx context.Context, log models.AuditLog) {
	_ = r.db.WithContext(ctx).Create(&log).Error
}

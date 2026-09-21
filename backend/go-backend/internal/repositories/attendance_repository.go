package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"classroom-attendance/go-backend/internal/models"
	"classroom-attendance/go-backend/internal/recognition"

	"gorm.io/gorm"
)

type AttendanceRepository struct {
	db *gorm.DB
}

func NewAttendanceRepository(db *gorm.DB) *AttendanceRepository {
	return &AttendanceRepository{db: db}
}

func (r *AttendanceRepository) LoadFaceVectors(ctx context.Context) ([]recognition.StudentVector, error) {
	vectors, legacyErr := r.loadLegacyFaceVectors(ctx)
	enrollmentVectors, enrollmentErr := r.loadEnrollmentFaceVectors(ctx)
	if enrollmentErr == nil {
		vectors = append(vectors, enrollmentVectors...)
	}
	teacherVectors, teacherErr := r.loadTeacherEnrollmentFaceVectors(ctx)
	if teacherErr == nil {
		vectors = append(vectors, teacherVectors...)
	}
	if len(vectors) > 0 {
		return vectors, nil
	}
	if legacyErr != nil {
		return nil, legacyErr
	}
	if enrollmentErr != nil {
		return nil, enrollmentErr
	}
	return vectors, teacherErr
}

func (r *AttendanceRepository) loadLegacyFaceVectors(ctx context.Context) ([]recognition.StudentVector, error) {
	var rows []struct {
		StudentID         string `gorm:"column:student_id"`
		ExternalStudentID string `gorm:"column:external_student_id"`
		FullName          string `gorm:"column:full_name"`
		EmbeddingID       string `gorm:"column:embedding_id"`
		Embedding         string `gorm:"column:embedding"`
	}

	err := r.db.WithContext(ctx).
		Table("students AS s").
		Select(`
			s.id::text AS student_id,
			s.external_student_id,
			s.full_name,
			e.id::text AS embedding_id,
			e.embedding::text AS embedding`).
		Joins("JOIN student_face_embeddings AS e ON e.student_id = s.id").
		Where("s.is_active = ? AND e.is_active = ?", true, true).
		Order("s.full_name ASC, e.captured_at DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	vectors := make([]recognition.StudentVector, 0, len(rows))
	for _, row := range rows {
		parsed, err := parseVector128(row.Embedding)
		if err != nil {
			return nil, fmt.Errorf("embedding %s for student %s: %w", row.EmbeddingID, row.StudentID, err)
		}
		vectors = append(vectors, recognition.StudentVector{
			PersonType:        "student",
			StudentID:         row.StudentID,
			ExternalStudentID: row.ExternalStudentID,
			FullName:          row.FullName,
			EmbeddingID:       row.EmbeddingID,
			EmbeddingSource:   "student_face_embeddings",
			Vector:            parsed,
		})
	}
	return vectors, nil
}

func (r *AttendanceRepository) loadEnrollmentFaceVectors(ctx context.Context) ([]recognition.StudentVector, error) {
	var rows []struct {
		StudentID         string       `gorm:"column:student_id"`
		ExternalStudentID string       `gorm:"column:external_student_id"`
		FullName          string       `gorm:"column:full_name"`
		EmbeddingID       string       `gorm:"column:embedding_id"`
		EmbeddingJSON     models.JSONB `gorm:"column:embedding_json"`
		IsDemo            bool         `gorm:"column:is_demo"`
	}

	err := r.db.WithContext(ctx).
		Table("students AS s").
		Select(`
			s.id::text AS student_id,
			COALESCE(NULLIF(s.student_id, ''), s.external_student_id) AS external_student_id,
			s.full_name,
			e.id::text AS embedding_id,
			e.embedding_json,
			e.is_demo`).
		Joins("JOIN face_enrollments AS e ON e.student_id = s.id").
		Where("s.is_active = ? AND s.status = ?", true, "active").
		Order("s.full_name ASC, e.created_at DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	vectors := make([]recognition.StudentVector, 0, len(rows))
	for _, row := range rows {
		parsedVectors, err := parseJSONVectors128(row.EmbeddingJSON)
		if err != nil {
			return nil, fmt.Errorf("enrollment embedding %s for student %s: %w", row.EmbeddingID, row.StudentID, err)
		}
		for index, parsed := range parsedVectors {
			source := "face_enrollments"
			if index > 0 {
				source = "face_enrollments:scale_variant"
			}
			vectors = append(vectors, recognition.StudentVector{
				PersonType:        "student",
				StudentID:         row.StudentID,
				ExternalStudentID: row.ExternalStudentID,
				FullName:          row.FullName,
				EmbeddingID:       row.EmbeddingID,
				EmbeddingSource:   source,
				IsDemo:            row.IsDemo,
				Vector:            parsed,
			})
		}
	}
	return vectors, nil
}

func (r *AttendanceRepository) loadTeacherEnrollmentFaceVectors(ctx context.Context) ([]recognition.StudentVector, error) {
	var rows []struct {
		TeacherID     string       `gorm:"column:teacher_id"`
		FullName      string       `gorm:"column:full_name"`
		Email         string       `gorm:"column:email"`
		EmbeddingID   string       `gorm:"column:embedding_id"`
		EmbeddingJSON models.JSONB `gorm:"column:embedding_json"`
		IsDemo        bool         `gorm:"column:is_demo"`
	}

	err := r.db.WithContext(ctx).
		Table("users AS u").
		Select(`
			u.id::text AS teacher_id,
			u.name AS full_name,
			u.email,
			e.id::text AS embedding_id,
			e.embedding_json,
			e.is_demo`).
		Joins("JOIN teacher_face_enrollments AS e ON e.teacher_id = u.id").
		Where("u.role = ? AND u.status = ?", "teacher", "active").
		Order("u.name ASC, e.created_at DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	vectors := make([]recognition.StudentVector, 0, len(rows))
	for _, row := range rows {
		parsedVectors, err := parseJSONVectors128(row.EmbeddingJSON)
		if err != nil {
			return nil, fmt.Errorf("teacher enrollment embedding %s for teacher %s: %w", row.EmbeddingID, row.TeacherID, err)
		}
		for index, parsed := range parsedVectors {
			source := "teacher_face_enrollments"
			if index > 0 {
				source = "teacher_face_enrollments:scale_variant"
			}
			vectors = append(vectors, recognition.StudentVector{
				PersonType:      "teacher",
				TeacherID:       row.TeacherID,
				FullName:        row.FullName,
				Email:           row.Email,
				EmbeddingID:     row.EmbeddingID,
				EmbeddingSource: source,
				IsDemo:          row.IsDemo,
				Vector:          parsed,
			})
		}
	}
	return vectors, nil
}

func (r *AttendanceRepository) UpsertAttendance(ctx context.Context, sessionID string, studentID string, confidence float64, embeddingID string) (string, error) {
	now := time.Now().UTC()
	log := models.AttendanceLog{
		SessionID:             sessionID,
		StudentID:             studentID,
		Status:                "present",
		FirstSeenAt:           now,
		LastSeenAt:            now,
		RecognitionConfidence: &confidence,
		MatchedEmbeddingID:    nullableStringPtr(embeddingID),
		FramesMatched:         1,
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing models.AttendanceLog
		err := tx.Where("session_id = ? AND student_id = ?", sessionID, studentID).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&log).Error
		}
		if err != nil {
			return err
		}

		log.ID = existing.ID
		bestConfidence := confidence
		if existing.RecognitionConfidence != nil && *existing.RecognitionConfidence > bestConfidence {
			bestConfidence = *existing.RecognitionConfidence
		}
		updates := map[string]any{
			"last_seen_at":           now,
			"recognition_confidence": &bestConfidence,
			"frames_matched":         existing.FramesMatched + 1,
			"updated_at":             now,
		}
		if embeddingID != "" {
			updates["matched_embedding_id"] = embeddingID
		}
		return tx.Model(&existing).Updates(updates).Error
	})
	return log.ID, err
}

func (r *AttendanceRepository) InsertCognitiveSnapshot(ctx context.Context, input models.CognitiveSnapshotInput) error {
	now := time.Now().UTC()
	snapshot := models.CognitiveSnapshot{
		SessionID:          input.SessionID,
		StudentID:          nullableStringPtr(input.StudentID),
		AttendanceLogID:    nullableStringPtr(input.AttendanceLogID),
		CapturedAt:         now,
		Emotion:            input.Emotion,
		EmotionConfidence:  input.EmotionConfidence,
		Gaze:               input.Gaze,
		GazeConfidence:     input.GazeConfidence,
		IsLive:             input.IsLive,
		LivenessConfidence: input.LivenessConfidence,
		FaceBoxX:           input.Box.X,
		FaceBoxY:           input.Box.Y,
		FaceBoxWidth:       input.Box.Width,
		FaceBoxHeight:      input.Box.Height,
		ProcessingMS:       input.ProcessingMS,
		ServiceVersion:     input.ServiceVersion,
		RawResponse:        models.JSONB(input.RawResponseJSON),
		CreatedAt:          now,
	}
	return r.db.WithContext(ctx).Create(&snapshot).Error
}

func (r *AttendanceRepository) InsertUnknownFaceEvent(ctx context.Context, sessionID string, reason string, bestDistance float64, box recognition.Box) error {
	now := time.Now().UTC()
	event := models.UnknownFaceEvent{
		SessionID:     nullableStringPtr(sessionID),
		CapturedAt:    now,
		Reason:        reason,
		BestDistance:  nullableDistance(bestDistance),
		FaceBoxX:      box.X,
		FaceBoxY:      box.Y,
		FaceBoxWidth:  box.Width,
		FaceBoxHeight: box.Height,
		CreatedAt:     now,
	}
	return r.db.WithContext(ctx).Create(&event).Error
}

func parseVector128(raw string) ([128]float32, error) {
	var vector [128]float32
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "[")
	cleaned = strings.TrimSuffix(cleaned, "]")
	parts := strings.Split(cleaned, ",")
	if len(parts) != 128 {
		return vector, fmt.Errorf("expected 128 dimensions, got %d", len(parts))
	}
	for i, part := range parts {
		value, err := strconv.ParseFloat(strings.TrimSpace(part), 32)
		if err != nil {
			return vector, err
		}
		vector[i] = float32(value)
	}
	return vector, nil
}

func parseJSONVectors128(raw models.JSONB) ([][128]float32, error) {
	var values []float64
	if err := json.Unmarshal([]byte(raw), &values); err == nil && len(values) > 0 {
		vector, err := floatSliceToVector128(values)
		if err != nil {
			return nil, err
		}
		return [][128]float32{vector}, nil
	}

	var payload struct {
		Embedding []float64   `json:"embedding"`
		Vector    []float64   `json:"vector"`
		Variants  [][]float64 `json:"variants"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}
	base := payload.Embedding
	if len(base) == 0 {
		base = payload.Vector
	}
	vector, err := floatSliceToVector128(base)
	if err != nil {
		return nil, err
	}
	vectors := [][128]float32{vector}
	for _, variant := range payload.Variants {
		parsed, err := floatSliceToVector128(variant)
		if err != nil {
			return nil, err
		}
		vectors = append(vectors, parsed)
	}
	return vectors, nil
}

func floatSliceToVector128(values []float64) ([128]float32, error) {
	var vector [128]float32
	if len(values) != 128 {
		return vector, fmt.Errorf("expected 128 dimensions, got %d", len(values))
	}
	for i, value := range values {
		vector[i] = float32(value)
	}
	return vector, nil
}

func nullableStringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func nullableDistance(value float64) *float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value > 99.999999 {
		return nil
	}
	return &value
}

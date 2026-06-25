package services

import (
	"context"
	"errors"

	"classroom-attendance/go-backend/internal/models"
	"classroom-attendance/go-backend/internal/repositories"
)

var requiredPoses = map[string]bool{
	"front":     true,
	"left":      true,
	"right":     true,
	"look_up":   true,
	"look_down": true,
}

type EnrollmentService struct {
	repository *repositories.PlatformRepository
	embedding  *EmbeddingService
	cache      *FaceCacheService
}

func NewEnrollmentService(repository *repositories.PlatformRepository, embedding *EmbeddingService, cache ...*FaceCacheService) *EnrollmentService {
	var cacheService *FaceCacheService
	if len(cache) > 0 {
		cacheService = cache[0]
	}
	return &EnrollmentService{repository: repository, embedding: embedding, cache: cacheService}
}

func (s *EnrollmentService) EnrollStudent(ctx context.Context, orgID string, request models.EnrollmentRequest) (models.EnrollmentResponse, error) {
	if !requiredPoses[request.Pose] {
		return models.EnrollmentResponse{}, errors.New("pose must be one of front, left, right, look_up, look_down")
	}
	result, err := s.embedding.Generate(request.ImageBase64, request.Pose)
	if err != nil {
		return models.EnrollmentResponse{}, err
	}
	embeddingJSON, err := s.embedding.JSON(result)
	if err != nil {
		return models.EnrollmentResponse{}, err
	}
	_, err = s.repository.SaveStudentEnrollment(ctx, models.FaceEnrollment{
		OrganizationID: orgID,
		StudentID:      request.EntityID,
		Pose:           request.Pose,
		ImageSHA256:    result.ImageSHA256,
		EmbeddingJSON:  models.JSONB(embeddingJSON),
		ModelPath:      result.ModelPath,
		ModelMode:      result.ModelMode,
		IsDemo:         result.IsDemo,
		QualityScore:   result.QualityScore,
	})
	if err != nil {
		return models.EnrollmentResponse{}, err
	}
	if err := s.reloadFaceCache(ctx); err != nil {
		return models.EnrollmentResponse{}, err
	}
	return responseFromEmbedding(request.EntityID, request.Pose, result), nil
}

func (s *EnrollmentService) DeleteStudentEnrollment(ctx context.Context, orgID string, studentID string) error {
	if studentID == "" {
		return errors.New("student id is required")
	}
	if err := s.repository.DeleteStudentEnrollment(ctx, orgID, studentID); err != nil {
		return err
	}
	return s.reloadFaceCache(ctx)
}

func (s *EnrollmentService) EnrollTeacher(ctx context.Context, orgID string, request models.EnrollmentRequest) (models.EnrollmentResponse, error) {
	if !requiredPoses[request.Pose] {
		return models.EnrollmentResponse{}, errors.New("pose must be one of front, left, right, look_up, look_down")
	}
	result, err := s.embedding.Generate(request.ImageBase64, request.Pose)
	if err != nil {
		return models.EnrollmentResponse{}, err
	}
	embeddingJSON, err := s.embedding.JSON(result)
	if err != nil {
		return models.EnrollmentResponse{}, err
	}
	_, err = s.repository.SaveTeacherEnrollment(ctx, models.TeacherFaceEnrollment{
		OrganizationID: orgID,
		TeacherID:      request.EntityID,
		Pose:           request.Pose,
		ImageSHA256:    result.ImageSHA256,
		EmbeddingJSON:  models.JSONB(embeddingJSON),
		ModelPath:      result.ModelPath,
		ModelMode:      result.ModelMode,
		IsDemo:         result.IsDemo,
		QualityScore:   result.QualityScore,
	})
	if err != nil {
		return models.EnrollmentResponse{}, err
	}
	if err := s.reloadFaceCache(ctx); err != nil {
		return models.EnrollmentResponse{}, err
	}
	return responseFromEmbedding(request.EntityID, request.Pose, result), nil
}

func (s *EnrollmentService) DeleteTeacherEnrollment(ctx context.Context, orgID string, teacherID string) error {
	if teacherID == "" {
		return errors.New("teacher id is required")
	}
	if err := s.repository.DeleteTeacherEnrollment(ctx, orgID, teacherID); err != nil {
		return err
	}
	return s.reloadFaceCache(ctx)
}

func (s *EnrollmentService) reloadFaceCache(ctx context.Context) error {
	if s.cache == nil {
		return nil
	}
	_, err := s.cache.Reload(ctx)
	return err
}

func responseFromEmbedding(entityID string, pose string, result EmbeddingResult) models.EnrollmentResponse {
	return models.EnrollmentResponse{
		EntityID:     entityID,
		Pose:         pose,
		ImageSHA256:  result.ImageSHA256,
		ModelPath:    result.ModelPath,
		ModelMode:    result.ModelMode,
		IsDemo:       result.IsDemo,
		QualityScore: result.QualityScore,
		VariantCount: len(result.Variants),
	}
}

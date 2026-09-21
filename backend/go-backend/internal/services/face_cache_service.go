package services

import (
	"context"

	"classroom-attendance/go-backend/internal/recognition"
	"classroom-attendance/go-backend/internal/repositories"
)

type FaceCacheService struct {
	repository *repositories.AttendanceRepository
	cache      *recognition.FaceCache
}

func NewFaceCacheService(repository *repositories.AttendanceRepository, cache *recognition.FaceCache) *FaceCacheService {
	return &FaceCacheService{repository: repository, cache: cache}
}

func (s *FaceCacheService) Size() int {
	return s.cache.Size()
}

func (s *FaceCacheService) Reload(ctx context.Context) (int, error) {
	vectors, err := s.repository.LoadFaceVectors(ctx)
	if err != nil {
		return 0, err
	}
	s.cache.Replace(vectors)
	return len(vectors), nil
}

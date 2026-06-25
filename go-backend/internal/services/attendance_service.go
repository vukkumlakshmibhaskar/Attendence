package services

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"classroom-attendance/go-backend/internal/cognitive"
	"classroom-attendance/go-backend/internal/models"
	"classroom-attendance/go-backend/internal/recognition"
	"classroom-attendance/go-backend/internal/repositories"
)

type AttendanceService struct {
	repository *repositories.AttendanceRepository
	cache      *recognition.FaceCache
	matcher    *recognition.Matcher
	recognizer recognition.Recognizer
	cognitive  *cognitive.Client
}

func NewAttendanceService(
	repository *repositories.AttendanceRepository,
	cache *recognition.FaceCache,
	matcher *recognition.Matcher,
	recognizer recognition.Recognizer,
	cognitiveClient *cognitive.Client,
) *AttendanceService {
	return &AttendanceService{
		repository: repository,
		cache:      cache,
		matcher:    matcher,
		recognizer: recognizer,
		cognitive:  cognitiveClient,
	}
}

func (s *AttendanceService) Process(ctx context.Context, request models.FrameRequest) (models.FrameResponse, error) {
	if request.SessionID == "" {
		return models.FrameResponse{}, errors.New("session_id is required")
	}

	imageBytes, err := decodeBase64Image(request.ImageBase64)
	if err != nil {
		return models.FrameResponse{}, err
	}

	observations, err := s.recognizer.Recognize(ctx, imageBytes)
	if err != nil {
		return models.FrameResponse{}, err
	}

	response := models.FrameResponse{
		SessionID:     request.SessionID,
		ProcessedAt:   time.Now().UTC(),
		CacheSize:     s.cache.Size(),
		FacesDetected: len(observations),
		Results:       make([]models.FaceResult, 0, len(observations)),
	}

	candidates := s.cache.Students()
	for _, observation := range observations {
		match := s.matcher.Match(observation.Descriptor, candidates)
		if !match.Found {
			response.UnknownCount++
			_ = s.repository.InsertUnknownFaceEvent(ctx, request.SessionID, "no_embedding_under_threshold", match.Distance, observation.Box)
			response.Results = append(response.Results, models.FaceResult{
				AttendanceStatus:      "unknown",
				RecognitionDistance:   match.Distance,
				RecognitionConfidence: 0,
				Box:                   observation.Box,
			})
			continue
		}

		faceJPEG, err := cropJPEG(imageBytes, observation.Box)
		if err != nil {
			return response, err
		}

		cognitiveResult, rawJSON, cognitiveErr := s.cognitive.Analyze(ctx, faceJPEG)
		attendanceStatus := "present"
		attendanceLogID := ""
		shouldLogAttendance := true

		if cognitiveErr != nil {
			attendanceStatus = "present_cognitive_failed"
		} else if !cognitiveResult.Liveness.IsLive && cognitiveResult.Liveness.Confidence >= 0.70 {
			attendanceStatus = "blocked_liveness"
			shouldLogAttendance = false
		}

		if shouldLogAttendance {
			matchedEmbeddingID := ""
			if match.Student.EmbeddingSource == "student_face_embeddings" {
				matchedEmbeddingID = match.Student.EmbeddingID
			}
			attendanceLogID, err = s.repository.UpsertAttendance(
				ctx,
				request.SessionID,
				match.Student.StudentID,
				match.Confidence,
				matchedEmbeddingID,
			)
			if err != nil {
				return response, err
			}
			response.FacesRecognized++
		}

		if cognitiveErr == nil {
			if err := s.repository.InsertCognitiveSnapshot(ctx, models.CognitiveSnapshotInput{
				SessionID:          request.SessionID,
				StudentID:          match.Student.StudentID,
				AttendanceLogID:    attendanceLogID,
				Emotion:            cognitiveResult.Emotion.Label,
				EmotionConfidence:  cognitiveResult.Emotion.Confidence,
				Gaze:               cognitiveResult.Gaze.Label,
				GazeConfidence:     cognitiveResult.Gaze.Confidence,
				IsLive:             cognitiveResult.Liveness.IsLive,
				LivenessConfidence: cognitiveResult.Liveness.Confidence,
				Box:                observation.Box,
				ProcessingMS:       cognitiveResult.ProcessingMS,
				ServiceVersion:     cognitiveResult.ServiceVersion,
				RawResponseJSON:    rawJSON,
			}); err != nil {
				return response, err
			}
		}

		result := models.FaceResult{
			PersonType:            "student",
			StudentID:             match.Student.StudentID,
			ExternalStudentID:     match.Student.ExternalStudentID,
			FullName:              match.Student.FullName,
			MatchedFaceID:         match.Student.EmbeddingID,
			MatchedFaceSource:     match.Student.EmbeddingSource,
			AttendanceLogID:       attendanceLogID,
			AttendanceStatus:      attendanceStatus,
			RecognitionDistance:   match.Distance,
			RecognitionConfidence: match.Confidence,
			Box:                   observation.Box,
		}
		if cognitiveErr != nil {
			result.CognitiveError = cognitiveErr.Error()
		} else {
			var cognitivePayload map[string]any
			if err := json.Unmarshal(rawJSON, &cognitivePayload); err == nil {
				result.Cognitive = cognitivePayload
			} else {
				result.Cognitive = cognitiveResult
			}
		}
		response.Results = append(response.Results, result)
	}

	return response, nil
}

func (s *AttendanceService) Identify(ctx context.Context, request models.FrameRequest) (models.FrameResponse, error) {
	// Live recognition is a transient lookup only. Do not write attendance,
	// cognitive snapshots, or unknown-face events from this path.
	imageBytes, err := decodeBase64Image(request.ImageBase64)
	if err != nil {
		return models.FrameResponse{}, err
	}

	observations, err := s.recognizer.Recognize(ctx, imageBytes)
	if err != nil {
		return models.FrameResponse{}, err
	}

	response := models.FrameResponse{
		SessionID:     "",
		ProcessedAt:   time.Now().UTC(),
		CacheSize:     s.cache.Size(),
		FacesDetected: len(observations),
		Results:       make([]models.FaceResult, 0, len(observations)),
	}

	candidates := s.cache.All()
	for _, observation := range observations {
		match := s.matcher.Match(observation.Descriptor, candidates)
		if !match.Found {
			response.UnknownCount++
			response.Results = append(response.Results, models.FaceResult{
				AttendanceStatus:      "unknown",
				RecognitionDistance:   match.Distance,
				RecognitionConfidence: 0,
				Box:                   observation.Box,
			})
			continue
		}

		response.FacesRecognized++
		response.Results = append(response.Results, models.FaceResult{
			PersonType:            match.Student.PersonType,
			StudentID:             match.Student.StudentID,
			TeacherID:             match.Student.TeacherID,
			ExternalStudentID:     match.Student.ExternalStudentID,
			FullName:              match.Student.FullName,
			Email:                 match.Student.Email,
			MatchedFaceID:         match.Student.EmbeddingID,
			MatchedFaceSource:     match.Student.EmbeddingSource,
			AttendanceStatus:      "identified",
			RecognitionDistance:   match.Distance,
			RecognitionConfidence: match.Confidence,
			Box:                   observation.Box,
		})
	}

	return response, nil
}

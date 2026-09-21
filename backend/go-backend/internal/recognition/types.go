package recognition

import "context"

type Box struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type FaceObservation struct {
	Descriptor [128]float32
	Box        Box
	Quality    float32
}

type StudentVector struct {
	PersonType        string
	StudentID         string
	TeacherID         string
	ExternalStudentID string
	FullName          string
	Email             string
	EmbeddingID       string
	EmbeddingSource   string
	IsDemo            bool
	Vector            [128]float32
}

type Recognizer interface {
	Recognize(ctx context.Context, imageBytes []byte) ([]FaceObservation, error)
	Close() error
}

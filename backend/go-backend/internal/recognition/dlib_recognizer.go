//go:build dlib

package recognition

import (
	"context"
	"os"

	face "github.com/Kagami/go-face"
)

type DlibRecognizer struct {
	recognizer *face.Recognizer
}

func NewDlibRecognizer(modelDir string) (Recognizer, error) {
	recognizer, err := face.NewRecognizer(modelDir)
	if err != nil {
		return nil, err
	}
	return &DlibRecognizer{recognizer: recognizer}, nil
}

func (r *DlibRecognizer) Recognize(ctx context.Context, imageBytes []byte) ([]FaceObservation, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	tmp, err := os.CreateTemp("", "attendance-frame-*.jpg")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(imageBytes); err != nil {
		tmp.Close()
		return nil, err
	}
	if err := tmp.Close(); err != nil {
		return nil, err
	}

	faces, err := r.recognizer.RecognizeFile(tmp.Name())
	if err != nil {
		return nil, err
	}

	observations := make([]FaceObservation, 0, len(faces))
	for _, detected := range faces {
		var descriptor [128]float32
		copy(descriptor[:], detected.Descriptor[:])
		rect := detected.Rectangle
		observations = append(observations, FaceObservation{
			Descriptor: descriptor,
			Box: Box{
				X:      rect.Min.X,
				Y:      rect.Min.Y,
				Width:  rect.Dx(),
				Height: rect.Dy(),
			},
			Quality: 1,
		})
	}
	return observations, nil
}

func (r *DlibRecognizer) Close() error {
	r.recognizer.Close()
	return nil
}

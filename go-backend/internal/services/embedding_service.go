package services

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"

	"classroom-attendance/go-backend/internal/recognition"
)

type EmbeddingResult struct {
	Vector       []float64
	Variants     [][]float64
	ImageSHA256  string
	ModelPath    string
	ModelMode    string
	Descriptor   string
	IsDemo       bool
	QualityScore float64
}

type EmbeddingService struct {
	modelPath string
}

func NewEmbeddingService(modelPath string) *EmbeddingService {
	return &EmbeddingService{modelPath: modelPath}
}

func (s *EmbeddingService) Generate(imageBase64 string, pose string) (EmbeddingResult, error) {
	imageBytes, err := decodeEnrollmentImage(imageBase64)
	if err != nil {
		return EmbeddingResult{}, err
	}
	hash := sha256.Sum256(imageBytes)
	mode := "onnx_path_configured"
	isDemo := false
	if _, err := os.Stat(s.modelPath); err != nil {
		mode = "demo_embeddings"
		isDemo = true
	}

	descriptors, _, err := recognition.DescriptorVariantsFromImageBytes(imageBytes)
	if err != nil {
		return EmbeddingResult{}, err
	}
	if len(descriptors) == 0 {
		return EmbeddingResult{}, errors.New("no face descriptors were generated")
	}
	vector := make([]float64, 128)
	for i := 0; i < 128; i++ {
		vector[i] = float64(descriptors[0][i])
	}

	variants := make([][]float64, 0, len(descriptors)-1)
	for _, descriptor := range descriptors[1:] {
		variant := make([]float64, 128)
		for i := 0; i < 128; i++ {
			variant[i] = float64(descriptor[i])
		}
		variants = append(variants, variant)
	}

	return EmbeddingResult{
		Vector:       vector,
		Variants:     variants,
		ImageSHA256:  toHex(hash[:]),
		ModelPath:    s.modelPath,
		ModelMode:    mode,
		Descriptor:   recognition.ActiveDescriptorName,
		IsDemo:       isDemo,
		QualityScore: 0.86,
	}, nil
}

func (s *EmbeddingService) JSON(result EmbeddingResult) ([]byte, error) {
	return json.Marshal(struct {
		Descriptor string      `json:"descriptor"`
		Embedding  []float64   `json:"embedding"`
		Variants   [][]float64 `json:"variants,omitempty"`
	}{
		Descriptor: result.Descriptor,
		Embedding:  result.Vector,
		Variants:   result.Variants,
	})
}

func decodeEnrollmentImage(raw string) ([]byte, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, errors.New("image_base64 is required")
	}
	if comma := strings.Index(value, ","); comma >= 0 {
		value = value[comma+1:]
	}
	return base64.StdEncoding.DecodeString(value)
}

func toHex(bytes []byte) string {
	const alphabet = "0123456789abcdef"
	out := make([]byte, len(bytes)*2)
	for i, b := range bytes {
		out[i*2] = alphabet[b>>4]
		out[i*2+1] = alphabet[b&0x0f]
	}
	return string(out)
}

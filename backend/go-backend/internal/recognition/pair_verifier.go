package recognition

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"strings"
)

type PairVerifierModel struct {
	Version         int            `json:"version"`
	ModelType       string         `json:"model_type"`
	Descriptor      string         `json:"descriptor"`
	FeatureCount    int            `json:"feature_count"`
	Weights         []float64      `json:"weights"`
	Intercept       float64        `json:"intercept"`
	Threshold       float64        `json:"threshold"`
	TargetPrecision float64        `json:"target_precision"`
	Training        map[string]any `json:"training"`
}

func LoadPairVerifierModel(path string) (*PairVerifierModel, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	bytes, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var model PairVerifierModel
	if err := json.Unmarshal(bytes, &model); err != nil {
		return nil, err
	}
	if model.FeatureCount != 130 || len(model.Weights) != 130 {
		return nil, errors.New("face verifier model must contain 130 linear weights")
	}
	if model.Threshold <= 0 || model.Threshold >= 1 {
		return nil, errors.New("face verifier threshold must be between 0 and 1")
	}
	if model.Descriptor != "" && model.Descriptor != ActiveDescriptorName {
		return nil, errors.New("face verifier descriptor does not match active recognizer descriptor")
	}
	return &model, nil
}

func (m *PairVerifierModel) Score(first [128]float32, second [128]float32) float64 {
	if m == nil || len(m.Weights) != 130 {
		return 0
	}

	logit := m.Intercept
	var euclideanSquared float64
	var dot float64
	for i := 0; i < 128; i++ {
		a := float64(first[i])
		b := float64(second[i])
		diff := math.Abs(a - b)
		logit += m.Weights[i] * diff
		euclideanSquared += diff * diff
		dot += a * b
	}
	logit += m.Weights[128] * math.Sqrt(euclideanSquared)
	logit += m.Weights[129] * (1.0 - dot)
	return sigmoid(logit)
}

func sigmoid(value float64) float64 {
	if value >= 0 {
		z := math.Exp(-value)
		return 1 / (1 + z)
	}
	z := math.Exp(value)
	return z / (1 + z)
}

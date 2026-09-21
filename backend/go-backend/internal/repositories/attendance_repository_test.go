package repositories

import (
	"encoding/json"
	"testing"

	"classroom-attendance/go-backend/internal/models"
)

func TestParseJSONVectors128SupportsLegacyArray(t *testing.T) {
	raw, err := json.Marshal(sequence128(0))
	if err != nil {
		t.Fatalf("marshal legacy vector: %v", err)
	}

	vectors, err := parseJSONVectors128(models.JSONB(raw))
	if err != nil {
		t.Fatalf("parseJSONVectors128 failed: %v", err)
	}
	if len(vectors) != 1 {
		t.Fatalf("expected one legacy vector, got %d", len(vectors))
	}
}

func TestParseJSONVectors128SupportsEmbeddingVariants(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"descriptor": "cpu_hybrid_grid_v2_8x8_intensity_gradient",
		"embedding":  sequence128(0),
		"variants": []any{
			sequence128(1),
			sequence128(2),
		},
	})
	if err != nil {
		t.Fatalf("marshal variant payload: %v", err)
	}

	vectors, err := parseJSONVectors128(models.JSONB(raw))
	if err != nil {
		t.Fatalf("parseJSONVectors128 failed: %v", err)
	}
	if len(vectors) != 3 {
		t.Fatalf("expected primary vector plus two variants, got %d", len(vectors))
	}
}

func sequence128(offset float64) []float64 {
	values := make([]float64, 128)
	for i := range values {
		values[i] = offset + float64(i)/128
	}
	return values
}

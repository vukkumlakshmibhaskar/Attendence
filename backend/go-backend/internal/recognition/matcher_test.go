package recognition

import "testing"

func TestMatcherWithVerifierAcceptsCloseDistanceEvenWhenVerifierIsStrict(t *testing.T) {
	var query [128]float32
	var enrolled [128]float32
	query[0] = 0.10
	enrolled[0] = 0.12

	verifier := strictLowConfidenceVerifier()
	matcher := NewMatcher(0.55, verifier)

	result := matcher.Match(query, []StudentVector{
		{StudentID: "student-1", FullName: "Student One", Vector: enrolled},
	})

	if !result.Found {
		t.Fatalf("expected close enrolled face to match")
	}
	if result.Student.StudentID != "student-1" {
		t.Fatalf("expected student-1, got %q", result.Student.StudentID)
	}
}

func TestMatcherWithVerifierRejectsFarDistanceWhenVerifierIsStrict(t *testing.T) {
	var query [128]float32
	var enrolled [128]float32
	query[0] = 0
	enrolled[0] = 1

	verifier := strictLowConfidenceVerifier()
	matcher := NewMatcher(0.55, verifier)

	result := matcher.Match(query, []StudentVector{
		{StudentID: "student-1", FullName: "Student One", Vector: enrolled},
	})

	if result.Found {
		t.Fatalf("expected far face to be rejected")
	}
}

func TestMatcherWithVerifierReturnsNearestCandidateOnUnknown(t *testing.T) {
	var query [128]float32
	var near [128]float32
	var far [128]float32
	near[0] = 0.70
	far[0] = 1.30

	verifier := strictLowConfidenceVerifier()
	matcher := NewMatcher(0.55, verifier)

	result := matcher.Match(query, []StudentVector{
		{StudentID: "far", FullName: "Far", Vector: far},
		{StudentID: "near", FullName: "Near", Vector: near},
	})

	if result.Found {
		t.Fatalf("expected no accepted match")
	}
	if result.Student.StudentID != "near" {
		t.Fatalf("expected nearest candidate for diagnostics, got %q", result.Student.StudentID)
	}
}

func strictLowConfidenceVerifier() *PairVerifierModel {
	return &PairVerifierModel{
		Weights:   make([]float64, 130),
		Intercept: -10,
		Threshold: 0.95,
	}
}

package recognition

import "math"

type MatchResult struct {
	Found      bool
	Student    StudentVector
	Distance   float64
	Confidence float64
}

type Matcher struct {
	threshold float64
	verifier  *PairVerifierModel
}

func NewMatcher(threshold float64, verifier ...*PairVerifierModel) *Matcher {
	var selected *PairVerifierModel
	if len(verifier) > 0 {
		selected = verifier[0]
	}
	return &Matcher{threshold: threshold, verifier: selected}
}

func (m *Matcher) Match(query [128]float32, candidates []StudentVector) MatchResult {
	if m.verifier != nil {
		return m.matchWithVerifier(query, candidates)
	}

	best := MatchResult{Distance: math.MaxFloat64}
	for _, candidate := range candidates {
		distance := euclidean(query, candidate.Vector)
		if distance < best.Distance {
			best.Distance = distance
			best.Student = candidate
		}
	}

	if len(candidates) == 0 || best.Distance > m.threshold {
		return best
	}

	best.Found = true
	best.Confidence = clamp01(1.0 - (best.Distance / m.threshold))
	return best
}

func (m *Matcher) matchWithVerifier(query [128]float32, candidates []StudentVector) MatchResult {
	if len(candidates) == 0 {
		return MatchResult{Distance: math.MaxFloat64}
	}

	nearest := MatchResult{Distance: math.MaxFloat64}
	strongest := MatchResult{Distance: math.MaxFloat64}
	for _, candidate := range candidates {
		distance := euclidean(query, candidate.Vector)
		confidence := m.verifier.Score(query, candidate.Vector)
		result := MatchResult{
			Student:    candidate,
			Distance:   distance,
			Confidence: confidence,
		}
		if distance < nearest.Distance {
			nearest = result
		}
		if confidence > strongest.Confidence || (confidence == strongest.Confidence && distance < strongest.Distance) {
			strongest = result
		}
	}

	if nearest.Distance <= m.threshold {
		nearest.Found = true
		distanceConfidence := clamp01(1.0 - (nearest.Distance / m.threshold))
		if distanceConfidence > nearest.Confidence {
			nearest.Confidence = distanceConfidence
		}
		return nearest
	}

	if strongest.Confidence >= m.verifier.Threshold {
		strongest.Found = true
		return strongest
	}

	return nearest
}

func euclidean(a [128]float32, b [128]float32) float64 {
	var sum float64
	for i := 0; i < 128; i++ {
		delta := float64(a[i] - b[i])
		sum += delta * delta
	}
	return math.Sqrt(sum)
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

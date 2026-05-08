package rl

import (
	"math"
	"sort"
	"strconv"
)

// Metrics is the summary scoresheet produced by Evaluator.Run.
type Metrics struct {
	SampleCount       int
	Accuracy          float64 // [0,1]
	PassAtK           float64 // ≥ Accuracy when K > 1
	AverageLength     float64 // chars
	AverageSteps      float64
	FormatCorrectness float64 // [0,1] fraction passing validator
	AverageReward     float64
}

// ComputeAccuracy: fraction of trajectories whose extracted answer
// matches GroundTruth (using the same normalization as AccuracyReward).
func ComputeAccuracy(trajs []Trajectory, samples []Sample) float64 {
	if len(trajs) == 0 || len(samples) == 0 {
		return 0
	}
	gt := indexBy(samples, func(s Sample) string { return s.ID })
	matches := 0
	for _, t := range trajs {
		s, ok := gt[t.SampleID]
		if !ok {
			continue
		}
		if normalizeAnswer(extractFinalAnswer(t.Answer)) == normalizeAnswer(s.GroundTruth) {
			matches++
		}
	}
	return float64(matches) / float64(len(trajs))
}

// ComputePassAtK: per Sample, true if any of K trajectories matches.
// trajsByID is grouping of K runs per sample. K is the requested K
// (used only to validate input — actual K = len of group).
func ComputePassAtK(trajsByID map[string][]Trajectory, samples []Sample, k int) float64 {
	if len(samples) == 0 {
		return 0
	}
	gt := indexBy(samples, func(s Sample) string { return s.ID })
	passed := 0
	for id, group := range trajsByID {
		s, ok := gt[id]
		if !ok {
			continue
		}
		// K is at most len(group); cap silently.
		limit := k
		if limit > len(group) || limit <= 0 {
			limit = len(group)
		}
		for i := 0; i < limit; i++ {
			if normalizeAnswer(extractFinalAnswer(group[i].Answer)) == normalizeAnswer(s.GroundTruth) {
				passed++
				break
			}
		}
	}
	return float64(passed) / float64(len(samples))
}

// ComputeFormatCorrectness: fraction of trajectories whose answer
// passes the supplied validator.
func ComputeFormatCorrectness(trajs []Trajectory, validator func(string) bool) float64 {
	if len(trajs) == 0 || validator == nil {
		return 0
	}
	ok := 0
	for _, t := range trajs {
		if validator(t.Answer) {
			ok++
		}
	}
	return float64(ok) / float64(len(trajs))
}

// ComputeNumericError returns mean / p50 / p95 of |predicted -
// ground_truth| for trajectories whose answers parse as floats.
// Trajectories with non-numeric answers are skipped (not penalized).
func ComputeNumericError(trajs []Trajectory, samples []Sample) (mean, p50, p95 float64) {
	if len(trajs) == 0 || len(samples) == 0 {
		return 0, 0, 0
	}
	gt := indexBy(samples, func(s Sample) string { return s.ID })
	errs := make([]float64, 0, len(trajs))
	var sum float64
	for _, t := range trajs {
		s, ok := gt[t.SampleID]
		if !ok {
			continue
		}
		predStr := normalizeAnswer(extractFinalAnswer(t.Answer))
		gtStr := normalizeAnswer(s.GroundTruth)
		pred, err1 := strconv.ParseFloat(predStr, 64)
		actual, err2 := strconv.ParseFloat(gtStr, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		e := math.Abs(pred - actual)
		errs = append(errs, e)
		sum += e
	}
	if len(errs) == 0 {
		return 0, 0, 0
	}
	mean = sum / float64(len(errs))
	sort.Float64s(errs)
	p50 = errs[len(errs)/2]
	idx95 := int(float64(len(errs)) * 0.95)
	if idx95 >= len(errs) {
		idx95 = len(errs) - 1
	}
	p95 = errs[idx95]
	return mean, p50, p95
}

// indexBy builds a map from element keys for O(1) lookup.
func indexBy[T any, K comparable](xs []T, key func(T) K) map[K]T {
	out := make(map[K]T, len(xs))
	for _, x := range xs {
		out[key(x)] = x
	}
	return out
}

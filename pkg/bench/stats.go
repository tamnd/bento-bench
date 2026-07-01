package bench

import (
	"math"
	"slices"
	"time"
)

// Stats summarizes a set of timed runs. Durations are kept in wall-clock time as
// measured from process start to exit, so they include startup, which is exactly
// what a user feels when they invoke a runtime.
type Stats struct {
	Runs   int           `json:"runs"`
	Min    time.Duration `json:"min"`
	Mean   time.Duration `json:"mean"`
	Median time.Duration `json:"median"`
	P90    time.Duration `json:"p90"`
	Max    time.Duration `json:"max"`
	Stddev time.Duration `json:"stddev"`
}

// summarize reduces raw samples to Stats. It expects at least one sample; an
// empty slice yields the zero value, which the report renders as "n/a".
func summarize(samples []time.Duration) Stats {
	n := len(samples)
	if n == 0 {
		return Stats{}
	}
	sorted := make([]time.Duration, n)
	copy(sorted, samples)
	slices.Sort(sorted)

	var total time.Duration
	for _, d := range sorted {
		total += d
	}
	mean := total / time.Duration(n)

	var sumSq float64
	for _, d := range sorted {
		diff := float64(d - mean)
		sumSq += diff * diff
	}
	stddev := time.Duration(math.Sqrt(sumSq / float64(n)))

	return Stats{
		Runs:   n,
		Min:    sorted[0],
		Mean:   mean,
		Median: percentile(sorted, 0.50),
		P90:    percentile(sorted, 0.90),
		Max:    sorted[n-1],
		Stddev: stddev,
	}
}

// percentile returns the value at the given quantile of an already sorted slice
// using nearest-rank, which is stable and needs no interpolation guesswork.
func percentile(sorted []time.Duration, q float64) time.Duration {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	rank := max(int(math.Ceil(q*float64(n)))-1, 0)
	if rank >= n {
		rank = n - 1
	}
	return sorted[rank]
}

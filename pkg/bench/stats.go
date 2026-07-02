package bench

import (
	"math"
	"slices"
	"time"
)

// sample is one timed run: its wall-clock duration, the peak resident memory the
// process held in bytes, and the in-process compute time the workload reported.
// RSS is zero when the platform could not report it, and compute is zero when the
// workload printed no compute marker; summarize treats each zero as absent rather
// than a real zero.
type sample struct {
	dur     time.Duration
	rss     int64
	compute time.Duration
}

// Stats summarizes a set of timed runs. Durations are kept in wall-clock time as
// measured from process start to exit, so they include startup, which is exactly
// what a user feels when they invoke a runtime. Memory is the peak resident set
// size the run reached, the high-water mark of physical memory it held.
type Stats struct {
	Runs   int           `json:"runs"`
	Min    time.Duration `json:"min"`
	Mean   time.Duration `json:"mean"`
	Median time.Duration `json:"median"`
	P90    time.Duration `json:"p90"`
	Max    time.Duration `json:"max"`
	Stddev time.Duration `json:"stddev"`

	// Peak RSS in bytes across the timed runs. Zero means no run reported memory,
	// so the report shows it as "n/a" rather than a misleading 0 B.
	MinRSS    int64 `json:"minRssBytes"`
	MedianRSS int64 `json:"medianRssBytes"`
	MaxRSS    int64 `json:"maxRssBytes"`

	// In-process compute time across the timed runs, the interval each workload
	// measured with performance.now() around its hot region and printed to stderr.
	// It excludes process startup and teardown, so it isolates the runtime's
	// compute from the cold-start floor the wall-clock durations include. Zero
	// means no run reported a compute number (a workload with no timed region, such
	// as the startup program), so the report shows it as "n/a".
	MinCompute    time.Duration `json:"minComputeNs"`
	MedianCompute time.Duration `json:"medianComputeNs"`
	MaxCompute    time.Duration `json:"maxComputeNs"`
}

// summarize reduces raw samples to Stats. It expects at least one sample; an
// empty slice yields the zero value, which the report renders as "n/a". Duration
// and memory are summarized independently: a run that could not report memory
// still counts toward the timing, and is simply dropped from the memory summary.
func summarize(samples []sample) Stats {
	n := len(samples)
	if n == 0 {
		return Stats{}
	}
	durs := make([]time.Duration, 0, n)
	for _, s := range samples {
		durs = append(durs, s.dur)
	}
	slices.Sort(durs)

	var total time.Duration
	for _, d := range durs {
		total += d
	}
	mean := total / time.Duration(n)

	var sumSq float64
	for _, d := range durs {
		diff := float64(d - mean)
		sumSq += diff * diff
	}
	stddev := time.Duration(math.Sqrt(sumSq / float64(n)))

	minRSS, medRSS, maxRSS := summarizeRSS(samples)
	minC, medC, maxC := summarizeCompute(samples)

	return Stats{
		Runs:          n,
		Min:           durs[0],
		Mean:          mean,
		Median:        percentile(durs, 0.50),
		P90:           percentile(durs, 0.90),
		Max:           durs[n-1],
		Stddev:        stddev,
		MinRSS:        minRSS,
		MedianRSS:     medRSS,
		MaxRSS:        maxRSS,
		MinCompute:    minC,
		MedianCompute: medC,
		MaxCompute:    maxC,
	}
}

// summarizeCompute reduces the in-process compute side of the samples, over only
// the runs that reported a nonzero compute time. When no run reported one the
// three values are zero, which the report reads as "n/a", the same way the memory
// summary treats a run with no rusage.
func summarizeCompute(samples []sample) (minC, medC, maxC time.Duration) {
	vals := make([]time.Duration, 0, len(samples))
	for _, s := range samples {
		if s.compute > 0 {
			vals = append(vals, s.compute)
		}
	}
	if len(vals) == 0 {
		return 0, 0, 0
	}
	slices.Sort(vals)
	return vals[0], percentile(vals, 0.50), vals[len(vals)-1]
}

// summarizeRSS reduces the peak-memory side of the samples, over only the runs
// that reported a nonzero RSS. When no run reported memory the three values are
// zero, which the report reads as "n/a".
func summarizeRSS(samples []sample) (minRSS, medRSS, maxRSS int64) {
	vals := make([]int64, 0, len(samples))
	for _, s := range samples {
		if s.rss > 0 {
			vals = append(vals, s.rss)
		}
	}
	if len(vals) == 0 {
		return 0, 0, 0
	}
	slices.Sort(vals)
	return vals[0], medianInt64(vals), vals[len(vals)-1]
}

// medianInt64 returns the nearest-rank median of an already sorted slice, the
// same rule percentile uses for durations so memory and timing read alike.
func medianInt64(sorted []int64) int64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	rank := max(int(math.Ceil(0.50*float64(n)))-1, 0)
	if rank >= n {
		rank = n - 1
	}
	return sorted[rank]
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

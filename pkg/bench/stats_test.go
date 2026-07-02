package bench

import (
	"testing"
	"time"
)

func TestSummarizeEmpty(t *testing.T) {
	if got := summarize(nil); got.Runs != 0 {
		t.Errorf("empty summarize runs = %d, want 0", got.Runs)
	}
}

func TestSummarize(t *testing.T) {
	samples := []sample{
		{dur: 10 * time.Millisecond, rss: 100},
		{dur: 20 * time.Millisecond, rss: 200},
		{dur: 30 * time.Millisecond, rss: 300},
		{dur: 40 * time.Millisecond, rss: 400},
		{dur: 50 * time.Millisecond, rss: 500},
	}
	s := summarize(samples)
	if s.Runs != 5 {
		t.Fatalf("runs = %d, want 5", s.Runs)
	}
	if s.Min != 10*time.Millisecond {
		t.Errorf("min = %v, want 10ms", s.Min)
	}
	if s.Max != 50*time.Millisecond {
		t.Errorf("max = %v, want 50ms", s.Max)
	}
	if s.Mean != 30*time.Millisecond {
		t.Errorf("mean = %v, want 30ms", s.Mean)
	}
	if s.Median != 30*time.Millisecond {
		t.Errorf("median = %v, want 30ms", s.Median)
	}
	if s.MinRSS != 100 || s.MedianRSS != 300 || s.MaxRSS != 500 {
		t.Errorf("rss min/median/max = %d/%d/%d, want 100/300/500", s.MinRSS, s.MedianRSS, s.MaxRSS)
	}
}

// TestSummarizeMissingRSS pins that runs with no memory reading (rss zero, as on
// a platform without rusage) are dropped from the memory summary, leaving it
// zero rather than pulling the median down with false zeros.
func TestSummarizeMissingRSS(t *testing.T) {
	samples := []sample{
		{dur: 10 * time.Millisecond, rss: 0},
		{dur: 20 * time.Millisecond, rss: 0},
	}
	s := summarize(samples)
	if s.MedianRSS != 0 {
		t.Errorf("medianRSS = %d, want 0 when no run reported memory", s.MedianRSS)
	}
	if s.Runs != 2 {
		t.Errorf("runs = %d, want 2 (timing still counts)", s.Runs)
	}
}

func TestPercentile(t *testing.T) {
	sorted := []time.Duration{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	if got := percentile(sorted, 0.90); got != 9 {
		t.Errorf("p90 = %v, want 9", got)
	}
	if got := percentile(sorted, 0.50); got != 5 {
		t.Errorf("p50 = %v, want 5", got)
	}
	if got := percentile(nil, 0.5); got != 0 {
		t.Errorf("empty percentile = %v, want 0", got)
	}
}

func TestSummarizeCompute(t *testing.T) {
	samples := []sample{
		{dur: 10 * time.Millisecond, compute: 3 * time.Millisecond},
		{dur: 20 * time.Millisecond, compute: 0}, // no marker this run
		{dur: 30 * time.Millisecond, compute: 5 * time.Millisecond},
		{dur: 40 * time.Millisecond, compute: 7 * time.Millisecond},
	}
	s := summarize(samples)
	if s.MinCompute != 3*time.Millisecond {
		t.Errorf("minCompute = %v, want 3ms", s.MinCompute)
	}
	if s.MedianCompute != 5*time.Millisecond {
		t.Errorf("medianCompute = %v, want 5ms", s.MedianCompute)
	}
	if s.MaxCompute != 7*time.Millisecond {
		t.Errorf("maxCompute = %v, want 7ms", s.MaxCompute)
	}
}

func TestSummarizeComputeAbsent(t *testing.T) {
	samples := []sample{
		{dur: 10 * time.Millisecond},
		{dur: 20 * time.Millisecond},
	}
	if s := summarize(samples); s.MedianCompute != 0 {
		t.Errorf("medianCompute = %v, want 0 when no run reported compute", s.MedianCompute)
	}
}

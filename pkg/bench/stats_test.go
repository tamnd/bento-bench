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
	samples := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
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

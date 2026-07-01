package bench

import (
	"strings"
	"testing"
	"time"
)

func sampleResults() []WorkloadResult {
	return []WorkloadResult{
		{
			Workload: "fib",
			Category: "compute",
			Measurements: []Measurement{
				{Runtime: "node", Stats: Stats{Median: 40 * time.Millisecond}},
				{Runtime: "bun", Stats: Stats{Median: 30 * time.Millisecond}},
				{Runtime: "bento", Stats: Stats{Median: 60 * time.Millisecond}},
			},
		},
		{
			Workload: "hello",
			Category: "startup",
			Measurements: []Measurement{
				{Runtime: "node", Stats: Stats{Median: 45 * time.Millisecond}},
				{Runtime: "bun", Failed: true, Note: "timeout"},
				{Runtime: "bento", Stats: Stats{Median: 12 * time.Millisecond}},
			},
		},
	}
}

func TestFastest(t *testing.T) {
	res := sampleResults()
	if got := fastest(res[0]); got != "bun" {
		t.Errorf("fastest(compute/fib) = %q, want bun", got)
	}
	if got := fastest(res[1]); got != "bento" {
		t.Errorf("fastest(startup/hello) = %q, want bento", got)
	}
}

func TestFmtDur(t *testing.T) {
	cases := map[time.Duration]string{
		1500 * time.Millisecond: "1.50s",
		12 * time.Millisecond:   "12.0ms",
		800 * time.Microsecond:  "800us",
		42 * time.Nanosecond:    "42ns",
	}
	for in, want := range cases {
		if got := fmtDur(in); got != want {
			t.Errorf("fmtDur(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestWriteReport(t *testing.T) {
	var b strings.Builder
	if err := WriteReport(&b, sampleResults(), Options{Runs: 10, Warmup: 2}); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if !strings.Contains(out, "compute/fib") {
		t.Error("report missing workload row")
	}
	if !strings.Contains(out, "*30.0ms") {
		t.Error("report should star the fastest cell")
	}
	if !strings.Contains(out, "fail") {
		t.Error("report should show a failed run")
	}
}

func TestWriteMarkdown(t *testing.T) {
	var b strings.Builder
	if err := WriteMarkdown(&b, sampleResults(), Options{Runs: 10, Warmup: 2}); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if !strings.Contains(out, "| workload |") {
		t.Error("markdown missing table header")
	}
	if !strings.Contains(out, "bento versus the fastest other runtime") {
		t.Error("markdown missing speedup section")
	}
}

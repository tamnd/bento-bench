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
				{Runtime: "node", Stats: Stats{Median: 40 * time.Millisecond, MedianRSS: 60 * 1024 * 1024}},
				{Runtime: "bun", Stats: Stats{Median: 30 * time.Millisecond, MedianRSS: 55 * 1024 * 1024}},
				{Runtime: "bento", Stats: Stats{Median: 60 * time.Millisecond, MedianRSS: 12 * 1024 * 1024}},
			},
		},
		{
			Workload: "hello",
			Category: "startup",
			Measurements: []Measurement{
				{Runtime: "node", Stats: Stats{Median: 45 * time.Millisecond, MedianRSS: 40 * 1024 * 1024}},
				{Runtime: "bun", Failed: true, Note: "timeout"},
				{Runtime: "bento", Stats: Stats{Median: 12 * time.Millisecond, MedianRSS: 8 * 1024 * 1024}},
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

func TestWriteReportMemory(t *testing.T) {
	var b strings.Builder
	if err := WriteReport(&b, sampleResults(), Options{Runs: 10, Warmup: 2}); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if !strings.Contains(out, "Median peak memory") {
		t.Error("report missing memory section")
	}
	if !strings.Contains(out, "*12.0MB") {
		t.Error("report should star the leanest memory cell (bento on fib)")
	}
	if !strings.Contains(out, "Startup cost") {
		t.Error("report missing startup section")
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
	if !strings.Contains(out, "## Peak memory") {
		t.Error("markdown missing memory table")
	}
	if !strings.Contains(out, "**30.0ms**") {
		t.Error("markdown should bold the fastest wall-clock cell")
	}
	if !strings.Contains(out, "**12.0MB**") {
		t.Error("markdown should bold the leanest memory cell")
	}
	if !strings.Contains(out, "## Startup cost") {
		t.Error("markdown missing startup section")
	}
	if !strings.Contains(out, "bento versus the fastest other runtime") {
		t.Error("markdown missing speedup section")
	}
	if strings.Contains(out, "## Compile step") {
		t.Error("single-phase results should not render a compile section")
	}
}

// TestWriteMarkdownCompile pins that a two-phase (AOT) measurement renders the
// compile-step table, so the cost of compiling to a binary is reported next to
// the binary's speed.
func TestWriteMarkdownCompile(t *testing.T) {
	results := []WorkloadResult{
		{
			Workload: "fib",
			Category: "compute",
			Measurements: []Measurement{
				{Runtime: "node", Stats: Stats{Median: 40 * time.Millisecond}},
				{
					Runtime: "bento",
					Stats:   Stats{Median: 5 * time.Millisecond},
					Compile: &Stats{Median: 250 * time.Millisecond},
				},
			},
		},
	}
	var b strings.Builder
	if err := WriteMarkdown(&b, results, Options{Runs: 10, Warmup: 2}); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if !strings.Contains(out, "## Compile step") {
		t.Fatal("markdown missing compile section for a two-phase runtime")
	}
	if !strings.Contains(out, "250.0ms") {
		t.Error("compile section missing the bento compile median")
	}
}

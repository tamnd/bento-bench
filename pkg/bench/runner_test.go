package bench

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestDiscoverWorkloads(t *testing.T) {
	dir := t.TempDir()
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(dir, "compute"), 0o755))
	must(os.MkdirAll(filepath.Join(dir, "fs"), 0o755))
	must(os.WriteFile(filepath.Join(dir, "compute", "fib.mjs"), []byte("1"), 0o644))
	must(os.WriteFile(filepath.Join(dir, "fs", "rw.mjs"), []byte("1"), 0o644))
	must(os.WriteFile(filepath.Join(dir, "notes.md"), []byte("skip"), 0o644))

	got, err := DiscoverWorkloads(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 workloads, got %d", len(got))
	}
	if got[0].Category != "compute" || got[0].Name != "fib" {
		t.Errorf("first workload = %+v, want compute/fib", got[0])
	}
}

func TestRuntimeRunArgs(t *testing.T) {
	rt := Runtime{Name: "deno", Bin: "deno", Args: []string{"run", "--quiet"}}
	args := rt.runArgs("/tmp/x.mjs")
	want := []string{"run", "--quiet", "/tmp/x.mjs"}
	if len(args) != len(want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("args[%d] = %q, want %q", i, args[i], want[i])
		}
	}
}

// TestCollectBudgetStopsEarly pins that a tiny budget cuts the timed pass down
// to the minimum floor of samples instead of the full run count, which is how a
// slow runtime is kept from dominating a run.
func TestCollectBudgetStopsEarly(t *testing.T) {
	bin, err := exec.LookPath("true")
	if err != nil {
		t.Skip("no true binary to time")
	}
	opts := Options{Warmup: 0, Runs: 20, Timeout: 5 * time.Second, Budget: time.Nanosecond}
	stats, err := collect(context.Background(), bin, nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Runs != minTimedRuns {
		t.Errorf("Runs = %d, want the budget floor of %d", stats.Runs, minTimedRuns)
	}
}

// TestCollectNoBudgetTakesAllRuns pins that with no budget the timed pass takes
// exactly the requested number of runs.
func TestCollectNoBudgetTakesAllRuns(t *testing.T) {
	bin, err := exec.LookPath("true")
	if err != nil {
		t.Skip("no true binary to time")
	}
	opts := Options{Warmup: 0, Runs: 5, Timeout: 5 * time.Second}
	stats, err := collect(context.Background(), bin, nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Runs != 5 {
		t.Errorf("Runs = %d, want 5", stats.Runs)
	}
}

func TestCompileStepCommand(t *testing.T) {
	c := &CompileStep{Bin: "bento", Args: []string{"build"}}
	bin, args := c.command("/tmp/w.ts", "/tmp/out/w")
	if bin != "bento" {
		t.Errorf("bin = %q, want bento", bin)
	}
	want := []string{"build", "-o", "/tmp/out/w", "/tmp/w.ts"}
	if len(args) != len(want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("args[%d] = %q, want %q", i, args[i], want[i])
		}
	}
}

package bench

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Workload is a single benchmark program. Category groups related workloads
// (startup, compute, json, fs) so the report can be read by area.
type Workload struct {
	Name     string
	Category string
	Path     string
}

// DiscoverWorkloads walks a directory and returns every .mjs, .js, and .ts file
// as a workload, using the immediate parent directory as the category. Results
// are sorted so runs are deterministic.
func DiscoverWorkloads(root string) ([]Workload, error) {
	var out []Workload
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".mjs", ".js", ".ts":
		default:
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		category := filepath.Dir(rel)
		if category == "." {
			category = "misc"
		}
		out = append(out, Workload{
			Name:     strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
			Category: filepath.ToSlash(category),
			Path:     path,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Category != out[j].Category {
			return out[i].Category < out[j].Category
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// Options controls how each workload is measured.
type Options struct {
	Warmup  int           // discarded runs before timing, to prime caches
	Runs    int           // timed runs collected into Stats
	Timeout time.Duration // per-run wall-clock limit
}

// Measurement is the timing of one runtime on one workload.
type Measurement struct {
	Runtime string `json:"runtime"`
	Stats   Stats  `json:"stats"`
	Failed  bool   `json:"failed"`
	Note    string `json:"note,omitempty"`
}

// WorkloadResult is every runtime's measurement for a single workload.
type WorkloadResult struct {
	Workload     string        `json:"workload"`
	Category     string        `json:"category"`
	Measurements []Measurement `json:"measurements"`
}

// timeOnce runs a workload once under a runtime and returns the wall-clock
// duration. A nonzero exit, a timeout, or a spawn failure is reported as an
// error so the caller can mark the runtime as failing this workload.
func timeOnce(ctx context.Context, rt Runtime, workload string, timeout time.Duration) (time.Duration, error) {
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	bin, args := rt.command(workload)
	cmd := exec.CommandContext(runCtx, bin, args...)
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	cmd.Stdout = nil
	cmd.Stderr = nil

	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start)

	if runCtx.Err() == context.DeadlineExceeded {
		return elapsed, errors.New("timeout")
	}
	if err != nil {
		return elapsed, err
	}
	return elapsed, nil
}

// measure runs the warmup and timed passes for one runtime on one workload.
func measure(ctx context.Context, rt Runtime, w Workload, opts Options) Measurement {
	for i := 0; i < opts.Warmup; i++ {
		if _, err := timeOnce(ctx, rt, w.Path, opts.Timeout); err != nil {
			return Measurement{Runtime: rt.Name, Failed: true, Note: warmupNote(err)}
		}
	}
	samples := make([]time.Duration, 0, opts.Runs)
	for i := 0; i < opts.Runs; i++ {
		d, err := timeOnce(ctx, rt, w.Path, opts.Timeout)
		if err != nil {
			return Measurement{Runtime: rt.Name, Failed: true, Note: err.Error()}
		}
		samples = append(samples, d)
	}
	return Measurement{Runtime: rt.Name, Stats: summarize(samples)}
}

func warmupNote(err error) string {
	return "warmup failed: " + err.Error()
}

// Run measures every available runtime against every workload and returns one
// result per workload. Runtimes that are not installed are skipped entirely.
func Run(ctx context.Context, workloads []Workload, runtimes []Runtime, opts Options) []WorkloadResult {
	live := make([]Runtime, 0, len(runtimes))
	for _, rt := range runtimes {
		if rt.Available() {
			live = append(live, rt)
		}
	}

	results := make([]WorkloadResult, 0, len(workloads))
	for _, w := range workloads {
		wr := WorkloadResult{Workload: w.Name, Category: w.Category}
		for _, rt := range live {
			wr.Measurements = append(wr.Measurements, measure(ctx, rt, w, opts))
		}
		results = append(results, wr)
	}
	return results
}

package bench

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	// Budget bounds the wall-clock spent on the timed runs of a single runtime on
	// a single workload. Once it is exceeded the timed pass stops early, as long
	// as at least minTimedRuns samples were collected, so a slow runtime does not
	// grind through every run while a fast one still gets the full count. Zero
	// means no budget: always take exactly Runs samples.
	Budget time.Duration

	// Progress, when set, receives a human-readable line as each workload begins
	// and as each measurement finishes, so a long run shows what it is doing
	// instead of sitting silent. It is never serialized.
	Progress io.Writer `json:"-"`
}

// minTimedRuns is the floor of timed samples the budget will never cut below, so
// even a very slow runtime still yields a median worth reporting.
const minTimedRuns = 3

// Measurement is the timing of one runtime on one workload. For a two-phase
// runtime (bento's AOT path), Compile carries the separate timing of the
// compile step and Stats is the timing of the produced binary; for a single
// phase runtime Compile is nil and Stats is the whole invocation.
type Measurement struct {
	Runtime string `json:"runtime"`
	Stats   Stats  `json:"stats"`
	Compile *Stats `json:"compile,omitempty"`
	Failed  bool   `json:"failed"`
	Note    string `json:"note,omitempty"`
}

// WorkloadResult is every runtime's measurement for a single workload.
type WorkloadResult struct {
	Workload     string        `json:"workload"`
	Category     string        `json:"category"`
	Measurements []Measurement `json:"measurements"`
}

// runCommand runs one command to completion and returns its wall-clock duration
// and peak resident memory. A nonzero exit, a timeout, or a spawn failure is an
// error so the caller can mark the runtime as failing this workload. Memory is
// read from the finished process rusage, which is zero when the platform cannot
// report it.
func runCommand(ctx context.Context, bin string, args []string, timeout time.Duration) (sample, error) {
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, bin, args...)
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	cmd.Stdout = nil
	cmd.Stderr = nil

	start := time.Now()
	err := cmd.Run()
	elapsed := time.Since(start)

	s := sample{dur: elapsed, rss: maxRSSBytes(cmd.ProcessState)}

	if runCtx.Err() == context.DeadlineExceeded {
		return s, errors.New("timeout")
	}
	if err != nil {
		return s, err
	}
	return s, nil
}

// collect runs one command through the warmup and timed passes and reduces the
// timed samples to Stats. A failure in any pass stops the collection and returns
// the error, which the caller turns into a failed measurement with a note.
func collect(ctx context.Context, bin string, args []string, opts Options) (Stats, error) {
	for i := 0; i < opts.Warmup; i++ {
		if _, err := runCommand(ctx, bin, args, opts.Timeout); err != nil {
			return Stats{}, err
		}
	}
	samples := make([]sample, 0, opts.Runs)
	start := time.Now()
	for i := 0; i < opts.Runs; i++ {
		s, err := runCommand(ctx, bin, args, opts.Timeout)
		if err != nil {
			return Stats{}, err
		}
		samples = append(samples, s)
		floor := min(minTimedRuns, opts.Runs)
		if opts.Budget > 0 && len(samples) >= floor && time.Since(start) >= opts.Budget {
			break
		}
	}
	return summarize(samples), nil
}

// measure runs the warmup and timed passes for one runtime on one workload. A
// single-phase runtime times the workload directly. A two-phase runtime first
// times the compile step, then times the binary it produced; a compile failure
// (which is what surfaces while bento's AOT build is still a stub) fails the
// whole measurement with the compiler's own message, rather than falling back to
// a different path and reporting a number the AOT column did not earn.
func measure(ctx context.Context, rt Runtime, w Workload, opts Options) Measurement {
	if rt.Compile == nil {
		stats, err := collect(ctx, rt.Bin, rt.runArgs(w.Path), opts)
		if err != nil {
			return failed(rt.Name, err)
		}
		return Measurement{Runtime: rt.Name, Stats: stats}
	}

	bin, cleanup, err := tempBinaryPath(w)
	if err != nil {
		return failed(rt.Name, err)
	}
	defer cleanup()

	cbin, cargs := rt.Compile.command(w.Path, bin)
	compile, err := collect(ctx, cbin, cargs, opts)
	if err != nil {
		return failed(rt.Name, errors.New("compile: "+err.Error()))
	}
	run, err := collect(ctx, bin, nil, opts)
	if err != nil {
		return failed(rt.Name, errors.New("run: "+err.Error()))
	}
	return Measurement{Runtime: rt.Name, Stats: run, Compile: &compile}
}

// tempBinaryPath returns a path for a compiled workload binary and a cleanup
// that removes its directory. The name carries the workload so a stray file left
// by a killed run is still identifiable.
func tempBinaryPath(w Workload) (string, func(), error) {
	dir, err := os.MkdirTemp("", "bento-bench-*")
	if err != nil {
		return "", func() {}, err
	}
	return filepath.Join(dir, w.Name), func() { _ = os.RemoveAll(dir) }, nil
}

func failed(name string, err error) Measurement {
	return Measurement{Runtime: name, Failed: true, Note: err.Error()}
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
	for i, w := range workloads {
		logf(opts.Progress, "[%d/%d] %s/%s\n", i+1, len(workloads), w.Category, w.Name)
		wr := WorkloadResult{Workload: w.Name, Category: w.Category}
		for _, rt := range live {
			logf(opts.Progress, "    %-6s running...\n", rt.Name)
			m := measure(ctx, rt, w, opts)
			logCell(opts.Progress, m, opts.Runs)
			wr.Measurements = append(wr.Measurements, m)
		}
		results = append(results, wr)
	}
	return results
}

// logf writes a progress line when a progress writer is set, and is a no-op
// otherwise so measurement never depends on progress being wired.
func logf(w io.Writer, format string, args ...any) {
	if w == nil {
		return
	}
	_, _ = fmt.Fprintf(w, format, args...)
}

// logCell reports the outcome of one measurement: the median duration and peak
// memory with the number of runs kept, or the failure note. When the budget cut
// the timed pass below the requested count it is called out, so a shortened cell
// is never mistaken for the full one.
func logCell(w io.Writer, m Measurement, wantRuns int) {
	if w == nil {
		return
	}
	if m.Failed {
		_, _ = fmt.Fprintf(w, "    %-6s fail: %s\n", m.Runtime, m.Note)
		return
	}
	runs := fmt.Sprintf("%d runs", m.Stats.Runs)
	if m.Stats.Runs < wantRuns {
		runs += ", budget capped"
	}
	compile := ""
	if m.Compile != nil {
		compile = fmt.Sprintf(", compile %s", fmtDur(m.Compile.Median))
	}
	_, _ = fmt.Fprintf(w, "    %-6s %9s  %9s  (%s%s)\n",
		m.Runtime, fmtDur(m.Stats.Median), fmtBytes(m.Stats.MedianRSS), runs, compile)
}

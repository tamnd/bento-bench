// Command bento-bench times the same workloads on bento, Node, Bun, and Deno and
// reports the median wall-clock of each. It prints a text table, and can write a
// JSON record and a Markdown summary for CI to publish. bento is located through
// BENTO_BIN when set, otherwise it is taken from PATH like the others.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/tamnd/bento-bench/pkg/bench"
)

func main() {
	var (
		dir     = flag.String("workloads", "workloads", "directory of workload programs")
		warmup  = flag.Int("warmup", 1, "discarded runs before timing")
		runs    = flag.Int("runs", 5, "timed runs per workload")
		timeout = flag.Duration("timeout", 60*time.Second, "per-run timeout")
		budget  = flag.Duration("budget", 1500*time.Millisecond, "soft wall-clock ceiling on the timed runs of one runtime on one workload; a slow runtime stops early after enough samples instead of taking all --runs (0 means no ceiling). The default keeps a full workload round across every runtime under about ten seconds; pass --budget 0 --runs 20 for a high-fidelity publish run")
		jsonOut = flag.String("json", "", "write raw results as JSON to this path")
		mdOut   = flag.String("markdown", "", "write a Markdown summary to this path")
		skip    = flag.String("skip", "", "comma-separated runtime names to leave out of this run (for example \"bento\" to skip the ahead-of-time build on a runner without the Go toolchain)")
	)
	flag.Parse()

	if *runs < 1 {
		fmt.Fprintln(os.Stderr, "bento-bench: --runs must be at least 1")
		os.Exit(2)
	}

	workloads, err := bench.DiscoverWorkloads(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bento-bench: %v\n", err)
		os.Exit(2)
	}
	if len(workloads) == 0 {
		fmt.Fprintf(os.Stderr, "bento-bench: no workloads found under %s\n", *dir)
		os.Exit(2)
	}

	runtimes := bench.Without(bench.DefaultRuntimes(), *skip)
	reportAvailability(runtimes)

	opts := bench.Options{Warmup: *warmup, Runs: *runs, Timeout: *timeout, Budget: *budget, Progress: os.Stderr}
	results := bench.Run(context.Background(), workloads, runtimes, opts)

	if err := bench.WriteReport(os.Stdout, results, opts); err != nil {
		fmt.Fprintf(os.Stderr, "bento-bench: write report: %v\n", err)
		os.Exit(2)
	}

	if *jsonOut != "" {
		if err := writeJSON(*jsonOut, results, opts); err != nil {
			fmt.Fprintf(os.Stderr, "bento-bench: write json: %v\n", err)
			os.Exit(2)
		}
	}
	if *mdOut != "" {
		if err := writeMarkdown(*mdOut, results, opts); err != nil {
			fmt.Fprintf(os.Stderr, "bento-bench: write markdown: %v\n", err)
			os.Exit(2)
		}
	}
}

func reportAvailability(runtimes []bench.Runtime) {
	fmt.Fprintln(os.Stderr, "Runtimes:")
	for _, rt := range runtimes {
		status := "not found"
		if rt.Available() {
			status = "ready"
		}
		fmt.Fprintf(os.Stderr, "  %-6s %s\n", rt.Name, status)
	}
	fmt.Fprintln(os.Stderr)
}

// record is the JSON shape written to disk: the options used and every result.
type record struct {
	Options bench.Options          `json:"options"`
	Results []bench.WorkloadResult `json:"results"`
}

func writeJSON(path string, results []bench.WorkloadResult, opts bench.Options) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(record{Options: opts, Results: results})
}

func writeMarkdown(path string, results []bench.WorkloadResult, opts bench.Options) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	err = bench.WriteMarkdown(f, results, opts)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

package bench

import (
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"
	"time"
)

// printer writes formatted lines to an io.Writer and remembers the first error,
// so the report code stays readable without an error check on every line.
type printer struct {
	w   io.Writer
	err error
}

func (p *printer) printf(format string, a ...any) {
	if p.err != nil {
		return
	}
	_, p.err = fmt.Fprintf(p.w, format, a...)
}

func (p *printer) line(s string) {
	if p.err != nil {
		return
	}
	_, p.err = fmt.Fprintln(p.w, s)
}

// writef appends a formatted fragment to a builder. Builder writes never fail,
// and splitting the Sprintf from the WriteString keeps both errcheck and
// staticcheck happy when a line is built piece by piece.
func writef(b *strings.Builder, format string, a ...any) {
	s := fmt.Sprintf(format, a...)
	b.WriteString(s)
}

// runtimeOrder returns the runtimes present in the results, in first-seen order,
// which keeps bento last where DefaultRuntimes puts it.
func runtimeOrder(results []WorkloadResult) []string {
	var order []string
	seen := map[string]bool{}
	for _, wr := range results {
		for _, m := range wr.Measurements {
			if !seen[m.Runtime] {
				seen[m.Runtime] = true
				order = append(order, m.Runtime)
			}
		}
	}
	return order
}

// fastest returns the name of the runtime with the lowest median in a workload,
// ignoring failures. It returns "" when nobody succeeded.
func fastest(wr WorkloadResult) string {
	best := ""
	var bestMed time.Duration
	for _, m := range wr.Measurements {
		if m.Failed {
			continue
		}
		if best == "" || m.Stats.Median < bestMed {
			best = m.Runtime
			bestMed = m.Stats.Median
		}
	}
	return best
}

// cell renders one runtime's median for the text and markdown tables. A failed
// run shows as "fail"; the fastest runtime is marked with a leading star.
func cell(m Measurement, isFastest bool) string {
	if m.Failed {
		return "fail"
	}
	s := fmtDur(m.Stats.Median)
	if isFastest {
		return "*" + s
	}
	return s
}

// fmtDur prints a duration in a compact, human-scaled form.
func fmtDur(d time.Duration) string {
	switch {
	case d >= time.Second:
		return fmt.Sprintf("%.2fs", d.Seconds())
	case d >= time.Millisecond:
		return fmt.Sprintf("%.1fms", float64(d)/float64(time.Millisecond))
	case d >= time.Microsecond:
		return fmt.Sprintf("%.0fus", float64(d)/float64(time.Microsecond))
	default:
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
}

// fmtBytes prints a byte count in a compact, human-scaled form. A zero or
// negative count is "n/a", which is what a run that could not report memory
// carries, so the report never shows a false 0 B.
func fmtBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case b <= 0:
		return "n/a"
	case b >= gb:
		return fmt.Sprintf("%.2fGB", float64(b)/gb)
	case b >= mb:
		return fmt.Sprintf("%.1fMB", float64(b)/mb)
	case b >= kb:
		return fmt.Sprintf("%.0fKB", float64(b)/kb)
	default:
		return fmt.Sprintf("%dB", b)
	}
}

// leanest returns the runtime with the lowest median peak memory in a workload,
// ignoring failures and runs with no memory reading. It returns "" when nobody
// reported memory.
func leanest(wr WorkloadResult) string {
	best := ""
	var bestRSS int64
	for _, m := range wr.Measurements {
		if m.Failed || m.Stats.MedianRSS <= 0 {
			continue
		}
		if best == "" || m.Stats.MedianRSS < bestRSS {
			best = m.Runtime
			bestRSS = m.Stats.MedianRSS
		}
	}
	return best
}

// memCell renders one runtime's median peak memory. A failed run shows as
// "fail", a run with no memory reading as "n/a"; the leanest runtime is starred.
func memCell(m Measurement, isLeanest bool) string {
	if m.Failed {
		return "fail"
	}
	s := fmtBytes(m.Stats.MedianRSS)
	if isLeanest && s != "n/a" {
		return "*" + s
	}
	return s
}

func measurementFor(wr WorkloadResult, runtime string) (Measurement, bool) {
	for _, m := range wr.Measurements {
		if m.Runtime == runtime {
			return m, true
		}
	}
	return Measurement{}, false
}

// WriteReport prints a text table: one row per workload, one column per runtime,
// each cell the median wall-clock time. The fastest cell in a row is starred.
func WriteReport(w io.Writer, results []WorkloadResult, opts Options) error {
	p := &printer{w: w}
	runtimes := runtimeOrder(results)

	p.printf("Median wall-clock over %d runs (%d warmup), lower is better. * marks fastest.\n\n", opts.Runs, opts.Warmup)

	var header strings.Builder
	writef(&header, "%-22s", "workload")
	for _, rt := range runtimes {
		writef(&header, "  %-10s", rt)
	}
	p.line(header.String())
	p.line(strings.Repeat("-", header.Len()))

	for _, wr := range results {
		fast := fastest(wr)
		var row strings.Builder
		writef(&row, "%-22s", wr.Category+"/"+wr.Workload)
		for _, rt := range runtimes {
			m, ok := measurementFor(wr, rt)
			text := "-"
			if ok {
				text = cell(m, rt == fast)
			}
			writef(&row, "  %-10s", text)
		}
		p.line(row.String())
	}

	p.line("")
	p.line("Median peak memory (RSS), lower is better. * marks leanest.")
	p.line("")
	var memHeader strings.Builder
	writef(&memHeader, "%-22s", "workload")
	for _, rt := range runtimes {
		writef(&memHeader, "  %-10s", rt)
	}
	p.line(memHeader.String())
	p.line(strings.Repeat("-", memHeader.Len()))
	for _, wr := range results {
		lean := leanest(wr)
		var row strings.Builder
		writef(&row, "%-22s", wr.Category+"/"+wr.Workload)
		for _, rt := range runtimes {
			m, ok := measurementFor(wr, rt)
			text := "-"
			if ok {
				text = memCell(m, rt == lean)
			}
			writef(&row, "  %-10s", text)
		}
		p.line(row.String())
	}

	writeStartupText(p, results, runtimes)
	writeBinarySizeText(p, results, runtimes)
	return p.err
}

// startupCost returns each runtime's startup duration and peak memory, taken as
// the median over the startup-category workloads (the smallest programs, whose
// wall-clock is dominated by process start and teardown). A runtime missing from
// the map either had no startup workload or failed every one.
func startupCost(results []WorkloadResult, runtimes []string) (map[string]time.Duration, map[string]int64) {
	durs := map[string][]time.Duration{}
	mems := map[string][]int64{}
	for _, wr := range results {
		if wr.Category != "startup" {
			continue
		}
		for _, rt := range runtimes {
			m, ok := measurementFor(wr, rt)
			if !ok || m.Failed {
				continue
			}
			durs[rt] = append(durs[rt], m.Stats.Median)
			if m.Stats.MedianRSS > 0 {
				mems[rt] = append(mems[rt], m.Stats.MedianRSS)
			}
		}
	}
	dOut := map[string]time.Duration{}
	for rt, ds := range durs {
		slices.Sort(ds)
		dOut[rt] = percentile(ds, 0.50)
	}
	mOut := map[string]int64{}
	for rt, ms := range mems {
		slices.Sort(ms)
		mOut[rt] = medianInt64(ms)
	}
	return dOut, mOut
}

// writeStartupText adds a short startup section to the text report, one line per
// runtime, so the cold-start cost a user pays on every invocation is called out
// on its own rather than buried in the workload grid.
func writeStartupText(p *printer, results []WorkloadResult, runtimes []string) {
	durs, mems := startupCost(results, runtimes)
	if len(durs) == 0 {
		return
	}
	p.line("")
	p.line("Startup cost (median of the startup workloads):")
	for _, rt := range runtimes {
		d, ok := durs[rt]
		if !ok {
			continue
		}
		p.printf("  %-6s %-10s %s\n", rt, fmtDur(d), fmtBytes(mems[rt]))
	}
}

// WriteMarkdown renders the results as a Markdown table for the CI job summary.
func WriteMarkdown(w io.Writer, results []WorkloadResult, opts Options) error {
	p := &printer{w: w}
	runtimes := runtimeOrder(results)

	p.line("# bento benchmark")
	p.line("")
	p.printf("Median wall-clock over %d runs (%d warmup), lower is better. A star marks the fastest runtime in each row.\n\n", opts.Runs, opts.Warmup)

	var head, sep strings.Builder
	head.WriteString("| workload |")
	sep.WriteString("| --- |")
	for _, rt := range runtimes {
		writef(&head, " %s |", rt)
		sep.WriteString(" --- |")
	}
	p.line(head.String())
	p.line(sep.String())

	for _, wr := range results {
		fast := fastest(wr)
		var row strings.Builder
		writef(&row, "| %s/%s |", wr.Category, wr.Workload)
		for _, rt := range runtimes {
			m, ok := measurementFor(wr, rt)
			text := "-"
			if ok {
				text = cell(m, rt == fast)
			}
			writef(&row, " %s |", text)
		}
		p.line(row.String())
	}

	writeMemoryMarkdown(p, results, runtimes)
	writeStartupMarkdown(p, results, runtimes)
	writeCompileMarkdown(p, results, runtimes)
	writeBinarySizeMarkdown(p, results, runtimes)
	writeSpeedup(p, results, runtimes)
	return p.err
}

// writeMemoryMarkdown renders the peak-memory table: one row per workload, one
// column per runtime, each cell the median peak RSS, the leanest starred.
func writeMemoryMarkdown(p *printer, results []WorkloadResult, runtimes []string) {
	p.line("")
	p.line("## Peak memory")
	p.line("")
	p.line("Median peak resident memory over the timed runs, lower is better. A star marks the leanest runtime in each row.")
	p.line("")

	var head, sep strings.Builder
	head.WriteString("| workload |")
	sep.WriteString("| --- |")
	for _, rt := range runtimes {
		writef(&head, " %s |", rt)
		sep.WriteString(" --- |")
	}
	p.line(head.String())
	p.line(sep.String())

	for _, wr := range results {
		lean := leanest(wr)
		var row strings.Builder
		writef(&row, "| %s/%s |", wr.Category, wr.Workload)
		for _, rt := range runtimes {
			m, ok := measurementFor(wr, rt)
			text := "-"
			if ok {
				text = memCell(m, rt == lean)
			}
			writef(&row, " %s |", text)
		}
		p.line(row.String())
	}
}

// writeStartupMarkdown renders the startup cost per runtime, the median duration
// and memory of the startup-category workloads, so cold start is called out on
// its own.
func writeStartupMarkdown(p *printer, results []WorkloadResult, runtimes []string) {
	durs, mems := startupCost(results, runtimes)
	if len(durs) == 0 {
		return
	}
	p.line("")
	p.line("## Startup cost")
	p.line("")
	p.line("Median wall-clock and peak memory of the startup workloads, the floor you pay on every invocation.")
	p.line("")
	p.line("| runtime | startup | memory |")
	p.line("| --- | --- | --- |")
	for _, rt := range runtimes {
		d, ok := durs[rt]
		if !ok {
			continue
		}
		p.printf("| %s | %s | %s |\n", rt, fmtDur(d), fmtBytes(mems[rt]))
	}
}

// writeCompileMarkdown renders the compile step for the runtimes that have one
// (bento, bun, and deno all compile a workload to a single executable), so the
// cost of turning TypeScript into a binary is visible next to the speed of the
// binary it produced. The section is omitted when no runtime reported a compile
// step, which is the case for node's interpreter path.
func writeCompileMarkdown(p *printer, results []WorkloadResult, runtimes []string) {
	twoPhase := make([]string, 0, len(runtimes))
	for _, rt := range runtimes {
		if anyCompile(results, rt) {
			twoPhase = append(twoPhase, rt)
		}
	}
	if len(twoPhase) == 0 {
		return
	}
	p.line("")
	p.line("## Compile step")
	p.line("")
	p.line("Median wall-clock of the ahead-of-time compile that turns each workload into a native binary. The speed table above times the binary this step produced.")
	p.line("")

	var head, sep strings.Builder
	head.WriteString("| workload |")
	sep.WriteString("| --- |")
	for _, rt := range twoPhase {
		writef(&head, " %s |", rt)
		sep.WriteString(" --- |")
	}
	p.line(head.String())
	p.line(sep.String())

	for _, wr := range results {
		var row strings.Builder
		writef(&row, "| %s/%s |", wr.Category, wr.Workload)
		for _, rt := range twoPhase {
			m, ok := measurementFor(wr, rt)
			text := "-"
			switch {
			case ok && m.Compile != nil:
				text = fmtDur(m.Compile.Median)
			case ok && m.Failed:
				text = "fail"
			}
			writef(&row, " %s |", text)
		}
		p.line(row.String())
	}
}

// anyCompile reports whether a runtime produced a compile measurement on any
// workload, which marks it as a two-phase (ahead-of-time) runtime in the report.
func anyCompile(results []WorkloadResult, runtime string) bool {
	for _, wr := range results {
		if m, ok := measurementFor(wr, runtime); ok && m.Compile != nil {
			return true
		}
	}
	return false
}

// binarySize returns each compiling runtime's median produced-binary size in
// bytes, taken over the workloads where it reported one. A runtime that never
// compiled a binary (the interpreter path) is absent from the map, so it carries
// no size cell rather than a false zero.
func binarySize(results []WorkloadResult, runtimes []string) map[string]int64 {
	sizes := map[string][]int64{}
	for _, wr := range results {
		for _, rt := range runtimes {
			m, ok := measurementFor(wr, rt)
			if !ok || m.Failed || m.BinarySize <= 0 {
				continue
			}
			sizes[rt] = append(sizes[rt], m.BinarySize)
		}
	}
	out := map[string]int64{}
	for rt, ss := range sizes {
		slices.Sort(ss)
		out[rt] = medianInt64(ss)
	}
	return out
}

// writeBinarySizeMarkdown renders the size of the single executable each
// compiling runtime produces, so the native footprint is comparable like with
// like: bento's Go binary against bun's and deno's bundled runtimes. node is
// absent because it stays the interpreter reference with no compiled binary. The
// section is omitted when no runtime produced a binary.
func writeBinarySizeMarkdown(p *printer, results []WorkloadResult, runtimes []string) {
	sizes := binarySize(results, runtimes)
	if len(sizes) == 0 {
		return
	}
	p.line("")
	p.line("## Binary size")
	p.line("")
	p.line("Median size of the single executable each runtime compiles a workload into, lower is better. node stays the interpreter reference, so it compiles no binary and is left out.")
	p.line("")
	p.line("| runtime | binary |")
	p.line("| --- | --- |")
	for _, rt := range runtimes {
		s, ok := sizes[rt]
		if !ok {
			continue
		}
		p.printf("| %s | %s |\n", rt, fmtBytes(s))
	}
}

// writeBinarySizeText adds the binary-size summary to the text report, one line
// per compiling runtime, so the native footprint sits next to the speed and
// memory tables.
func writeBinarySizeText(p *printer, results []WorkloadResult, runtimes []string) {
	sizes := binarySize(results, runtimes)
	if len(sizes) == 0 {
		return
	}
	p.line("")
	p.line("Binary size (median single executable each runtime compiles a workload into):")
	for _, rt := range runtimes {
		s, ok := sizes[rt]
		if !ok {
			continue
		}
		p.printf("  %-6s %s\n", rt, fmtBytes(s))
	}
}

// writeSpeedup adds a short section showing bento's median relative to the
// fastest other runtime per category, so regressions are easy to spot.
func writeSpeedup(p *printer, results []WorkloadResult, runtimes []string) {
	if !slices.Contains(runtimes, "bento") {
		return
	}
	type acc struct{ bento, best float64 }
	byCat := map[string]*acc{}
	var cats []string
	for _, wr := range results {
		bm, ok := measurementFor(wr, "bento")
		if !ok || bm.Failed {
			continue
		}
		var bestOther time.Duration
		haveOther := false
		for _, m := range wr.Measurements {
			if m.Runtime == "bento" || m.Failed {
				continue
			}
			if !haveOther || m.Stats.Median < bestOther {
				bestOther = m.Stats.Median
				haveOther = true
			}
		}
		if !haveOther {
			continue
		}
		a := byCat[wr.Category]
		if a == nil {
			a = &acc{}
			byCat[wr.Category] = a
			cats = append(cats, wr.Category)
		}
		a.bento += float64(bm.Stats.Median)
		a.best += float64(bestOther)
	}
	if len(cats) == 0 {
		return
	}
	sort.Strings(cats)

	p.line("")
	p.line("## bento versus the fastest other runtime")
	p.line("")
	p.line("| category | ratio | reading |")
	p.line("| --- | --- | --- |")
	for _, c := range cats {
		a := byCat[c]
		if a.best == 0 {
			continue
		}
		ratio := a.bento / a.best
		p.printf("| %s | %.2fx | %s |\n", c, ratio, reading(ratio))
	}
}

func reading(ratio float64) string {
	switch {
	case ratio <= 1.05:
		return "on par or faster"
	case ratio <= 2:
		return "within 2x"
	default:
		return "slower, room to close"
	}
}

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
	return p.err
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

	writeSpeedup(p, results, runtimes)
	return p.err
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

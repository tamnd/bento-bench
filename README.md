# bento-bench

Benchmark harness for [bento](https://github.com/tamnd/bento), the TypeScript runtime built in Go.

It runs the same workload on bento, Node, Bun, and Deno, times each one end to end, and reports the median wall-clock side by side.
The number that matters is the one a user actually feels: process start to process exit.
That means every workload includes startup cost, which is fair because you pay it on every invocation.

## What it measures

Each workload under `workloads/` is an ordinary ES module that all four runtimes can execute unchanged.
The harness runs each one a few warmup times to prime caches, then times a batch of runs and reduces them to min, mean, median, p90, and standard deviation.
With `--budget` set, the timed pass for a single runtime on a single workload stops once it has spent that much wall-clock, as long as it collected at least three samples, so a slow runtime does not grind through every run while a fast one still takes the full count. Each Stats records how many runs it actually kept, so a shortened cell is honest about it.
A run counts as failed if the process exits nonzero or times out, and a failed cell is shown as `fail` rather than a fake number.

For every run it captures two things: the wall-clock duration from process start to exit, and the peak resident memory the process held, read from the kernel rusage of the finished process. So the report has three parts: a speed table, a peak-memory table, and a startup section that pulls the startup workloads out on their own, since cold start is the cost you pay on every single invocation. Peak memory is reported in bytes on macOS and normalized from the kilobytes Linux reports, and a run that cannot report memory shows `n/a` rather than a false zero.

The workloads are grouped by area so the table can be read by what it stresses:

- `startup/` the smallest possible program, which isolates cold start and teardown.
- `compute/` call-heavy and numeric loops (Fibonacci, Mandelbrot, string churn, primitive coercion, number formatting, the bit-exact Math methods, the Math and Number constants, template-literal building, String.fromCharCode construction, and string replace and replaceAll rewriting) that lean on the engine.
- `json/` stringify and parse round trips that mix serialization with allocation.
- `fs/` batches of small file writes and reads through node:fs, which leans on the host bridge.

## Reading the results

bento starts from a different place than the others.
Node, Bun, and Deno all ship an optimizing JIT (V8 or JavaScriptCore), so on tight compute loops they have a real head start today.
bento runs the pure-Go quickjs engine by default, which interprets, so it trades peak throughput for a tiny binary, no cgo, and fast startup.
You can see both sides in the table: bento tends to win cold start and trail on heavy compute.

That gap is the point of tracking it.
As bento's AOT TypeScript-to-Go compiler lands, compute-bound workloads should move toward the pack, and this harness is how that progress gets measured instead of asserted.
For that reason CI currently runs with `--skip bento`: on a shared runner the interpreter's compute workloads would push the job past its time budget, and the number worth publishing is the ahead-of-time one, which is the next thing to land. The interpreter's own numbers still come from local and release runs, where nothing is skipped.

### The ahead-of-time path

`--bento-aot` measures the path bento is being built for: instead of the interpreter, each workload is compiled to a native Go binary and the binary is what gets timed. The compile step is timed on its own and shown in a separate compile-step table, so you can read the one-time cost of turning TypeScript into a binary apart from the speed of the binary it produced. Until `bento build` lands end to end, the compile step fails and the bento column shows `fail` with the compiler's own message, which is honest rather than a fallback to the interpreter reported under the AOT label. The default mode stays the interpreter, so the numbers a plain run reports are always real today.

## Running it

You need Go, and whichever runtimes you want to compare on your PATH.
Point the harness at a bento binary with `BENTO_BIN`, otherwise it takes `bento` from PATH.

```
go run ./cmd/bento-bench
BENTO_BIN=/path/to/bento go run ./cmd/bento-bench --runs 20 --warmup 3
```

Useful flags:

- `--workloads DIR` the workload directory, default `workloads`
- `--runs N` timed runs per workload, default 10
- `--warmup N` discarded runs before timing, default 2
- `--timeout D` per-run timeout, default 60s
- `--budget D` soft ceiling on the timed runs of one runtime on one workload, so a slow runtime stops after a few samples instead of taking all `--runs`, while fast runtimes still take the full count; default off
- `--skip names` comma-separated runtime names to leave out of a run, for example `bento` to drop the slow interpreter; default none
- `--json PATH` write the raw results as JSON
- `--markdown PATH` write a Markdown summary
- `--bento-aot` measure bento by compiling each workload to a Go binary and timing the binary, instead of the interpreter

Each workload also honors a `BENCH_SCALE` environment variable that scales the per-run work it does, unset or `1` being the full size and a smaller value doing proportionally less. It is the size knob for a run: CI sets it low so a run is quick, while a local run leaves it at full size for real numbers. The workloads that assert a fixed checksum still agree across runtimes at any fixed scale, and their documented checksum is the one at the default scale.

A runtime that is not installed is skipped, not failed, so the harness is useful even with only some runtimes present.

While it runs, the harness logs progress to stderr: a line as each workload begins, a line as each runtime starts, and a line with the median duration, peak memory, and run count as each cell finishes, with the compile time shown too on the AOT path. A cell the budget cut short is marked so a shortened run is never mistaken for a full one. The tables still go to stdout, so `--json` and `--markdown` and a redirected stdout stay clean.

## A note on fairness

Wall-clock benchmarks are sensitive to the machine, the load, and the runtime versions.
Treat a single local run as a snapshot, not a verdict.
CI runs the same workloads on a fixed runner and publishes the numbers as a job summary and an artifact, which is the more stable place to compare across time.

## License

MIT

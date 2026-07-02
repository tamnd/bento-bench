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

For every run it captures two things: the wall-clock duration from process start to exit, and the peak resident memory the process held, read from the kernel rusage of the finished process. So the report has a speed table, a peak-memory table, a startup section that pulls the startup workloads out on their own since cold start is the cost you pay on every single invocation, a compile-step table for the runtimes that compile to a binary, and a binary-size table that compares the single executables they produce. Peak memory is reported in bytes on macOS and normalized from the kilobytes Linux reports, and a run that cannot report memory shows `n/a` rather than a false zero.

The workloads are grouped by area so the table can be read by what it stresses:

- `startup/` the smallest possible program, which isolates cold start and teardown.
- `compute/` call-heavy and numeric loops (Fibonacci, Mandelbrot, string churn, primitive coercion, number formatting, the bit-exact Math methods, the Math and Number constants, template-literal building, String.fromCharCode construction, and string replace and replaceAll rewriting) that lean on the engine.
- `json/` stringify and parse round trips that mix serialization with allocation.
- `fs/` batches of small file writes and reads through node:fs, which leans on the host bridge.

## Reading the results

bento starts from a different place than the others.
Node, Bun, and Deno all ship an optimizing JIT (V8 or JavaScriptCore), so on tight compute loops they have a real head start.
bento is measured through its ahead-of-time path: each workload is compiled to a native Go binary with `bento build` and the binary is what gets timed, which is the number the compiler exists to move. The tree-walking interpreter behind `bento run` is not a benchmark target, since publishing an interpreter number under bento's name would measure the thing bento is built to replace.

To compare like with like, Bun and Deno are measured the same way: each workload is compiled to a single executable with `bun build --compile` and `deno compile`, and that binary is timed. So the run column, the compile-step table, and the binary-size table all line up across bento, Bun, and Deno.

Node stays the interpreter reference. Its single-executable path (SEA) is experimental, needs the external `postject` tool, and does not accept TypeScript directly, so compiling a `.ts` workload to one node binary is not the clean single command the other three offer. Node is timed as `node workload.ts` and carries no binary-size cell rather than a number earned a different way.

### The compile-to-binary path

Each workload is compiled to a single native binary and the binary is what gets timed. The compile step is timed on its own and shown in a separate compile-step table, so you can read the one-time cost of turning TypeScript into a binary apart from the speed of the binary it produced. The binary-size table then compares those executables directly, which is where bento's small Go binary stands out against the bundled JavaScript runtimes. A construct the bento compiler does not yet lower fails the build, and the bento column shows `fail` with the compiler's own message, which is honest rather than a fallback to another path reported under the bento label.

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
- `--skip names` comma-separated runtime names to leave out of a run, for example `bento` to skip the ahead-of-time build on a machine without the Go toolchain; default none
- `--json PATH` write the raw results as JSON
- `--markdown PATH` write a Markdown summary

The workloads are a fixed size, so a run is directly comparable to any other run and the fast runtimes stay quick under `--budget`. The workloads that reduce to a checksum print the same value on every runtime, and that value is written into the workload's own comment.

Because bento compiles each workload with `bento build` and links its value runtime in, the build needs to find the bento module source. It is discovered from the `BENTO_BIN` binary when that binary sits inside a checkout; when it does not, point `BENTO_MODULE_ROOT` at a checkout of the bento repository.

A runtime that is not installed is skipped, not failed, so the harness is useful even with only some runtimes present.

While it runs, the harness logs progress to stderr: a line as each workload begins, a line as each runtime starts, and a line with the median duration, peak memory, and run count as each cell finishes, with the compile time shown too on the AOT path. A cell the budget cut short is marked so a shortened run is never mistaken for a full one. The tables still go to stdout, so `--json` and `--markdown` and a redirected stdout stay clean.

## Runtime versions

The published numbers come from CI, which pins each runtime to a concrete version so a comparison across time moves only when we deliberately bump one, not whenever a floating tag rolls forward:

- Node [24.18.0](https://github.com/nodejs/node/releases/tag/v24.18.0), the current LTS line, which runs TypeScript directly through native type stripping.
- Deno [2.7.7](https://github.com/denoland/deno/releases/tag/v2.7.7).
- Bun [1.3.11](https://github.com/oven-sh/bun/releases/tag/bun-v1.3.11).
- [Go 1.26](https://go.dev/doc/devel/release#go1.26) from `go.mod`, which builds bento and compiles each AOT binary.

The CI run prints the exact resolved versions in a "Versions" step before the benchmark, so every published result carries the versions it was produced with. A local run uses whatever `node`, `deno`, `bun`, and `go` are on your PATH, which is why a local run is a snapshot rather than the number of record.

## A note on fairness

Wall-clock benchmarks are sensitive to the machine, the load, and the runtime versions.
Treat a single local run as a snapshot, not a verdict.
CI runs the same workloads on a fixed runner and publishes the numbers as a job summary and an artifact, which is the more stable place to compare across time.

## License

MIT

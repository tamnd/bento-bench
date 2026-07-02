// A recursive Fibonacci is a dense call-heavy microbenchmark. It stresses the
// call path and integer arithmetic more than allocation, so it separates raw
// interpreter or JIT throughput from garbage collector behavior.
function fib(n) {
  if (n < 2) return n;
  return fib(n - 1) + fib(n - 2);
}

// BENCH_SCALE scales the outer repeat count so CI can run a lighter size
// without editing the workload; unset or 1 is the full size.
const SCALE = Number(process.env.BENCH_SCALE ?? "1");
const reps = Math.max(1, Math.round(5 * SCALE));
let acc = 0;
for (let i = 0; i < reps; i++) {
  acc += fib(32);
}
process.stdout.write(String(acc) + "\n");

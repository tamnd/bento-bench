// A recursive Fibonacci is a dense call-heavy microbenchmark. It stresses the
// call path and integer arithmetic more than allocation, so it separates raw
// interpreter or JIT throughput from garbage collector behavior.
function fib(n: number): number {
  if (n < 2) return n;
  return fib(n - 1) + fib(n - 2);
}

const reps = 5;
let acc = 0;
const t0 = performance.now();
for (let i = 0; i < reps; i++) {
  acc += fib(32);
}
const t1 = performance.now();
console.error("compute_ms=" + (t1 - t0));
console.log(acc);

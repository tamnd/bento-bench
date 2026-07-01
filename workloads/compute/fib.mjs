// A recursive Fibonacci is a dense call-heavy microbenchmark. It stresses the
// call path and integer arithmetic more than allocation, so it separates raw
// interpreter or JIT throughput from garbage collector behavior.
function fib(n) {
  if (n < 2) return n;
  return fib(n - 1) + fib(n - 2);
}

let acc = 0;
for (let i = 0; i < 5; i++) {
  acc += fib(32);
}
process.stdout.write(String(acc) + "\n");

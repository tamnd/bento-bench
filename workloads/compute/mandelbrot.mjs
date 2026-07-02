// A tight numeric loop over a small Mandelbrot grid. It is float heavy and
// branch heavy with no allocation in the inner loop, so it rewards runtimes
// with a strong optimizing compiler.
// BENCH_SCALE scales the number of rows so CI can run a lighter size without
// editing the workload; unset or 1 is the full grid.
const SCALE = Number(process.env.BENCH_SCALE ?? "1");
const width = 200;
const height = Math.max(1, Math.round(200 * SCALE));
const maxIter = 100;
let checksum = 0;

for (let py = 0; py < height; py++) {
  const y0 = (py / height) * 2 - 1;
  for (let px = 0; px < width; px++) {
    const x0 = (px / width) * 3 - 2;
    let x = 0;
    let y = 0;
    let iter = 0;
    while (x * x + y * y <= 4 && iter < maxIter) {
      const xt = x * x - y * y + x0;
      y = 2 * x * y + y0;
      x = xt;
      iter++;
    }
    checksum += iter;
  }
}
process.stdout.write(String(checksum) + "\n");

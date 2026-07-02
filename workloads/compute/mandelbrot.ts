// A tight numeric loop over a small Mandelbrot grid. It is float heavy and
// branch heavy with no allocation in the inner loop, so it rewards runtimes
// with a strong optimizing compiler.
const width = 200;
const height = 200;
const maxIter = 100;
let checksum = 0;

const t0 = performance.now();
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
const t1 = performance.now();
console.error("compute_ms=" + (t1 - t0));
console.log(checksum);

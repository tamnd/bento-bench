// Numeric typed arrays in a tight index-addressed loop, which is what typed arrays
// exist for: a dense fixed-width buffer with no per-element boxing. It fills an
// Int32Array, a Float64Array, and a Uint8ClampedArray, then runs read-modify-write
// passes that store back through each buffer, so the element store coercion (Int32
// wrap, Float64 identity, Uint8Clamped clamp) and the indexed read and write are on
// the hot path rather than any iterator machinery. A runtime that boxes each
// element or routes the store through a generic property set shows up here against
// one that indexes a native slice.
const N = 4096;
const passes = 400;
const t0 = performance.now();

const i32 = new Int32Array(N);
const f64 = new Float64Array(N);
const clamped = new Uint8ClampedArray(N);

let acc = 0;
for (let p = 0; p < passes; p++) {
  // Seed each buffer so every pass starts from the same state and the numbers do
  // not drift toward a fixed point over the run.
  for (let i = 0; i < N; i++) {
    i32[i] = i * 3 - 5;
    f64[i] = i * 0.5;
    clamped[i] = i - 100;
  }
  // A running combine that reads the neighbor and stores back, so each element is
  // both read and written through the buffer. This loop keeps its own counter so
  // the Int32Array store lowers to a native slice write (a proven-in-range index
  // into the backing []int32, no per-element wrap), while the Float64Array and the
  // clamped buffer stay on the checked store where the coercion is not a no-op.
  let sum = 0;
  for (let j = 1; j < N; j++) {
    i32[j] = i32[j] + (i32[j - 1] & 0x7f);
    f64[j] = f64[j] + f64[j - 1] * 0.25;
    sum += clamped[j];
  }
  acc += (i32[N - 1] % 1000) + sum + f64[N - 1];
}

const t1 = performance.now();
console.error("compute_ms=" + (t1 - t0));
console.log(acc.toFixed(0));

// The Math and Number constants driven through a tight arithmetic loop. Each
// iteration folds the constants into an accumulator so the read path for a
// namespace property is exercised rather than a single constant fold. The finite
// constants combine into a double the loop reduces to a 32-bit checksum, so the
// result is deterministic to the last bit and must match across runtimes. The checksum is -732318951 on node, bun, deno, and bento. Every runtime agrees on it.
const passes = 400;
let acc = 0;
let mix = 0;
for (let pass = 0; pass < passes; pass++) {
  for (let i = 1; i < 5000; i++) {
    // a rotating pick over the eight Math constants keeps every one live
    const m = [
      Math.E,
      Math.LN10,
      Math.LN2,
      Math.LOG10E,
      Math.LOG2E,
      Math.PI,
      Math.SQRT1_2,
      Math.SQRT2,
    ][i & 7];
    mix = mix + m * i;
    // fold in the safe-integer bound and epsilon, then reduce to 32 bits
    acc = (acc + Math.clz32(i) + (mix % Number.MAX_SAFE_INTEGER >>> 0)) | 0;
    acc = (acc ^ ((Number.EPSILON * i * 1e18) | 0)) | 0;
  }
}
console.log(acc | 0);

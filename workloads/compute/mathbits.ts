// The bit-exact Math methods in a tight loop: fround (a single-precision round
// trip), clz32 (leading zeros of the ToUint32 coercion), and imul (a 32-bit
// integer multiply that keeps the low half). These are the integer and
// single-precision operations, not the transcendental ones, so the result is
// deterministic to the last bit and the checksum must match across runtimes. This
// isolates the 32-bit coercion path from the rest of a program. The checksum is
// -1303942704 on node, bun, deno, and bento, and every runtime agrees on it.
const passes = 400;
let acc = 0;
for (let pass = 0; pass < passes; pass++) {
  for (let i = 1; i < 5000; i++) {
    acc = Math.imul(acc ^ i, 2654435761); // a 32-bit mix through imul
    acc = (acc + Math.clz32(i)) | 0; // leading-zero count of the index
    const f = Math.fround(i * 1.1); // round to single precision and back
    acc = (acc + Math.clz32(f)) | 0; // clz32 coerces the double to uint32
  }
}
console.log(acc | 0);

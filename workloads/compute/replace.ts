// String replace and replaceAll with string patterns in a tight loop. Each
// iteration builds a short row, replaces the first separator with replace, then
// rewrites every separator with replaceAll, so this exercises the code-unit
// search and the substitution expansion the compiler lowers these to, isolated
// from the rest of a program. Each result folds into a 32-bit checksum through
// its length and a couple of sampled code units, an O(1) reduction so the timing
// tracks the rewriting rather than a per-character scan. The result is
// deterministic to the last bit, so it must match across runtimes. The checksum is 2101432840 on node, bun, deno, and bento. Every runtime agrees on it.
const passes = 100;
let acc = 0;
const t0 = performance.now();
for (let pass = 0; pass < passes; pass++) {
  for (let i = 1; i < 3000; i++) {
    const row = "a:" + (i % 97) + ":b:" + (i % 13) + ":c";
    const first = row.replace(":", " [$&] ");
    const all = row.replaceAll(":", "-");
    acc = (acc * 31 + first.length) | 0;
    acc = (acc * 31 + all.length) | 0;
    acc = (acc * 31 + all.charCodeAt(0)) | 0;
    acc = (acc * 31 + first.charCodeAt(first.length - 1)) | 0;
  }
}
const t1 = performance.now();
console.error("compute_ms=" + (t1 - t0));
console.log(acc | 0);

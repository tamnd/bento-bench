// Template literals in a tight loop: a head, a number substitution, a string
// substitution, and a boolean substitution joined per iteration. This exercises
// the ToString coercion and the string join the compiler lowers a template to,
// isolated from the rest of a program. Each built string folds into a 32-bit
// checksum through its length and a couple of sampled code units, an O(1)
// reduction so the timing tracks template building rather than a per-character
// scan. The result is deterministic to the last bit, so it must match across
// runtimes. The checksum is 1174080576 on node, bun, deno,
// and bento. Every runtime agrees on it.
const passes = 300;
let acc = 0;
const words = ["alpha", "beta", "gamma", "delta"];
for (let pass = 0; pass < passes; pass++) {
  for (let i = 1; i < 3000; i++) {
    const s = `row ${i}: ${words[i & 3]} = ${i * 1.5} (${(i & 1) === 0})`;
    // fold the built string in O(1): its length and the code units at both ends
    acc = (acc * 31 + s.length) | 0;
    acc = (acc * 31 + s.charCodeAt(0)) | 0;
    acc = (acc * 31 + s.charCodeAt(s.length - 1)) | 0;
  }
}
console.log(acc | 0);

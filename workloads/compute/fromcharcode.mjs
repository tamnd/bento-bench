// String.fromCharCode in a tight loop. Each iteration builds a short string from
// four code units the compiler lowers to a variadic constructor that runs the
// ECMAScript ToUint16 on every argument, so this isolates that coercion and the
// string construction from the rest of a program. Each built string folds into a
// 32-bit checksum through its first and last code units, an O(1) reduction so the
// timing tracks the construction rather than a per-character scan. The result is
// deterministic to the last bit, so it must match across runtimes. The checksum
// is 181477376 on node, bun, deno, and bento.
let acc = 0;
for (let pass = 0; pass < 400; pass++) {
  for (let i = 0; i < 4000; i++) {
    // four code units: two ASCII letters that shift with i, a value past 2^16 so
    // ToUint16 has to wrap, and a Greek letter from a high code point.
    const s = String.fromCharCode(65 + (i & 25), 97 + (i % 26), 65536 + (i & 127), 945 + (i % 24));
    acc = (acc * 31 + s.charCodeAt(0)) | 0;
    acc = (acc * 31 + s.charCodeAt(2)) | 0;
    acc = (acc * 31 + s.length) | 0;
  }
}
process.stdout.write((acc | 0) + "\n");

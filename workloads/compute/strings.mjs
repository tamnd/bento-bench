// String building, splitting, and searching. Runtimes differ a lot on how they
// represent and concatenate strings, so this catches rope versus flat tradeoffs
// and the cost of the common split and join and replace path.
// BENCH_SCALE scales both the string length and the outer pass count so CI can
// run a lighter size without editing the workload; unset or 1 is the full size.
const SCALE = Number(process.env.BENCH_SCALE ?? "1");
const words = Math.max(1, Math.round(20000 * SCALE));
const passes = Math.max(1, Math.round(30 * SCALE));

let s = "";
for (let i = 0; i < words; i++) {
  s += "word" + (i % 100) + " ";
}

let count = 0;
for (let pass = 0; pass < passes; pass++) {
  const parts = s.split(" ");
  count += parts.length;
  const joined = parts.join("-");
  count += joined.length - joined.replace(/word/g, "W").length;
}
process.stdout.write(String(count) + "\n");

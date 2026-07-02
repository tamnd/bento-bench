// Primitive coercion in a tight loop: parse a string to a number, format a number
// back to a string, and reduce a truthiness test. This stresses the ToNumber
// grammar, Number::toString, and ToBoolean paths that a runtime hits constantly
// when it crosses the string and number boundary, so a slow formatter or parser
// shows up here rather than hiding behind heavier work.
// BENCH_SCALE scales the outer pass count so CI can run a lighter size without
// editing the workload; unset or 1 is the full size.
const SCALE = Number(process.env.BENCH_SCALE ?? "1");
const passes = Math.max(1, Math.round(400 * SCALE));
let acc = 0;
let truthy = 0;
for (let pass = 0; pass < passes; pass++) {
  for (let i = 0; i < 2000; i++) {
    const n = i * 1.5 - pass;
    // Number::toString, then Number() parses it straight back.
    const s = String(n);
    acc += Number(s);
    // A radix string and a decimal string through the parse grammar.
    acc += Number("0x" + (i & 0xff).toString(16));
    // ToBoolean over a number and a string, folded into a counter.
    if (Boolean(n) && Boolean(s)) truthy++;
  }
}
process.stdout.write(acc.toFixed(0) + " " + truthy + "\n");

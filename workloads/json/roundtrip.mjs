// Build a moderately nested object, stringify it, and parse it back many times.
// This exercises the JSON serializer and parser plus the allocation and garbage
// collection that comes with churning short-lived objects and strings.
function makeRecord(i) {
  return {
    id: i,
    name: "item-" + i,
    active: i % 2 === 0,
    tags: ["a", "b", "c", String(i % 10)],
    meta: { created: i * 1000, score: (i % 7) / 7, nested: { depth: i % 5 } },
  };
}

// BENCH_SCALE scales the record count and the outer pass count so CI can run a
// lighter size without editing the workload; unset or 1 is the full size.
const SCALE = Number(process.env.BENCH_SCALE ?? "1");
const recordCount = Math.max(1, Math.round(2000 * SCALE));
const passes = Math.max(1, Math.round(20 * SCALE));

const records = [];
for (let i = 0; i < recordCount; i++) records.push(makeRecord(i));

let total = 0;
for (let pass = 0; pass < passes; pass++) {
  const text = JSON.stringify(records);
  const back = JSON.parse(text);
  total += back.length + text.length;
}
process.stdout.write(String(total) + "\n");

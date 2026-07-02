// Build a moderately nested object, stringify it, and parse it back many times.
// This exercises the JSON serializer and parser plus the allocation and garbage
// collection that comes with churning short-lived objects and strings.
function makeRecord(i: number) {
  return {
    id: i,
    name: "item-" + i,
    active: i % 2 === 0,
    tags: ["a", "b", "c", String(i % 10)],
    meta: { created: i * 1000, score: (i % 7) / 7, nested: { depth: i % 5 } },
  };
}

const recordCount = 800;
const passes = 10;

const t0 = performance.now();
const records: ReturnType<typeof makeRecord>[] = [];
for (let i = 0; i < recordCount; i++) records.push(makeRecord(i));

let total = 0;
for (let pass = 0; pass < passes; pass++) {
  const text = JSON.stringify(records);
  const back = JSON.parse(text);
  total += back.length + text.length;
}
const t1 = performance.now();
console.error("compute_ms=" + (t1 - t0));
console.log(total);

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

const recordCount = 2000;
const passes = 20;

const records: ReturnType<typeof makeRecord>[] = [];
for (let i = 0; i < recordCount; i++) records.push(makeRecord(i));

let total = 0;
for (let pass = 0; pass < passes; pass++) {
  const text = JSON.stringify(records);
  const back = JSON.parse(text);
  total += back.length + text.length;
}
console.log(total);

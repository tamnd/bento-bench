// String building, splitting, and searching. Runtimes differ a lot on how they
// represent and concatenate strings, so this catches rope versus flat tradeoffs
// and the cost of the common split and join and replace path.
const words = 8000;
const passes = 12;

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
console.log(count);

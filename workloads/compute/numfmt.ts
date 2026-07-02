// Number::toString across the full range of magnitudes: ordinary integers and
// fractions, the values that cross into exponential notation, and the small
// magnitudes that stay decimal. The shortest-round-trip digit search and the
// exponent placement are the expensive part, so this isolates the formatter from
// the rest of a program.
const passes = 300;
let total = 0;
for (let pass = 0; pass < passes; pass++) {
  for (let i = 1; i < 3000; i++) {
    const x = i * 1.000001;
    total += String(x).length;
    total += String(x * 1e18).length;   // large, some cross 1e21 into exponential
    total += String(x / 1e12).length;   // small, some cross 1e-7 into exponential
    total += String(-x).length;
  }
}
console.log(total);

// Write a batch of small files, read them all back, then delete them. This is a
// syscall-bound workload that leans on the runtime's node:fs layer rather than
// its JavaScript engine, so it shows the cost of the host bridge.
import { mkdtempSync, writeFileSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

// BENCH_SCALE scales the number of files per pass so CI can run a lighter size
// without editing the workload; unset or 1 is the full size.
const SCALE = Number(process.env.BENCH_SCALE ?? "1");
const dir = mkdtempSync(join(tmpdir(), "bento-bench-"));
const count = Math.max(1, Math.round(200 * SCALE));
const payload = "x".repeat(512);

let bytes = 0;
for (let pass = 0; pass < 3; pass++) {
  for (let i = 0; i < count; i++) {
    writeFileSync(join(dir, "f" + i + ".txt"), payload);
  }
  for (let i = 0; i < count; i++) {
    bytes += readFileSync(join(dir, "f" + i + ".txt"), "utf8").length;
  }
}
rmSync(dir, { recursive: true, force: true });
process.stdout.write(String(bytes) + "\n");

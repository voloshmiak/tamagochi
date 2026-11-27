import { readFileSync } from "node:fs";
import { join } from "node:path";
import { gzipSync } from "node:zlib";

const LIMIT = 300 * 1024;
const dist = new URL("../dist/", import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, "$1");
const html = readFileSync(join(dist, "index.html"), "utf8");

const initial = new Set();
for (const m of html.matchAll(/<script[^>]+src="([^"]+\.js)"/g)) initial.add(m[1]);
for (const m of html.matchAll(/<link[^>]+rel="modulepreload"[^>]+href="([^"]+\.js)"/g)) initial.add(m[1]);

let total = 0;
for (const file of initial) {
  const size = gzipSync(readFileSync(join(dist, file.replace(/^\//, "")))).length;
  total += size;
  console.log(`${(size / 1024).toFixed(1).padStart(8)} KB  ${file}`);
}
console.log(`${(total / 1024).toFixed(1).padStart(8)} KB  initial JS (gzip), limit ${LIMIT / 1024} KB`);
if (total > LIMIT) {
  console.error("Initial JavaScript is over the budget (NFR-PERF-02)");
  process.exit(1);
}

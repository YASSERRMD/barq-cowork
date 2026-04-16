import fs from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { build } from "esbuild";

const rootDir = path.dirname(fileURLToPath(import.meta.url));
const outDir = path.resolve(rootDir, "../internal/tools/assets");

await fs.mkdir(outDir, { recursive: true });

await build({
  entryPoints: [path.resolve(rootDir, "src/render-pptx.ts")],
  bundle: true,
  platform: "node",
  format: "cjs",
  target: "node20",
  loader: { ".svg": "text" },
  outfile: path.resolve(outDir, "pptx-renderer.cjs"),
});
